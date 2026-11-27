# Development

[简体中文](development.zh-CN.md)

Run these commands from this repository root. Use the Go version declared in `go.mod` (managed with asdf locally); the SDK stays pinned there. Build the CLI with the same component toolchain:

```fish
go mod download github.com/AlonaBot/AlonaBot
env GOTOOLCHAIN=(go env GOVERSION) go -C (go list -m -f '{{.Dir}}' github.com/AlonaBot/AlonaBot) install ./cmd/alonabot
```

Keep the Go bin directory on PATH. For cross-repository development, a local `go.work` may select the corresponding Core checkout; verify releases using the locked SDK.

The existing GORM dependency uses upstream releases pinned in `go.mod`.

## Verify a change

Select checks by the affected behavior. CI runs the following Go checks for Go, dependency, and CI gate changes; component resources and checking workflows also trigger `alonabot check .`.

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

The schema generator is `internal/schema/main.go`, invoked by `go generate ./...`; compare the result with `adapter_config_schema.json` and keep generation deterministic.

For documentation and GitHub metadata, check the changed links, YAML and scripts, plus `git diff --check`. In CI, Repository compares the event base SHA with HEAD; docs-only changes skip Go checks. Record actual commands, platform, failures, and unrun checks in the PR.

## Source map

- `onebot11.go`: public construction entry
- `internal/onebot11/`: component implementation
- `plugin.yaml`: component identity and packaging
- `client/`, `segment/`, `event/`, `perm/`, `onebottest/`, `integration/`: public APIs and simulator-based verification

[Core development](https://github.com/AlonaBot/AlonaBot/blob/main/docs/development.md) · [Component developer docs](https://alonabot.github.io/en/docs/tutorial/project)

Local checks have covered typed handlers for complete notices/requests, reverse authentication and account rechecks through Core HTTP, SDK media URL/shared-path lease cleanup, and slow-fetch cancellation. Simulator checks and real-platform acceptance provide separate evidence. See the [recorded SnowLuma issue](usage.md#known-platform-issue).

For cross-repository rich-message development, keep a dedicated `tmp/go.work` in the ignored directory, listing only this repository and its Core checkout, and set `GOWORK` explicitly. Set `GOWORK=off` when validating the remote SDK pinned in `go.mod`. `alonabot test .` uses a real offline Session to validate extension purpose/example/schema registration. `TestRichMessageDelivery` drives the public adapter and `onebottest` to verify bot-authored/message-reference nodes and raw extension delivery. The development candidate uses `0.4.0-dev.0` with Unreleased bilingual changelog headings.

The current CLI development build uses `-modfile`, which conflicts with workspace mode. To validate against a local Core candidate, copy a source snapshot under `tmp/`, add a local SDK `replace` only to that snapshot’s `go.mod`, and run `alonabot test` or `alonabot run` with `GOWORK=off`. Keep absolute paths and the snapshot in the ignored directory.
