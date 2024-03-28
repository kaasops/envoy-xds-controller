#!/bin/bash

npx vite-inject-env set -d /usr/share/nginx/html

nginx -g 'daemon off;'