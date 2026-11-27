# 0.4.0-dev.0 — Unreleased

## Added

- Declare standard formats, media sources, and composition rules; existing OneBot extensions provide purposes and payload examples for on-demand Agent discovery.
- Add readonly preflight that reuses the sending plan and reports concrete limits for Markdown, buttons, text entities, file captions, and quote combinations before delivery.

## Fixed

- Enable delivery of complete `onebot11.segment` envelopes, preserving data fields and integer precision; forwards retain mixed bot-authored and same-session message-reference nodes.
