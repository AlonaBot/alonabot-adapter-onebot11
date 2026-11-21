package onebot11

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coder/websocket"
)

func (a *onebotAdapter) reverseHTTP(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/ws/")
	a.mu.Lock()
	i := a.instances[id]
	a.mu.Unlock()
	if i == nil || i.receiveCtx.Err() != nil {
		http.NotFound(w, r)
		return
	}
	provided := r.Header.Get("Authorization")
	if provided == "" {
		provided = r.URL.Query().Get("access_token")
	} else {
		if !strings.HasPrefix(provided, "Bearer ") {
			http.Error(w, "unauthorized", 401)
			return
		}
		provided = strings.TrimPrefix(provided, "Bearer ")
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(i.config.accessToken)) != 1 {
		http.Error(w, "unauthorized", 401)
		return
	}

	if r.Header.Get("X-Self-ID") != i.account.ID || r.Header.Get("X-Client-Role") != "Universal" {
		http.Error(w, "forbidden", 403)
		return
	}
	// 握手和连接占位共用锁，避免两个并发连入都通过空闲检查。
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.receiveCtx.Err() != nil {
		http.NotFound(w, r)
		return
	}
	if i.conn != nil {
		http.Error(w, "conflict", 409)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(-1)
	i.conn = conn
	i.incoming <- conn
}

func (i *instance) maintainReverse() {
	defer close(i.done)
	defer i.cancel()
	defer func() {
		i.stopReceive()
		<-i.deliveryDone
		select {
		case failure := <-i.failures:
			i.reportFailure(failure.Code, failure.Message)
		default:
		}
	}()
	for {
		var conn *websocket.Conn
		select { // Stop 取消等待；已交接的连接由 stopReceive 回收。
		case <-i.receiveCtx.Done():
			return
		case conn = <-i.incoming:
		}
		source := i.startRead(conn)
		account, err := i.login(i.receiveCtx)
		valid := err == nil && account.ID == i.account.ID
		if valid {
			i.mu.Lock()
			i.ready = true
			i.displayName = account.DisplayName
			i.mu.Unlock()
		} else {
			if err == nil {
				err = fmt.Errorf("onebot account mismatch: expected %s, received %s", i.account.ID, account.ID)
			}
			slog.Warn("onebot reverse login rejected", "adapter_instance_id", i.id, "error", err)
			conn.CloseNow()
		}
		source.valid = valid
		close(source.checked)
		<-source.readDone
	}
}
