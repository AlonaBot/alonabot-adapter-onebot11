package onebot11

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/AlonaBot/AlonaBot/sdk"
	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	msg "github.com/AlonaBot/AlonaBot/sdk/message"
)

type wireSegment struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}
type sendPart struct {
	action  string
	chain   msg.Chain
	indexes []uint32
	reply   *wireSegment
}

func validateChain(chain msg.Chain, nested bool) error {
	for _, s := range chain {
		switch s := s.(type) {
		case msg.TextSegment:
			if s.Format == msg.FormatMarkdown || len(s.Entities) > 0 {
				return errcode.Unsupported.New(plainTextRule + ": rich text is unsupported")
			}
		case msg.ImageSegment:
			if s.Alt != "" {
				return errcode.Unsupported.New(mediaRule + ": image alt is unsupported")
			}
			if err := validateInline(s.Source); err != nil {
				return err
			}
		case msg.VideoSegment:
			if err := validateInline(s.Source); err != nil {
				return err
			}
		case msg.VoiceSegment:
			if err := validateInline(s.Source); err != nil {
				return err
			}
		case msg.FileSegment:
			if s.Caption != "" {
				return errcode.Unsupported.New(mediaRule + ": file captions are unsupported")
			}
			if err := validateInline(s.Source); err != nil {
				return err
			}
		case msg.MentionSegment:
		case msg.ReferenceSegment:
			if nested {
				return errcode.Unsupported.New(replyRule + ": nested reply is unsupported")
			}
		case msg.ForwardSegment:
			if s.PlatformForwardID != "" || len(s.Items) == 0 {
				return errcode.Unsupported.New(forwardRule + ": send forward requires nonempty items")
			}
			for _, item := range s.Items {
				if (len(item.Chain) == 0) == (item.PlatformMessageID == "") {
					return errcode.InvalidArgument.New(forwardRule + ": forward item requires content or message ID")
				}
				if err := validateChain(item.Chain, true); err != nil {
					return err
				}
			}
		case msg.ExtensionSegment:
			if s.AdapterID != AdapterID || s.Version != 1 {
				return errcode.Unsupported.New(extensionRule + ": unsupported extension")
			}
			if _, err := extensionSegment(s); err != nil {
				return err
			}
		default:
			return errcode.Unsupported.New(unsupportedRule + ": unsupported onebot segment")
		}
	}
	return nil
}

func (i *instance) plan(to string, chain msg.Chain) ([]sendPart, error) {
	if _, err := targetParams(to); err != nil {
		return nil, err
	}
	if err := validateChain(chain, false); err != nil {
		return nil, err
	}
	var parts []sendPart
	var reply *wireSegment
	for index, s := range chain {
		if ref, ok := s.(msg.ReferenceSegment); ok {
			sid, id, instanceID, err := msg.ParseRef(ref.Ref)
			if err != nil {
				return nil, err
			}
			if sid != to || (instanceID != "" && instanceID != i.id) || reply != nil {
				return nil, errcode.Unsupported.New(replyRule + ": reply must reference this instance and target")
			}
			reply = &wireSegment{"reply", map[string]any{"id": id}}
			continue
		}
		action := "send_msg"
		separate := false
		switch s.(type) {
		case msg.VideoSegment, msg.VoiceSegment:
			separate = true
		case msg.FileSegment:
			action = "upload_file"
			separate = true
		case msg.ExtensionSegment:
			if s.(msg.ExtensionSegment).Name == "onebot11.forward_ref" {
				action = "send_forward_msg"
				separate = true
			}
		case msg.ForwardSegment:
			action = "send_forward_msg"
			separate = true
		}
		if separate || len(parts) == 0 || parts[len(parts)-1].action != "send_msg" {
			parts = append(parts, sendPart{action: action})
		} else if len(parts[len(parts)-1].chain) > 0 {
			switch parts[len(parts)-1].chain[0].(type) {
			case msg.VideoSegment, msg.VoiceSegment:
				parts = append(parts, sendPart{action: action})
			}
		}
		part := &parts[len(parts)-1]
		part.chain = append(part.chain, s)
		part.indexes = append(part.indexes, uint32(index))
	}
	if reply != nil {
		if len(parts) == 0 || parts[0].action != "send_msg" {
			return nil, errcode.Unsupported.New(replyRule + ": reply requires a message as first operation")
		}
		parts[0].reply = reply
		for n, s := range chain {
			if _, ok := s.(msg.ReferenceSegment); ok {
				parts[0].indexes = append([]uint32{uint32(n)}, parts[0].indexes...)
			}
		}
	}
	return parts, nil
}

