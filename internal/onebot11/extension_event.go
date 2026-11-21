package onebot11

import (
	"encoding/json"
	"fmt"

	"github.com/AlonaBot/AlonaBot/sdk/message"
)

func (i *instance) extensionEvent(raw json.RawMessage, post onebotPost, event *message.Event) (*message.Event, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	var category string
	if post.PostType == "request" {
		json.Unmarshal(fields["request_type"], &category)
		category += "_request"
	} else {
		category = post.NoticeType
		if category == "notify" {
			category = post.SubType
		}
	}
	names := map[string][]string{
		"poke": {"target_id"}, "lucky_king": {"target_id"}, "honor": {"honor_type"}, "group_admin": {"sub_type", "user_id"}, "group_ban": {"sub_type", "user_id", "duration"}, "group_upload": {"file"}, "friend_add": {}, "friend_request": {"comment", "flag"}, "group_request": {"sub_type", "comment", "flag"}, "input_status": {"status_text", "event_type"},
	}
	payload := map[string]json.RawMessage{}
	keys, known := names[category]
	if !known {
		category = post.PostType
		payload["raw"], _ = json.Marshal(string(raw))
	} else {
		for _, key := range keys {
			value, ok := fields[key]
			if !ok {
				return nil, fmt.Errorf("missing onebot %s %s", category, key)
			}
			if key == "file" {
				var file map[string]json.RawMessage
				if err := json.Unmarshal(value, &file); err != nil {
					return nil, fmt.Errorf("decode group upload file: %w", err)
				}
				declared := map[string]json.RawMessage{}
				for _, field := range []string{"id", "name", "size", "busid"} {
					part, ok := file[field]
					if !ok {
						return nil, fmt.Errorf("missing group upload file %s", field)
					}
					declared[field] = part
				}
				encoded, err := json.Marshal(declared)
				if err != nil {
					return nil, err
				}
				value = encoded
			}
			if key == "target_id" || key == "user_id" {
				var number json.Number
				if err := json.Unmarshal(value, &number); err != nil {
					return nil, err
				}
				value, _ = json.Marshal(number.String())
			}
			payload[key] = value
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	event.Type = "onebot11." + category
	event.Payload = &message.AdapterEventPayload{Version: 1, PayloadJSON: string(encoded)}
	if post.UserID != nil {
		event.Actor = i.actor(*post.UserID)
	}
	if post.OperatorID != nil && *post.OperatorID != 0 {
		event.Actor = i.actor(*post.OperatorID)
	}
	return event, nil
}
