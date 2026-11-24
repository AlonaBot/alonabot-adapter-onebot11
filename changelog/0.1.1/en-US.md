# alonabot-adapter-onebot11 0.1.1

## Added

- Reverse Universal WebSocket with token authentication and account verification.
- Typed OneBot events, platform actions, queries and native message segments, with original platform data retained.
- Recursive forward and reply queries, nested forward sending, file URL queries and inbound Markdown.
- Uploaded media delivery through shared directories or temporary Core URLs, with cancellation and lease cleanup.

## Changed

- Verify the account after reconnecting and retain partial send results and errors.

## Upgrade requirements

- `ReadNodes` requires Core #293's node-query and media-authorization implementation. Upgrade Core before using these capabilities.

## Known issues

- SnowLuma v2.4.2 can return nested forward XML with an empty `m_resid`, losing content and child nodes. `ReadNodes` retains parsed content and returns the original error with its node path.
