package client

import (
	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	msg "github.com/AlonaBot/AlonaBot/sdk/message"
)

// ReadReply 经 Core 查询引用内容，并取得当前调用有权读取的节点媒体。
func (c *Client) ReadReply(reference msg.ReferenceSegment) ([]msg.ForwardItem, error) {
	session, id, instance, err := msg.ParseRef(reference.Ref)
	if err != nil {
		return nil, err
	}
	if session != c.adapter.Session() || (instance != "" && instance != c.adapter.InstanceID()) {
		return nil, errcode.InvalidArgument.New("reference must belong to the current session and instance")
	}
	return c.context.ReadNodes(msg.ForwardSegment{Items: []msg.ForwardItem{{PlatformMessageID: id}}})
}
