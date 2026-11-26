# 源码开发

[English](development.md)

在本仓根目录执行。使用 `go.mod` 声明的 Go 版本，本地由 asdf 管理；SDK 以该文件固定版本为准。CLI 沿用组件工具链构建：

```fish
go mod download github.com/AlonaBot/AlonaBot
env GOTOOLCHAIN=(go env GOVERSION) go -C (go list -m -f '{{.Dir}}' github.com/AlonaBot/AlonaBot) install ./cmd/alonabot
```

将 Go bin 目录加入 PATH。跨仓开发可用本地 `go.work` 指向对应 Core checkout；发行验证使用锁定的 SDK。

现有 GORM 依赖使用 `go.mod` 固定的上游正式版本。

## 验证改动

按影响选择检查。Go、依赖和 CI 门禁变更触发以下 Go 检查；组件资源和检查工作流变更也触发 `alonabot check .`。

```fish
go generate ./...
gofmt -l (git ls-files '*.go')
go vet ./...
go test -race ./...
for target in linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64
    set -l parts (string split / $target)
    env GOOS=$parts[1] GOARCH=$parts[2] CGO_ENABLED=0 go build ./...
end
alonabot check .
```

Schema 生成入口为 `internal/schema/main.go`，通过 `go generate ./...` 调用；核对 `adapter_config_schema.json`，保持生成结果一致。

文档和 GitHub 元数据检查改动链接、YAML、脚本及 `git diff --check`。CI 的 Repository 比较事件 base SHA 与 HEAD；纯文档改动跳过 Go 检查。PR 记录实际命令、平台、失败与未运行项。

## 源码导航

- `onebot11.go`: 公开构造入口
- `internal/onebot11/`: 组件实现
- `plugin.yaml`: 组件身份与打包声明
- `client/`、`segment/`、`event/`、`perm/`、`onebottest/`、`integration/`：公开 API 与模拟端验证

[Core 源码开发](https://github.com/AlonaBot/AlonaBot/blob/main/docs/development.zh-CN.md) · [组件开发文档](https://alonabot.github.io/docs/tutorial/project)

本地验证曾覆盖完整通知/请求的类型化 Handler、Core HTTP 入口的反向鉴权与账号复核、SDK 媒体 URL 与共享路径的租约回收，以及慢取件取消。模拟端检查与真实平台验收分别记录。见[SnowLuma 实测问题](usage.zh-CN.md#已知平台问题)。
