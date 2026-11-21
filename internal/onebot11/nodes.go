package onebot11

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	msg "github.com/AlonaBot/AlonaBot/sdk/message"
)

// ReadNodes 查询转发资源或消息引用，祖先集合仅覆盖当前路径，允许兄弟重复引用。
func (i *instance) ReadNodes(ctx context.Context, to string, forward msg.ForwardSegment) ([]msg.ForwardItem, *errcode.Error) {
	if _, err := targetParams(to); err != nil {
		return nil, safePublic(err)
	}
	items, err := i.readNodes(ctx, to, forward, map[string]bool{}, "nodes")
	if err != nil {
		err = fmt.Errorf("nodes: %w", err)
	}
	return items, safePublic(err)
}

func (i *instance) readNodes(ctx context.Context, to string, forward msg.ForwardSegment, ancestors map[string]bool, path string) (items []msg.ForwardItem, err error) {
	if forward.PlatformForwardID != "" {
		key := "forward:" + forward.PlatformForwardID
		if ancestors[key] {
			return nil, fmt.Errorf("%s: cyclic forward resource %s", path, forward.PlatformForwardID)
		}
		ancestors[key] = true
		defer delete(ancestors, key)
		var raw json.RawMessage
		callErr := i.callAPI(ctx, "get_forward_msg", map[string]any{"id": forward.PlatformForwardID}, &raw, true)
		var response struct {
			Message  []json.RawMessage `json:"message"`
			Messages []json.RawMessage `json:"messages"`
		}
		if len(raw) > 0 {
			err = json.Unmarshal(raw, &response)
		}
		err = errors.Join(callErr, err)
		nodes := response.Message
		if nodes == nil {
			nodes = response.Messages
		}
		if len(nodes) == 0 && err == nil {
			return nil, fmt.Errorf("%s: missing forward nodes", path)
		}
		for index, rawNode := range nodes {
			nodePath := fmt.Sprintf("%s[%d]", path, index)
			item, nodeErr := i.readNode(ctx, to, rawNode, ancestors, nodePath)
			if len(item.Chain) > 0 {
				items = append(items, item)
			}
			if nodeErr != nil {
				return items, errors.Join(err, nodeErr)
			}
		}
		return items, err
	}
	for index, item := range forward.Items {
		nodePath := fmt.Sprintf("%s[%d]", path, index)
		if item.PlatformMessageID != "" {
			key := "message:" + item.PlatformMessageID
			if ancestors[key] {
				return items, fmt.Errorf("%s: cyclic message reference %s", nodePath, item.PlatformMessageID)
			}
			ancestors[key] = true
			var raw json.RawMessage
			callErr := i.callAPI(ctx, "get_msg", map[string]any{"message_id": item.PlatformMessageID}, &raw, true)
			var readErr error
			if len(raw) > 0 {
				item, readErr = i.readNode(ctx, to, raw, ancestors, nodePath)
			}
			delete(ancestors, key)
			if len(item.Chain) > 0 {
				items = append(items, item)
			}
			if joined := errors.Join(callErr, readErr); joined != nil {
				return items, fmt.Errorf("%s: %w", nodePath, joined)
			}
		} else {
			chain, readErr := i.expandQueryChain(ctx, to, item.Chain, ancestors, nodePath)
			item.Chain = chain
			if len(item.Chain) > 0 {
				items = append(items, item)
			}
			if readErr != nil {
				return items, readErr
			}
		}
	}
	return items, nil
}

func (i *instance) readNode(ctx context.Context, to string, raw json.RawMessage, ancestors map[string]bool, path string) (item msg.ForwardItem, err error) {
	var node struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err = json.Unmarshal(raw, &node); err != nil {
		return item, fmt.Errorf("%s: %w", path, err)
	}
	if node.Type == "node" {
		raw = node.Data
	}
	var data struct {
		ID        json.Number `json:"id"`
		MessageID json.Number `json:"message_id"`
		UserID    json.Number `json:"user_id"`
		Nickname  string      `json:"nickname"`
		Name      string      `json:"name"`
		Time      int64       `json:"time"`
		Sender    struct {
			UserID   json.Number `json:"user_id"`
			Nickname string      `json:"nickname"`
			Card     string      `json:"card"`
		} `json:"sender"`
		Content json.RawMessage `json:"content"`
		Message json.RawMessage `json:"message"`
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		return item, fmt.Errorf("%s: %w", path, err)
	}
	content := data.Content
	if len(content) == 0 {
		content = data.Message
	}
	if len(content) == 0 && data.ID != "" {
		items, readErr := i.readNodes(ctx, to, msg.ForwardSegment{Items: []msg.ForwardItem{{PlatformMessageID: string(data.ID)}}}, ancestors, path+".id")
		if len(items) > 0 {
			item = items[0]
		}
		return item, readErr
	}
	user := string(data.UserID)
	if user == "" {
		user = string(data.Sender.UserID)
	}
	name := data.Nickname
	if name == "" {
		name = data.Name
	}
	if name == "" {
		name = data.Sender.Card
	}
	if name == "" {
		name = data.Sender.Nickname
	}
	item.Source = &msg.ForwardSource{DisplayName: name, ExternalUserID: user}
	if data.Time != 0 {
		item.OccurredAt = time.Unix(data.Time, 0)
	}
	content, parseErr := queryContent(content)
	if content == nil {
		return item, fmt.Errorf("%s.content: %w", path, parseErr)
	}
	item.Chain, err = i.queryChain(ctx, content, to, ancestors, path+".content")
	err = errors.Join(parseErr, err)
	if err == nil && len(item.Chain) == 0 {
		err = errors.New("node content is empty")
	}
	if err != nil {
		return item, fmt.Errorf("%s.content: %w", path, err)
	}
	return item, err
}

