# alonabot-adapter-onebot11 0.1.1

## 新增

- 可以通过反向 Universal WebSocket 接入，连接时会验证凭据和账号身份。
- 插件可以处理类型化 OneBot 事件、动作、查询和原生消息段，同时读取平台原始数据。
- 插件可以用 `ReadNodes` 递归读取引用和合并转发消息；节点查询和媒体读取需要 Core v0.1.1 ([#293](https://github.com/AlonaBot/AlonaBot/pull/293))。
- Adapter 可以发送嵌套转发、查询文件 URL，并接收入站 Markdown。
- 上传媒体可以通过共享目录或 Core 的临时 URL 交付。

## 变更

- 重连后复核账号，保留部分发送结果与原始错误。

## 已知问题

- SnowLuma v2.4.2 可能返回 `m_resid` 为空的嵌套转发 XML，导致正文与子节点缺失。`ReadNodes` 保留已解析内容，并返回包含节点路径的原始错误。
