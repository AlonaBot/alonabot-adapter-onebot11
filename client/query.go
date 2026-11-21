package client

import (
	"encoding/json"
	"errors"
)

// Result 同时保留类型化查询结果和平台完整 JSON，包括实现端额外字段。
type Result[T any] struct {
	Value T
	Raw   json.RawMessage
}

// User 保存好友或陌生人资料，数字 ID 保留 int64 精度。
type User struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Remark   string `json:"remark"`
	Card     string `json:"card,omitempty"`
	Sex      string `json:"sex"`
	Age      int64  `json:"age"`
}

// Group 保存群资料。
type Group struct {
	GroupID        int64  `json:"group_id"`
	GroupName      string `json:"group_name"`
	MemberCount    int64  `json:"member_count"`
	MaxMemberCount int64  `json:"max_member_count"`
}

// Member 保存群成员资料。
type Member struct {
	User
	GroupID         int64  `json:"group_id"`
	Card            string `json:"card"`
	Role            string `json:"role"`
	Area            string `json:"area"`
	JoinTime        int64  `json:"join_time"`
	LastSentTime    int64  `json:"last_sent_time"`
	Level           string `json:"level"`
	Title           string `json:"title"`
	TitleExpireTime int64  `json:"title_expire_time"`
	CardChangeable  bool   `json:"card_changeable"`
	Unfriendly      bool   `json:"unfriendly"`
}

// Message 保存查询消息，Message 保留原始段数组与实现端扩展。
type Message struct {
	MessageID   int64           `json:"message_id"`
	RealID      int64           `json:"real_id"`
	Time        int64           `json:"time"`
	MessageType string          `json:"message_type"`
	Group       bool            `json:"group"`
	Sender      User            `json:"sender"`
	Message     json.RawMessage `json:"message"`
}

// ForwardMessage 保存标准节点序列和实现端提供的消息序列。
type ForwardMessage struct {
	Message  []ForwardNode `json:"message"`  // OneBot 11 标准响应
	Messages []Message     `json:"messages"` // SnowLuma 响应
}

// ForwardNode 保存合并转发响应中的 node 消息段。
type ForwardNode struct {
	Type string          `json:"type"`
	Data ForwardNodeData `json:"data"`
}

// ForwardNodeData 保存转发节点的发送者及原始内容。
type ForwardNodeData struct {
	UserID    json.Number     `json:"user_id"` // 接受十进制字符串或整数，保留数字精度
	Nickname  string          `json:"nickname"`
	Name      string          `json:"name"`
	Sender    User            `json:"sender"`
	Time      int64           `json:"time"`
	ID        json.Number     `json:"id"`
	MessageID json.Number     `json:"message_id"`
	Content   json.RawMessage `json:"content"` // 消息段数组或 CQ 码字符串
}

func decode[T any](raw json.RawMessage, callError error) (*Result[T], error) {
	result := &Result[T]{Raw: raw}
	if len(raw) == 0 {
		return result, callError
	}
	err := json.Unmarshal(raw, &result.Value)
	return result, errors.Join(callError, err)
}
