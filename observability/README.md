# Observability stack

Configuration for the three pillars — metrics, logs, traces — plus
dashboards and alerting. All of it targets standard upstream Helm charts;
nothing here is bespoke infrastructure.

| Concern    | Chart                                                | Config in this repo                                  |
|------------|-------------------------------------------------------|--------------------------------------------------------|
| Metrics    | `prometheus-community/kube-prometheus-stack`           | `prometheus/prometheus-values.yaml`, `prometheus/alert-rules.yaml` |
| Dashboards | Grafana (bundled with kube-prometheus-stack)           | `grafana/dashboards/cloudforge-api.json`, `grafana/dashboard-configmap.yaml` |
| Logs       | `grafana/loki-stack` (Loki + Promtail)                 | `loki/loki-values.yaml`                                 |
| Traces     | `open-telemetry/opentelemetry-collector`               | `otel/otel-collector-config.yaml`, `otel/otel-collector-values.yaml` |

## Install order (kind/minikube demo)

```bash
kubectl create namespace observability

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update

# Metrics + Grafana + Alertmanager
helm install prometheus prometheus-community/kube-prometheus-stack \
  -f observability/prometheus/prometheus-values.yaml -n observability
kubectl apply -f observability/prometheus/alert-rules.yaml -n observability

# Dashboard + datasources (auto-picked up by the Grafana sidecar)
kubectl apply -f observability/grafana/dashboard-configmap.yaml
kubectl create configmap cloudforge-datasources \
  --from-file=observability/grafana/provisioning/datasources/datasources.yaml \
  -n observability --dry-run=client -o yaml | \
  kubectl label --local -f - grafana_datasource=true -o yaml --dry-run=client | \
  kubectl apply -f -

# Logs
helm install loki grafana/loki-stack -f observability/loki/loki-values.yaml -n observability

# Traces
helm install otel-collector open-telemetry/opentelemetry-collector \
  -f observability/otel/otel-collector-values.yaml \
  --set-file config=observability/otel/otel-collector-config.yaml \
  -n observability
```

Then port-forward Grafana:

```bash
kubectl port-forward -n observability svc/prometheus-grafana 3000:80
# default creds: admin / prom-operator (kube-prometheus-stack default) unless overridden
```

The `cloudforge-api` dashboard ships with panels for request rate, error
rate, p50/p95/p99 latency, in-flight requests, replica count vs HPA target,
CPU/memory, and goroutine count — all sourced directly from the
`http_requests_total` / `http_request_duration_seconds` metrics emitted by
the app (see [internal/middleware/metrics.go](../internal/middleware/metrics.go)).
