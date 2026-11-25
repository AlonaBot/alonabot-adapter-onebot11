# alonabot-adapter-onebot11 0.1.1

## Added

- You can connect via reverse Universal WebSocket, with token authentication and account verification.
- Plugins can use typed OneBot events, actions, queries and native message segments and still access the original platform data.
- Plugins can use `ReadNodes` to read replies and forwarded messages recursively. Node queries and media access require Core v0.1.1 ([#293](https://github.com/AlonaBot/AlonaBot/pull/293)).
- The adapter can send nested forwards, query file URLs and receive Markdown.
- Uploaded media can be delivered through a shared directory or a temporary Core URL.

## Changed

- Verify the account after reconnecting and retain partial send results and errors.

## Known issues

- SnowLuma v2.4.2 can return nested forward XML with an empty `m_resid`, losing content and child nodes. `ReadNodes` retains parsed content and returns the original error with its node path.
