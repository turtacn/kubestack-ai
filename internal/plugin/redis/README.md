# Redis Plugin - Deprecated

**Status:** DEPRECATED - Moved to `internal/plugin/redis_legacy/`

This Redis plugin implementation using `github.com/redis/go-redis/v9` has been deprecated.

## Current Active Implementation

The active Redis plugin is located at `plugins/redis/` and uses `github.com/go-redis/redis/v8`.

## Migration Note

As part of Phase 01 (Contract Alignment), we consolidated Redis plugin implementations to use a single client library version to avoid dependency conflicts and ensure consistent behavior.

The legacy implementation has been moved to `internal/plugin/redis_legacy/` for reference but is not built by default.

## For Developers

If you need to reference the v9-based implementation, see `internal/plugin/redis_legacy/`.

For all new development, use the canonical implementation at `plugins/redis/`.