func (i *instance) expandQueryChain(ctx context.Context, to string, chain msg.Chain, ancestors map[string]bool, path string) (msg.Chain, error) {
	for index, segment := range chain {
		if forward, ok := segment.(msg.ForwardSegment); ok && len(forward.Items) == 0 {
			items, err := i.readNodes(ctx, to, forward, ancestors, fmt.Sprintf("%s[%d]", path, index))
			forward.Items = items
			chain[index] = forward
			if err != nil {
				if len(items) == 0 {
					return chain[:index], err
				}
				return chain[:index+1], err
			}
		}
	}
	return chain, nil
}

// queryContent 仅为查询响应解析 CQ 字符串；实时上报仍要求数组。
func queryContent(raw json.RawMessage) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, errors.New("missing node content")
	}
	if raw[0] != '"' {
		return raw, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil, err
	}
	var segments []wireSegment
	unescape := func(value string) string {
		return strings.NewReplacer("&#44;", ",", "&#91;", "[", "&#93;", "]", "&amp;", "&").Replace(value)
	}
	for len(text) > 0 {
		start := strings.Index(text, "[CQ:")
		if start < 0 {
			segments = append(segments, wireSegment{"text", map[string]string{"text": unescape(text)}})
			break
		}
		if start > 0 {
			segments = append(segments, wireSegment{"text", map[string]string{"text": unescape(text[:start])}})
		}
		end := strings.IndexByte(text[start:], ']')
		if end < 0 {
			data, encodeErr := json.Marshal(segments)
			return data, errors.Join(errors.New("unterminated CQ segment"), encodeErr)
		}
		end += start
		fields := strings.Split(text[start+4:end], ",")
		data := map[string]string{}
		for _, field := range fields[1:] {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				encoded, encodeErr := json.Marshal(segments)
				return encoded, errors.Join(fmt.Errorf("invalid CQ field %s", strconv.Quote(field)), encodeErr)
			}
			data[key] = unescape(value)
		}
		segments = append(segments, wireSegment{fields[0], data})
		text = text[end+1:]
	}
	if segments == nil {
		segments = []wireSegment{}
	}
	return json.Marshal(segments)
}

func (i *instance) queryChain(ctx context.Context, raw json.RawMessage, to string, ancestors map[string]bool, path string) (chain msg.Chain, err error) {
	var segments []json.RawMessage
	if err = json.Unmarshal(raw, &segments); err != nil {
		return chain, err
	}
	for index, rawSegment := range segments {
		var header struct {
			Type string `json:"type"`
		}
		if err = json.Unmarshal(rawSegment, &header); err != nil {
			return chain, err
		}
		if header.Type == "xml" {
			forward, recognized, parseErr := queryXMLForward(rawSegment)
			if recognized {
				if parseErr != nil {
					part, decodeErr := messageChain(append(append([]byte{'['}, rawSegment...), ']'), to, i.id)
					return append(chain, part...), fmt.Errorf("%s[%d]: %w", path, index, errors.Join(parseErr, decodeErr))
				}
				rawSegment, err = json.Marshal(wireSegment{"forward", map[string]string{"id": forward}})
				if err != nil {
					return chain, err
				}
				header.Type = "forward"
			}
		}
		if header.Type == "node" {
			item, readErr := i.readNode(ctx, to, rawSegment, ancestors, fmt.Sprintf("%s[%d]", path, index))
			if len(item.Chain) > 0 {
				chain = append(chain, msg.ForwardSegment{Items: []msg.ForwardItem{item}})
			}
			if readErr != nil {
				return chain, readErr
			}
		} else {
			part, readErr := messageChain(append(append([]byte{'['}, rawSegment...), ']'), to, i.id)
			if readErr == nil {
				part, readErr = i.expandQueryChain(ctx, to, part, ancestors, fmt.Sprintf("%s[%d]", path, index))
			}
			chain = append(chain, part...)
			if readErr != nil {
				return chain, fmt.Errorf("%s[%d]: %w", path, index, readErr)
			}
		}
	}
	return chain, nil
}

// queryXMLForward 识别查询中的旧式转发卡片，缺少资源 ID 时保留原段并报告展开失败。
func queryXMLForward(raw json.RawMessage) (string, bool, error) {
	var segment struct {
		Data struct {
			Data string `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &segment); err != nil {
		return "", false, err
	}
	decoder := xml.NewDecoder(strings.NewReader(segment.Data.Data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", false, nil
		}
		root, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if root.Name.Local != "msg" {
			return "", false, nil
		}
		attributes := map[string]string{}
		for _, attr := range root.Attr {
			attributes[attr.Name.Local] = attr.Value
		}
		if attributes["serviceID"] != "35" || attributes["action"] != "viewMultiMsg" {
			return "", false, nil
		}
		id := attributes["m_resid"]
		if id == "" {
			return "", true, errcode.Unsupported.New("forward XML card has no resource ID; protocol implementation returned an incomplete node")
		}
		return id, true, nil
	}
}
