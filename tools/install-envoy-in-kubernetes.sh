#!/bin/sh
set -o errexit

# namespace="envoy-xds-controller"

# 0. Check commands
function check_command() {
    local command=$1

    if ! command -v $command &> /dev/null; then
        echo "Error: ${command} not found"
        exit 1
    fi
}

check_command kubectl

# 1. Get params
# Set defaults params
namespace="envoy-xds-controller"
nodeid="test"
envoy_version=v1.30.2

while getopts "h?niv:" flag
do
    case "${opt}" in
    h|\?)
      echo "Usage: $0 [-n namespace] [-i nodeid] [-v envoy_version]"
      exit 0
      ;;
    n) namespace=${OPTARG}
      ;;
    i) nodeid=${OPTARG}
      ;;
    v) envoy_version=${OPTARG}
      ;;
    esac
done


# 1. Create ConfigMap with envoy.yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: envoy-config
  namespace: ${namespace}
data:
  envoy.yaml: |
    admin:
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 19000
    dynamic_resources:
      ads_config:
        api_type: DELTA_GRPC
        transport_api_version: V3
        set_node_on_first_message_only: true
        grpc_services:
        - envoy_grpc:
            cluster_name: xds_cluster
      lds_config:
        resource_api_version: V3
        ads: {}
      cds_config:
        resource_api_version: V3
        ads: {}
    node:
      cluster: ${nodeid}
      id: ${nodeid}
    static_resources:
      clusters:
      - typed_extension_protocol_options:
          envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
            "@type": type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
            explicit_http_config:
              http2_protocol_options:
                connection_keepalive:
                  interval: 30s
                  timeout: 50s
        connect_timeout: 100s
        load_assignment:
          cluster_name: xds_cluster
          endpoints:
          - lb_endpoints:
            - endpoint:
                address:
                  socket_address:
                    address: exc-envoy-xds-controller
                    port_value: 9000
        http2_protocol_options: {}
        name: xds_cluster
        type: LOGICAL_DNS
    layered_runtime:
      layers:
        - name: runtime-0
          rtds_layer:
            rtds_config:
              resource_api_version: V3
              api_config_source:
                transport_api_version: V3
                api_type: GRPC
                grpc_services:
                  envoy_grpc:
                    cluster_name: xds_cluster
            name: runtime-0
EOF

# 2. Create Deployment

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: envoy
  name: envoy
  namespace: ${namespace}
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: envoy
  template:
    metadata:
      labels:
        app.kubernetes.io/name: envoy
    spec:
      containers:
      - args:
        - -c /etc/envoy/envoy.yaml
        - --log-level debug
        image: envoyproxy/envoy:${envoy_version}
        imagePullPolicy: IfNotPresent
        name: envoy
        ports:
        - containerPort: 19000
          name: admin
        - containerPort: 10080
          name: http
        - containerPort: 10443
          name: https
        securityContext:
          allowPrivilegeEscalation: true
          capabilities:
            add:
            - NET_BIND_SERVICE
        volumeMounts:
        - mountPath: /etc/envoy
          name: config
          readOnly: true
      nodeSelector:
        envoy: "true"
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      restartPolicy: Always
      volumes:
      - name: config
        configMap:
          defaultMode: 420
          name: envoy-config
EOF

# 3. Create Service
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: envoy
  name: envoy
  namespace: ${namespace}
spec:
  ports:
  - name: admin
    port: 19000
    protocol: TCP
    targetPort: admin
  - name: http
    port: 80
    protocol: TCP
    targetPort: http
  - name: https
    port: 443
    protocol: TCP
    targetPort: https
  selector:
    app.kubernetes.io/name: envoy
  type: ClusterIP
EOF