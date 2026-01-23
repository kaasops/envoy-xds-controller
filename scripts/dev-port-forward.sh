#!/bin/bash
# Start all port-forwards for dev environment

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}Stopping port-forwards...${NC}"
    kill $(jobs -p) 2>/dev/null
    exit 0
}

trap cleanup SIGINT SIGTERM

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Starting Port Forwards${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

PIDS=()

# UI port-forward
if kubectl -n envoy-xds-controller get svc exc-envoy-xds-controller-ui &>/dev/null; then
    echo -e "${GREEN}Starting UI port-forward (8080)...${NC}"
    kubectl -n envoy-xds-controller port-forward svc/exc-envoy-xds-controller-ui 8080:8080 &
    PIDS+=($!)
else
    echo -e "${YELLOW}UI service not found, skipping...${NC}"
fi

# Dex port-forward
if kubectl -n dex get svc dex &>/dev/null; then
    echo -e "${GREEN}Starting Dex port-forward (5556)...${NC}"
    kubectl -n dex port-forward svc/dex 5556:5556 &
    PIDS+=($!)

    # Check /etc/hosts
    if ! grep -q "dex.dex" /etc/hosts 2>/dev/null; then
        echo ""
        echo -e "${RED}Warning: dex.dex not in /etc/hosts${NC}"
        echo -e "Run: ${YELLOW}echo \"127.0.0.1 dex.dex\" | sudo tee -a /etc/hosts${NC}"
    fi
else
    echo -e "${YELLOW}Dex service not found, skipping...${NC}"
fi

# Envoy port-forward
if kubectl -n default get svc envoy &>/dev/null; then
    echo -e "${GREEN}Starting Envoy port-forward (10080/http, 10443/https, 19000/admin)...${NC}"
    kubectl -n default port-forward svc/envoy 10080:80 10443:443 19000:19000 &
    PIDS+=($!)
else
    echo -e "${YELLOW}Envoy service not found, skipping...${NC}"
fi

if [ ${#PIDS[@]} -eq 0 ]; then
    echo -e "${RED}No services to forward. Is the dev environment running?${NC}"
    echo -e "Run ${YELLOW}make dev${NC} first."
    exit 1
fi

echo ""
echo -e "${GREEN}Port forwards active:${NC}"
kubectl -n envoy-xds-controller get svc exc-envoy-xds-controller-ui &>/dev/null && echo "  - UI:    http://localhost:8080"
kubectl -n dex get svc dex &>/dev/null && echo "  - Dex:   http://dex.dex:5556"
if kubectl -n default get svc envoy &>/dev/null; then
    echo "  - Envoy: http://localhost:10080 (http), https://localhost:10443 (https)"
    echo "           http://localhost:19000 (admin)"
fi
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop all port-forwards${NC}"
echo ""

# Wait for all background processes
wait
