package onebot11_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/AlonaBot/AlonaBot/sdk/adapter"
	msg "github.com/AlonaBot/AlonaBot/sdk/message"
	"github.com/AlonaBot/alonabot-adapter-onebot11/onebottest"
	"github.com/AlonaBot/alonabot-adapter-onebot11/segment"
)

// TestRichMessageDelivery 经公开 Adapter 入口验证混排节点及原始扩展的实际平台请求。
func TestRichMessageDelivery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server := onebottest.NewServer(t, "")
	server.SetLogin(42, "Alona")
	for _, action := range []string{"send_group_forward_msg", "send_msg"} {
		server.Respond(action, func(map[string]any) (string, int, any) {
			return "ok", 0, map[string]any{"message_id": 101}
		})
	}
	running := startInstance(t, ctx, server, "", newRecordingCore(t))
	target := msg.SID("onebot11", "group:9", "")
	chain := msg.Chain{
		msg.ForwardSegment{Items: []msg.ForwardItem{
			{Chain: msg.Chain{msg.Text("Summary")}},
			{PlatformMessageID: "77"},
		}},
		msg.Extension("onebot11", "onebot11.segment", 1, segment.Raw{Data: `{"type":"poke","data":{"type":"1","id":9007199254740993}}`}),
	}
	if status, failure := running.(adapter.MessageChecker).CheckMessage(ctx, target, chain); status != msg.SupportSupported || failure != nil {
		t.Fatalf("preflight = %s, %v", status, failure)
	}
	if calls := server.Calls(); len(calls) != 1 || calls[0].Action != "get_login_info" {
		t.Fatalf("preflight requested platform: %v", calls)
	}
	result, failure := running.Send(ctx, target, chain)
	if failure != nil || result.Status != msg.SendAccepted || len(result.PartResults) != 2 {
		t.Fatalf("send = %+v, %v", result, failure)
	}
	for n, part := range result.PartResults {
		if part.Status != msg.PartAccepted || !reflect.DeepEqual(part.SegmentIndexes, []uint32{uint32(n)}) || !reflect.DeepEqual(part.PlatformMessageIDs, []string{"101"}) {
			t.Fatalf("part %d = %+v", n, part)
		}
	}
	calls := server.Calls()[1:]
	wantNodes := []any{
		map[string]any{"type": "node", "data": map[string]any{"user_id": "42", "nickname": "Alona", "content": []any{map[string]any{"type": "text", "data": map[string]any{"text": "Summary"}}}}},
		map[string]any{"type": "node", "data": map[string]any{"id": "77"}},
	}
	wantSegment := []any{map[string]any{"type": "poke", "data": map[string]any{"type": "1", "id": json.Number("9007199254740993")}}}
	if len(calls) != 2 || calls[0].Action != "send_group_forward_msg" || calls[1].Action != "send_msg" || !reflect.DeepEqual(calls[0].Params["messages"], wantNodes) || !reflect.DeepEqual(calls[1].Params["message"], wantSegment) {
		t.Fatalf("platform calls = %#v", calls)
	}
}