func (i *instance) segments(ctx context.Context, chain msg.Chain) (out []wireSegment, cleanups []func() error, err error) {
	media := func(source msg.MediaSource) (string, error) {
		file, cleanup, err := i.mediaFile(ctx, source)
		cleanups = append(cleanups, cleanup)
		return file, err
	}
	for _, s := range chain {
		switch s := s.(type) {
		case msg.TextSegment:
			out = append(out, wireSegment{"text", map[string]any{"text": s.Text}})
		case msg.MentionSegment:
			out = append(out, wireSegment{"at", map[string]any{"qq": s.ExternalUserID}})
		case msg.ImageSegment:
			file, e := media(s.Source)
			if e != nil {
				return out, cleanups, e
			}
			out = append(out, wireSegment{"image", map[string]any{"file": file}})
			if s.Caption != "" {
				out = append(out, wireSegment{"text", map[string]any{"text": s.Caption}})
			}
		case msg.VideoSegment:
			file, e := media(s.Source)
			if e != nil {
				return out, cleanups, e
			}
			out = append(out, wireSegment{"video", map[string]any{"file": file}})
			if s.Caption != "" {
				out = append(out, wireSegment{"text", map[string]any{"text": s.Caption}})
			}
		case msg.VoiceSegment:
			file, e := media(s.Source)
			if e != nil {
				return out, cleanups, e
			}
			out = append(out, wireSegment{"record", map[string]any{"file": file}})
			if s.Caption != "" {
				out = append(out, wireSegment{"text", map[string]any{"text": s.Caption}})
			}
		case msg.FileSegment:
			file, e := media(s.Source)
			if e != nil {
				return out, cleanups, e
			}
			out = append(out, wireSegment{"file", map[string]any{"file": file, "name": s.Filename}})
		case msg.ForwardSegment:
			nodes, cleanup, e := i.forwardNodes(ctx, s.Items)
			cleanups = append(cleanups, cleanup...)
			if e != nil {
				return out, cleanups, e
			}
			out = append(out, nodes...)
		case msg.ExtensionSegment:
			segment, e := extensionSegment(s)
			if e != nil {
				return out, cleanups, e
			}
			out = append(out, segment)
		}
	}
	return
}

func (i *instance) executePart(ctx context.Context, to string, part sendPart, onUpload func(error, bool)) (id string, unavailable bool, accepted bool, err error) {
	params, err := targetParams(to)
	if err != nil {
		return "", false, false, err
	}
	var cleanups []func() error
	defer func() {
		for _, cleanup := range cleanups {
			err = errors.Join(err, cleanup())
		}
	}()
	action := part.action
	switch action {
	case "upload_file":
		file := part.chain[0].(msg.FileSegment)
		var cleanup func() error
		params["file"], cleanup, err = i.mediaFile(ctx, file.Source)
		cleanups = append(cleanups, cleanup)
		if err != nil {
			return
		}
		params["name"] = file.Filename
		if params["message_type"] == "group" {
			action = "upload_group_file"
		} else {
			action = "upload_private_file"
		}
		delete(params, "message_type")
		unavailable = true
	case "send_forward_msg":
		nodes, cleanup, e := i.forwardNodes(ctx, part.chain[0].(msg.ForwardSegment).Items)
		cleanups = append(cleanups, cleanup...)
		if e != nil {
			return "", false, false, e
		}
		nodes, e = i.materializeMixed(ctx, to, nodes, onUpload)
		if e != nil {
			return "", false, false, e
		}
		params["messages"] = nodes
		if params["message_type"] == "group" {
			action = "send_group_forward_msg"
		} else {
			action = "send_private_forward_msg"
		}
		delete(params, "message_type")
	default:
		segments, cleanup, e := i.segments(ctx, part.chain)
		cleanups = append(cleanups, cleanup...)
		if e != nil {
			return "", false, false, e
		}
		if part.reply != nil {
			segments = append([]wireSegment{*part.reply}, segments...)
		}
		params["message"] = segments
	}
	var response struct {
		MessageID json.Number `json:"message_id"`
	}
	err = i.callAPI(ctx, action, params, &response, true)
	accepted = err == nil
	id = response.MessageID.String()
	if id == "0" {
		id = ""
	}
	unavailable = unavailable || id == ""
	return
}

