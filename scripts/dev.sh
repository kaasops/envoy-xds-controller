#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Image settings (match Makefile variable names)
LOCAL_REGISTRY="${LOCAL_REGISTRY:-localhost:5001}"
GIT_COMMIT=$(git rev-parse --short HEAD)
IMAGE_TAG="${IMAGE_TAG:-$GIT_COMMIT}"

IMG_REPO="${LOCAL_REGISTRY}/envoy-xds-controller"
UI_IMG_REPO="${LOCAL_REGISTRY}/envoy-xds-controller-ui"
INIT_CERT_IMG_REPO="${LOCAL_REGISTRY}/envoy-xds-controller-init-cert"

# Defaults
UI_ENABLED=true
AUTH_ENABLED=true
PROMETHEUS_ENABLED=false
DEV_MODE=true
DEPLOY_ENVOY=true
APPLY_TEST_RESOURCES=true

# Banner
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Envoy xDS Controller - Dev Setup${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check Kind cluster
if ! kind get clusters 2>/dev/null | grep -q 'kind'; then
    echo -e "${RED}Error: No Kind cluster found.${NC}"
    echo -e "Run ${YELLOW}make kr${NC} first to create cluster with registry."
    exit 1
fi

echo -e "${GREEN}Kind cluster found.${NC}"
echo ""

# Interactive prompts
echo "Select options for local development:"
echo ""

read -p "Enable UI? [Y/n]: " ui_choice
[[ "$ui_choice" =~ ^[Nn]$ ]] && UI_ENABLED=false

read -p "Enable Auth (OIDC via Dex)? [Y/n]: " auth_choice
[[ "$auth_choice" =~ ^[Nn]$ ]] && AUTH_ENABLED=false

read -p "Install Prometheus Operator? [y/N]: " prom_choice
[[ "$prom_choice" =~ ^[Yy]$ ]] && PROMETHEUS_ENABLED=true

read -p "Development mode (verbose logging)? [Y/n]: " dev_choice
[[ "$dev_choice" =~ ^[Nn]$ ]] && DEV_MODE=false

read -p "Deploy test Envoy proxy? [Y/n]: " envoy_choice
[[ "$envoy_choice" =~ ^[Nn]$ ]] && DEPLOY_ENVOY=false

read -p "Apply test resources (VirtualServices, etc.)? [Y/n]: " resources_choice
[[ "$resources_choice" =~ ^[Nn]$ ]] && APPLY_TEST_RESOURCES=false

echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  UI:              $UI_ENABLED"
echo "  Auth:            $AUTH_ENABLED"
echo "  Prometheus:      $PROMETHEUS_ENABLED"
echo "  Dev mode:        $DEV_MODE"
echo "  Deploy Envoy:    $DEPLOY_ENVOY"
echo "  Test resources:  $APPLY_TEST_RESOURCES"
echo ""

# Auth setup (if enabled)
if [ "$AUTH_ENABLED" = true ]; then
    echo -e "${BLUE}Setting up Auth (Dex + LDAP)...${NC}"
    bash "${SCRIPT_DIR}/dev-auth.sh"
fi

# Prometheus setup (if enabled)
if [ "$PROMETHEUS_ENABLED" = true ]; then
    echo -e "${BLUE}Installing Prometheus Operator...${NC}"
    make -C "$ROOT_DIR" install-prometheus
fi

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

if [ "$UI_ENABLED" = true ]; then
    echo -e "${BLUE}Building UI image...${NC}"
    docker build -t "${UI_IMG_REPO}:${IMAGE_TAG}" -f "$ROOT_DIR/ui/Dockerfile" "$ROOT_DIR/ui"
    docker push "${UI_IMG_REPO}:${IMAGE_TAG}"
fi

# Uninstall existing release if present
echo -e "${BLUE}Removing existing Helm release (if any)...${NC}"
helm uninstall exc -n envoy-xds-controller 2>/dev/null || true

# Deploy via Helm
echo -e "${BLUE}Deploying with Helm...${NC}"
helm install exc \
    --set metrics.address=:8443 \
    --set metrics.secure=false \
    --set metrics.serviceMonitor.enabled="$PROMETHEUS_ENABLED" \
    --set development="$DEV_MODE" \
    --set auth.enabled="$AUTH_ENABLED" \
    --set ui.enabled="$UI_ENABLED" \
    --set cacheAPI.enabled=true \
    --set resourceAPI.enabled=true \
    --set 'watchNamespaces={default}' \
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
    --create-namespace "$ROOT_DIR/helm/charts/envoy-xds-controller" \
    --timeout='5m' --wait

# Deploy Envoy (if enabled)
if [ "$DEPLOY_ENVOY" = true ]; then
    echo -e "${BLUE}Deploying test Envoy proxy...${NC}"
    kubectl apply -f "$ROOT_DIR/dev/envoy"
fi

# Apply test resources (if enabled)
if [ "$APPLY_TEST_RESOURCES" = true ]; then
    echo -e "${BLUE}Applying test resources...${NC}"
    kubectl -n envoy-xds-controller apply -f "$ROOT_DIR/dev/testdata/common"
fi

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  Deployment complete!${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""
echo "Useful commands:"
echo "  kubectl -n envoy-xds-controller get pods"
echo "  kubectl -n envoy-xds-controller logs -f deployment/exc-envoy-xds-controller"
if [ "$DEPLOY_ENVOY" = true ]; then
    echo "  kubectl -n default logs -f deployment/envoy"
fi
if [ "$APPLY_TEST_RESOURCES" = true ]; then
    echo "  kubectl -n envoy-xds-controller get virtualservices"
fi

# Port forward section
if [ "$UI_ENABLED" = true ] || [ "$AUTH_ENABLED" = true ] || [ "$DEPLOY_ENVOY" = true ]; then
    echo ""
    echo -e "${YELLOW}Port forwards (run in separate terminals or use 'make dev-port-forward'):${NC}"
    if [ "$UI_ENABLED" = true ]; then
        echo "  kubectl -n envoy-xds-controller port-forward svc/exc-envoy-xds-controller-ui 8080:8080"
    fi
    if [ "$AUTH_ENABLED" = true ]; then
        echo "  kubectl -n dex port-forward svc/dex 5556:5556"
    fi
    if [ "$DEPLOY_ENVOY" = true ]; then
        echo "  kubectl -n default port-forward svc/envoy 10080:80 10443:443 19000:19000"
    fi
fi

# /etc/hosts hint for Dex
if [ "$AUTH_ENABLED" = true ]; then
    echo ""
    if grep -q "dex.dex" /etc/hosts 2>/dev/null; then
        echo -e "${GREEN}/etc/hosts: dex.dex is configured${NC}"
    else
        echo -e "${RED}Important: Add dex.dex to /etc/hosts for authentication to work:${NC}"
        echo -e "  ${YELLOW}echo \"127.0.0.1 dex.dex\" | sudo tee -a /etc/hosts${NC}"
    fi
fi

# Show test credentials if auth is enabled
if [ "$AUTH_ENABLED" = true ]; then
    bash "${SCRIPT_DIR}/dev-creds.sh"
fi

# Ask to open browser
if [ "$UI_ENABLED" = true ]; then
    echo ""
    read -p "Start port-forwards and open UI in browser? [Y/n]: " open_choice
    if [[ ! "$open_choice" =~ ^[Nn]$ ]]; then
        echo ""
        echo -e "${BLUE}Starting port-forwards...${NC}"

        # Start port-forwards in background
        kubectl -n envoy-xds-controller port-forward svc/exc-envoy-xds-controller-ui 8080:8080 &>/dev/null &
        PF_UI_PID=$!

        if [ "$AUTH_ENABLED" = true ]; then
            kubectl -n dex port-forward svc/dex 5556:5556 &>/dev/null &
            PF_DEX_PID=$!
        fi

        if [ "$DEPLOY_ENVOY" = true ]; then
            kubectl -n default port-forward svc/envoy 10080:80 10443:443 19000:19000 &>/dev/null &
            PF_ENVOY_PID=$!
        fi

        # Wait a moment for port-forwards to establish
        sleep 2

        # Open browser (macOS: open, Linux: xdg-open)
        if command -v open &>/dev/null; then
            open "http://localhost:8080"
        elif command -v xdg-open &>/dev/null; then
            xdg-open "http://localhost:8080"
        else
            echo -e "${YELLOW}Could not detect browser opener. Please open http://localhost:8080 manually.${NC}"
        fi

        echo ""
        echo -e "${GREEN}Port-forwards running in background.${NC}"
        echo -e "To stop them: ${YELLOW}pkill -f 'port-forward'${NC}"
        echo -e "Or run: ${YELLOW}make dev-port-forward${NC} for managed port-forwards with Ctrl+C support."
    fi
fi
