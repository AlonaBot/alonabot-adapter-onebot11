# alonabot-adapter-onebot11 0.2.0 - 2026-11-22

### Changed

- You can install this release from one platform archive without separate manifest, signature or public-key files. This format requires the archive installer introduced in Core v0.2.0.

### Known issues

- SnowLuma v2.4.2 may return nested-forward XML with an empty `m_resid`, losing content and child nodes. `ReadNodes` preserves parsed content and returns the original error with its node path.
