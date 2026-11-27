package onebot11

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	msg "github.com/AlonaBot/AlonaBot/sdk/message"
)

// extensionSegment 在出站边界解析声明的载荷；原始 JSON 保留数字精度与扩展字段。
func extensionSegment(extension msg.ExtensionSegment) (wireSegment, error) {
	name, ok := strings.CutPrefix(extension.Name, "onebot11.")
	if !ok {
		return wireSegment{}, errcode.Unsupported.New(extensionRule + ": unsupported onebot segment " + extension.Name)
	}
	switch name {
	case "at_all":
		if !json.Valid([]byte(extension.PayloadJSON)) {
			return wireSegment{}, errcode.InvalidArgument.New(extensionRule + ": invalid extension JSON")
		}
		return wireSegment{"at", map[string]any{"qq": "all"}}, nil
	case "face":
		if !json.Valid([]byte(extension.PayloadJSON)) {
			return wireSegment{}, errcode.InvalidArgument.New(extensionRule + ": invalid extension JSON")
		}
		return wireSegment{"face", json.RawMessage(extension.PayloadJSON)}, nil
	case "forward_ref":
		var ref struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(extension.PayloadJSON), &ref); err != nil {
			return wireSegment{}, fmt.Errorf("%s: decode forward reference: %w", extensionRule, err)
		}
		if ref.ID == "" {
			return wireSegment{}, errcode.InvalidArgument.New(extensionRule + ": missing forward resource ID")
		}
		return wireSegment{"forward", map[string]any{"id": ref.ID}}, nil
	case "segment":
	default:
		if !rawSegment(name) {
			return wireSegment{}, errcode.Unsupported.New(extensionRule + ": unsupported onebot segment " + extension.Name)
		}
	}
	var raw struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal([]byte(extension.PayloadJSON), &raw); err != nil {
		return wireSegment{}, fmt.Errorf("%s: decode extension: %w", extensionRule, err)
	}
	if !json.Valid([]byte(raw.Data)) {
		return wireSegment{}, errcode.InvalidArgument.New(extensionRule + ": invalid raw segment data JSON")
	}
	if name == "segment" {
		var envelope struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal([]byte(raw.Data), &envelope); err != nil {
			return wireSegment{}, fmt.Errorf("%s: decode segment envelope: %w", extensionRule, err)
		}
		if envelope.Type == "" || len(envelope.Data) == 0 || envelope.Data[0] != '{' {
			return wireSegment{}, errcode.InvalidArgument.New(extensionRule + ": segment requires type and object data")
		}
		return wireSegment{envelope.Type, envelope.Data}, nil
	}
	return wireSegment{name, json.RawMessage(raw.Data)}, nil
}
