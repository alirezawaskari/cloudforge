# k6 load test

`script.js` runs a smoke scenario (2 constant VUs hitting `/livez` for 30s)
followed by a ramping load scenario (0 → 50 VUs) that exercises the full
CRUD path — create, get, list, delete — against `/api/v1/items`.

## Run locally

```bash
# against docker-compose
docker compose up -d
k6 run load-test/k6/script.js

# against a port-forwarded cluster deployment
kubectl port-forward svc/cloudforge -n cloudforge-dev 8080:80 &
k6 run -e BASE_URL=http://localhost:8080 load-test/k6/script.js
```

## Thresholds

The run fails (non-zero exit) if:
- overall HTTP failure rate exceeds 1%
- p95 latency exceeds 300ms or p99 exceeds 800ms during the load scenario
- p95 latency exceeds 200ms during the smoke scenario

## In CI

Point `BASE_URL` at a deployed environment and run:

```bash
k6 run --summary-export=k6-results/summary.json -e BASE_URL=$BASE_URL load-test/k6/script.js
```

`k6-results/` is gitignored — treat it as CI artifact output, not a
committed fixture.
