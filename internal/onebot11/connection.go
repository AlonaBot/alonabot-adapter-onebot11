package onebot11

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/AlonaBot/AlonaBot/sdk"
	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	"github.com/coder/websocket"
)

const forwardMode = "forward"

type connectionConfig struct {
	wsURL       string
	accessToken string
	safeURL     string
	redact      *strings.Replacer
}

func parseConfig(raw []byte) (connectionConfig, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return connectionConfig{}, invalidField("config")
	}
	stringField := func(name string) (string, error) {
		value, exists := fields[name]
		if !exists {
			return "", nil
		}
		var text *string
		if err := json.Unmarshal(value, &text); err != nil || text == nil {
			return "", invalidField(name)
		}
		return *text, nil
	}
	endpoint, err := stringField("ws_url")
	if err != nil {
		return connectionConfig{}, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "ws" && parsed.Scheme != "wss") || parsed.Fragment != "" {
		return connectionConfig{}, invalidField("ws_url")
	}
	mode, err := stringField("mode")
	if err != nil {
		return connectionConfig{}, err
	}
	if _, exists := fields["mode"]; exists && mode != forwardMode {
		return connectionConfig{}, invalidField("mode")
	}
	token, err := stringField("access_token")
	if err != nil {
		return connectionConfig{}, err
	}

	// URL 的用户信息和查询字符串只供连接使用，诊断路径提前生成脱敏表示。
	clean := *parsed
	clean.User, clean.RawQuery, clean.ForceQuery = nil, "", false
	var replacements []string
	for _, secret := range []string{token, parsed.User.String(), parsed.RawQuery} {
		if secret != "" {
			quoted := strconv.Quote(secret)
			replacements = append(replacements, secret, "<redacted>", url.PathEscape(secret), "<redacted>", url.QueryEscape(secret), "<redacted>", quoted[1:len(quoted)-1], "<redacted>")
		}
	}
	redact := strings.NewReplacer(replacements...)
	safeURL := redact.Replace(clean.String())
	httpURL := *parsed
	if parsed.Scheme == "ws" {
		httpURL.Scheme = "http"
	} else {
		httpURL.Scheme = "https"
	}
	clean.Scheme = httpURL.Scheme
	replacements = append([]string{endpoint, safeURL, httpURL.String(), redact.Replace(clean.String())}, replacements...)
	return connectionConfig{wsURL: endpoint, accessToken: token, safeURL: safeURL, redact: strings.NewReplacer(replacements...)}, nil
}

func invalidField(name string) error {
	return &sdk.Error{Code: errcode.InvalidArgument, Message: name}
}

func (c *connectionConfig) dial(ctx context.Context) (*websocket.Conn, error) {
	headers := make(http.Header)
	if c.accessToken != "" {
		headers.Set("Authorization", "Bearer "+c.accessToken)
	}
	// 凭证属于配置的实现端，保持在这次握手内，避免随重定向传给其他端点。
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	conn, response, err := websocket.Dial(ctx, c.wsURL, &websocket.DialOptions{HTTPHeader: headers, HTTPClient: client})
	if err != nil {
		code := errcode.Unavailable
		if response != nil && (response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden) {
			code = errcode.PermissionDenied
		}
		return nil, &sdk.Error{Code: code, Message: c.safeError(err).Error()}
	}
	conn.SetReadLimit(-1) // 上报大小由平台决定，此处只限制队列条数。
	return conn, nil
}

// safeError 保留取消等错误判断，文本先移除 URL 凭证和 access_token。
func (c *connectionConfig) safeError(err error) error {
	message := err.Error()
	var requestError *url.Error
	if errors.As(err, &requestError) {
		// net/http 会先把密码改成 ***；直接替换其带引号的 URL，避免依赖密码表示。
		message = strings.ReplaceAll(message, strconv.Quote(requestError.URL), strconv.Quote(c.safeURL))
	}
	return &sanitizedError{message: c.redact.Replace(message), cause: err}
}

type sanitizedError struct {
	message string
	cause   error
}

func (e *sanitizedError) Error() string        { return e.message }
func (e *sanitizedError) Is(target error) bool { return errors.Is(e.cause, target) }

type apiResponse struct {
	Status  string          `json:"status"`
	Retcode *int            `json:"retcode"`
	Data    json.RawMessage `json:"data"`
}

type apiResult struct {
	response apiResponse
	err      error
}

