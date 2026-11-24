# alonabot-adapter-onebot11 0.1.1

## 新增

- 反向 Universal WebSocket，支持凭证鉴权与账号复核。
- 类型化 OneBot 事件、平台动作、查询及原生消息段，保留平台原始数据。
- 递归转发与引用查询、嵌套转发发送、文件 URL 查询及入站 Markdown。
- 上传媒体通过共享目录或 Core 临时 URL 交付，支持取消与租约回收。

## 变更

- 重连后复核账号，保留部分发送结果与原始错误。

## 升级要求

- `ReadNodes` 需要 Core #293 的节点查询与媒体授权实现；使用这些能力前请先升级 Core。

## 已知问题

- SnowLuma v2.4.2 可能返回 `m_resid` 为空的嵌套转发 XML，导致正文与子节点缺失。`ReadNodes` 保留已解析内容，并返回包含节点路径的原始错误。
