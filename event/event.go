// Package event 发布 OneBot 扩展事件与类型化载荷。
package event

import (
	"github.com/AlonaBot/AlonaBot/sdk"
)

// Empty 表示平台没有附加字段的通知。
type Empty struct{}

// Target 保存被戳或运气王的号码。
type Target struct {
	TargetID string `json:"target_id"`
}

// HonorPayload 保存荣誉类别。
type HonorPayload struct {
	HonorType string `json:"honor_type"`
}

// AdminPayload 保存管理员变动。
type AdminPayload struct {
	SubType string `json:"sub_type"`
	UserID  string `json:"user_id"`
}

// BanPayload 保存禁言变动，Duration 单位为秒。
type BanPayload struct {
	SubType  string `json:"sub_type"`
	UserID   string `json:"user_id"`
	Duration int64  `json:"duration"`
}

// UploadedFile 保存群文件元数据。
type UploadedFile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Size  uint64 `json:"size"`
	BusID int64  `json:"busid"`
}

// UploadPayload 保存群文件上传通知。
type UploadPayload struct {
	File UploadedFile `json:"file"`
}

// FriendRequestPayload 保存好友审批所需的 flag。
type FriendRequestPayload struct {
	Comment string `json:"comment"`
	Flag    string `json:"flag"`
}

// GroupRequestPayload 保存入群审批类型和 flag。
type GroupRequestPayload struct {
	SubType string `json:"sub_type"`
	Comment string `json:"comment"`
	Flag    string `json:"flag"`
}

// InputStatusPayload 保存实现端上报的输入状态。
type InputStatusPayload struct {
	StatusText string `json:"status_text"`
	EventType  int64  `json:"event_type"`
}

// GenericPayload 保留未知通知或请求的完整 JSON 原文字符串。
type GenericPayload struct {
	Raw string `json:"raw"`
}

var (
	Poke          = sdk.AdapterEvent[Target]("onebot11.poke", 1)                         // Poke 订阅戳一戳。
	LuckyKing     = sdk.AdapterEvent[Target]("onebot11.lucky_king", 1)                   // LuckyKing 订阅运气王。
	Honor         = sdk.AdapterEvent[HonorPayload]("onebot11.honor", 1)                  // Honor 订阅荣誉变化。
	GroupAdmin    = sdk.AdapterEvent[AdminPayload]("onebot11.group_admin", 1)            // GroupAdmin 订阅管理员变化。
	GroupBan      = sdk.AdapterEvent[BanPayload]("onebot11.group_ban", 1)                // GroupBan 订阅禁言变化。
	GroupUpload   = sdk.AdapterEvent[UploadPayload]("onebot11.group_upload", 1)          // GroupUpload 订阅群文件上传。
	FriendAdd     = sdk.AdapterEvent[Empty]("onebot11.friend_add", 1)                    // FriendAdd 订阅新增好友。
	FriendRequest = sdk.AdapterEvent[FriendRequestPayload]("onebot11.friend_request", 1) // FriendRequest 订阅好友申请。
	GroupRequest  = sdk.AdapterEvent[GroupRequestPayload]("onebot11.group_request", 1)   // GroupRequest 订阅入群申请。
	InputStatus   = sdk.AdapterEvent[InputStatusPayload]("onebot11.input_status", 1)     // InputStatus 订阅输入状态。
	Notice        = sdk.AdapterEvent[GenericPayload]("onebot11.notice", 1)               // Notice 订阅未知通知。
	Request       = sdk.AdapterEvent[GenericPayload]("onebot11.request", 1)              // Request 订阅未知请求。
)
