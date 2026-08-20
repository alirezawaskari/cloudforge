# Disaster recovery

## Failure scenarios and response

### Single pod crash / OOM kill

**Detection:** liveness probe fails 3x (30s) → kubelet restarts the
container. `CloudforgePodCrashLooping` alert fires if it happens more than
3 times in 15 minutes.

**Impact:** none, if `replicaCount >= 2` — the Service only routes to pods
passing the readiness probe, and the PodDisruptionBudget guarantees at
least one other replica stays up during any voluntary disruption.

**Recovery:** automatic. If it's a genuine OOM (check
`kubectl describe pod` for `OOMKilled`), raise `resources.limits.memory` in
the relevant `values-<env>.yaml`.

### Node failure

**Detection:** node goes `NotReady`; kubelet stops heartbeating; pods on
that node are marked `Unknown` and, after the pod-eviction-timeout
(default 5m), rescheduled elsewhere.

**Impact:** pods on the failed node stop serving; remaining replicas on
other nodes (see `affinity.podAntiAffinity` — pods are preferentially
spread across nodes) continue serving traffic. If the in-cluster
PostgreSQL/Redis StatefulSet pod was on that node, a **new pod is
scheduled but its PVC (if using a `WaitForFirstConsumer`-bound
`StorageClass`) may pin it to the same node** until that node returns —
this is the sharpest edge of running stateful data in-cluster and is
exactly why [values-production.yaml](../deploy/helm/cloudforge/values-production.yaml)
recommends managed databases (RDS/Cloud SQL/etc., which handle this via
multi-AZ replication) for real production use.

**Recovery:** automatic for the stateless API tier. For the datastore
tier: either wait for the node to return, or (cloud-managed StorageClass
permitting) delete the PVC binding and let it reschedule with a fresh
volume, restoring from the most recent backup (see below).

### Full cluster loss

**Detection:** total control-plane/API unreachability, or a cloud-provider
region-level outage.

**Recovery:**
1. Provision a new cluster (out of scope for this repo — your platform
   team's cluster-provisioning process).
2. `terraform apply` the relevant `terraform/environments/<env>` against
   the new cluster's kubeconfig — reproduces namespaces, ingress
   controller, and the observability stack from code.
3. Restore PostgreSQL from the most recent backup (see Backup strategy
   below) into the new cluster (or, in production, your managed database
   should already be running independently of the compute cluster and
   need no restore at all — this is the strongest argument for not running
   production data stores in-cluster).
4. `helm upgrade --install` the app chart pointed at the restored
   database.
5. Verify with `helm test` and the k6 smoke scenario before reopening
   traffic (DNS/ingress cutover).

**RTO target:** ~30–60 minutes for the compute layer (cluster + Terraform
apply + Helm install is scriptable and fast); RTO for data depends entirely
on backup recency and restore method — see below.

**RPO target:** depends on backup frequency (below); with continuous WAL
archiving this can be seconds, with nightly snapshots it's up to 24h.

### Bad deploy (broken image, migration failure)

**Detection:** rollout stalls — `RollingUpdate` with `maxUnavailable: 0`
means Kubernetes won't finish replacing old pods with new ones that never
pass readiness, so `kubectl rollout status` blocks/times out rather than
taking the whole service down.

**Recovery:** `kubectl rollout undo deployment/cloudforge -n <namespace>`.
Because `revisionHistoryLimit: 5` is set, the last 5 ReplicaSets are kept
around for instant rollback without needing to rebuild an image.

### Dependency outage (Postgres or Redis unreachable)

**Detection:** `/readyz` starts failing on the `database`/`cache` check
(see [internal/handlers/health.go](../internal/handlers/health.go)) while
`/livez` keeps passing — the process itself is healthy, it just can't serve
correctly, so Kubernetes doesn't restart it (that would just cause a crash
loop without fixing anything) but the Service stops routing to it.

**Recovery:** the app's startup/reconnect logic
(`connectDependencies` in [cmd/api/main.go](../cmd/api/main.go)) retries
with exponential backoff rather than crashing, so once the dependency
recovers, the pod recovers automatically without a restart.

## Backup strategy

This repo ships the in-cluster PostgreSQL StatefulSet for **dev/demo
convenience only** — it uses a single replica and a single PVC, no backup
job. For any environment that matters:

- **Managed database** (recommended — see `values-production.yaml`):
  RDS/Cloud SQL/Azure Database automated snapshots + point-in-time
  recovery via WAL/binlog shipping. This is the actual production answer;
  everything below is what you'd do if you insisted on running Postgres
  in-cluster.
- **In-cluster, if you must:** a `CronJob` running `pg_dump` on a schedule,
  writing to object storage (S3/GCS/Azure Blob) with a lifecycle policy —
  or a proper operator like CloudNativePG/Zalando's postgres-operator that
  handles continuous WAL archiving and PITR for you. Neither is included
  here since it would need real object storage credentials to demonstrate
  honestly.
- **Redis:** treated as a cache, not a system of record — `secrets.data`
  and `items` table live in Postgres; losing Redis only costs a cold cache,
  not data. No backup needed by design.

## Runbook checklist (practice this before you need it)

- [ ] Can you `kubectl rollout undo` in under 2 minutes?
- [ ] Do you know your database's actual last-successful-backup timestamp
      right now, without looking it up?
- [ ] Has anyone actually restored from that backup in the last 90 days
      (not just verified it exists)?
- [ ] Does `terraform apply` against a brand-new cluster succeed without
      manual intervention?
- [ ] Are the Prometheus alerts in
      [observability/prometheus/alert-rules.yaml](../observability/prometheus/alert-rules.yaml)
      actually routed to a human (Alertmanager receiver configured), or do
      they fire into the void?
