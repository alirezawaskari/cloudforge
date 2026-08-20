# Terraform — generic Kubernetes infrastructure

This Terraform targets **any** conformant Kubernetes cluster via the
`kubernetes` and `helm` providers, authenticated through your kubeconfig.
There is no cloud provider block, no `aws`/`google`/`azurerm` provider, and
no credentials of any kind checked into this repo — cluster provisioning
(EKS/GKE/AKS/etc.) is assumed to happen separately, with your platform
team's own cloud-specific Terraform or the cloud console. This layer picks
up after a cluster already exists and its kubeconfig is available.

## Layout

```
terraform/
  modules/
    namespace/       generic namespace + optional ResourceQuota/LimitRange
    app/              installs the cloudforge Helm chart (deploy/helm/cloudforge),
                       whose own templates provision PostgreSQL/Redis StatefulSets
                       when postgresql.enabled/redis.enabled are true
    ingress-nginx/    ingress-nginx controller
    monitoring/       kube-prometheus-stack + Loki + OTel Collector
  environments/
    dev/              targets kind/minikube; chart-bundled datastores, no monitoring by default
    staging/          production-like topology, chart-bundled datastores, monitoring on
    production/       external managed datastores via secret reference, ingress + monitoring
```

## Usage

```bash
cd terraform/environments/dev
cp terraform.tfvars.example terraform.tfvars   # edit as needed
terraform init
terraform plan
terraform apply
```

Repeat for `staging` / `production` with their own `terraform.tfvars`.
Never commit a populated `.tfvars` file — they're gitignored; only the
`.example` files are tracked.

## Why no cloud resources here

Provisioning the Kubernetes cluster itself (VPC, node pools, IAM) is
cloud-specific and out of scope for a "generic cloud Kubernetes environment"
deliverable — that Terraform would need real credentials and would only be
valid for one provider. Instead, this project demonstrates the layer that
*is* cloud-agnostic: everything you'd run identically against EKS, GKE, AKS,
or a self-managed cluster, using only the Kubernetes API. Swapping providers
means changing `kube_context`, not rewriting modules.

## Secrets

- The `staging` environment takes `database_password` as a `sensitive`
  Terraform variable — set it via `TF_VAR_database_password` or an
  environment-specific secrets backend (e.g. a CI secret store), never in
  a committed `.tfvars` file.
- The `production` environment does not create secrets at all — it
  references a pre-existing Kubernetes Secret (`secret_name`) that your
  external secret manager is expected to populate. See
  [../docs/SECRETS.md](../docs/SECRETS.md).
