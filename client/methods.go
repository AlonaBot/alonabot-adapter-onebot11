package client

import "encoding/json"

// SetGroupBanParams 保存 set_group_ban 的参数，数字 ID 采用十进制字符串。
type SetGroupBanParams struct {
	UserID   string `json:"user_id"`
	Duration int64  `json:"duration"`
}

// SetGroupBan 执行 set_group_ban 并保留完整平台结果。
func (c *Client) SetGroupBan(p SetGroupBanParams) (json.RawMessage, error) {
	return c.Call("set_group_ban", p)
}

// SetGroupWholeBanParams 保存 set_group_whole_ban 的参数，数字 ID 采用十进制字符串。
type SetGroupWholeBanParams struct {
	Enable bool `json:"enable"`
}

// SetGroupWholeBan 执行 set_group_whole_ban 并保留完整平台结果。
func (c *Client) SetGroupWholeBan(p SetGroupWholeBanParams) (json.RawMessage, error) {
	return c.Call("set_group_whole_ban", p)
}

// SetGroupKickParams 保存 set_group_kick 的参数，数字 ID 采用十进制字符串。
type SetGroupKickParams struct {
	UserID           string `json:"user_id"`
	RejectAddRequest bool   `json:"reject_add_request"`
}

// SetGroupKick 执行 set_group_kick 并保留完整平台结果。
func (c *Client) SetGroupKick(p SetGroupKickParams) (json.RawMessage, error) {
	return c.Call("set_group_kick", p)
}

// SetGroupCardParams 保存 set_group_card 的参数，数字 ID 采用十进制字符串。
type SetGroupCardParams struct {
	UserID string `json:"user_id"`
	Card   string `json:"card"`
}

// SetGroupCard 执行 set_group_card 并保留完整平台结果。
func (c *Client) SetGroupCard(p SetGroupCardParams) (json.RawMessage, error) {
	return c.Call("set_group_card", p)
}

// SetGroupAdminParams 保存 set_group_admin 的参数，数字 ID 采用十进制字符串。
type SetGroupAdminParams struct {
	UserID string `json:"user_id"`
	Enable bool   `json:"enable"`
}

// SetGroupAdmin 执行 set_group_admin 并保留完整平台结果。
func (c *Client) SetGroupAdmin(p SetGroupAdminParams) (json.RawMessage, error) {
	return c.Call("set_group_admin", p)
}

// SetGroupNameParams 保存 set_group_name 的参数，数字 ID 采用十进制字符串。
type SetGroupNameParams struct {
	GroupName string `json:"group_name"`
}

// SetGroupName 执行 set_group_name 并保留完整平台结果。
func (c *Client) SetGroupName(p SetGroupNameParams) (json.RawMessage, error) {
	return c.Call("set_group_name", p)
}

// SetGroupLeaveParams 保存 set_group_leave 的参数，数字 ID 采用十进制字符串。
type SetGroupLeaveParams struct {
	IsDismiss bool `json:"is_dismiss"`
}

// SetGroupLeave 执行 set_group_leave 并保留完整平台结果。
func (c *Client) SetGroupLeave(p SetGroupLeaveParams) (json.RawMessage, error) {
	return c.Call("set_group_leave", p)
}

// GetGroupInfoParams 保存 get_group_info 的参数，数字 ID 采用十进制字符串。
type GetGroupInfoParams struct {
	NoCache bool `json:"no_cache"`
}

// GetGroupInfo 执行 get_group_info 并保留完整平台结果。
func (c *Client) GetGroupInfo(p GetGroupInfoParams) (*Result[Group], error) {
	raw, err := c.Call("get_group_info", p)
	return decode[Group](raw, err)
}

// GetGroupMemberInfoParams 保存 get_group_member_info 的参数，数字 ID 采用十进制字符串。
type GetGroupMemberInfoParams struct {
	UserID  string `json:"user_id"`
	NoCache bool   `json:"no_cache"`
}

// GetGroupMemberInfo 执行 get_group_member_info 并保留完整平台结果。
func (c *Client) GetGroupMemberInfo(p GetGroupMemberInfoParams) (*Result[Member], error) {
	raw, err := c.Call("get_group_member_info", p)
	return decode[Member](raw, err)
}

// GetGroupMemberListParams 保存 get_group_member_list 的参数，数字 ID 采用十进制字符串。
type GetGroupMemberListParams struct {
}

// GetGroupMemberList 执行 get_group_member_list 并保留完整平台结果。
func (c *Client) GetGroupMemberList(p GetGroupMemberListParams) (*Result[[]Member], error) {
	raw, err := c.Call("get_group_member_list", p)
	return decode[[]Member](raw, err)
}

// GetMessageParams 保存 get_msg 的参数，数字 ID 采用十进制字符串。
type GetMessageParams struct {
	MessageID string `json:"message_id"`
}

