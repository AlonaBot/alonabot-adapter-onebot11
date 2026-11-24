package onebot11

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/AlonaBot/AlonaBot/sdk/adapter"
	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	"github.com/AlonaBot/alonabot-adapter-onebot11/client"
)

func actionDeclarations() []adapter.ActionDeclaration {
	return []adapter.ActionDeclaration{
		adapter.Action[client.GetGroupFileURLParams, map[string]any]("onebot11.get_group_file_url", 1, adapter.ActionScopeChat),
		adapter.Action[client.GetPrivateFileURLParams, map[string]any]("onebot11.get_private_file_url", 1, adapter.ActionScopeChat),
		adapter.Action[client.SetGroupBanParams, map[string]any]("onebot11.set_group_ban", 1, adapter.ActionScopeChat),
		adapter.Action[client.SetGroupWholeBanParams, map[string]any]("onebot11.set_group_whole_ban", 1, adapter.ActionScopeChat),
		adapter.Action[client.SetGroupKickParams, map[string]any]("onebot11.set_group_kick", 1, adapter.ActionScopeChat),
		adapter.Action[client.SetGroupCardParams, map[string]any]("onebot11.set_group_card", 1, adapter.ActionScopeChat),
		adapter.Action[client.SetGroupSpecialTitleParams, map[string]any]("onebot11.set_group_special_title", 1, adapter.ActionScopeChat),
		adapter.Action[client.SetGroupAdminParams, map[string]any]("onebot11.set_group_admin", 1, adapter.ActionScopeChat),
		adapter.Action[client.SetGroupNameParams, map[string]any]("onebot11.set_group_name", 1, adapter.ActionScopeChat),
		adapter.Action[client.SetGroupLeaveParams, map[string]any]("onebot11.set_group_leave", 1, adapter.ActionScopeChat),
		adapter.Action[client.GetGroupInfoParams, map[string]any]("onebot11.get_group_info", 1, adapter.ActionScopeChat),
		adapter.Action[client.GetGroupMemberInfoParams, map[string]any]("onebot11.get_group_member_info", 1, adapter.ActionScopeChat),
		adapter.Action[client.GetGroupMemberListParams, []map[string]any]("onebot11.get_group_member_list", 1, adapter.ActionScopeChat),
		adapter.Action[client.GetMessageParams, map[string]any]("onebot11.get_msg", 1, adapter.ActionScopeChat),
		adapter.Action[client.GetForwardMessageParams, map[string]any]("onebot11.get_forward_msg", 1, adapter.ActionScopeChat),
		adapter.Action[client.PokeParams, map[string]any]("onebot11.poke", 1, adapter.ActionScopeChat),
		adapter.Action[client.SetFriendAddRequestParams, map[string]any]("onebot11.set_friend_add_request", 1, adapter.ActionScopeAccount),
		adapter.Action[client.SetGroupAddRequestParams, map[string]any]("onebot11.set_group_add_request", 1, adapter.ActionScopeAccount),
		adapter.Action[client.GetStrangerInfoParams, map[string]any]("onebot11.get_stranger_info", 1, adapter.ActionScopeAccount),
		adapter.Action[client.GetFriendListParams, []map[string]any]("onebot11.get_friend_list", 1, adapter.ActionScopeAccount),
		adapter.Action[client.GetGroupListParams, []map[string]any]("onebot11.get_group_list", 1, adapter.ActionScopeAccount),
		adapter.Action[client.SendLikeParams, map[string]any]("onebot11.send_like", 1, adapter.ActionScopeAccount),
	}
}

// Action 只执行公开声明的动作，账号或会话身份来自 Core 路由。
func (i *instance) Action(ctx context.Context, to, name string, version uint32, raw json.RawMessage) (json.RawMessage, *errcode.Error) {
	if version != 1 {
		return nil, &errcode.Error{Code: errcode.Unsupported, Message: "unsupported action version"}
	}
	var params map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&params); err != nil {
		return nil, safePublic(err)
	}
	actionName := strings.TrimPrefix(name, "onebot11.")
	switch name {
	case "onebot11.get_group_file_url", "onebot11.get_private_file_url":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		kind := "group"
		key := "group_id"
		if actionName == "get_private_file_url" {
			kind = "private"
			key = "user_id"
		}
		if target["message_type"] != kind {
			return nil, errcode.InvalidArgument.New("file URL action requires matching session")
		}
		delete(params, "group_id")
		delete(params, "user_id")
		params[key] = target[key]
	case "onebot11.set_group_ban":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.set_group_whole_ban":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.set_group_kick":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.set_group_card", "onebot11.set_group_special_title":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.set_group_admin":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.set_group_name":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.set_group_leave":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.get_group_info":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.get_group_member_info":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.get_group_member_list":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] != "group" {
			return nil, &errcode.Error{Code: errcode.Unsupported, Message: "action requires group target"}
		}
		params["group_id"] = target["group_id"]
	case "onebot11.get_msg":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		_ = target
	case "onebot11.get_forward_msg":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		_ = target
	case "onebot11.poke":
		target, err := targetParams(to)
		if err != nil {
			return nil, safePublic(err)
		}
		if target["message_type"] == "group" {
			actionName = "group_poke"
			params["group_id"] = target["group_id"]
		} else {
			actionName = "friend_poke"
			params["user_id"] = target["user_id"]
		}
	case "onebot11.set_friend_add_request":
		if to != "" {
			return nil, &errcode.Error{Code: errcode.InvalidArgument, Message: "account action requires empty target"}
		}
	case "onebot11.set_group_add_request":
		if to != "" {
			return nil, &errcode.Error{Code: errcode.InvalidArgument, Message: "account action requires empty target"}
		}
	case "onebot11.get_stranger_info":
		if to != "" {
			return nil, &errcode.Error{Code: errcode.InvalidArgument, Message: "account action requires empty target"}
		}
	case "onebot11.get_friend_list":
		if to != "" {
			return nil, &errcode.Error{Code: errcode.InvalidArgument, Message: "account action requires empty target"}
		}
	case "onebot11.get_group_list":
		if to != "" {
			return nil, &errcode.Error{Code: errcode.InvalidArgument, Message: "account action requires empty target"}
		}
	case "onebot11.send_like":
		if to != "" {
			return nil, &errcode.Error{Code: errcode.InvalidArgument, Message: "account action requires empty target"}
		}
	default:
		return nil, &errcode.Error{Code: errcode.Unsupported, Message: "undeclared onebot action"}
	}
	var result json.RawMessage
	err := i.callAPI(ctx, actionName, params, &result, true)
	return result, safePublic(err)
}
