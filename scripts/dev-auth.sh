#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kubectl create ns ldap
kubectl create ns dex

kubectl apply -f "${SCRIPT_DIR}/ldap" -n ldap

helm repo add dex https://charts.dexidp.io

helm install dex -n dex dex/dex --values "${SCRIPT_DIR}/dex/values.yaml"