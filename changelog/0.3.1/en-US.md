# alonabot-adapter-onebot11 0.3.1 - 2026-11-24

### Added

- Set or clear a group member's special title through the `onebot11.set_group_special_title` chat action and typed client method. An empty title clears it; optional `duration` values are forwarded unchanged, with lifetime support depending on the OneBot implementation.
- Include the official OneBot icon in component packages and both repository READMEs, preserving the upstream MIT copyright and license notice.

### Distribution and upgrade

- The SDK and packager remain pinned to Core commit `25a0f851099f` (`v0.2.1-0.20261123105642-25a0f851099f`).
- Upgrade Core to v0.3.0 before installing or updating this release.