func safePublic(err error) *errcode.Error {
	if err == nil {
		return nil
	}
	var public *sdk.Error
	if errors.As(err, &public) {
		return &errcode.Error{Code: public.Code, Message: err.Error()}
	}
	code := errcode.Internal
	if errors.Is(err, context.Canceled) {
		code = errcode.Cancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		code = errcode.DeadlineExceeded
	}
	return &errcode.Error{Code: code, Message: err.Error()}
}

// Send 先完成整链计划，再逐项发送；首次失败后保留未尝试部分与已确认回执。
func (i *instance) Send(ctx context.Context, to string, chain msg.Chain) (*msg.SendResult, *errcode.Error) {
	chain, err := i.expandSendChain(ctx, to, chain)
	if err != nil {
		return nil, safePublic(err)
	}
	parts, err := i.plan(to, chain)
	if err != nil {
		return nil, safePublic(err)
	}
	result := &msg.SendResult{Session: to, AdapterInstanceID: i.id, Status: msg.SendAccepted}
	var failure *errcode.Error
	for index, part := range parts {
		receipt := msg.PartResult{PartIndex: uint32(index), SegmentIndexes: part.indexes, Status: msg.PartNotAttempted}
		if failure == nil {
			used := []string{"message.send"}
			if part.reply != nil {
				used = append(used, "message.reference")
			}
			for _, s := range part.chain {
				switch s.(type) {
				case msg.VoiceSegment:
					used = append(used, "message.voice")
				case msg.ForwardSegment:
					used = append(used, "message.forward")
				}
			}
			for _, name := range used {
				if !slices.ContainsFunc(result.CapabilitiesUsed, func(c msg.Capability) bool { return c.Name == name }) {
					result.CapabilitiesUsed = append(result.CapabilitiesUsed, msg.Capability{Name: name, Version: 1, Supported: true})
				}
			}
			uploadFailed := false
			onUpload := func(uploadError error, accepted bool) {
				upload := msg.PartResult{PartIndex: uint32(len(result.PartResults)), SegmentIndexes: part.indexes, Status: msg.PartAccepted, MessageIDsUnavailable: true}
				if uploadError != nil {
					uploadFailed = true
					upload.Error = safePublic(uploadError)
					if !accepted {
						upload.Status = msg.PartRejected
						var callError *apiCallError
						if errors.As(uploadError, &callError) && callError.attempted {
							upload.Status = msg.PartUnknown
						}
					}
				}
				result.PartResults = append(result.PartResults, upload)
			}
			id, unavailable, accepted, err := i.executePart(ctx, to, part, onUpload)
			if uploadFailed {
				failure = safePublic(err)
				result.Status = msg.SendRejected
				for _, previous := range result.PartResults {
					if previous.Status == msg.PartAccepted {
						result.Status = msg.SendPartial
						break
					}
					if previous.Status == msg.PartUnknown {
						result.Status = msg.SendUnknown
					}
				}
			} else if accepted {
				if err != nil {
					failure = safePublic(err)
					receipt.Error = failure
					if index < len(parts)-1 {
						result.Status = msg.SendPartial
					}
				}
				receipt.Status = msg.PartAccepted
				receipt.MessageIDsUnavailable = unavailable
				if id != "" {
					receipt.PlatformMessageIDs = []string{id}
					result.PlatformMessageIDs = append(result.PlatformMessageIDs, id)
				}
				result.ProviderAcceptedAt = time.Now()
			} else {
				failure = safePublic(err)
				receipt.Error = failure
				receipt.Status = msg.PartRejected
				var callError *apiCallError
				if errors.As(err, &callError) && callError.attempted {
					receipt.Status = msg.PartUnknown
				}
				if len(result.PartResults) > 0 {
					result.Status = msg.SendPartial
				} else if receipt.Status == msg.PartUnknown {
					result.Status = msg.SendUnknown
				} else {
					result.Status = msg.SendRejected
				}
			}
		}
		receipt.PartIndex = uint32(len(result.PartResults))
		result.PartResults = append(result.PartResults, receipt)
	}
	if len(result.PlatformMessageIDs) > 0 {
		result.PrimaryPlatformMessageID = result.PlatformMessageIDs[0]
	}
	return result, failure
}

