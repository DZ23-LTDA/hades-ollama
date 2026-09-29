# PostgreSQL HA and failover rehearsal

**Status: local rehearsal passed; production HA is not approved.** This document captures an observed PostgreSQL 16 test and the minimum controlled promotion sequence. It is not a substitute for an HA manager, externally managed TLS/CA, quorum/fencing, or an operator-run production disaster-recovery exercise.

## What was exercised

A disposable primary and standby were provisioned as separate PostgreSQL 16 clusters. The primary enabled `wal_level=replica`, WAL senders and replication slots, then accepted a dedicated `REPLICATION LOGIN` over TLS. A `pg_basebackup -X stream -R -C -S ...` created the standby and slot. The standby's `primary_conninfo` used `sslmode=verify-full` and an explicit CA. The test waited for `pg_stat_replication.state='streaming'`, committed two marker rows, and waited until the standby replayed both. It then stopped/fenced the primary before running `pg_promote(true, 30)`, verified `pg_is_in_recovery() = false`, and committed a third row on the promoted node.

Observed result: `REPLICATION_STATE=streaming`, `STANDBY_ROWS_BEFORE_PROMOTION=2`, and after promotion `PROMOTED_NODE_ROW_COUNTS=3/1`. The rehearsal ran on disposable local clusters with self-signed certificates containing the loopback IP/DNS SAN. The clusters, generated replication slot, CA, keys, and temporary data were removed afterward. This does **not** validate cloud networking, real certificate issuance/rotation, automated failover, synchronous replication, backup retention, or an operational RPO/RTO.

## Controlled promotion principles

1. Confirm which node is authoritative using the deployment's quorum/fencing mechanism. **Fence the former primary before promoting a standby.** A network partition without fencing can create split-brain: two writable primaries and diverging timelines.
2. Measure the actual replication/replay position and compare it to the latest acknowledged commit. Asynchronous replication can lose commits that were acknowledged by an isolated primary but not replayed on the standby; choose and test synchronous-commit behavior against the service's RPO requirement.
3. Promote only the selected caught-up standby. Confirm it is out of recovery and that the read/write smoke succeeds on the intended endpoint; record the promotion timestamp and RTO.
4. Keep the old primary fenced. Do not simply restart it as writable. Rebuild it from the new primary or use a tested `pg_rewind` procedure with the required WAL/history and prerequisites, then verify it is streaming as a standby before returning it to service.
5. Re-run database readiness, tenant-isolation smoke tests, queue/worker connectivity and application health checks before reopening traffic. Preserve logs and timeline/system identifiers, without recording secrets.

## Production gate

Before production approval, execute this in a dedicated staging environment using the actual HA topology and certificate authority. Record the topology, fencing/quorum mechanism, replication mode, observed RPO/RTO, backup age, restore result, timeline recovery procedure, and alert coverage. Exercise primary loss, standby loss, network partition, stale former-primary restart, and restoration from backup. Obtain an independent operations/security review. The local PostgreSQL rehearsal above is evidence of the manual replication/promotion mechanics only; it is not evidence that those production gates passed.

## Referências oficiais (PostgreSQL 16)

- [Failover and the former primary](https://www.postgresql.org/docs/16/warm-standby-failover.html) — PostgreSQL warns that the old primary must be prevented from returning as writable; it describes STONITH/fencing and `pg_promote()`.
- [`pg_basebackup`](https://www.postgresql.org/docs/16/app-pgbasebackup.html) — documents WAL streaming (`-X stream`), `-R`, and the role of a replication slot.
- [libpq SSL verification](https://www.postgresql.org/docs/16/libpq-ssl.html) — `verify-full` checks the certificate chain and the host identity; configure an explicit trusted root certificate.
- [Administrative recovery functions](https://www.postgresql.org/docs/16/functions-admin.html) — `pg_promote(wait, wait_seconds)` waits for completion up to its timeout and returns a boolean result.
