package onebot11

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	urlpkg "net/url"
	"strconv"
	"time"

	"github.com/AlonaBot/AlonaBot/sdk"
	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	msg "github.com/AlonaBot/AlonaBot/sdk/message"
	"github.com/AlonaBot/alonabot-adapter-onebot11/perm"
	segmentpkg "github.com/AlonaBot/alonabot-adapter-onebot11/segment"
)

var errStringMessage = errors.New("onebot implementation must post messages in array format")

type receivedEvent struct {
	raw    json.RawMessage
	source *receiveConnection
}

// onebotPost 只读取标准化所需字段；原始上报始终保留在 Event.PlatformPayload。
type onebotPost struct {
	Time        *int64          `json:"time"`
	SelfID      int64           `json:"self_id"`
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	NoticeType  string          `json:"notice_type"`
	SubType     string          `json:"sub_type"`
	MessageID   *int64          `json:"message_id"`
	UserID      *int64          `json:"user_id"`
	GroupID     *int64          `json:"group_id"`
	OperatorID  *int64          `json:"operator_id"`
	Message     json.RawMessage `json:"message"`
	Sender      struct {
		Card     string `json:"card"`
		Nickname string `json:"nickname"`
		Role     string `json:"role"`
	} `json:"sender"`
}

func (i *instance) deliverEvents() {
	defer close(i.deliveryDone)
	for i.receiveCtx.Err() == nil {
		var received receivedEvent
		select { // 等待上报时允许 Stop 立即取消。
		case <-i.receiveCtx.Done():
			return
		case received = <-i.events:
		}
		select { // 上报可以先于登录应答到达，必须等待其所属连接的账号复核。
		case <-i.receiveCtx.Done():
			return
		case <-received.source.checked:
		}
		if !received.source.valid || i.receiveCtx.Err() != nil {
			continue
		}
		event, err := i.normalize(i.receiveCtx, received.raw)
		if errors.Is(err, errStringMessage) {
			i.fail(errcode.FailedPrecondition, errStringMessage.Error())
			return
		}
		if err != nil {
			slog.Warn("skip malformed onebot event", "adapter_instance_id", i.id, "error", i.config.safeError(err))
			continue
		}
		if event == nil || i.receiveCtx.Err() != nil {
			continue
		}
		// 等待本条应答后才取下一条；accepted=false 同样继续，OneBot 上报没有重放。
		if _, err := i.core.Deliver(i.receiveCtx, event, nil); err != nil {
			if i.receiveCtx.Err() != nil {
				return
			}
			var public *sdk.Error
			if errors.As(err, &public) && public.Code == errcode.FailedPrecondition {
				i.stopReceive()
				return
			}
			slog.Warn("deliver onebot event failed", "adapter_instance_id", i.id, "error", i.config.safeError(err))
		}
	}
}

