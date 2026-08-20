#!/usr/bin/env bash
# Creates the local kind cluster and installs ingress-nginx, wired for the
# extraPortMappings in deploy/kind/kind-config.yaml (host 8080/8443).
set -euo pipefail

CLUSTER_NAME="cloudforge"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  echo "kind cluster '${CLUSTER_NAME}' already exists, skipping create"
else
  kind create cluster --config "${ROOT_DIR}/deploy/kind/kind-config.yaml"
fi

echo "Installing ingress-nginx (kind-specific manifest, NodePort via hostPort mapping)..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

echo "Waiting for ingress-nginx controller to be ready..."
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=180s

echo "Cluster ready. Context: kind-${CLUSTER_NAME}"
kubectl cluster-info --context "kind-${CLUSTER_NAME}"
