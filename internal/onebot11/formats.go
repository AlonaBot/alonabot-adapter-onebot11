package onebot11

import (
	"context"
	"encoding/json"

	"github.com/AlonaBot/AlonaBot/sdk/errcode"
	msg "github.com/AlonaBot/AlonaBot/sdk/message"
)

const (
	plainTextRule   = "onebot11.plain_text"
	mediaRule       = "onebot11.media"
	forwardRule     = "onebot11.forward_nodes"
	replyRule       = "onebot11.reply_first"
	extensionRule   = "onebot11.extensions"
	unsupportedRule = "onebot11.unsupported_formats"
)

func messageFormats() []msg.MessageFormat {
	formats := msg.StandardFormats()
	for n := range formats {
		format := &formats[n]
		format.Status = msg.SupportSupported
		switch format.Name {
		case "text":
			format.Description = "Plain text only; text entities and Markdown text format are unsupported (" + plainTextRule + ")."
			format.ExampleJSON = json.RawMessage(`{"parts":[{"type":"text","text":"Details"},{"type":"image","source":{"https_url":"https://example.com/image.jpg"}}]}`)
		case "markdown", "buttons":
			format.Status = msg.SupportUnsupported
			format.Description += " Unsupported by OneBot 11 standard messages (" + unsupportedRule + ")."
		case "image", "video", "voice", "file":
			format.Sources = []string{"https_url", "media_ref", "incoming", "inline_base64", "platform_file"}
			format.Description += " Core authorizes incoming/media_ref and prepares them for send; platform_file is adapter-only. Captions become following text segments (" + mediaRule + ")."
			if format.Name == "image" {
				format.Description += " Image alt is unsupported."
			}
			if format.Name == "file" {
				format.Description = "File upload with filename; caption is unsupported, MIME does not change routing. Inside forward nodes files stay native file segments. Core authorizes incoming/media_ref; platform_file is adapter-only (" + mediaRule + ")."
			}
		case "quote":
			format.Description += " At most one quote; the first operation must be an ordinary message (text/image/mention/extensions, video or voice), not a file or forward. Quotes inside forward nodes are unsupported (" + replyRule + ")."
		case "forward":
			format.Description += " Bot-authored nodes use the bot account and nickname; SDK source/time metadata is preserved when provided. Nodes can contain nested forwards and files, but no quotes. Pure node arrays send directly; body mixed with child nodes uploads child resources before the parent. Each top-level forward is a separate operation (" + forwardRule + ")."
		}
	}
	return formats
}

func combinationRules() []msg.CombinationRule {
	return []msg.CombinationRule{
		{Name: plainTextRule, Formats: []string{"text"}, Description: "Text must be plain and have no entities.", ExampleJSON: json.RawMessage(`{"parts":[{"type":"text","text":"Details"},{"type":"mention","user_id":"7"}]}`)},
		{Name: mediaRule, Formats: []string{"text", "image", "video", "voice", "file"}, Description: "Adjacent ordinary segments combine. Video, voice, file and forward each start a separate operation. Image/video/voice captions become following text; file captions and image alt are unsupported.", ExampleJSON: json.RawMessage(`{"parts":[{"type":"image","source":{"https_url":"https://example.com/image.jpg"},"caption":"Image"},{"type":"file","source":{"media_ref":"<uploaded-media-ref>"},"filename":"report.pdf"}]}`)},
		{Name: forwardRule, Formats: []string{"forward", "text", "image", "video", "voice", "file", "mention", "quote", "onebot11.forward_ref"}, Description: "Mix bot-authored nodes and confirmed messages from this target and instance. Nested forwards/files are supported; nested quotes are unsupported. Mixed body/child nodes upload child resources before the parent. A forward_ref resource needs platform expansion, so readonly preflight is unknown for its contents.", ExampleJSON: json.RawMessage(`{"parts":[{"type":"forward","nodes":[{"parts":[{"type":"text","text":"Summary"}]},{"message_ref":"<confirmed-message-ref>"}]}]}`)},
		{Name: replyRule, Formats: []string{"quote", "text", "image", "video", "voice", "file", "forward", "onebot11.forward_ref"}, Description: "One quote from this target/instance attaches to the first ordinary send. File/forward as first operation and nested quotes are unsupported.", ExampleJSON: json.RawMessage(`{"parts":[{"type":"quote","message_ref":"<confirmed-message-ref>"},{"type":"text","text":"Reply"}]}`)},
		{Name: extensionRule, Formats: []string{"onebot11.segment"}, Description: "Declared version-1 extensions preserve their complete data. onebot11.segment takes Raw.data containing one complete {type,data} envelope and sends that envelope unchanged; implementation support is determined by its response.", ExampleJSON: json.RawMessage(`{"parts":[{"type":"onebot11.segment","version":1,"payload":{"data":"{\"type\":\"poke\",\"data\":{\"type\":\"1\",\"id\":\"1\"}}"}}]}`)},
		{Name: unsupportedRule, Formats: []string{"markdown", "buttons"}, Description: "Standard Markdown and buttons are unsupported, including inside forward nodes. The platform-specific onebot11.markdown extension remains available."},
	}
}

// CheckMessage 复用纯发送计划；资源引用缺少平台展开内容时保留 unknown。
func (i *instance) CheckMessage(ctx context.Context, to string, chain msg.Chain) (msg.SupportStatus, *errcode.Error) {
	if err := ctx.Err(); err != nil {
		return msg.SupportUnknown, safePublic(err)
	}
	if _, err := i.plan(to, chain); err != nil {
		return msg.SupportUnsupported, safePublic(err)
	}
	if hasForwardReference(chain) {
		return msg.SupportUnknown, nil
	}
	return msg.SupportSupported, nil
}

func hasForwardReference(chain msg.Chain) bool {
	for _, segment := range chain {
		switch value := segment.(type) {
		case msg.ExtensionSegment:
			if value.Name == "onebot11.forward_ref" {
				return true
			}
		case msg.ForwardSegment:
			for _, item := range value.Items {
				if hasForwardReference(item.Chain) {
					return true
				}
			}
		}
	}
	return false
}
