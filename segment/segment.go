// Package segment 发布 OneBot 专有消息片段载荷。
package segment

import (
	"encoding/json"

	msg "github.com/AlonaBot/AlonaBot/sdk/message"
)

// Empty 表示没有附加字段的片段。
type Empty struct{}

// Face 保存平台表情编号。
type Face struct {
	ID string `json:"id"`
}

// Raw 保留平台专有片段的完整 data JSON 字符串，包括实现端扩展字段。
type Raw struct {
	Data string `json:"data"`
}

// JSON 保存 JSON 卡片文本。
type JSON struct {
	Data string `json:"data"`
}

// XML 保存 XML 卡片文本。
type XML struct {
	Data string `json:"data"`
}

// Music 保存音乐平台曲目或自定义音乐字段。
type Music struct {
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	URL     string `json:"url,omitempty"`
	Audio   string `json:"audio,omitempty"`
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
	Image   string `json:"image,omitempty"`
}

// Share 保存链接分享卡片。
type Share struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
	Image   string `json:"image,omitempty"`
}

// Location 保存经纬度及地点说明。
type Location struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Title   string  `json:"title,omitempty"`
	Content string  `json:"content,omitempty"`
}

// Contact 保存用户或群名片。
type Contact struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// ForwardReference 保存仅由 OneBot 展开的资源 ID。
type ForwardReference struct {
	ID string `json:"id"`
}

func raw(name string, value any) msg.Segment {
	data, err := json.Marshal(value)
	if err != nil {
		return msg.Extension("onebot11", "onebot11."+name, 1, value)
	}
	return msg.Extension("onebot11", "onebot11."+name, 1, Raw{Data: string(data)})
}

// NewFace 构造平台表情。
func NewFace(id string) msg.Segment {
	return msg.Extension("onebot11", "onebot11.face", 1, Face{ID: id})
}

// AtAll 构造全体提及。
func AtAll() msg.Segment { return msg.Extension("onebot11", "onebot11.at_all", 1, Empty{}) }

// NewJSON 构造 JSON 卡片。
func NewJSON(data string) msg.Segment { return raw("json", JSON{Data: data}) }

// NewXML 构造 XML 卡片。
func NewXML(data string) msg.Segment { return raw("xml", XML{Data: data}) }

// NewMusic 构造音乐分享。
func NewMusic(value Music) msg.Segment { return raw("music", value) }

// NewShare 构造链接分享。
func NewShare(value Share) msg.Segment { return raw("share", value) }

// NewLocation 构造位置片段。
func NewLocation(value Location) msg.Segment { return raw("location", value) }

// RPS 构造猜拳片段。
func RPS() msg.Segment { return raw("rps", Empty{}) }

// Dice 构造骰子片段。
func Dice() msg.Segment { return raw("dice", Empty{}) }

// Shake 构造窗口抖动片段。
func Shake() msg.Segment { return raw("shake", Empty{}) }

// NewContact 构造名片。
func NewContact(value Contact) msg.Segment { return raw("contact", value) }

// ForwardRef 构造平台资源引用，发送时按目标展开并重建。
func ForwardRef(id string) msg.Segment {
	return msg.Extension("onebot11", "onebot11.forward_ref", 1, ForwardReference{ID: id})
}
