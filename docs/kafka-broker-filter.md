# Kafka Broker Filter Support

This document describes how to proxy Kafka traffic through Envoy using the Kafka Broker Filter with automatic broker address rewriting.

## Overview

The Envoy Kafka Broker Filter enables transparent Kafka proxying by intercepting Kafka protocol messages and rewriting broker addresses in metadata responses. This allows external clients to connect to Kafka through Envoy without modifying Kafka's `advertised.listeners` configuration.

## Requirements

| Requirement | Value | Notes |
|-------------|-------|-------|
| **Kafka version** | **≤ 3.8.0** | Kafka 4.0+ is NOT supported |
| **Envoy image** | `envoyproxy/envoy-contrib` | Standard `envoy` image does not include the filter |
| **Protocol** | Plaintext | TLS not supported with address rewriting |

> **Important**: The Kafka Broker Filter is **experimental** and under active development. Configuration structures may change. Thorough testing is recommended before production use.

## Architecture

Each Kafka broker requires a dedicated port on Envoy:

```
Client ──► Envoy :19092 ──► kafka-broker-0:9092
       ──► Envoy :19093 ──► kafka-broker-1:9092
       ──► Envoy :19094 ──► kafka-broker-2:9092
```

The filter rewrites broker addresses in Kafka protocol responses (Metadata, FindCoordinator, etc.) so clients receive Envoy addresses instead of internal broker addresses.

## Configuration

### Listener

Create a Listener for each broker with the Kafka Broker Filter **before** TCP Proxy:

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: Listener
metadata:
  name: kafka-broker-0-listener
spec:
  name: kafka-broker-0-listener
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 19092
  filter_chains:
    - filters:
        - name: envoy.filters.network.kafka_broker
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.network.kafka_broker.v3.KafkaBroker
            stat_prefix: kafka-broker-0
            id_based_broker_address_rewrite_spec:
              rules:
                - id: 0
                  host: envoy.example.com
                  port: 19092
                - id: 1
                  host: envoy.example.com
                  port: 19093
                - id: 2
                  host: envoy.example.com
                  port: 19094
        - name: envoy.filters.network.tcp_proxy
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
            stat_prefix: kafka-broker-0
            cluster: kafka-broker-0
```

**Key points**:
- `id_based_broker_address_rewrite_spec.rules` must contain ALL brokers in the cluster
- `id` corresponds to Kafka's `broker.id` or `node.id`
- `host` and `port` are the external Envoy addresses that clients will use
- The same rules should be replicated in all broker listeners

### Cluster

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: Cluster
metadata:
  name: kafka-broker-0
spec:
  name: kafka-broker-0
  connect_timeout: 5s
  type: STRICT_DNS
  load_assignment:
    cluster_name: kafka-broker-0
    endpoints:
      - lb_endpoints:
          - endpoint:
              address:
                socket_address:
                  address: kafka-broker-0.internal
                  port_value: 9092
```

### VirtualService

```yaml
apiVersion: envoy.kaasops.io/v1alpha1
kind: VirtualService
metadata:
  name: kafka-broker-0
  annotations:
    envoy.kaasops.io/node-id: <envoy-node-id>
spec:
  listener:
    name: kafka-broker-0-listener
```

## Verification

### Check address rewriting

```bash
kafka-broker-api-versions.sh --bootstrap-server <envoy>:19092
```

**Expected output** (Envoy addresses):
```
envoy.example.com:19092 (id: 0 rack: null) -> ...
envoy.example.com:19093 (id: 1 rack: null) -> ...
envoy.example.com:19094 (id: 2 rack: null) -> ...
```

**Problem** (internal addresses visible):
```
kafka-broker-0.internal:9092 (id: 0 rack: null) -> ...
```

### Check metrics

```bash
curl -s http://<envoy>:19000/stats | grep "kafka.*unknown"
```

If `request.unknown > 0`, the Kafka version is incompatible.

## Troubleshooting

| Symptom | Cause | Solution |
|---------|-------|----------|
| `Bootstrap broker disconnected` | Kafka 4.0+ or standard Envoy image | Use Kafka ≤3.8.0, `envoy-contrib` image |
| `request.unknown > 0` | Incompatible Kafka version | Downgrade to Kafka 3.8.0 |
| Internal addresses in metadata | Wrong broker ID in rules | Verify broker IDs match |
| Filter not applied | Filter order | `kafka_broker` must be before `tcp_proxy` |
| Consumer TimeoutException | Slow FindCoordinator via proxy | Increase client timeout |

## Limitations

1. **Kafka 4.0+** — not supported due to protocol changes
2. **TLS** — not supported with address rewriting
3. **Filter status** — experimental in Envoy
4. **Scaling** — adding brokers requires updating rules in ALL listeners

## Alternative: SNI-based Routing

For TLS support or lower latency requirements, consider SNI-based routing instead. This approach:
- Supports TLS
- Has minimal latency overhead (L4 proxy only)
- Requires configuring `advertised.listeners` on Kafka brokers

## References

- [Envoy Kafka Broker Filter Documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/network_filters/kafka_broker_filter)
- [Proxying Kafka with Envoy without changing advertised.listeners](https://adam-kotwasinski.medium.com/proxying-kafka-with-envoy-without-changing-advertised-listeners-by-using-rewrite-rules-387792cce066)