// Delete 按输入顺序撤回，失败后的条目保持未尝试。
func (i *instance) Delete(ctx context.Context, to string, ids []string) (*msg.DeleteResult, *errcode.Error) {
	if _, err := targetParams(to); err != nil {
		return nil, safePublic(err)
	}
	result := &msg.DeleteResult{Session: to, AdapterInstanceID: i.id}
	var failure *errcode.Error
	for _, id := range ids {
		item := msg.DeleteItemResult{PlatformMessageID: id, Status: msg.PartNotAttempted}
		if failure == nil {
			err := i.callAPI(ctx, "delete_msg", map[string]any{"message_id": id}, nil, true)
			if err == nil {
				item.Status = msg.PartAccepted
			} else {
				failure = safePublic(err)
				item.Error = failure
				item.Status = msg.PartRejected
				var callError *apiCallError
				if errors.As(err, &callError) && callError.attempted {
					item.Status = msg.PartUnknown
				}
			}
		}
		result.Items = append(result.Items, item)
	}
	return result, failure
}

// forwardNodes 保留节点来源与时间，所有临时文件归最终父操作统一清理。
func (i *instance) forwardNodes(ctx context.Context, items []msg.ForwardItem) (nodes []wireSegment, cleanups []func() error, err error) {
	for _, item := range items {
		data := map[string]any{}
		if item.PlatformMessageID != "" && len(item.Chain) == 0 {
			data["id"] = item.PlatformMessageID
		} else {
			content, cleanup, e := i.segments(ctx, item.Chain)
			cleanups = append(cleanups, cleanup...)
			if e != nil {
				return nodes, cleanups, e
			}
			data["content"] = content
			data["user_id"] = i.account.ID
			data["nickname"] = i.Account().DisplayName
			if item.Source != nil {
				if item.Source.ExternalUserID != "" {
					data["user_id"] = item.Source.ExternalUserID
				}
				if item.Source.DisplayName != "" {
					data["nickname"] = item.Source.DisplayName
				}
			}
			if !item.OccurredAt.IsZero() {
				data["time"] = item.OccurredAt.Unix()
			}
		}
		nodes = append(nodes, wireSegment{"node", data})
	}
	return
}

// decodeRawNode 保留未知字段和数字原文，只将数组 content 转成递归发送所需的片段。
func decodeRawNode(raw json.RawMessage) (map[string]any, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("decode raw node data: %w", err)
	}
	data := make(map[string]any, len(fields))
	for name, value := range fields {
		data[name] = value
	}
	content := fields["content"]
	if len(content) == 0 || content[0] != '[' {
		return data, nil
	}
	var children []struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(content, &children); err != nil {
		return nil, fmt.Errorf("decode raw node content: %w", err)
	}
	segments := make([]wireSegment, len(children))
	for n, child := range children {
		segments[n] = wireSegment{child.Type, child.Data}
	}
	data["content"] = segments
	return data, nil
}

