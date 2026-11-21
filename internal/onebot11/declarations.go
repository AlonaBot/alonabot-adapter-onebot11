package onebot11

import (
	"github.com/AlonaBot/AlonaBot/sdk/adapter"
	"github.com/AlonaBot/alonabot-adapter-onebot11/event"
	"github.com/AlonaBot/alonabot-adapter-onebot11/segment"
)

func extensionDeclarations() []adapter.ExtensionDeclaration {
	out := []adapter.ExtensionDeclaration{adapter.Extension[segment.ForwardReference]("onebot11.forward_ref", 1), adapter.Extension[segment.Empty]("onebot11.at_all", 1), adapter.Extension[segment.Face]("onebot11.face", 1)}
	for _, name := range []string{"json", "xml", "music", "share", "location", "rps", "dice", "shake", "contact", "markdown", "segment"} {
		out = append(out, adapter.Extension[segment.Raw]("onebot11."+name, 1))
	}
	return out
}
func eventDeclarations() []adapter.EventDeclaration {
	return []adapter.EventDeclaration{
		adapter.Event[event.Target]("onebot11.poke", 1),
		adapter.Event[event.Target]("onebot11.lucky_king", 1),
		adapter.Event[event.HonorPayload]("onebot11.honor", 1),
		adapter.Event[event.AdminPayload]("onebot11.group_admin", 1),
		adapter.Event[event.BanPayload]("onebot11.group_ban", 1),
		adapter.Event[event.UploadPayload]("onebot11.group_upload", 1),
		adapter.Event[event.Empty]("onebot11.friend_add", 1),
		adapter.Event[event.FriendRequestPayload]("onebot11.friend_request", 1),
		adapter.Event[event.GroupRequestPayload]("onebot11.group_request", 1),
		adapter.Event[event.InputStatusPayload]("onebot11.input_status", 1),
		adapter.Event[event.GenericPayload]("onebot11.notice", 1),
		adapter.Event[event.GenericPayload]("onebot11.request", 1),
	}
}

func rawSegment(name string) bool {
	switch name {
	case "json", "xml", "music", "share", "location", "rps", "dice", "shake", "contact", "markdown":
		return true
	}
	return false
}