func (i *instance) normalize(ctx context.Context, raw json.RawMessage) (*msg.Event, error) {
	// 1. 身份来自已复核的登录账号，错误账号的上报不进入 Core。
	var post onebotPost
	if err := json.Unmarshal(raw, &post); err != nil {
		return nil, fmt.Errorf("decode onebot event: %w", err)
	}
	if strconv.FormatInt(post.SelfID, 10) != i.account.ID {
		slog.Warn("skip onebot event with mismatched self_id", "adapter_instance_id", i.id)
		return nil, nil
	}
	if post.PostType == "message" && len(post.Message) != 0 && post.Message[0] == '"' {
		return nil, errStringMessage
	}

	// 2. 标准通知归一化，其余通知与请求走已声明的扩展事件。
	group := post.MessageType == "group"
	switch post.PostType {
	case "message":
		if post.MessageType != "private" && post.MessageType != "group" {
			i.ignore(post)
			return nil, nil
		}
	case "notice", "request":
		group = post.GroupID != nil
	default:
		i.ignore(post)
		return nil, nil
	}
	chatID, prefix := post.UserID, "private:"
	if group {
		chatID, prefix = post.GroupID, "group:"
	}
	if post.Time == nil || (chatID == nil && post.PostType == "message") {
		return nil, errors.New("missing onebot event time or chat ID")
	}
	session := ""
	if chatID != nil {
		session = msg.SID(AdapterID, prefix+strconv.FormatInt(*chatID, 10), "")
	}
	event := &msg.Event{AdapterID: AdapterID, AdapterInstanceID: i.id, AccountID: i.account.ID, Session: session, OccurredAt: time.Unix(*post.Time, 0), PlatformPayload: raw}

	event.ChatTitle = i.chatTitle(ctx, post)
	if chatID != nil {
		event.ChatType = msg.ChatPrivate
		if group {
			event.ChatType = msg.ChatGroup
		}
	}
	// 3. 消息与通知分别构造对应的 payload，Core 保留字段留空。
	if post.PostType == "message" {
		return i.messageEvent(ctx, post, event)
	}
	if post.PostType == "request" || (post.NoticeType != "group_recall" && post.NoticeType != "friend_recall" && post.NoticeType != "group_increase" && post.NoticeType != "group_decrease") {
		return i.extensionEvent(raw, post, event)
	}
	return i.noticeEvent(post, event)
}

func (i *instance) ignore(post onebotPost) {
	i.ignored++
	slog.Debug("ignore unsupported onebot event", "adapter_instance_id", i.id,
		"post_type", i.config.redact.Replace(post.PostType), "notice_type", i.config.redact.Replace(post.NoticeType), "ignored", i.ignored)
}

func (i *instance) messageEvent(ctx context.Context, post onebotPost, event *msg.Event) (*msg.Event, error) {
	// 1. 先检查来源标识与数组片段，避免向 Core 交付不完整的消息。
	if post.MessageID == nil || (post.SubType != "anonymous" && post.UserID == nil) {
		return nil, errors.New("missing onebot message_id or user_id")
	}
	chain, err := messageChain(post.Message, event.Session, i.id)
	if err != nil {
		return nil, err
	}
	i.enrichMentions(ctx, post, chain)
	// 2. 权限只写平台事实，隐含关系留给 Core 展开。
	if post.SubType != "anonymous" {
		event.Actor = i.actor(*post.UserID)
		name := post.Sender.Card
		if name == "" {
			name = post.Sender.Nickname
		}
		if name != "" {
			event.Actor.DisplayName = name
		}
		if post.MessageType == "group" {
			switch post.Sender.Role {
			case "owner":
				event.Actor.Permissions = []string{perm.GroupOwner}
			case "admin":
				event.Actor.Permissions = []string{perm.GroupAdmin}
			}
		} else if post.SubType == "friend" {
			event.Actor.Permissions = []string{perm.PrivateFriend}
		}
	}
	// 3. 来源消息与新消息 payload 使用同一份 Chain。
	platformID := strconv.FormatInt(*post.MessageID, 10)
	event.Type = "message.created"
	event.SourceEventID = platformID
	event.Message = &msg.PlatformMessage{PlatformMessageID: platformID, Chain: chain}
	event.Payload = &msg.CreatedPayload{PlatformMessageID: platformID, Chain: chain, ToMe: i.toMe(ctx, post.MessageType == "private", chain)}
	return event, nil
}

