package onebot11_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlonaBot/AlonaBot/adapters/onebot11"
	"github.com/AlonaBot/AlonaBot/adapters/onebot11/onebottest"
	"github.com/AlonaBot/AlonaBot/adapters/onebot11/perm"
	pb "github.com/AlonaBot/AlonaBot/contracts/pb/alonabotv1"
	"github.com/AlonaBot/AlonaBot/sdk"
	"github.com/AlonaBot/AlonaBot/sdk/adapter"
	"google.golang.org/protobuf/proto"
)

type recordingCore struct {
	adapter.Core
	t             *testing.T
	mu            sync.Mutex
	events        []*pb.Event
	delivered     chan struct{}
	failures      chan *pb.ErrorInfo
	deliveryError error
}

func newRecordingCore(t *testing.T) *recordingCore {
	return &recordingCore{t: t, delivered: make(chan struct{}, 16), failures: make(chan *pb.ErrorInfo, 2)}
}

func (c *recordingCore) Deliver(_ context.Context, event *pb.Event, cursor []byte) (bool, error) {
	if cursor != nil {
		c.t.Errorf("cursor = %v, want nil", cursor)
	}
	c.mu.Lock()
	c.events = append(c.events, event)
	count := len(c.events)
	c.mu.Unlock()
	c.delivered <- struct{}{}
	return count != 2, c.deliveryError
}

func (c *recordingCore) ReportFailure(_ context.Context, failure *pb.ErrorInfo) error {
	c.failures <- failure
	return nil
}

func (c *recordingCore) snapshot() []*pb.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*pb.Event(nil), c.events...)
}

func startRequest(t *testing.T, server *onebottest.Server, token string) adapter.StartRequest {
	t.Helper()
	config, err := json.Marshal(map[string]string{"ws_url": server.URL(), "access_token": token})
	if err != nil {
		t.Fatal(err)
	}
	return adapter.StartRequest{InstanceID: "integration", ConfigJSON: config}
}

func startInstance(t *testing.T, ctx context.Context, server *onebottest.Server, token string, core *recordingCore) adapter.Instance {
	t.Helper()
	running, err := onebot11.New().Start(ctx, startRequest(t, server, token), core)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := running.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return running
}

