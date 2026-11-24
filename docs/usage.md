# Using OneBot 11

[简体中文](usage.zh-CN.md)

Use Core v0.3.0 or later, which includes ReadNodes and media authorization. See [README](../README.md) for installation. `go.mod` pins the SDK and packager.

## Configure an instance

Under Core's `adapters.onebot11`, set `package: "alonabot-adapter-onebot11"` and configure accounts in `instances`. Forward mode:

```json
{
  "mode": "forward",
  "ws_url": "ws://127.0.0.1:3001",
  "access_token": "the implementation's credential"
}
```

Forward startup calls `get_login_info`. Reconnection rechecks the account and reports a fault if it changed. Business requests already written to the connection return an unknown outcome. Configure the implementation to report array-form messages.

Reverse mode starts immediately and waits for a connection:

```json
{"mode":"reverse","self_id":"123456789","access_token":"platform credential"}
```

The implementation connects to `<Core base_url>/plugins/alonabot-adapter-onebot11/ws/<instance_id>`. Reverse `access_token` must be nonempty. Authenticate with `Authorization: Bearer <access_token>` or the `access_token` query parameter, and include `X-Self-ID: <self_id>` and `X-Client-Role: Universal`. Unknown instances return 404, bad credentials 401, wrong identity or role 403, and duplicate connections 409. Every connection is rechecked with `get_login_info`. Offline sends and actions return `UNAVAILABLE`.

## Deploy media access

HTTPS URLs pass directly to the implementation. Inbound media from the same instance reuses platform file identifiers; inline sources retain `base64://`. Uploaded `media_ref` supports two deployment paths:

- **Shared directory**: configure `"media_dir":{"local":"/adapter-visible/path","remote":"/implementation-visible/path"}`. Empty or omitted `remote` uses `local`. Both paths map to the same directory. The adapter writes random files, and the implementation reads them through `file://`. Files are removed when their last parent operation ends, including failure, cancellation, and unknown outcomes. Create the directory first and grant both processes read/write access.
- **Separate volumes or remote implementation**: omit `media_dir` and use Core temporary media URLs. Set `media_url_base` to a Core address reachable by the implementation, or leave it empty to use Core settings. Retrieval failures preserve the platform error.

Inbound image, video, record, and file become SDK media segments. HTTP/HTTPS `url` values support on-demand reads, and cancellation stops slow retrievals. Files with only `file_id` retain their segment, name, size, busid, and file_hash; reads query the URL in the originating group/private chat. Other media without a readable source is marked `Unreadable`. `PlatformPayload` always retains the complete original payload.

## Messages, events, and actions

Supported content includes text, images, videos, voice, files, mentions, references, faces, at-all, and merged forwards. Sending preflights the entire chain before executing in order: adjacent ordinary segments combine; videos, voice, files, and forwards form separate operations. A reference precedes the first ordinary message. Referencing a first operation that is a file or forward is rejected before any request. Generic Markdown, rich text, buttons, file captions, and other content without a direct mapping return `UNSUPPORTED`.

Merged forwards support files and recursive nodes; depth constraints follow the implementation's errors. Pure node arrays are submitted directly. Mixed body text and child forwards upload child resources with `upload_forward_msg` before sending the parent. Node files use native `file` segments to avoid separate file-upload actions creating extra chats.

`SendResult.PartResults` includes child-resource uploads and parent sends. Each physical operation retains segment indices, platform message ID, explicit rejection, unknown outcome, and unattempted state. Midway failures preserve successful receipts and return overall `PARTIAL`. File uploads and asynchronous responses can succeed without a message ID. Deletion calls `delete_msg` in input order.

### Segments

The `segment` package provides `Face`, `JSON`, `XML`, `Music`, `Share`, `Location`, `Contact`, `Empty`, and `Raw` payloads, plus `NewFace`, `AtAll`, `NewJSON`, `NewXML`, `NewMusic`, `NewShare`, `NewLocation`, `RPS`, `Dice`, `Shake`, and `NewContact` constructors. `Raw.Data` holds the complete data JSON string; SDK declarations transmit it as a string to preserve all fields and numeric precision.

`onebot11.face` and `onebot11.at_all` support inbound and outbound messages. json, xml, music, share, location, rps, dice, shake, contact, and markdown preserve full `data` in their declared extensions and can also be sent to the implementation. Unknown segments remain in the read-only `onebot11.segment` extension. Inbound forwards retain the platform resource ID and expand through `c.ReadNodes`; sends use SDK `ForwardItem`. `segment.ForwardRef(id)` is a OneBot resource reference that the adapter queries and rebuilds for the target before sending.

### Events

The `event` package provides `sdk.EventType` values and typed payloads for `Poke`, `LuckyKing`, `Honor`, `GroupAdmin`, `GroupBan`, `GroupUpload`, `FriendAdd`, `FriendRequest`, `GroupRequest`, `InputStatus`, `Notice`, and `Request`. Names use `onebot11.<snake_case>` and version 1. The last two retain complete unknown notice/request JSON strings in `Raw`.

Unknown reports without group or user IDs retain account identity with an empty `Session`. Group-file notices extract declared file fields, while implementation-specific fields remain in `PlatformPayload`. Standard messages, recalls, and member joins/leaves use standard SDK events.

### Typed client

Plugins import `github.com/AlonaBot/alonabot-adapter-onebot11/client`. This independent client package depends only on the SDK:

```go
bound, err := client.From(c)
if err != nil {
    return err
}
result, err := bound.GetGroupInfo(client.GetGroupInfoParams{NoCache: true})
if err != nil {
    return err
}
c.Logger().Info("group queried", "name", result.Value.GroupName)
```

