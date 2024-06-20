#!/bin/bash
set -o errexit

# namespace="envoy-xds-controller"

# 0. Check commands
function check_command() {
    local command=$1

    if ! command -v $command &> /dev/null; then
        echo "Error: ${command} not found"
        exit 1
    fi
}

check_command kubectl

# 1. Get params
# Set defaults params
namespace="envoy-xds-controller"

while getopts "n:" flag
do
    case "${opt}" in
    n) namespace=${OPTARG}
      ;;
    esac
done

# 2. Delete Deployment
kubectl delete deployments.apps -n ${namespace} envoy

# 3. Delete ConfigMap
kubectl  delete configmaps -n ${namespace} envoy-config

# 4. Delete Service
kubectl delete service -n ${namespace} envoy