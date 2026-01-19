#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Generate LDAP config from dev-users.yaml
bash "${SCRIPT_DIR}/ldap/generate.sh"

kubectl create ns ldap --dry-run=client -o yaml | kubectl apply -f -
kubectl create ns dex --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f "${SCRIPT_DIR}/ldap" -n ldap

helm repo add dex https://charts.dexidp.io

helm install dex -n dex dex/dex --values "${SCRIPT_DIR}/dex/values.yaml"