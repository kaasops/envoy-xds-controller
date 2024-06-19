#!/bin/bash
set -o errexit

# 1. Stop and delete registry container if it exists
reg_name='kind-registry'
if [ "$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)" == 'true' ]; then
  docker stop "${reg_name}"
  docker rm "${reg_name}"
fi

# 2. Delete kind cluster
tools/bin/kind delete cluster --name exc