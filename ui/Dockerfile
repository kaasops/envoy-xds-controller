FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.25.3-alpine AS runtime

ENV API_PROXY_PASS=http://exc-envoy-xds-controller-cache-api:9999 \
    GRPC_API_PROXY_PASS=http://exc-envoy-xds-controller-grpc-api:10000

RUN apk add --no-cache nodejs npm gettext \
 && npm i -g vite-inject-env@latest \
 && rm -rf /root/.npm/_cacache

WORKDIR /usr/share/nginx/html

COPY --from=build --chown=65532:65532 /app/dist /usr/share/nginx/html
COPY --chown=65532:65532 nginx.cfg/template.conf /etc/nginx/conf.d/template.conf

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh \
 && chown 65532:65532 /entrypoint.sh

RUN mkdir -p /var/run /var/log/nginx /var/cache/nginx /var/lib/nginx \
 && touch /var/run/nginx.pid \
 && chown -R 65532:65532 /var/run /var/log/nginx /var/cache/nginx /var/lib/nginx \
 && chown -R 65532:65532 /etc/nginx/conf.d /usr/share/nginx/html \
 && chmod -R u+rwX,g+rX,o+rX /usr/share/nginx/html /etc/nginx/conf.d \
 && chmod -R u+rwX /var/run /var/log/nginx /var/cache/nginx /var/lib/nginx

RUN sed -ri '/^\s*pid\s+/d; s/^\s*user\s+\S+;/# &/' /etc/nginx/nginx.conf

EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/entrypoint.sh"]