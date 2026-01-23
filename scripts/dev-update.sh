#!/bin/bash
# Rebuild and redeploy to existing dev cluster
# Usage:
#   ./dev-update.sh              # Update all components
#   COMPONENTS=backend ./dev-update.sh   # Update only backend (controller + init-cert)
#   COMPONENTS=frontend ./dev-update.sh  # Update only frontend (UI)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Image settings
LOCAL_REGISTRY="${LOCAL_REGISTRY:-localhost:5001}"
GIT_COMMIT=$(git rev-parse --short HEAD)
IMAGE_TAG="${IMAGE_TAG:-$GIT_COMMIT}"

IMG_REPO="${LOCAL_REGISTRY}/envoy-xds-controller"
UI_IMG_REPO="${LOCAL_REGISTRY}/envoy-xds-controller-ui"
INIT_CERT_IMG_REPO="${LOCAL_REGISTRY}/envoy-xds-controller-init-cert"

# Components to build (default: all)
COMPONENTS="${COMPONENTS:-all}"

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Updating dev deployment (${IMAGE_TAG})${NC}"
echo -e "${BLUE}  Components: ${COMPONENTS}${NC}"
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

# Build backend images
if [ "$COMPONENTS" = "all" ] || [ "$COMPONENTS" = "backend" ]; then
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
fi

# Build frontend image
if [ "$COMPONENTS" = "all" ] || [ "$COMPONENTS" = "frontend" ]; then
    if [ "$UI_ENABLED" = "true" ]; then
        echo -e "${BLUE}Building UI image...${NC}"
        docker build -t "${UI_IMG_REPO}:${IMAGE_TAG}" -f "$ROOT_DIR/ui/Dockerfile" "$ROOT_DIR/ui"
        docker push "${UI_IMG_REPO}:${IMAGE_TAG}"
    else
        echo -e "${YELLOW}UI is disabled, skipping frontend build${NC}"
    fi
fi

# Upgrade release
echo -e "${BLUE}Upgrading Helm release...${NC}"
helm upgrade exc \
    --reuse-values \
    --set image.repository="$IMG_REPO" \
    --set image.tag="$IMAGE_TAG" \
    --set image.pullPolicy=Always \
    --set ui.image.repository="$UI_IMG_REPO" \
    --set ui.image.tag="$IMAGE_TAG" \
    --set ui.image.pullPolicy=Always \
    --set initCert.image.repository="$INIT_CERT_IMG_REPO" \
    --set initCert.image.tag="$IMAGE_TAG" \
    --set initCert.image.pullPolicy=Always \
    --namespace envoy-xds-controller \
    "$ROOT_DIR/helm/charts/envoy-xds-controller" \
    --timeout='5m' --wait

# Restart pods to pick up new images (needed when tag doesn't change)
echo -e "${BLUE}Restarting pods to pick up new images...${NC}"
if [ "$COMPONENTS" = "all" ] || [ "$COMPONENTS" = "backend" ]; then
    kubectl -n envoy-xds-controller rollout restart deployment -l app.kubernetes.io/name=envoy-xds-controller
fi
if [ "$COMPONENTS" = "all" ] || [ "$COMPONENTS" = "frontend" ]; then
    if [ "$UI_ENABLED" = "true" ]; then
        kubectl -n envoy-xds-controller rollout restart deployment -l app.kubernetes.io/name=envoy-xds-controller-ui
    fi
fi

# Wait for rollout
echo -e "${BLUE}Waiting for rollout to complete...${NC}"
kubectl -n envoy-xds-controller rollout status deployment -l app.kubernetes.io/instance=exc --timeout=120s

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  Update complete!${NC}"
echo -e "${GREEN}============================================${NC}"
