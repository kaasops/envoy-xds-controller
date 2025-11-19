#!/bin/sh
set -euo pipefail

HTML_DIR="/usr/share/nginx/html"
mkdir -p "$HTML_DIR/build"
cd "$HTML_DIR"

# Inject environment variables into Vite build
vite-inject-env set -d /usr/share/nginx/html

# Determine deployment mode and configure nginx
if [ -f "/etc/nginx/templates/nginx.conf" ]; then
  # Helm deployment mode: use ConfigMap-provided config
  echo "INFO: Using Helm-provided nginx configuration from ConfigMap"
  rm -f /etc/nginx/conf.d/*.conf
  cp /etc/nginx/templates/nginx.conf /etc/nginx/conf.d/default.conf
else
  # Standalone deployment mode: use envsubst with environment variables
  echo "INFO: Using standalone nginx configuration with envsubst"

  # Validate required environment variables for standalone mode
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
fi

exec nginx -g 'pid /tmp/nginx.pid; daemon off;'