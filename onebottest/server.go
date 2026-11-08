// Package onebottest 提供本地 OneBot 11 正向 WebSocket 模拟服务。
//
// [NewServer] 创建由测试拥有的 [Server]；测试通过 Push 注入上报，
// 通过 Calls 检查 API 请求，通过 Respond 控制响应。
package onebottest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/coder/websocket"
)

const unknownActionRetcode = 1404

// Call 记录一次 OneBot API 请求。
type Call struct {
	Action string
	Params map[string]any
}

// Server 保存登录身份、待推送上报与 API 调用记录。
// mu 保护所有可变状态；每条连接的推送循环拥有事件写入顺序，API 响应可同时写入。
type Server struct {
	t           testing.TB
	http        *httptest.Server
	token       string
	mu          sync.Mutex
	userID      int64
	nickname    string
	conn        *websocket.Conn
	connections int
	pending     []json.RawMessage
	messages    map[string]json.RawMessage
	calls       []Call
	responses   map[string]func(map[string]any) (string, int, any)
	changed     chan struct{} // 入队时关闭并替换，广播唤醒推送循环。
	closed      bool
	handlers    sync.WaitGroup
}

// NewServer 在 / 提供通用端点；accessToken 非空时校验 Bearer，并在测试结束时关闭连接和服务。
func NewServer(t testing.TB, accessToken string) *Server {
	t.Helper()
	s := &Server{
		t: t, token: accessToken, userID: 1, nickname: "test_bot",
		messages:  make(map[string]json.RawMessage),
		responses: make(map[string]func(map[string]any) (string, int, any)),
		changed:   make(chan struct{}, 1),
	}
	s.http = httptest.NewServer(http.HandlerFunc(s.serveHTTP))
	t.Cleanup(func() {
		s.mu.Lock()
		s.closed = true
		conn := s.conn
		s.mu.Unlock()
		if conn != nil {
			conn.CloseNow()
		}
		s.http.Close()
		s.handlers.Wait() // httptest 无法等待已升级的连接，显式等待其读写循环。
	})
	return s
}

// URL 返回可直接作为 ws_url 的地址。
func (s *Server) URL() string { return "ws" + strings.TrimPrefix(s.http.URL, "http") }

// SetLogin 设置 get_login_info 的账号和昵称。
func (s *Server) SetLogin(userID int64, nickname string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userID, s.nickname = userID, nickname
}

// Push 复制并按顺序排队上报，记录 message 的 sender 供 get_msg 查询；非法 JSON 使测试失败。
func (s *Server) Push(events ...json.RawMessage) {
	s.t.Helper()
	pending := make([]json.RawMessage, 0, len(events))
	messages := make(map[string]json.RawMessage)
	for _, event := range events {
		var fields struct {
			PostType  string          `json:"post_type"`
			MessageID json.Number     `json:"message_id"`
			Sender    json.RawMessage `json:"sender"`
		}
		if err := json.Unmarshal(event, &fields); err != nil {
			s.t.Fatalf("decode event: %v", err)
		}
		pending = append(pending, slices.Clone(event))
		if fields.PostType == "message" {
			messages[fields.MessageID.String()] = slices.Clone(fields.Sender)
		}
	}
	s.mu.Lock()
	s.pending = append(s.pending, pending...)
	for id, sender := range messages {
		s.messages[id] = sender
	}
	s.mu.Unlock()
	s.notify()
}

// Disconnect 立即关闭当前连接以模拟断线。
func (s *Server) Disconnect() {
	s.mu.Lock()
	conn := s.conn
	s.conn = nil // 先撤下连接，避免旧推送循环取走随后排队的离线上报。
	s.mu.Unlock()
	if conn != nil {
		conn.CloseNow()
	}
}

// Connections 返回累计接受的 WebSocket 连接数。
func (s *Server) Connections() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connections
}

// Calls 返回按接收顺序记录的请求深副本。
func (s *Server) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Call, len(s.calls))
	for i, call := range s.calls {
		result[i] = Call{Action: call.Action, Params: cloneParams(call.Params)}
	}
	return result
}