func waitDelivery(t *testing.T, ctx context.Context, core *recordingCore) {
	t.Helper()
	select {
	case <-core.delivered:
	case failure := <-core.failures:
		t.Fatalf("unexpected failure: %v", failure)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func waitConnections(t *testing.T, ctx context.Context, server *onebottest.Server, count int) {
	t.Helper()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for server.Connections() < count {
		select {
		case <-tick.C:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if got := server.Connections(); got != count {
		t.Fatalf("connections = %d, want %d", got, count)
	}
}

func TestReceiveLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server := onebottest.NewServer(t, "integration-secret-token")
	server.SetLogin(42, "Alona")
	core := newRecordingCore(t)
	running := startInstance(t, ctx, server, "integration-secret-token", core)
	if got := running.Account(); got != (adapter.Account{ID: "42", DisplayName: "Alona"}) {
		t.Fatalf("account = %+v", got)
	}
	raw := []json.RawMessage{
		json.RawMessage(`{"time":123,"self_id":42,"post_type":"message","message_type":"private","sub_type":"friend","message_id":1,"user_id":7,"sender":{"nickname":"Friend"},"message":[{"type":"text","data":{"text":"hello"}}]}`),
		json.RawMessage(`{"time":123,"self_id":42,"post_type":"message","message_type":"group","message_id":2,"user_id":7,"group_id":9,"sender":{"card":"Owner","nickname":"fallback","role":"owner"},"message":[{"type":"at","data":{"qq":"42"}}]}`),
		json.RawMessage(`{"time":123,"self_id":42,"post_type":"message","message_type":"group","sub_type":"anonymous","message_id":3,"user_id":80000000,"group_id":9,"sender":{"nickname":"anonymous"},"message":[{"type":"image","data":{"file":"image"}}]}`),
		json.RawMessage(`{"time":123,"self_id":42,"post_type":"notice","notice_type":"group_recall","group_id":9,"user_id":7,"operator_id":8,"message_id":2}`),
		json.RawMessage(`{"time":123,"self_id":42,"post_type":"notice","notice_type":"group_increase","group_id":9,"user_id":10,"operator_id":8}`),
		json.RawMessage(`{"time":123,"self_id":42,"post_type":"request","request_type":"friend","user_id":11}`),
	}
	server.Push(raw...)
	for range 5 {
		waitDelivery(t, ctx, core)
	}
	// 完整比较同时检查 Core 保留字段、可选字段和原始上报。
	want := make([]*pb.Event, 5)
	for n := range want {
		chat := "group:9"
		if n == 0 {
			chat = "private:7"
		}
		want[n] = &pb.Event{AdapterId: proto.String(onebot11.AdapterID), AdapterInstanceId: proto.String("integration"), AccountId: proto.String("42"),
			Target: &pb.Target{AdapterId: onebot11.AdapterID, AdapterInstanceId: "integration", AccountId: "42", ChatId: chat}, OccurredAtUnixMs: 123000, PlatformPayload: raw[n]}
	}
	chains := []*pb.MessageChain{
		{Segments: []*pb.Segment{{Data: &pb.Segment_Text{Text: &pb.TextSegment{Text: "hello", Format: pb.TextFormat_TEXT_FORMAT_PLAIN.Enum()}}}}},
		{Segments: []*pb.Segment{{Data: &pb.Segment_Mention{Mention: &pb.MentionSegment{ExternalUserId: "42"}}}}},
		{},
	}
	for n := range 3 {
		id := fmt.Sprint(n + 1)
		want[n].EventType = "message.created"
		want[n].SourceEventId = proto.String(id)
		want[n].Message = &pb.PlatformMessage{PlatformMessageId: id, Chain: chains[n]}
		want[n].EventPayload = &pb.EventPayload{Kind: &pb.EventPayload_MessageCreated{MessageCreated: &pb.MessageCreatedPayload{PlatformMessageId: id, Chain: chains[n], ToMe: n < 2}}}
	}
	want[0].Actor = &pb.Actor{ExternalUserId: "7", AdapterInstanceId: "integration", DisplayName: proto.String("Friend"), Permissions: []string{perm.PrivateFriend}}
	want[1].Actor = &pb.Actor{ExternalUserId: "7", AdapterInstanceId: "integration", DisplayName: proto.String("Owner"), Permissions: []string{perm.GroupOwner}}
	want[3].EventType = "message.deleted"
	want[3].Actor = &pb.Actor{ExternalUserId: "8", AdapterInstanceId: "integration"}
	want[3].EventPayload = &pb.EventPayload{Kind: &pb.EventPayload_MessageDeleted{MessageDeleted: &pb.MessageDeletedPayload{PlatformMessageIds: []string{"2"}}}}
	want[4].EventType = "member.joined"
	want[4].Actor = &pb.Actor{ExternalUserId: "8", AdapterInstanceId: "integration"}
	want[4].EventPayload = &pb.EventPayload{Kind: &pb.EventPayload_MemberJoined{MemberJoined: &pb.MemberPayload{MemberExternalUserId: "10"}}}
	got := core.snapshot()
	if len(got) != len(want) {
		t.Fatalf("events = %d, want %d", len(got), len(want))
	}
	for n := range want {
		if !proto.Equal(got[n], want[n]) {
			t.Errorf("event %d:\ngot %v\nwant %v", n, got[n], want[n])
		}
	}
	server.Disconnect()
	waitConnections(t, ctx, server, 2)
	next := json.RawMessage(`{"time":124,"self_id":42,"post_type":"message","message_type":"private","sub_type":"friend","message_id":4,"user_id":7,"sender":{"nickname":"Friend"},"message":[]}`)
	server.Push(next)
	waitDelivery(t, ctx, core)
	got = core.snapshot()
	if len(got) != 6 || got[5].GetSourceEventId() != "4" || !bytes.Equal(got[5].PlatformPayload, next) {
		t.Fatalf("events after reconnect: %v", got)
	}
	server.SetLogin(99, "Other")
	server.Disconnect()
	select {
	case failure := <-core.failures:
		if failure.Code != pb.ErrorCode_ERROR_CODE_FAILED_PRECONDITION || failure.SafeMessage != "onebot account changed" {
			t.Fatalf("failure = %v", failure)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := running.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if len(core.snapshot()) != 6 || len(core.failures) != 0 || server.Connections() != 3 {
		t.Fatal("unexpected repeated delivery, failure or connection")
	}
}

func TestCredentialRedaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const token = "credential-boundary-secret"
	var output bytes.Buffer
	previous, writer, flags := slog.Default(), log.Writer(), log.Flags()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous); log.SetOutput(writer); log.SetFlags(flags) })
	server := onebottest.NewServer(t, token)
	core := newRecordingCore(t)
	_, err := onebot11.New().Start(ctx, startRequest(t, server, token+"-wrong"), core)
	var public *sdk.Error
	if !errors.As(err, &public) || public.Code != pb.ErrorCode_ERROR_CODE_PERMISSION_DENIED || strings.Contains(err.Error(), token) {
		t.Fatalf("unsafe authentication error: %v", err)
	}
	server.Respond("get_login_info", func(map[string]any) (string, int, any) { return "rejected-" + token, 1401, nil })
	_, err = onebot11.New().Start(ctx, startRequest(t, server, token), core)
	if !errors.As(err, &public) || public.Code != pb.ErrorCode_ERROR_CODE_UNAVAILABLE || strings.Contains(err.Error(), token) || !strings.Contains(err.Error(), "rejected-") {
		t.Fatalf("unsafe login error: %v", err)
	}
	server.Respond("get_login_info", func(map[string]any) (string, int, any) {
		return "ok", 0, map[string]any{"user_id": 42, "nickname": "Alona"}
	})
	core.deliveryError = errors.New("delivery rejected: " + token)
	running := startInstance(t, ctx, server, token, core)
	server.Push(json.RawMessage(`{"time":123,"self_id":42,"post_type":"message","message_type":"private","message_id":1,"user_id":7,"message":[]}`))
	waitDelivery(t, ctx, core)
	server.Push(json.RawMessage(`{"time":123,"self_id":42,"post_type":"message","message_type":"private","message_id":2,"user_id":7,"message":[]}`))
	waitDelivery(t, ctx, core)
	if err := running.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	// 先等日志所属循环退出，再读取缓冲区，避免与 slog 写入竞争。
	if logs := output.String(); strings.Contains(logs, token) || !strings.Contains(logs, "delivery rejected:") || !strings.Contains(logs, "<redacted>") {
		t.Fatalf("unsafe logs: %s", logs)
	}
}