// materializeMixed 原生纯 node 数组直接递归；混合内容仅上传子资源，最后发送父消息。
func (i *instance) materializeMixed(ctx context.Context, to string, segments []wireSegment, onUpload func(error, bool)) ([]wireSegment, error) {
	hasNode, hasOther := false, false
	for index, segment := range segments {
		if segment.Type != "node" {
			hasOther = true
			continue
		}
		hasNode = true
		data, constructed := segment.Data.(map[string]any)
		if !constructed {
			var err error
			data, err = decodeRawNode(segment.Data.(json.RawMessage))
			if err != nil {
				return segments, err
			}
		}
		if content, ok := data["content"].([]wireSegment); ok {
			next, err := i.materializeMixed(ctx, to, content, onUpload)
			if err != nil {
				return segments, err
			}
			data["content"] = next
		}
		segments[index].Data = data
	}
	if !hasNode || !hasOther {
		return segments, nil
	}
	var out []wireSegment
	for index := 0; index < len(segments); {
		if segments[index].Type != "node" {
			out = append(out, segments[index])
			index++
			continue
		}
		end := index + 1
		for end < len(segments) && segments[end].Type == "node" {
			end++
		}
		params, err := targetParams(to)
		if err != nil {
			return segments, err
		}
		delete(params, "message_type")
		delete(params, "user_id")
		params["messages"] = segments[index:end]
		var result struct {
			ResID     string `json:"res_id"`
			ForwardID string `json:"forward_id"`
		}
		if err = i.callAPI(ctx, "upload_forward_msg", params, &result, true); err != nil {
			onUpload(err, false)
			return segments, err
		}
		id := result.ResID
		if id == "" {
			id = result.ForwardID
		}
		if id == "" {
			err = errors.New("upload_forward_msg response has no resource ID")
			onUpload(err, true)
			return segments, err
		}
		onUpload(nil, true)
		out = append(out, wireSegment{"forward", map[string]string{"id": id}})
		index = end
	}
	return out, nil
}

func (i *instance) expandSendChain(ctx context.Context, to string, chain msg.Chain) (msg.Chain, error) {
	out := make(msg.Chain, 0, len(chain))
	for _, segment := range chain {
		if extension, ok := segment.(msg.ExtensionSegment); ok && extension.AdapterID == AdapterID && extension.Name == "onebot11.forward_ref" && extension.Version == 1 {
			var ref struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(extension.PayloadJSON), &ref); err != nil {
				return out, err
			}
			if ref.ID == "" {
				return out, errcode.InvalidArgument.New("missing forward resource ID")
			}
			items, err := i.ReadNodes(ctx, to, msg.ForwardSegment{PlatformForwardID: ref.ID})
			if err != nil {
				return out, err
			}
			segment = msg.ForwardSegment{Items: items}
			segment = queriedMediaForSend(segment)
		}
		if forward, ok := segment.(msg.ForwardSegment); ok {
			if forward.PlatformForwardID != "" && len(forward.Items) == 0 {
				return out, errcode.Unsupported.New("send forward requires items; use onebot11 ForwardRef for resources")
			}
			for index, item := range forward.Items {
				if len(item.Chain) > 0 {
					content, err := i.expandSendChain(ctx, to, item.Chain)
					if err != nil {
						return out, err
					}
					item.Chain = content
					item.PlatformMessageID = ""
					forward.Items[index] = item
				}
			}
			forward.PlatformForwardID = ""
			segment = forward
		}
		out = append(out, segment)
	}
	return out, nil
}

// queriedMediaForSend 只在 Adapter 内部复用本次平台查询来源，插件侧媒体仍经 Core 授权。
func queriedMediaForSend(segment msg.Segment) msg.Segment {
	source := func(value msg.MediaSource) msg.MediaSource {
		if value.Incoming != nil {
			return msg.MediaSource{PlatformFile: value.Incoming.Token}
		}
		return value
	}
	switch value := segment.(type) {
	case msg.ImageSegment:
		value.Source = source(value.Source)
		return value
	case msg.VideoSegment:
		value.Source = source(value.Source)
		return value
	case msg.VoiceSegment:
		value.Source = source(value.Source)
		return value
	case msg.FileSegment:
		value.Source = source(value.Source)
		return value
	case msg.ForwardSegment:
		for index, item := range value.Items {
			for n, child := range item.Chain {
				item.Chain[n] = queriedMediaForSend(child)
			}
			value.Items[index] = item
		}
		return value
	}
	return segment
}
