package onebot11

import (
	"github.com/AlonaBot/AlonaBot/sdk/adapter"
	"github.com/AlonaBot/alonabot-adapter-onebot11/event"
	"github.com/AlonaBot/alonabot-adapter-onebot11/segment"
)

func extensionDeclarations() []adapter.ExtensionDeclaration {
	out := []adapter.ExtensionDeclaration{
		adapter.Extension[segment.ForwardReference]("onebot11.forward_ref", 1).Describe("Query and rebuild a OneBot forward resource for this target; readonly preflight cannot inspect its contents.", segment.ForwardReference{ID: "resource-id"}),
		adapter.Extension[segment.Empty]("onebot11.at_all", 1).Describe("Mention all members of a group; platform permissions apply.", segment.Empty{}),
		adapter.Extension[segment.Face]("onebot11.face", 1).Describe("Send a OneBot platform face by its ID.", segment.Face{ID: "14"}),
	}
	for _, extension := range []struct{ name, description, data string }{
		{"json", "Send a JSON card with full implementation fields.", `{"data":"{\"app\":\"example\"}"}`},
		{"xml", "Send an XML card with full implementation fields.", `{"data":"<msg/>"}`},
		{"music", "Share a music track or custom music card.", `{"type":"qq","id":"123"}`},
		{"share", "Share a URL card with title and optional description/image.", `{"url":"https://example.com","title":"Example"}`},
		{"location", "Share a geographic location.", `{"lat":1.3,"lon":103.8,"title":"Singapore"}`},
		{"rps", "Send a rock-paper-scissors segment.", `{}`},
		{"dice", "Send a dice segment.", `{}`},
		{"shake", "Send a window shake.", `{}`},
		{"contact", "Share a user or group contact card.", `{"type":"qq","id":"7"}`},
		{"markdown", "Send implementation-specific Markdown data; availability depends on the OneBot backend.", `{"content":"**Details**"}`},
		{"segment", "Send a complete raw OneBot {type,data} segment envelope; availability depends on the OneBot backend.", `{"type":"poke","data":{"type":"1","id":"1"}}`},
	} {
		out = append(out, adapter.Extension[segment.Raw]("onebot11."+extension.name, 1).Describe(extension.description, segment.Raw{Data: extension.data}))
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
