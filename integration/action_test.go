package onebot11_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlonaBot/AlonaBot/sdk/adapter"
	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	"github.com/AlonaBot/AlonaBot/sdk/message"
	"github.com/AlonaBot/alonabot-adapter-onebot11/onebottest"
)

func TestGroupSpecialTitleAction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server := onebottest.NewServer(t, "")
	running := startInstance(t, ctx, server, "", newRecordingCore(t))
	executor := running.(adapter.ActionExecutor)
	const action = "set_group_special_title"
	group := message.SID("onebot11", "group:9", "")
	for _, test := range []struct {
		name     string
		target   string
		params   string
		status   string
		retcode  int
		data     any
		wantCode errcode.Code
		want     map[string]any
	}{
		{
			name: "set", target: group,
			params: `{"user_id":"7","special_title":"热心群友"}`,
			status: "ok",
			want:   map[string]any{"group_id": json.Number("9"), "user_id": "7", "special_title": "热心群友"},
		},
		{
			name: "clear", target: group,
			params: `{"user_id":"7","special_title":""}`,
			status: "ok",
			want:   map[string]any{"group_id": json.Number("9"), "user_id": "7", "special_title": ""},
		},
		{
			name: "duration", target: group,
			params: `{"user_id":"7","special_title":"Helper","duration":3600}`,
			status: "ok",
			want:   map[string]any{"group_id": json.Number("9"), "user_id": "7", "special_title": "Helper", "duration": json.Number("3600")},
		},
		{
			name: "zero duration", target: group,
			params: `{"user_id":"7","special_title":"Helper","duration":0}`,
			status: "ok",
			want:   map[string]any{"group_id": json.Number("9"), "user_id": "7", "special_title": "Helper", "duration": json.Number("0")},
		},
		{
			name: "permanent", target: group,
			params: `{"user_id":"7","special_title":"Helper","duration":-1}`,
			status: "ok", data: map[string]any{"extra": "retained"},
			want: map[string]any{"group_id": json.Number("9"), "user_id": "7", "special_title": "Helper", "duration": json.Number("-1")},
		},
		{
			name: "private", target: message.SID("onebot11", "private:7", ""),
			params:   `{"user_id":"7","special_title":"Helper"}`,
			wantCode: errcode.Unsupported,
		},
		{
			name: "permission denied", target: group,
			params: `{"user_id":"7","special_title":"Helper"}`,
			status: "failed", retcode: 1403, data: map[string]any{"reason": "permission denied"},
			wantCode: errcode.FailedPrecondition,
			want:     map[string]any{"group_id": json.Number("9"), "user_id": "7", "special_title": "Helper"},
		},
		{
			name: "backend unsupported", target: group,
			params: `{"user_id":"7","special_title":"Helper"}`,
			status: "failed", retcode: 1404,
			wantCode: errcode.FailedPrecondition,
			want:     map[string]any{"group_id": json.Number("9"), "user_id": "7", "special_title": "Helper"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server.Respond(action, func(map[string]any) (string, int, any) {
				return test.status, test.retcode, test.data
			})
			before := len(server.Calls())
			raw, failure := executor.Action(ctx, test.target, "onebot11."+action, 1, json.RawMessage(test.params))
			if test.wantCode == 0 {
				if failure != nil {
					t.Fatal(failure)
				}
			} else if failure == nil || failure.Code != test.wantCode {
				t.Fatalf("failure = %v, want %s", failure, test.wantCode)
			}
			calls := server.Calls()[before:]
			if test.want == nil {
				if len(calls) != 0 || len(raw) != 0 {
					t.Fatalf("rejected target: calls = %v, data = %s", calls, raw)
				}
				return
			}
			if len(calls) != 1 || calls[0].Action != action || !reflect.DeepEqual(calls[0].Params, test.want) {
				t.Fatalf("calls = %v, want %s %v", calls, action, test.want)
			}
			var data any
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(data, test.data) {
				t.Fatalf("data = %v, want %v", data, test.data)
			}
			if failure != nil && (!strings.Contains(failure.Message, action) || !strings.Contains(failure.Message, "retcode "+strconv.Itoa(test.retcode))) {
				t.Fatalf("missing platform error details: %v", failure)
			}
		})
	}
}