// GetMessage 执行 get_msg 并保留完整平台结果。
func (c *Client) GetMessage(p GetMessageParams) (*Result[Message], error) {
	raw, err := c.Call("get_msg", p)
	return decode[Message](raw, err)
}

// GetForwardMessageParams 保存 get_forward_msg 的参数，数字 ID 采用十进制字符串。
type GetForwardMessageParams struct {
	ID string `json:"id"`
}

// GetForwardMessage 执行 get_forward_msg 并保留完整平台结果。
func (c *Client) GetForwardMessage(p GetForwardMessageParams) (*Result[ForwardMessage], error) {
	raw, err := c.Call("get_forward_msg", p)
	return decode[ForwardMessage](raw, err)
}

// PokeParams 保存 poke 的参数，数字 ID 采用十进制字符串。
type PokeParams struct {
	UserID string `json:"user_id"`
}

// Poke 执行 poke 并保留完整平台结果。
func (c *Client) Poke(p PokeParams) (json.RawMessage, error) { return c.Call("poke", p) }

// SetFriendAddRequestParams 保存 set_friend_add_request 的参数，数字 ID 采用十进制字符串。
type SetFriendAddRequestParams struct {
	Flag    string `json:"flag"`
	Approve bool   `json:"approve"`
	Remark  string `json:"remark"`
}

// SetFriendAddRequest 执行 set_friend_add_request 并保留完整平台结果。
func (c *Client) SetFriendAddRequest(p SetFriendAddRequestParams) (json.RawMessage, error) {
	return c.Call("set_friend_add_request", p)
}

// SetGroupAddRequestParams 保存 set_group_add_request 的参数，数字 ID 采用十进制字符串。
type SetGroupAddRequestParams struct {
	Flag    string `json:"flag"`
	SubType string `json:"sub_type"`
	Approve bool   `json:"approve"`
	Reason  string `json:"reason"`
}

// SetGroupAddRequest 执行 set_group_add_request 并保留完整平台结果。
func (c *Client) SetGroupAddRequest(p SetGroupAddRequestParams) (json.RawMessage, error) {
	return c.Call("set_group_add_request", p)
}

// GetStrangerInfoParams 保存 get_stranger_info 的参数，数字 ID 采用十进制字符串。
type GetStrangerInfoParams struct {
	UserID  string `json:"user_id"`
	NoCache bool   `json:"no_cache"`
}

// GetStrangerInfo 执行 get_stranger_info 并保留完整平台结果。
func (c *Client) GetStrangerInfo(p GetStrangerInfoParams) (*Result[User], error) {
	raw, err := c.Call("get_stranger_info", p)
	return decode[User](raw, err)
}

// GetFriendListParams 保存 get_friend_list 的参数，数字 ID 采用十进制字符串。
type GetFriendListParams struct {
}

// GetFriendList 执行 get_friend_list 并保留完整平台结果。
func (c *Client) GetFriendList(p GetFriendListParams) (*Result[[]User], error) {
	raw, err := c.Call("get_friend_list", p)
	return decode[[]User](raw, err)
}

// GetGroupListParams 保存 get_group_list 的参数，数字 ID 采用十进制字符串。
type GetGroupListParams struct {
}

// GetGroupList 执行 get_group_list 并保留完整平台结果。
func (c *Client) GetGroupList(p GetGroupListParams) (*Result[[]Group], error) {
	raw, err := c.Call("get_group_list", p)
	return decode[[]Group](raw, err)
}

// SendLikeParams 保存 send_like 的参数，数字 ID 采用十进制字符串。
type SendLikeParams struct {
	UserID string `json:"user_id"`
	Times  int64  `json:"times"`
}

// SendLike 执行 send_like 并保留完整平台结果。
func (c *Client) SendLike(p SendLikeParams) (json.RawMessage, error) {
	return c.Call("send_like", p)
}

// GetGroupFileURLParams 指定群文件及可选业务编号，会话由 Core 注入。
type GetGroupFileURLParams struct {
	FileID string `json:"file_id"`
	BusID  *int64 `json:"busid,omitempty"`
}

// GetPrivateFileURLParams 指定私聊文件及可选哈希，会话由 Core 注入。
type GetPrivateFileURLParams struct {
	FileID   string `json:"file_id"`
	FileHash string `json:"file_hash,omitempty"`
}

// FileURL 保存协议端提供的文件下载地址。
type FileURL struct {
	URL string `json:"url"`
}

// GetGroupFileURL 查询当前群的文件地址并保留原始响应。
func (c *Client) GetGroupFileURL(p GetGroupFileURLParams) (*Result[FileURL], error) {
	raw, err := c.Call("get_group_file_url", p)
	return decode[FileURL](raw, err)
}

// GetPrivateFileURL 查询当前私聊的文件地址并保留原始响应。
func (c *Client) GetPrivateFileURL(p GetPrivateFileURLParams) (*Result[FileURL], error) {
	raw, err := c.Call("get_private_file_url", p)
	return decode[FileURL](raw, err)
}
