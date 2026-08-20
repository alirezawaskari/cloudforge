#!/usr/bin/env bash
# Builds the app image, loads it into the kind cluster, and deploys the
# cloudforge Helm chart with the dev overlay. Assumes `kind-up.sh` has
# already been run.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLUSTER_NAME="cloudforge"
IMAGE_TAG="${IMAGE_TAG:-dev}"
NAMESPACE="${NAMESPACE:-cloudforge-dev}"

echo "Building cloudforge-api:${IMAGE_TAG}..."
docker build -t "cloudforge-api:${IMAGE_TAG}" --build-arg VERSION="${IMAGE_TAG}" "${ROOT_DIR}"

echo "Loading image into kind..."
kind load docker-image "cloudforge-api:${IMAGE_TAG}" --name "${CLUSTER_NAME}"

echo "Deploying to namespace ${NAMESPACE}..."
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

helm upgrade --install cloudforge "${ROOT_DIR}/deploy/helm/cloudforge" \
  -f "${ROOT_DIR}/deploy/helm/cloudforge/values-dev.yaml" \
  --set image.repository=cloudforge-api \
  --set image.tag="${IMAGE_TAG}" \
  --set image.pullPolicy=Never \
  --namespace "${NAMESPACE}" \
  --wait --timeout 5m

echo "Running smoke test..."
helm test cloudforge --namespace "${NAMESPACE}" --logs

echo
echo "Deployed. Port-forward with:"
echo "  kubectl port-forward svc/cloudforge -n ${NAMESPACE} 8080:80"
