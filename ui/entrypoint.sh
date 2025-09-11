#!/bin/sh
set -euo pipefail

HTML_DIR="/usr/share/nginx/html"
mkdir -p "$HTML_DIR/build"
cd "$HTML_DIR"

vite-inject-env set -d /usr/share/nginx/html

if [ -z "$API_PROXY_PASS" ]; then
  echo "ERROR: environment variable API_PROXY_PASS not set" >&2
  exit 1
fi

if [ -z "$GRPC_API_PROXY_PASS" ]; then
  echo "ERROR: environment variable GRPC_API_PROXY_PASS not set" >&2
  exit 1
fi

envsubst '${API_PROXY_PASS},${GRPC_API_PROXY_PASS}' < /etc/nginx/conf.d/template.conf > /etc/nginx/conf.d/default.conf
rm -f /etc/nginx/conf.d/template.conf
exec nginx -g 'pid /tmp/nginx.pid; daemon off;'