# AlonaBot OneBot 11 Adapter

[![Go component](https://github.com/AlonaBot/alonabot-adapter-onebot11/actions/workflows/ci.yml/badge.svg)](https://github.com/AlonaBot/alonabot-adapter-onebot11/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

通过 AlonaBot SDK 接入 OneBot 11，实现正向、反向 Universal WebSocket、多实例账号绑定、断线重连、消息与通知接收、消息发送和平台动作。组件版本为 `0.1.1`。

## 配置实例

在 Core 的 `adapters.onebot11` 下指定 `package: "alonabot-adapter-onebot11"`，在 `instances` 中配置各实例：

```json
{
  "mode": "forward",
  "ws_url": "ws://127.0.0.1:3001",
  "access_token": "平台端设置的凭证"
}
```

正向连接启动时调用 `get_login_info`。重连后复核账号，账号改变时报告故障；已经写出的业务请求按结果未知返回。实现端应上报数组格式的消息。

反向连接配置如下，实例启动即返回并等待实现端连入：

```json
{"mode":"reverse","self_id":"123456789","access_token":"平台凭证"}
```

实现端连接 `<Core base_url>/plugins/alonabot-adapter-onebot11/ws/<instance_id>`，reverse 的 `access_token` 必须非空。实现端可用 `Authorization: Bearer <access_token>` 或 `access_token` 查询参数鉴权，并携带`X-Self-ID: <self_id>` 和 `X-Client-Role: Universal`。未知实例返回 404，凭证错误返回 401，身份或角色错误返回 403，重复连接返回 409。每次连入均通过 `get_login_info` 复核；离线期间发送与动作返回 `UNAVAILABLE`。

## 媒体部署

HTTPS URL 直接交给协议端，同实例的入站媒体使用平台文件标识复用。内联来源保留 `base64://`。上传的 `media_ref` 可选两条路径：

- **共享目录**：配置 `"media_dir":{"local":"/Adapter可见目录","remote":"/协议端可见目录"}`。`remote` 省略或留空时沿用 `local`。两个路径映射同一目录；Adapter 写随机文件，协议端通过 `file://` 读取，最后一次使用它的父操作结束后删除，包括失败、取消和结果未知。目录需预先存在，并允许两个进程读写。
- **隔离卷或远程协议端**：省略 `media_dir`，通过 Core 的临时媒体 URL 取件。配置 `media_url_base` 为协议端可达的 Core 地址；留空沿用 Core 配置。取件失败保留平台错误。

入站 image、video、record 和 file 转成 SDK 媒体片段。`url` 为 HTTP/HTTPS 时支持按需读取，调用取消会终止慢取件；文件只有 `file_id` 时保留片段及名称、大小、busid、file_hash，在读取时按来源群/私聊查询地址。其他缺少可读来源的媒体标记 `Unreadable`。完整原文始终保存在 `PlatformPayload`。

## 消息、事件与动作

支持文本、图片、视频、语音、文件、提及、引用、表情、全体提及与合并转发。发送先预检整链，再按顺序执行：连续普通片段合并，视频、语音、文件、转发各自分开。引用放在首条普通消息之前；文件或转发作为首操作时带引用会在请求前拒绝。通用 Markdown、富文本、按钮和文件说明等缺乏直接映射的内容返回 `UNSUPPORTED`。合并转发支持文件及递归节点，深度限制遵循当前协议端返回的错误；纯 node 数组直接提交，正文与子转发混合时先通过 `upload_forward_msg` 上传子资源，再发送父消息。节点文件随原生 `file` 段提交，避免独立文件上传动作产生额外聊天。

`SendResult.PartResults` 包含子资源上传与父消息发送，保留每个物理操作对应的片段索引、平台消息 ID、明确拒绝、结果未知及未尝试状态。中途失败时保留已经成功的回执，整体返回 `PARTIAL`；文件上传和异步响应可以成功且没有消息 ID。撤回按输入顺序调用 `delete_msg`。

`segment` 包提供 `Face`、`JSON`、`XML`、`Music`、`Share`、`Location`、`Contact`、`Empty` 和 `Raw` 载荷，以及 `NewFace`、`AtAll`、`NewJSON`、`NewXML`、`NewMusic`、`NewShare`、`NewLocation`、`RPS`、`Dice`、`Shake`、`NewContact` 构造函数。`Raw.Data` 保存完整 data JSON 字符串，SDK 声明按字符串传输以保留全部字段与数字精度。`onebot11.face` 与 `onebot11.at_all` 支持收发；json、xml、music、share、location、rps、dice、shake、contact、markdown 通过对应声明的 extension 保留完整 `data`，也可交给协议端发送。未知段保存在只读的 `onebot11.segment` 扩展中。入站 forward 保留平台资源标识，通过 `c.ReadNodes` 展开；发送使用 SDK `ForwardItem`。`segment.ForwardRef(id)` 为 OneBot 专有资源引用，Adapter 在发送前自行查询，并按目标重建节点。

`event` 包提供 `sdk.EventType`：`Poke`、`LuckyKing`、`Honor`、`GroupAdmin`、`GroupBan`、`GroupUpload`、`FriendAdd`、`FriendRequest`、`GroupRequest`、`InputStatus`、`Notice`、`Request`，及其类型化载荷。事件名为 `onebot11.<snake_case>`，版本均为 1；末两项的 `Raw` 字段保留未知通知、请求的完整 JSON 原文字符串。未知上报没有群号或用户号时保留账号身份，`Session` 为空。群文件通知按声明提取文件字段，实现端附加字段仍保留在 `PlatformPayload`。标准消息、撤回和成员进退群使用 SDK 标准事件。

普通插件导入 `github.com/AlonaBot/alonabot-adapter-onebot11/client`，独立 client 包只依赖 SDK：

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

类型化方法覆盖群管理、好友和群审批、群/成员/好友/陌生人查询、消息和转发查询、群及私聊文件 URL 查询、戳一戳与点赞。查询的 `Result[T]` 同时保留 `Value` 和完整 JSON `Raw`。`Call("get_msg", params)` 只接受声明的 API 名；Core 继续执行声明范围、实例、权限和父调用检查。业务调用沿用 Handler 的取消与期限。异步动作仍保留响应 `data`，包括对象、数组与 `null`；异步发送标记消息 ID 不可用。

### 查询提及昵称与引用内容

会话标题取私聊发送者昵称，群名在进程内首次遇到时查询一次，失败记诊断并缓存空结果。入站提及会优先使用事件发送者与已验证的账号昵称，其余通过群成员或陌生人查询补充 `MentionSegment.Display`；同一事件中的重复提及共用查询结果。查询失败记录 `onebot mention display lookup failed`，原消息和提及仍交付。插件也可调用 `GetGroupMemberInfo` 或 `GetStrangerInfo` 按需取得最新资料。

引用携带消息句柄，原消息内容可直接查询：

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

这里 `message` 指 `github.com/AlonaBot/AlonaBot/sdk/message`，`bound` 沿用 `client.From(c)` 的结果。`GetForwardMessage` 的标准响应通过 `Value.Message` 取得 `ForwardNode` 序列：`node.Data.UserID` 使用 `json.Number` 保留十进制字符串或整数 ID 的精度，`Nickname` 保存昵称，`Content` 保留消息段数组或 CQ 码字符串的原始 JSON。SnowLuma 的 `messages` 响应继续通过 `Value.Messages` 取得类型化消息；两种响应的完整原文和扩展字段均保留在 `Raw`。

### 节点、Markdown 与显式文本转换

```go
nodes, err := c.ReadNodes(message.ForwardSegment{PlatformForwardID: resourceID})
```

查询同时支持标准 `message: []ForwardNode` 和 SnowLuma `messages: []Message`；节点中的 CQ 字符串仅在查询入口解析。查询中的旧式 XML 转发卡片按资源 ID 展开；卡片缺少资源 ID 时保留原段并返回带路径的错误。资源 ID 与消息 ID 分开使用；展开成功的节点使用 Chain，原消息引用 ID 由原始查询保留。发送者、时间和整数精度保持原值。子资源与消息引用递归展开，循环错误包含具体节点路径，合法的重复兄弟节点保留。查询出错时返回已解析的合法节点和原错误；返回媒体先由 Core 登记，再按当前调用授权。`GetMessage`、`GetForwardMessage` 继续提供 `Result[T]{Value, Raw}` 原始查询能力。

```go
groupFile, err := bound.GetGroupFileURL(client.GetGroupFileURLParams{FileID: fileID})
privateFile, err := bound.GetPrivateFileURL(client.GetPrivateFileURLParams{FileID: fileID, FileHash: fileHash})
```

文件查询的目标来自当前可信会话。群文件可通过 `BusID: &busID` 传入业务编号；两个方法均保留完整响应及错误。

入站 markdown 的 `data.markdown` 或 `data.content` 文本映射为通用 `message.MarkdownSegment`，模板字段继续保存在 `onebot11.markdown` 扩展和事件 `PlatformPayload`。显式调用 `md.PlainText()` 可得到可发送的普通文本；旧的 `TextSegment.FormatMarkdown` 保持原有语义。

```go
md := message.Markdown("**标题**\n正文")
plain := md.PlainText()
buttonText := client.ButtonsText(buttons)
sentences := client.SplitSentences(text)
```

`ButtonsText` 保留标签和 URL，并给回调按钮添加「无法交互」提示。`SplitSentences` 一次性按句末标点或换行切分，连接全部结果可还原原文；发送仍使用既有 `Send` / `Stream`。

## 开发与发行

需要 asdf 管理的 Go 1.27.1 或更高版本。Schema 源位于 `internal/schema/main.go`，更新后运行：

```sh
go generate ./...
go test -race ./...
go build ./...
alonabot check .
alonabot pack . --target linux/amd64 --out bin
```

本地跨仓开发通过 `go.work` 引用对应 Core checkout。普通单元测试与本地 SDK 边界验证放在 `*_local_test.go`。本地测试覆盖完整通知/请求的类型化 Handler、Core HTTP 入口的反向鉴权与账号复核、SDK 媒体 URL 与共享路径的租约回收，以及慢取件取消。真实平台验收独立于模拟端测试。当前 SnowLuma v2.4.2 实测中，部分嵌套资源在接收端被返回为 `m_resid=""` 的 XML，正文与子节点会缺失；此时 ReadNodes 明确返回错误。该协议端问题影响部分群聊反向发送及混合正文/子转发的完整读取。

运行中的 Core 可执行 `alonabot install --source AlonaBot/alonabot-adapter-onebot11 --yes`。发行说明位于 `changelog/<版本>/`。

`0.1.1` 已验证可在公开 Core `v0.1.0` 上安装、启用及重启；`ReadNodes` 节点查询需要 Core #293 的节点查询与媒体授权实现，已在 `562fbcb57a11` 验证通过。Core `v0.1.0` 调用该接口返回 `NOT_FOUND: Core service not found`。需要节点查询及查询结果媒体授权时，先升级 Core。

协议字段依据 [OneBot 11 消息段](https://github.com/botuniverse/onebot-11/blob/master/message/segment.md)和[公开 API](https://github.com/botuniverse/onebot-11/blob/master/api/public.md)，上传文件、poke 和 markdown 的具体支持取决于实现端响应。

采用 [MIT](LICENSE)。
