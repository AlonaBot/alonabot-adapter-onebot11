// Package onebot11 实现官方 OneBot 11 Adapter 的正向 WebSocket 连接与事件接收。
//
// [New] 创建供 sdk.RunAdapter 使用的 Adapter。每个实例独占连接、读取循环和
// 逐条交付循环；API 请求通过实例内递增的 echo 配对，生命周期由 Stop 管理。
package onebot11

import (
	"context"
	"log/slog"
	"strconv"
	"sync"

	"github.com/AlonaBot/AlonaBot/adapters/onebot11/perm"
	"github.com/AlonaBot/AlonaBot/sdk"
	"github.com/AlonaBot/AlonaBot/sdk/adapter"
	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	msg "github.com/AlonaBot/AlonaBot/sdk/message"
	"github.com/coder/websocket"
)

// AdapterID 标识 OneBot 11 平台。
const AdapterID = "onebot11"

// eventQueueCapacity 遵循 CONTRACTS §8 的实例接收队列容量。
const eventQueueCapacity = 256

type onebotAdapter struct{}

// New 创建 OneBot 11 Adapter，账号身份在 Start 中通过 get_login_info 验证。
func New() adapter.Adapter { return &onebotAdapter{} }

func receiveCapabilities() []msg.Capability {
	return []msg.Capability{
		{Name: "message.created", Version: 1, Supported: true},
		{Name: "message.deleted", Version: 1, Supported: true},
		{Name: "member.joined", Version: 1, Supported: true},
		{Name: "member.left", Version: 1, Supported: true},
	}
}

// Descriptor 声明接收能力和平台事实权限。
func (*onebotAdapter) Descriptor() *adapter.Descriptor {
	return &adapter.Descriptor{
		AdapterID: AdapterID, Capabilities: receiveCapabilities(),
		Permissions: []adapter.PermissionDeclaration{
			{Name: perm.GroupOwner, Implies: []string{perm.GroupAdmin}, LabelKey: "permission.group_owner"},
			{Name: perm.GroupAdmin, Implies: []string{"chat_admin"}, LabelKey: "permission.group_admin"},
			{Name: perm.PrivateFriend, LabelKey: "permission.private_friend"},
		},
	}
}

// Start 验证配置、连接和账号，成功后将读取与交付循环交给 Stop 管理。
func (*onebotAdapter) Start(ctx context.Context, request adapter.StartRequest, core adapter.Core) (adapter.Instance, error) {
	// 1. 配置原文含凭证，校验错误只返回字段名。
	config, err := parseConfig(request.ConfigJSON)
	if err != nil {
		return nil, err
	}
	conn, err := config.dial(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 读取循环先就绪，才能在同一连接上等待登录 API 的 echo 响应。
	lifetime, cancel := context.WithCancel(context.WithoutCancel(ctx))
	receiveCtx, cancelReceive := context.WithCancel(lifetime)
	i := &instance{
		config: config, core: core, id: request.InstanceID,
		conn: conn, pending: make(map[string]chan apiResult),
		ctx: lifetime, cancel: cancel, done: make(chan struct{}),
		receiveCtx: receiveCtx, cancelReceive: cancelReceive,
		deliveryDone: make(chan struct{}), failures: make(chan *errcode.Error, 1),
		events: make(chan receivedEvent, eventQueueCapacity),
	}
	source := i.startRead(conn)
	account, err := i.login(ctx)
	if err != nil {
		cancel()
		conn.CloseNow()
		<-source.readDone // Start 失败不启动交付与重连，先收回唯一的读取循环。
		return nil, &sdk.Error{Code: errcode.Unavailable, Message: config.safeError(err).Error()}
	}
	i.account = account
	source.valid = true
	close(source.checked) // 先发布账号复核结果，再让交付循环消费该连接的上报。
	// 3. 登录成功后才启动生命周期管理，Start 失败始终不重试。
	go i.deliverEvents()
	go i.maintainConnection(source)
	slog.Debug("connected onebot websocket", "adapter_instance_id", i.id, "url", config.safeURL)
	return i, nil
}

// instance 由 mu 保护连接、echo 序号和在途调用；account 在 Start 返回前写入，之后只读。
// 读取循环唯一写入 events；交付循环按连接的账号复核结果消费上报。
// receiveCtx 控制平台读写与交付；致命故障先结束这些操作，再用 ctx 报告故障，Stop 可取消两者。
type instance struct {
	config        connectionConfig
	core          adapter.Core
	id            string
	account       adapter.Account
	ctx           context.Context
	cancel        context.CancelFunc
	receiveCtx    context.Context
	cancelReceive context.CancelFunc
	done          chan struct{} // 生命周期循环回收读与交付循环并完成故障报告后关闭。
	deliveryDone  chan struct{}
	failures      chan *errcode.Error // 容量为 1，保留首个致命故障供生命周期循环报告。
	events        chan receivedEvent
	dropped       uint64 // 仅由读取循环累加。
	ignored       uint64 // 仅由交付循环累加。

	mu       sync.Mutex
	conn     *websocket.Conn
	nextEcho uint64
	pending  map[string]chan apiResult
}

// Account 返回 get_login_info 验证过的账号身份。
func (i *instance) Account() adapter.Account { return i.account }

// Capabilities 返回实例的接收能力。
func (*instance) Capabilities() []msg.Capability { return receiveCapabilities() }

// Send 返回 UNSUPPORTED，且不会发起平台请求。
func (*instance) Send(context.Context, msg.Session, msg.Chain) (*msg.SendResult, *errcode.Error) {
	return nil, &errcode.Error{Code: errcode.Unsupported, Message: "onebot sending is unsupported"}
}

// Stop 取消退避、握手、读取和交付，等待全部循环与故障报告结束后返回。
func (i *instance) Stop(context.Context) error {
	i.cancel()
	i.stopReceive()
	<-i.done
	return nil
}

func (i *instance) stopReceive() {
	i.cancelReceive()
	i.mu.Lock()
	conn := i.conn
	i.mu.Unlock()
	if conn != nil {
		conn.CloseNow()
	}
}

func (i *instance) login(ctx context.Context) (adapter.Account, error) {
	var login struct {
		UserID   *int64  `json:"user_id"`
		Nickname *string `json:"nickname"`
	}
	if err := i.call(ctx, "get_login_info", nil, &login); err != nil {
		return adapter.Account{}, err
	}
	if login.UserID == nil || *login.UserID <= 0 || login.Nickname == nil {
		return adapter.Account{}, &sdk.Error{Code: errcode.Unavailable, Message: "invalid get_login_info data"}
	}
	return adapter.Account{ID: strconv.FormatInt(*login.UserID, 10), DisplayName: *login.Nickname}, nil
}
