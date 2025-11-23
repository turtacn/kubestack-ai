# Redis Plugin Specification

## Overview
The Redis plugin provides diagnostics for Redis 5.x, 6.x, and 7.x.

## Collected Data
- **INFO**: Server, Clients, Memory, Persistence, Stats, Replication, CPU, Cluster, Keyspace.
- **SLOWLOG**: Recent slow queries.
- **CONFIG**: Runtime configuration.

## Metrics
- `memory_used_bytes`
- `connected_clients`
- `hit_rate`
- `keyspace_hits/misses`

## Health Checks
1. **Connectivity**: PING check.
2. **Replication**:
   - Role verification.
   - Master link status.
   - Replication lag (seconds).
3. **Memory**:
   - Usage vs Maxmemory.

## Configuration
```yaml
plugin:
  name: redis
  settings:
    address: "localhost:6379"
    password: "..."
```
