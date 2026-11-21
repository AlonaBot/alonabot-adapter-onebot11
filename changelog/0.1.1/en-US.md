# alonabot-adapter-onebot11 0.1.1

## Added

- Reverse Universal WebSocket with token authentication and account verification.
- Typed OneBot events, platform actions, queries and native message segments, with original platform data retained.
- Recursive forward and reply queries, nested forward sending, file URL queries and inbound Markdown.
- Uploaded media delivery through shared directories or temporary Core URLs, with cancellation and lease cleanup.

## Changed

- Verify the account after reconnecting and retain partial send results and errors.
- Pin the SDK to Core `v0.1.1-0.20261121053743-7ca08ec8967e` for node queries and media authorization.
- Installation, activation and restart work on Core `v0.1.0`. `ReadNodes` requires the node-query and media-authorization implementation from Core #293, verified on `562fbcb57a11`. Upgrade Core before using node queries and media authorization for queried nodes.

## Known issues

- SnowLuma v2.4.2 can return nested forward XML with an empty `m_resid`, losing content and child nodes. `ReadNodes` retains parsed content and returns the original error with its node path.
