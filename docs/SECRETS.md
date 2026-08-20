# Secrets management

## What's in this repo

For local development and the kind/minikube demo path, the chart renders a
plain Kubernetes `Secret` (`deploy/helm/cloudforge/templates/secret.yaml`)
from `values.yaml`'s `secrets.data` map. It holds `DATABASE_URL` and
`REDIS_ADDR`/`REDIS_PASSWORD`. This is intentionally the simplest possible
mechanism — good for a laptop, **not good enough for a shared cluster**:

- Values in a Helm `values.yaml` land in `helm history` and (unless you're
  careful) in git. `values-production.yaml` in this repo never sets a real
  password — it points at `secrets.existingSecret` instead (see below).
- A raw Kubernetes `Secret` is only base64-encoded, not encrypted, at rest
  unless your cluster has [encryption at rest](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/)
  configured for the `Secret` resource type.
- Anyone with `get secrets` RBAC in the namespace can read it in plaintext.

## What staging/production actually do

`values-production.yaml` sets `secrets.existingSecret: cloudforge-production-secrets`.
When set, the chart **does not render its own Secret** — see the
`{{- if not .Values.secrets.existingSecret }}` guard in
`templates/secret.yaml` — and instead references a Secret by name that's
expected to already exist in the namespace, created by something outside
Helm's control. That "something" should be one of:

| Option | How it works | When to use it |
|---|---|---|
| **External Secrets Operator** | Syncs secrets from AWS Secrets Manager / GCP Secret Manager / Azure Key Vault / Vault into a native `Secret` on a schedule, via a `SecretStore` + `ExternalSecret` CRD. | Most common choice on any of the big three clouds; keeps the source of truth in the cloud provider's secret store. |
| **HashiCorp Vault + Vault Agent / CSI driver** | Injects secrets as files or env vars at pod startup directly from Vault, no Kubernetes `Secret` object created at all. | Multi-cloud or on-prem shops already standardized on Vault. |
| **Sealed Secrets (Bitnami)** | Secrets are encrypted client-side into a `SealedSecret` CRD that's safe to commit to git; the in-cluster controller decrypts it into a normal `Secret`. | Small teams wanting GitOps-friendly secrets without running an external secret store. |
| **SOPS + age/KMS** | Secrets are encrypted files in git, decrypted at deploy time (e.g. via `helm-secrets` plugin or a CI step) using a KMS key or age key. | Teams already using SOPS for other config; works well with Terraform too. |

None of these are wired up in this repo — that would require a real cloud
account and secret store to demonstrate honestly. What *is* demonstrated is
the seam: `secrets.existingSecret` is the integration point, and the chart
never assumes it knows how the Secret got there.

## Terraform

`terraform/environments/production` deliberately does not create or read
secret values — see the comment in `main.tf`. It only passes
`var.secret_name` (a name, not a value) to the app module. Populating that
Secret is out of scope for generic-Kubernetes Terraform and belongs to
whichever secret manager integration your org uses.

## Rotation

Whatever mechanism you choose, prefer one that supports rotation without a
pod restart (Vault Agent sidecar, ESO's refresh interval) over baking
secrets into a Deployment's env at rollout time — the latter means every
credential rotation requires a rolling restart to pick up.
