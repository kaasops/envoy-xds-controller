#!/usr/bin/env bash

NAMESPACE=dex

kubectl create namespace $NAMESPACE

TEST_STATIC_PASSWORD=$(echo password | htpasswd -BinC 10 admin | cut -d: -f2)

cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: dex-config
data:
  config.yaml: |
    issuer: http://dex.${NAMESPACE}:5556
    storage:
      type: sqlite3
      config:
        file: /tmp/dex.db
    web:
      http: 0.0.0.0:5556
      allowedOrigins: ['*']
    connectors:
    - type: mockCallback
      id: mock
      name: Example
    staticClients:
    - id: envoy-xds-controller
      redirectURIs:
      - 'http://localhost:8080/callback'
      name: 'Envoy xDS controller'
      public: true
    enablePasswordDB: true
    staticPasswords:
    - email: "admin@example.com"
      hash: "${TEST_STATIC_PASSWORD}"
      username: "admin"
      userID: "08a8684b-db88-4b73-90a9-3cd1661f5466"
EOF

# Создаем Deployment для Dex
cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dex
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dex
  template:
    metadata:
      labels:
        app: dex
    spec:
      containers:
      - name: dex
        image: ghcr.io/dexidp/dex:v2.41.1
        command: ["dex"]
        args: ["serve", "/etc/dex/config.yaml"]
        ports:
        - containerPort: 5556
        volumeMounts:
        - name: config
          mountPath: /etc/dex
      volumes:
      - name: config
        configMap:
          name: dex-config
EOF

cat <<EOF | kubectl apply -n $NAMESPACE -f -
apiVersion: v1
kind: Service
metadata:
  name: dex
spec:
  type: ClusterIP
  ports:
  - port: 5556
    targetPort: 5556
  selector:
    app: dex
EOF
