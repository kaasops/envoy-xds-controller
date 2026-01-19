#!/bin/bash
# Rebuild and redeploy to existing dev cluster

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Image settings
LOCAL_REGISTRY="${LOCAL_REGISTRY:-localhost:5001}"
GIT_COMMIT=$(git rev-parse --short HEAD)
IMAGE_TAG="${IMAGE_TAG:-$GIT_COMMIT}"

IMG_REPO="${LOCAL_REGISTRY}/envoy-xds-controller"
UI_IMG_REPO="${LOCAL_REGISTRY}/envoy-xds-controller-ui"
INIT_CERT_IMG_REPO="${LOCAL_REGISTRY}/envoy-xds-controller-init-cert"

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Updating dev deployment (${IMAGE_TAG})${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check if release exists
if ! helm status exc -n envoy-xds-controller &>/dev/null; then
    echo -e "${RED}Error: No existing deployment found.${NC}"
    echo -e "Run ${GREEN}make dev${NC} first to create initial deployment."
    exit 1
fi

# Get current values
UI_ENABLED=$(helm get values exc -n envoy-xds-controller -o json 2>/dev/null | grep -o '"ui":{"enabled":[^,}]*' | grep -o 'true\|false' || echo "true")

# Build images
echo -e "${BLUE}Building controller image...${NC}"
docker build \
    --build-arg VERSION="$IMAGE_TAG" \
    --build-arg COMMIT_HASH="$GIT_COMMIT" \
    --build-arg BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
    -t "${IMG_REPO}:${IMAGE_TAG}" "$ROOT_DIR"
docker push "${IMG_REPO}:${IMAGE_TAG}"

echo -e "${BLUE}Building init-cert image...${NC}"
docker build -t "${INIT_CERT_IMG_REPO}:${IMAGE_TAG}" -f "$ROOT_DIR/cmd/init-cert/Dockerfile" "$ROOT_DIR"
docker push "${INIT_CERT_IMG_REPO}:${IMAGE_TAG}"

if [ "$UI_ENABLED" = "true" ]; then
    echo -e "${BLUE}Building UI image...${NC}"
    docker build -t "${UI_IMG_REPO}:${IMAGE_TAG}" -f "$ROOT_DIR/ui/Dockerfile" "$ROOT_DIR/ui"
    docker push "${UI_IMG_REPO}:${IMAGE_TAG}"
fi

# Upgrade release
echo -e "${BLUE}Upgrading Helm release...${NC}"
helm upgrade exc \
    --reuse-values \
    --set image.repository="$IMG_REPO" \
    --set image.tag="$IMAGE_TAG" \
    --set ui.image.repository="$UI_IMG_REPO" \
    --set ui.image.tag="$IMAGE_TAG" \
    --set initCert.image.repository="$INIT_CERT_IMG_REPO" \
    --set initCert.image.tag="$IMAGE_TAG" \
    --namespace envoy-xds-controller \
    "$ROOT_DIR/helm/charts/envoy-xds-controller" \
    --timeout='5m' --wait

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  Update complete!${NC}"
echo -e "${GREEN}============================================${NC}"
