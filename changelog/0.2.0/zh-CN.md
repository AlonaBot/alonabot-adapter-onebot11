# alonabot-adapter-onebot11 0.2.0 - 2026-11-22

### 变更

- 安装包改为五个平台的 `.tar.gz` 归档，包含 Windows amd64；源码压缩包由 GitHub 提供，移除独立 `release.json`、签名与公钥资产。

### 升级要求

- 安装或更新本版前先升级 Core v0.2.0。

### 已知问题

- SnowLuma v2.4.2 可能返回 `m_resid` 为空的嵌套转发 XML，导致正文与子节点缺失；`ReadNodes` 保留已解析内容，并返回包含节点路径的原始错误。
