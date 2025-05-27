FROM node:22.11.0-alpine as build

RUN npm install -g vite
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build




FROM nginx:1.25.3

ENV API_PROXY_PASS=http://exc-envoy-xds-controller-cache-api:9999
ENV GRPC_API_PROXY_PASS=http://exc-envoy-xds-controller-grpc-api:10000

COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.cfg/template.conf /etc/nginx/conf.d/template.conf

COPY entrypoint.sh /entrypoint.sh

RUN apt-get update && \
    apt-get install -yq --no-install-recommends \
    curl \
    wget \
    git \
    gnupg

RUN curl -fsSL https://deb.nodesource.com/setup_current.x | bash - && \
    apt-get install -y nodejs \
    build-essential && \
    node --version && \
    npm --version

WORKDIR /usr/share/nginx/html

RUN npm install vite-inject-env

RUN set -ex \
    && usermod -u 1000 nginx \
    && groupmod -g 1000 nginx \
    && chown -R nginx:nginx /var/cache/nginx \
    && chown -R nginx:nginx /var/log/nginx \
    && chown -R nginx:nginx /etc/nginx/conf.d \
    && chmod -R 755 /var/log/nginx \
    && chmod -R 755 /etc/nginx/conf.d \
    && touch /var/run/nginx.pid \
    && chown -R nginx:nginx /var/run/nginx.pid \
    && chown -R nginx:nginx /usr/share/nginx\
    && chmod -R 777 /var/run/nginx.pid \
    && mkdir /nonexistent \
    && chown -R nginx:nginx /nonexistent

EXPOSE 8080

USER nginx
ENTRYPOINT [ "/entrypoint.sh" ]