func (i *instance) call(ctx context.Context, action string, params any, result any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// 1. 在写出请求前登记 echo，响应先到也能保留在独立的缓冲 channel。
	i.mu.Lock()
	conn := i.conn
	if conn == nil || i.receiveCtx.Err() != nil {
		i.mu.Unlock()
		return &sdk.Error{Code: errcode.Unavailable, Message: "onebot connection is closed"}
	}
	i.nextEcho++
	echo := strconv.FormatUint(i.nextEcho, 10)
	response := make(chan apiResult, 1)
	i.pending[echo] = response
	i.mu.Unlock()
	defer func() {
		i.mu.Lock()
		delete(i.pending, echo)
		i.mu.Unlock()
	}()

	request := struct {
		Action string `json:"action"`
		Params any    `json:"params,omitempty"`
		Echo   string `json:"echo"`
	}{action, params, echo}
	raw, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode onebot request: %w", i.config.safeError(err))
	}
	if err := conn.Write(ctx, websocket.MessageText, raw); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &sdk.Error{Code: errcode.Unavailable, Message: i.config.safeError(err).Error()}
	}

	// 2. ctx 仅限制本次调用；等待应答期间不会暂停读取其他响应或上报。
	select {
	case <-ctx.Done():
		return ctx.Err()
	case received := <-response:
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if received.err != nil {
			return received.err
		}
		if received.response.Retcode == nil {
			return &sdk.Error{Code: errcode.Unavailable, Message: "onebot response has no retcode"}
		}
		if received.response.Status != "ok" || *received.response.Retcode != 0 {
			message := fmt.Sprintf("onebot API %s returned status %q retcode %d", action, received.response.Status, *received.response.Retcode)
			return &sdk.Error{Code: errcode.Unavailable, Message: i.config.redact.Replace(message)}
		}
		if err := json.Unmarshal(received.response.Data, result); err != nil {
			return fmt.Errorf("decode onebot %s data: %w", action, i.config.safeError(err))
		}
		return nil
	}
}

func (i *instance) readConnection(source *receiveConnection) (readError error) {
	conn := source.conn
	defer conn.CloseNow()
	defer func() { i.disconnect(conn, readError) }()
	for {
		_, raw, err := conn.Read(i.receiveCtx)
		if err != nil {
			if i.receiveCtx.Err() == nil {
				slog.Warn("onebot websocket read failed", "adapter_instance_id", i.id, "error", i.config.safeError(err))
			}
			return err
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			slog.Warn("skip malformed onebot frame", "adapter_instance_id", i.id, "error", i.config.safeError(err))
			continue
		}
		if fields == nil {
			slog.Warn("skip malformed onebot frame", "adapter_instance_id", i.id, "error", "onebot frame must be a JSON object")
			continue
		}
		if echo, exists := fields["echo"]; exists {
			i.respond(echo, raw)
			continue
		}
		var postType string
		if err := json.Unmarshal(fields["post_type"], &postType); err != nil {
			slog.Warn("skip malformed onebot post_type", "adapter_instance_id", i.id, "error", i.config.safeError(err))
			continue
		}
		if postType == "meta_event" {
			var meta struct {
				Event   string `json:"meta_event_type"`
				SubType string `json:"sub_type"`
			}
			if err := json.Unmarshal(raw, &meta); err != nil {
				slog.Warn("skip malformed onebot meta_event", "adapter_instance_id", i.id, "error", i.config.safeError(err))
				continue
			}
			if meta.Event == "lifecycle" && meta.SubType == "connect" {
				slog.Debug("onebot lifecycle connected", "adapter_instance_id", i.id)
			}
			continue
		}
		select { // 上报排队不得阻塞读取 API 响应；队列满时丢弃新上报。
		case i.events <- receivedEvent{raw: raw, source: source}:
		default:
			i.dropped++
			slog.Warn("onebot event queue full", "adapter_instance_id", i.id, "dropped", i.dropped)
		}
	}
}

func (i *instance) respond(rawEcho, raw json.RawMessage) {
	var echo string
	if err := json.Unmarshal(rawEcho, &echo); err != nil {
		slog.Warn("skip invalid onebot response echo", "adapter_instance_id", i.id, "error", i.config.safeError(err))
		return
	}
	var result apiResult
	if err := json.Unmarshal(raw, &result.response); err != nil {
		result.err = &sdk.Error{Code: errcode.Unavailable, Message: i.config.safeError(err).Error()}
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if response, exists := i.pending[echo]; exists {
		delete(i.pending, echo)
		response <- result // 单次发送到容量为 1 的 channel；等待者超时也不会阻塞读循环。
	}
}

func (i *instance) disconnect(conn *websocket.Conn, cause error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.conn != conn {
		return
	}
	i.conn = nil
	for echo, response := range i.pending {
		response <- apiResult{err: &sdk.Error{Code: errcode.Unavailable, Message: i.config.safeError(cause).Error()}}
		delete(i.pending, echo)
	}
}
