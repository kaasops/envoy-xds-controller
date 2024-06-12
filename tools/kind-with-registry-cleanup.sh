#!/bin/sh
set -o errexit

# 0. Check commands
function check_command() {
    local command=$1

    if ! command -v $command &> /dev/null; then
        echo "Error: ${command} not found"
        exit 1
    fi
}

check_command kind

# 1. Stop and delete registry container if it exists
reg_name='kind-registry'
if [ "$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)" == 'true' ]; then
  docker stop "${reg_name}"
  docker rm "${reg_name}"
fi

# 2. Delete kind cluster
kind delete cluster --name exc