Typed methods cover group management, friend/group approvals, group/member/friend/stranger queries, messages and forwards, group/private file URLs, pokes, and likes. Query `Result[T]` retains both `Value` and full JSON `Raw`. `Call("get_msg", params)` accepts only declared API names; Core applies declaration scope, instance, permission, and parent-call checks. Calls inherit handler cancellation and deadlines. Asynchronous actions retain response `data`, including objects, arrays, and `null`; asynchronous sends mark the message ID unavailable.

### Set or clear a member's special title

From a group handler, use the bound client to set a member's title:

```go
_, err = bound.SetGroupSpecialTitle(client.SetGroupSpecialTitleParams{
    UserID:       "123456789",
    SpecialTitle: "Helper",
})
if err != nil {
    return err
}
```

Set `SpecialTitle: ""` to clear the title. The group comes from the current trusted conversation. Omit `Duration` to use the OneBot 11 default of `-1` (permanent); for an explicit lifetime in seconds, pass a pointer:

```go
duration := int64(3600)
_, err = bound.SetGroupSpecialTitle(client.SetGroupSpecialTitleParams{
    UserID:       "123456789",
    SpecialTitle: "Helper",
    Duration:     &duration,
})
if err != nil {
    return err
}
```

Explicit duration values, including `0` and `-1`, are forwarded unchanged; lifetime support depends on the OneBot implementation. The action is declared as `onebot11.set_group_special_title` version 1. Backend failures, including insufficient permissions and unsupported operations, retain the response data and error details for the plugin.

## Mention names and quoted contents

Private titles use the sender nickname. Group names are queried once per process on first encounter; failures produce diagnostics and cache an empty result. Inbound mentions prefer the event sender and verified account nickname; other names are queried from group members or strangers to populate `MentionSegment.Display`. Repeated mentions in one event share a query. Failures log `onebot mention display lookup failed` and still deliver the original message and mention. Plugins can call `GetGroupMemberInfo` or `GetStrangerInfo` for fresh details.

References carry message handles, allowing the original contents to be queried:

```go
for _, part := range c.Event().Message.Chain {
    reference, ok := part.(message.ReferenceSegment)
    if !ok {
        continue
    }
    nodes, err := bound.ReadReply(reference)
    if err != nil {
        // nodes 保留错误发生前已经解析的内容。
        return err
    }
    for _, node := range nodes {
        c.Logger().Info("quoted message queried", "content", node.Chain.Text())
    }
}
```

Here `message` is `github.com/AlonaBot/AlonaBot/sdk/message`, and `bound` is the result of `client.From(c)`. Standard `GetForwardMessage` responses expose `ForwardNode` values through `Value.Message`. `node.Data.UserID` uses `json.Number` to preserve decimal-string or integer ID precision; `Nickname` retains the nickname, and `Content` retains raw JSON for a segment array or CQ-code string. SnowLuma `messages` responses expose typed messages through `Value.Messages`. Both retain full raw responses and extensions in `Raw`.

## Nodes, Markdown, and explicit text conversion

```go
nodes, err := c.ReadNodes(message.ForwardSegment{PlatformForwardID: resourceID})
```

Queries accept standard `message: []ForwardNode` and SnowLuma `messages: []Message`. CQ strings inside nodes are parsed only at the query boundary. Legacy XML forward cards expand by resource ID; a missing ID preserves the original segment and returns an error with its path. Resource IDs and message IDs remain distinct. Expanded nodes use Chain, while raw queries retain original message-reference IDs. Sender, time, and integer precision stay intact.

Child resources and message references expand recursively. Cycle errors contain the node path, and valid repeated sibling nodes remain. Query failures return already parsed valid nodes with the original error. Core registers returned media and authorizes it for the current call. `GetMessage` and `GetForwardMessage` still provide raw query access through `Result[T]{Value, Raw}`.

```go
groupFile, err := bound.GetGroupFileURL(client.GetGroupFileURLParams{FileID: fileID})
privateFile, err := bound.GetPrivateFileURL(client.GetPrivateFileURLParams{FileID: fileID, FileHash: fileHash})
```

File queries use the current trusted conversation. Group files accept `BusID: &busID`. Both methods preserve the complete response and error.

Inbound markdown text in `data.markdown` or `data.content` becomes a generic `message.MarkdownSegment`. Template fields remain in `onebot11.markdown` and event `PlatformPayload`. Explicit `md.PlainText()` produces sendable plain text. Existing `TextSegment.FormatMarkdown` keeps its original meaning.

```go
md := message.Markdown("**标题**\n正文")
plain := md.PlainText()
buttonText := client.ButtonsText(buttons)
sentences := client.SplitSentences(text)
```

`ButtonsText` retains labels and URLs, and marks callback buttons as unavailable for interaction. `SplitSentences` splits once at sentence-ending punctuation or newlines; concatenating all results reproduces the original text. Sending still uses `Send` / `Stream`.

## Known platform issue

In recorded SnowLuma v2.4.2 tests, some nested resources arrived as XML with `m_resid=""`, losing body text and child nodes. ReadNodes returns an explicit error in this case. The implementation issue affects complete reads for some reverse group sends and mixed body/child-forward messages.

Protocol fields follow [OneBot 11 segments](https://github.com/botuniverse/onebot-11/blob/master/message/segment.md) and [public APIs](https://github.com/botuniverse/onebot-11/blob/master/api/public.md). File uploads, poke, and markdown support depend on the implementation's response.

[Development and verification](development.md) · [Packaging and releases](releasing.md)
