# Security

## Container

- **Multi-stage build**, static binary, no build toolchain in the runtime
  image ([Dockerfile](../Dockerfile)).
- **Non-root**: runtime user `app` (uid 10001), set both in the Dockerfile
  (`USER app:app`) and enforced again at the pod level
  (`podSecurityContext.runAsNonRoot: true`, `runAsUser: 10001`) so the pod
  is rejected even if the image were rebuilt without a `USER` directive.
- **Read-only root filesystem** (`securityContext.readOnlyRootFilesystem: true`)
  — the only writable path is an `emptyDir` mounted at `/tmp`, which the app
  doesn't currently use but is there for any future dependency that shells
  out or needs scratch space.
- **No privilege escalation, all Linux capabilities dropped**
  (`allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]`).
- **Seccomp**: `RuntimeDefault` profile applied at the pod level.
- Base image is `alpine:3.20` (not `distroless`) so `apk` is available for
  `ca-certificates` — chosen after `gcr.io/distroless` proved unreliable to
  pull in this environment; either is a reasonable choice for a real
  deployment, distroless trims a bit more attack surface at the cost of
  zero in-container debugging tools.

## Kubernetes

- **NetworkPolicy** (`templates/networkpolicy.yaml`) default-denies all
  ingress/egress except: traffic from the ingress controller and same-namespace
  pods on the app port, DNS, Postgres, Redis, and the OTel collector. Nothing
  else can reach the pod, and the pod can't reach anything else.
- **PodDisruptionBudget** keeps a minimum number of pods available during
  voluntary disruptions (node drains, cluster upgrades) — doesn't prevent
  attacks, but prevents an unrelated maintenance operation from taking the
  service down entirely.
- **Resource requests/limits** on every container, including the in-chart
  Postgres/Redis, so a single workload can't starve the node or trigger a
  cascading OOM across the namespace.
- **ServiceAccount** with `automountServiceAccountToken: false` by default —
  the app doesn't talk to the Kubernetes API, so it shouldn't have a token
  to do so.

## Application

- Postgres access goes through `pgx`'s parameterized queries exclusively
  (see [internal/handlers/items.go](../internal/handlers/items.go)) — no
  string-built SQL anywhere in the codebase.
- Request bodies are decoded with `encoding/json` directly into typed
  structs; no reflection-based or `interface{}`-keyed decoding that could
  be abused.
- `chi/middleware.Recoverer` catches panics per-request instead of crashing
  the process, so a single malformed request can't take down the pod.
- Every request gets a `Timeout` middleware (30s) to bound worst-case
  resource usage from a slow or stalled client.

## CI

- **gosec** static analysis on every push/PR, results uploaded as SARIF to
  GitHub code scanning.
- **govulncheck** checks the actual call graph against the Go vulnerability
  database — fewer false positives than a manifest-only scan.
- **Trivy** scans the built container image for CRITICAL/HIGH CVEs in the
  OS packages and Go module dependencies, also uploaded as SARIF.
- **Dependabot** (`.github/dependabot.yml`) keeps Go modules, the base
  Docker image, GitHub Actions, and Terraform providers current.

## What's intentionally out of scope here

- **mTLS between services** — would require a service mesh (Linkerd/Istio)
  or manual cert management; reasonable next step for a real multi-service
  environment, overkill for a single API + two datastores.
- **OPA/Gatekeeper or Kyverno policy enforcement** — the security postures
  above (non-root, read-only FS, no privilege escalation) are set correctly
  in this chart, but nothing *prevents* a different chart in the same
  cluster from violating them. A real platform would enforce these via
  admission control, not chart authoring discipline alone.
- **Image signing / SBOM attestation** (cosign, Sigstore) — natural next
  step once images are actually published to a registry with retention.
