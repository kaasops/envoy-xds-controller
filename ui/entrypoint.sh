#!/bin/bash

npx vite-inject-env set -d /usr/share/nginx/html

if [ -z "$API_PROXY_PASS" ]; then
  echo "ERROR: environment variable API_PROXY_PASS not set"
  exit 1
fi

if [ -z "$GRPC_API_PROXY_PASS" ]; then
  echo "ERROR: environment variable GRPC_API_PROXY_PASS not set"
  exit 1
fi

envsubst '${API_PROXY_PASS},${GRPC_API_PROXY_PASS}' < /etc/nginx/conf.d/template.conf > /etc/nginx/conf.d/default.conf
rm /etc/nginx/conf.d/template.conf
nginx -g 'daemon off;'