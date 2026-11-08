// Package perm 提供 OneBot 11 的平台权限名称。
//
// 插件以 sdk.PermissionName(perm.GroupOwner) 等方式引用权限；本包只含常量，
// 避免插件链接 Adapter 运行时代码。
package perm

const (
	// GroupOwner 标记群主身份，并隐含 GroupAdmin。
	GroupOwner = "onebot11.group_owner"
	// GroupAdmin 标记群管理员身份，并隐含 chat_admin。
	GroupAdmin = "onebot11.group_admin"
	// PrivateFriend 标记好友私聊。
	PrivateFriend = "onebot11.private_friend"
)
