// schema 命令生成 Core 的实例配置字段描述。
package main

import (
	"encoding/json"
	"os"
)

func main() {
	fields := map[string]any{
		"mode":           map[string]any{"type": "string", "options": []string{"forward", "reverse"}, "default": "forward"},
		"ws_url":         map[string]any{"type": "string", "condition": map[string]any{"mode": "forward"}},
		"self_id":        map[string]any{"type": "string", "condition": map[string]any{"mode": "reverse"}},
		"access_token":   map[string]any{"type": "string", "secret": true, "description": "reverse 模式必填；forward 模式可选，启动时按 mode 校验"},
		"media_url_base": map[string]any{"type": "string"},
		"media_dir":      map[string]any{"type": "object", "items": map[string]any{"local": map[string]any{"type": "string"}, "remote": map[string]any{"type": "string"}}},
	}
	raw, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("adapter_config_schema.json", append(raw, '\n'), 0644); err != nil {
		panic(err)
	}
}
