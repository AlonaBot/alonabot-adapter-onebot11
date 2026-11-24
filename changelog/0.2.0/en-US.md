# alonabot-adapter-onebot11 0.2.0 - 2026-11-22

### Changed

- Switch to five `.tar.gz` platform archives, including Windows amd64; GitHub supplies source archives. Standalone `release.json`, signatures and public-key assets are retired.

### Upgrade requirements

- Upgrade Core to v0.2.0 before installing or updating this release.

### Known issues

- SnowLuma v2.4.2 may return nested-forward XML with an empty `m_resid`, losing content and child nodes. `ReadNodes` preserves parsed content and returns the original error with its node path.
