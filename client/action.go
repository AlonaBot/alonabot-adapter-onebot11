// Package client 提供普通插件可调用的 OneBot 平台动作；运行时由 sdk 执行。
package client

import (
	"encoding/json"

	"github.com/AlonaBot/AlonaBot/sdk"
	"github.com/AlonaBot/AlonaBot/sdk/errcode"
)

// Client 绑定 Handler 的可信实例与调用上下文。
type Client struct {
	adapter *sdk.AdapterClient
	context *sdk.Context
}

// From 将平台动作绑定当前事件的实例与上下文。
func From(c *sdk.Context) (*Client, error) {
	bound, err := c.Adapter()
	if err != nil {
		return nil, err
	}
	if bound.AdapterID() != "onebot11" {
		return nil, errcode.InvalidArgument.New("expected onebot11 adapter")
	}
	return &Client{adapter: bound, context: c}, nil
}

// Call 执行已声明动作，完整保留响应 data 与错误。
func (c *Client) Call(name string, params any) (json.RawMessage, error) {
	if !Declared(name) {
		return nil, errcode.Unsupported.New("undeclared onebot action")
	}
	return c.adapter.Call("onebot11."+name, 1, params)
}

// Declared 报告 API 名是否属于公开的 v1 动作清单。
func Declared(name string) bool {
	switch name {
	case "get_group_file_url", "get_private_file_url", "set_group_ban", "set_group_whole_ban", "set_group_kick", "set_group_card", "set_group_admin", "set_group_name", "set_group_leave", "get_group_info", "get_group_member_info", "get_group_member_list", "get_msg", "get_forward_msg", "poke", "set_friend_add_request", "set_group_add_request", "get_stranger_info", "get_friend_list", "get_group_list", "send_like":
		return true
	}
	return false
}
