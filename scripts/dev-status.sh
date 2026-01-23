#!/bin/bash
# Show status of dev environment

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Dev Environment Status${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check Kind cluster
if ! kind get clusters 2>/dev/null | grep -q 'kind'; then
    echo -e "${RED}Kind cluster: Not running${NC}"
    echo -e "Run ${YELLOW}make kr${NC} to create cluster"
    exit 1
fi
echo -e "${GREEN}Kind cluster: Running${NC}"
echo ""

# Check Helm release
if helm status exc -n envoy-xds-controller &>/dev/null; then
    echo -e "${GREEN}Helm release 'exc': Installed${NC}"

    # Get current config
    VALUES=$(helm get values exc -n envoy-xds-controller -o json 2>/dev/null)
    UI_ENABLED=$(echo "$VALUES" | grep -o '"ui":{[^}]*"enabled":[^,}]*' | grep -o 'true\|false' | head -1 || echo "unknown")
    AUTH_ENABLED=$(echo "$VALUES" | grep -o '"auth":{[^}]*"enabled":[^,}]*' | grep -o 'true\|false' | head -1 || echo "unknown")

    echo "  UI enabled: $UI_ENABLED"
    echo "  Auth enabled: $AUTH_ENABLED"
else
    echo -e "${YELLOW}Helm release 'exc': Not installed${NC}"
    echo -e "Run ${YELLOW}make dev${NC} to deploy"
    exit 0
fi
echo ""

# Pods status
echo -e "${CYAN}Pods:${NC}"
kubectl -n envoy-xds-controller get pods -o wide 2>/dev/null || echo "  No pods found"
echo ""

# Check Dex
if kubectl get namespace dex &>/dev/null; then
    echo -e "${CYAN}Dex (Auth):${NC}"
    kubectl -n dex get pods -o wide 2>/dev/null || echo "  No pods found"
    echo ""
fi

# Check Envoy
if kubectl get deployment envoy -n default &>/dev/null; then
    echo -e "${CYAN}Envoy (test proxy):${NC}"
    kubectl -n default get pods -l app=envoy -o wide 2>/dev/null || echo "  No pods found"
    echo ""
fi

# Services
echo -e "${CYAN}Services:${NC}"
kubectl -n envoy-xds-controller get svc 2>/dev/null || echo "  No services found"
echo ""

# Port-forward reminder
echo -e "${YELLOW}Port forwards (run in separate terminals or use 'make dev-port-forward'):${NC}"
echo "  kubectl -n envoy-xds-controller port-forward svc/exc-envoy-xds-controller-ui 8080:8080"
if kubectl get namespace dex &>/dev/null; then
    echo "  kubectl -n dex port-forward svc/dex 5556:5556"
fi
if kubectl get deployment envoy -n default &>/dev/null; then
    echo "  kubectl -n default port-forward svc/envoy 10080:80 10443:443 19000:19000"
fi
echo ""

# Check /etc/hosts for dex
if kubectl get namespace dex &>/dev/null; then
    if grep -q "dex.dex" /etc/hosts 2>/dev/null; then
        echo -e "${GREEN}/etc/hosts: dex.dex configured${NC}"
    else
        echo -e "${RED}/etc/hosts: dex.dex NOT configured${NC}"
        echo -e "  Run: ${YELLOW}echo \"127.0.0.1 dex.dex\" | sudo tee -a /etc/hosts${NC}"
    fi
    echo ""
fi

# Quick links
echo -e "${CYAN}Access:${NC}"
echo "  UI: http://localhost:8080"
if kubectl get namespace dex &>/dev/null; then
    echo "  Dex: http://dex.dex:5556"
fi