func (i *instance) toMe(ctx context.Context, private bool, chain msg.Chain) bool {
	// 1. 私聊和显式提及已足够判断，无需查询引用消息。
	if private {
		return true
	}
	for _, segment := range chain {
		if mention, ok := segment.(msg.MentionSegment); ok && mention.ExternalUserID == i.account.ID {
			return true
		}
	}
	// 2. 引用只按 get_msg 的实际发送者判断，单次查询失败仍可检查后续引用。
	for _, segment := range chain {
		reference, ok := segment.(msg.ReferenceSegment)
		if !ok {
			continue
		}
		_, platformID, _, _ := msg.ParseRef(reference.Ref)
		var message struct {
			Sender struct {
				UserID *int64 `json:"user_id"`
			} `json:"sender"`
		}
		if err := i.call(ctx, "get_msg", map[string]any{"message_id": json.Number(platformID)}, &message); err != nil {
			slog.Debug("onebot get_msg failed", "adapter_instance_id", i.id, "error", i.config.safeError(err))
			continue
		}
		if message.Sender.UserID == nil {
			slog.Debug("onebot get_msg response has no sender user_id", "adapter_instance_id", i.id)
			continue
		}
		if strconv.FormatInt(*message.Sender.UserID, 10) == i.account.ID {
			return true
		}
	}
	return false
}

func (i *instance) noticeEvent(post onebotPost, event *msg.Event) (*msg.Event, error) {
	switch post.NoticeType {
	case "group_recall", "friend_recall":
		if post.MessageID == nil {
			return nil, errors.New("missing onebot recall message_id")
		}
		event.Type = "message.deleted"
		event.Payload = &msg.DeletedPayload{PlatformMessageIDs: []string{strconv.FormatInt(*post.MessageID, 10)}}
		if post.OperatorID != nil {
			event.Actor = i.actor(*post.OperatorID)
		}
	case "group_increase", "group_decrease":
		if post.UserID == nil {
			return nil, errors.New("missing onebot member user_id")
		}
		payload := &msg.MemberPayload{MemberExternalUserID: strconv.FormatInt(*post.UserID, 10)}
		if post.NoticeType == "group_increase" {
			event.Type = "member.joined"
			event.Payload = payload
		} else {
			event.Type = "member.left"
			event.Payload = payload
		}
		if post.OperatorID != nil && *post.OperatorID != 0 && *post.OperatorID != *post.UserID {
			event.Actor = i.actor(*post.OperatorID)
		}
	}
	return event, nil
}

func (i *instance) actor(userID int64) *msg.Actor {
	return &msg.Actor{ExternalUserID: strconv.FormatInt(userID, 10), AdapterInstanceID: i.id}
}