// Respond 替换 action 的响应；fn 返回空 status 时保持连接且跳过响应。
// fn 在锁外执行，并应自行结束，以便测试清理连接。
func (s *Server) Respond(action string, fn func(params map[string]any) (status string, retcode int, data any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[action] = fn
}

func (s *Server) notify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	close(s.changed) // 广播唤醒旧连接和新连接，避免交接时通知被旧循环取走。
	s.changed = make(chan struct{})
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. 在升级连接前完成端点和凭证检查。
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if s.token != "" && r.Header.Get("Authorization") != "Bearer "+s.token {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return // Accept 已写入 HTTP 错误。
	}
	conn.SetReadLimit(-1) // 模拟服务保留完整请求，不额外限制平台消息大小。
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		conn.CloseNow()
		return
	}
	previous := s.conn
	s.conn = conn
	s.connections++
	s.handlers.Add(1)
	s.mu.Unlock()
	if previous != nil {
		previous.CloseNow()
	}

	// 2. 读 API 与推送上报独立运行，推送堵塞时仍可处理请求。
	ctx, cancel := context.WithCancel(r.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.pushLoop(ctx, conn)
	}()
	s.notify()
	defer func() {
		cancel()
		conn.CloseNow()
		<-done
		s.mu.Lock()
		if s.conn == conn {
			s.conn = nil
		}
		s.mu.Unlock()
		s.handlers.Done()
	}()
	for {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			return // 断线由测试主动控制，结束该连接的循环。
		}
		if !s.respond(ctx, conn, raw) {
			return
		}
	}
}

func (s *Server) pushLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		s.mu.Lock()
		if s.conn != conn {
			s.mu.Unlock()
			return
		}
		if len(s.pending) != 0 {
			raw := s.pending[0]
			s.pending[0] = nil
			s.pending = s.pending[1:]
			s.mu.Unlock()
			if err := conn.Write(ctx, websocket.MessageText, raw); err != nil {
				return // 写失败已关闭连接，读循环负责收尾；上报不重放。
			}
			continue
		}
		changed := s.changed
		s.mu.Unlock()
		select { // 等待新上报，同时允许清理立即取消。
		case <-ctx.Done():
			return
		case <-changed:
		}
	}
}

func (s *Server) respond(ctx context.Context, conn *websocket.Conn, raw []byte) bool {
	// 1. 保留参数数字原文和 echo 类型，记录先于自定义响应执行。
	var request struct {
		Action string          `json:"action"`
		Params map[string]any  `json:"params"`
		Echo   json.RawMessage `json:"echo,omitempty"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&request); err != nil {
		s.t.Errorf("decode API request: %v", err)
		return false
	}
	s.mu.Lock()
	s.calls = append(s.calls, Call{Action: request.Action, Params: cloneParams(request.Params)})
	fn := s.responses[request.Action]
	status, retcode, data := s.defaultResponse(request.Action, request.Params)
	s.mu.Unlock()
	if fn != nil {
		status, retcode, data = fn(request.Params)
	}
	if status == "" {
		return true
	}

	// 2. 原样回送 echo，让客户端按请求标识配对。
	response := struct {
		Status  string          `json:"status"`
		Retcode int             `json:"retcode"`
		Data    any             `json:"data"`
		Echo    json.RawMessage `json:"echo,omitempty"`
	}{status, retcode, data, request.Echo}
	encoded, err := json.Marshal(response)
	if err != nil {
		s.t.Errorf("encode API response: %v", err)
		return false
	}
	return conn.Write(ctx, websocket.MessageText, encoded) == nil
}

// defaultResponse 由 respond 持有 mu 时调用。
func (s *Server) defaultResponse(action string, params map[string]any) (string, int, any) {
	switch action {
	case "get_login_info":
		return "ok", 0, map[string]any{"user_id": s.userID, "nickname": s.nickname}
	case "get_msg":
		id, ok := params["message_id"].(json.Number)
		if ok {
			if sender, found := s.messages[id.String()]; found {
				return "ok", 0, map[string]any{"sender": sender}
			}
		}
	}
	return "failed", unknownActionRetcode, nil
}

func cloneParams(params map[string]any) map[string]any {
	result := make(map[string]any, len(params))
	for key, value := range params {
		result[key] = cloneValue(value)
	}
	return result
}

func cloneValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		return cloneParams(value)
	case []any:
		result := make([]any, len(value))
		for i, child := range value {
			result[i] = cloneValue(child)
		}
		return result
	default:
		return value
	}
}
