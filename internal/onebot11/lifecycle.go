package onebot11

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/AlonaBot/AlonaBot/sdk"
	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	"github.com/coder/websocket"
)

const (
	initialReconnectBackoff = time.Second
	maxReconnectBackoff     = 60 * time.Second
)

// receiveConnection 绑定读取结束与账号复核门禁，防止重连登录前的上报提前交付。
type receiveConnection struct {
	conn     *websocket.Conn
	readDone chan struct{}
	checked  chan struct{} // 登录完成后关闭；交付循环可在关闭后读取 valid。
	valid    bool          // 只由登录方在关闭 checked 前写入。
}

// startRead 为当前连接启动唯一读循环；调用者等待 readDone 后才能建立下一条连接。
func (i *instance) startRead(conn *websocket.Conn) *receiveConnection {
	source := &receiveConnection{conn: conn, readDone: make(chan struct{}), checked: make(chan struct{})}
	go func() {
		defer close(source.readDone)
		i.readConnection(source)
	}()
	return source
}

func (i *instance) maintainConnection(source *receiveConnection) {
	defer close(i.done)
	defer i.cancel()
	defer func() {
		i.stopReceive()
		<-source.readDone
		<-i.deliveryDone // 先结束全部交付，再向 Core 报告首个致命故障。
		select {
		case failure := <-i.failures:
			i.reportFailure(failure.Code, failure.Message)
		default:
		}
	}()
	<-source.readDone // 初始连接由 Start 建立，先等其读取与在途调用收尾。
	backoff := initialReconnectBackoff
	for waitReconnect(i.receiveCtx, backoff) {
		// 1. 只有这个循环创建新连接；Stop 取消同一个 ctx，覆盖退避和握手。
		conn, err := i.config.dial(i.receiveCtx)
		if i.receiveCtx.Err() != nil {
			if conn != nil {
				conn.CloseNow()
			}
			return
		}
		if err != nil {
			var public *sdk.Error
			if errors.As(err, &public) && public.Code == errcode.PermissionDenied {
				i.fail(errcode.PermissionDenied, "onebot implementation rejected the access token")
				return
			}
			slog.Warn("onebot websocket reconnect failed", "adapter_instance_id", i.id, "error", i.config.safeError(err))
			backoff = nextReconnectBackoff(backoff)
			continue
		}

		// 2. 发布连接后立即恢复读取，让登录 API 也通过同一条 echo 通道配对。
		i.mu.Lock()
		if i.receiveCtx.Err() != nil {
			i.mu.Unlock()
			conn.CloseNow()
			return
		}
		i.conn = conn
		i.mu.Unlock()
		source = i.startRead(conn)
		account, err := i.login(i.receiveCtx)
		if err != nil || i.receiveCtx.Err() != nil || account.ID != i.account.ID {
			conn.CloseNow()
			<-source.readDone
			close(source.checked) // 失败连接的 valid 保持 false，其排队上报直接丢弃。
			if i.receiveCtx.Err() != nil {
				return
			}
			if err != nil {
				slog.Warn("onebot reconnect login failed", "adapter_instance_id", i.id, "error", i.config.safeError(err))
				backoff = nextReconnectBackoff(backoff)
				continue
			}
			i.fail(errcode.FailedPrecondition, "onebot account changed")
			return
		}

		// 3. 身份复核成功后恢复初始退避；不重放旧连接的任何请求。
		source.valid = true
		close(source.checked)
		slog.Debug("reconnected onebot websocket", "adapter_instance_id", i.id, "url", i.config.safeURL)
		backoff = initialReconnectBackoff
		<-source.readDone
	}
}

func (i *instance) fail(code errcode.Code, message string) {
	select { // 队列保留首个故障，接收与交付同时发现问题时也只报告一次。
	case i.failures <- &errcode.Error{Code: code, Message: message}:
	default:
	}
	i.stopReceive()
}

func (i *instance) reportFailure(code errcode.Code, message string) {
	if err := i.core.ReportFailure(i.ctx, &errcode.Error{Code: code, Message: message}); err != nil {
		slog.Error("report onebot failure", "adapter_instance_id", i.id, "error", i.config.safeError(err))
	}
}

func waitReconnect(ctx context.Context, backoff time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select { // Stop 必须能立即结束退避，无需等待计时器。
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}

func nextReconnectBackoff(current time.Duration) time.Duration {
	return min(current*2, maxReconnectBackoff)
}