func messageChain(raw json.RawMessage, session string, instanceID string) (msg.Chain, error) {
	chain := msg.Chain{}
	var segments []struct {
		Type string                     `json:"type"`
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &segments); err != nil {
		return chain, fmt.Errorf("decode onebot message array: %w", err)
	}
	if segments == nil {
		return chain, errors.New("missing onebot message array")
	}
	for _, segment := range segments {
		var field string
		switch segment.Type {
		case "text":
			field = "text"
		case "at":
			field = "qq"
		case "reply":
			field = "id"
		case "markdown":
			var text string
			for _, key := range []string{"markdown", "content"} {
				if value, ok := segment.Data[key]; ok {
					if json.Unmarshal(value, &text) == nil {
						break
					}
				}
			}
			if text != "" {
				chain = append(chain, msg.Markdown(text))
			}
			raw, err := json.Marshal(segment.Data)
			if err != nil {
				return chain, err
			}
			chain = append(chain, msg.Extension(AdapterID, "onebot11.markdown", 1, segmentpkg.Raw{Data: string(raw)}))
			continue
		case "image", "video", "record", "file":
			var file, url, name string
			for _, field := range []struct {
				key    string
				target *string
			}{{"file", &file}, {"url", &url}, {"name", &name}, {"file_name", &name}} {
				key, target := field.key, field.target
				if value, ok := segment.Data[key]; ok {
					if err := json.Unmarshal(value, target); err != nil {
						return chain, fmt.Errorf("decode media %s: %w", key, err)
					}
				}
			}
			readability := msg.Unreadable
			token := file
			if url != "" {
				token = url
				parsed, err := urlpkg.Parse(url)
				if err != nil {
					return chain, err
				}
				if parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https") {
					readability = msg.Readable
				}
			}
			if segment.Type == "file" && url == "" {
				var descriptor platformFile
				descriptor.Session = session
				descriptor.Data = segment.Data
				if value, ok := segment.Data["file_id"]; ok {
					if err := json.Unmarshal(value, &descriptor.ID); err != nil {
						return chain, err
					}
				} else {
					descriptor.ID = file
				}
				if descriptor.ID != "" {
					encoded, err := json.Marshal(descriptor)
					if err != nil {
						return chain, err
					}
					token = fileTokenPrefix + base64.RawURLEncoding.EncodeToString(encoded)
					readability = msg.Readable
				}
			}
			if token == "" {
				return chain, fmt.Errorf("missing onebot %s media source", segment.Type)
			}
			source := msg.MediaSource{Incoming: &msg.IncomingMedia{Token: token, Readability: readability}}
			if rawSize, ok := segment.Data["size"]; ok {
				var size json.Number
				if err := json.Unmarshal(rawSize, &size); err != nil {
					return chain, fmt.Errorf("decode media size: %w", err)
				}
				if size != "" {
					value, err := strconv.ParseUint(string(size), 10, 64)
					if err != nil {
						return chain, fmt.Errorf("parse media size: %w", err)
					}
					source.Incoming.Size = value
				}
			}
			switch segment.Type {
			case "image":
				chain = append(chain, msg.Image(source))
			case "video":
				chain = append(chain, msg.Video(source))
			case "record":
				chain = append(chain, msg.Voice(source))
			case "file":
				chain = append(chain, msg.File(source, name))
			}
			continue
		case "forward":
			var id string
			if err := json.Unmarshal(segment.Data["id"], &id); err != nil || id == "" {
				return chain, errors.New("invalid onebot forward id")
			}
			chain = append(chain, msg.ForwardSegment{PlatformForwardID: id})
			continue
		case "face":
			field = "id"
		default:
			raw, err := json.Marshal(segment.Data)
			if err != nil {
				return chain, err
			}
			name := "onebot11." + segment.Type
			if !rawSegment(segment.Type) {
				name = "onebot11.segment"
				raw, err = json.Marshal(segment)
				if err != nil {
					return chain, err
				}
			}
			chain = append(chain, msg.Extension(AdapterID, name, 1, segmentpkg.Raw{Data: string(raw)}))
			continue
		}
		var scalar json.Number
		var value *string
		if segment.Type == "face" || segment.Type == "at" || segment.Type == "reply" {
			if err := json.Unmarshal(segment.Data[field], &scalar); err == nil {
				v := scalar.String()
				value = &v
			}
		}
		if value == nil {
			if err := json.Unmarshal(segment.Data[field], &value); err != nil {
				return chain, fmt.Errorf("decode onebot %s segment: %w", segment.Type, err)
			}
		}
		if value == nil {
			return chain, fmt.Errorf("missing onebot %s segment value", segment.Type)
		}
		if segment.Type == "text" {
			chain = append(chain, msg.TextSegment{Text: *value, Format: msg.FormatPlain})
			continue
		}
		if segment.Type == "face" {
			chain = append(chain, msg.Extension(AdapterID, "onebot11.face", 1, segmentpkg.Face{ID: *value}))
			continue
		}
		if segment.Type == "at" && *value == "all" {
			chain = append(chain, msg.Extension(AdapterID, "onebot11.at_all", 1, segmentpkg.Empty{}))
			continue
		}
		id, err := strconv.ParseInt(*value, 10, 64)
		if err != nil {
			return chain, fmt.Errorf("parse onebot %s segment ID: %w", segment.Type, err)
		}
		platformID := strconv.FormatInt(id, 10)
		if segment.Type == "at" {
			chain = append(chain, msg.Mention(platformID))
		} else {
			chain = append(chain, msg.Quote(msg.Ref(session, platformID, instanceID)))
		}
	}
	return chain, nil
}
