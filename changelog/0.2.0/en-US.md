# alonabot-adapter-onebot11 0.2.0 - 2026-11-22

### Changed

- Rebuild against the reviewed Core SDK for archive-only packaging. Existing OneBot message, node-query, media and instance configuration behavior is retained.

### Distribution and upgrade

- Publish five `.tar.gz` platform archives, including Windows amd64; GitHub supplies source archives. Standalone `release.json`, signatures and public-key assets are retired.
- Use the SDK and packager from reviewed Core commit `2dbbded7184f` (`v0.1.2-0.20261122021903-2dbbded7184f`), which contains the v0.2.0 archive installation flow. The component uses the MIT License.
- Upgrade Core to v0.2.0 before installing or updating this release. Stop the component and back up its data before replacement; retain the same component ID, source and data directory. Historical sidecar cleanup follows verification of Core and WebUI.

### Known issues

- SnowLuma v2.4.2 may return nested-forward XML with an empty `m_resid`, losing content and child nodes. `ReadNodes` preserves parsed content and returns the original error with its node path.
