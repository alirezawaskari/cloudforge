# cloudforge Helm chart

Deploys the cloudforge API together with minimal, self-contained
PostgreSQL/Redis StatefulSets (plain `postgres:16-alpine` / `redis:7-alpine`,
no third-party subchart or external chart-repo dependency) for local/dev
use, or against externally managed datastores in staging/production.

## Environments

| File                     | Purpose                                            |
|---------------------------|-----------------------------------------------------|
| `values.yaml`              | Base defaults                                       |
| `values-dev.yaml`           | Single replica, no HA, ephemeral storage             |
| `values-staging.yaml`       | Production-like topology at reduced scale            |
| `values-production.yaml`    | HA replicas, autoscaling, strict PDB, TLS ingress     |

## Usage

```bash
# Install into dev
helm upgrade --install cloudforge deploy/helm/cloudforge \
  -f deploy/helm/cloudforge/values-dev.yaml \
  --namespace cloudforge-dev --create-namespace

# Render manifests without installing
helm template cloudforge deploy/helm/cloudforge -f deploy/helm/cloudforge/values-staging.yaml

# Smoke test after install
helm test cloudforge --namespace cloudforge-dev
```

## Notes

- `secrets.existingSecret` should be set in staging/production, pointing at a
  Secret populated by your external secret manager of choice — see
  [docs/SECRETS.md](../../../docs/SECRETS.md).
- In real production, set `postgresql.enabled=false` and `redis.enabled=false`
  and point `DATABASE_URL`/`REDIS_ADDR` at managed services instead of the
  in-cluster StatefulSets.
