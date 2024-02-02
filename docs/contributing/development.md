### Setup the development environment


If you don't need Validation WebHook for working - start localy Envoy xDS Controller with env `WEBHOOK_DISABLE` = `true`.

If you need full instalation with Validation Webhook logic on local instance Envoy xDS controller, you need Kubernetes with network access to workstation (Laptop). For example you can use [KIND](https://kind.sigs.k8s.io/).

Deploy Helm Envoy xDS Controller to you kubernetes:

```bash
cd helm/charts/envoy-xds-controller
helm upgrade envoy --install --namespace envoy-xds-controller --create-namespace .
```

Wait when Pod starting. After this - set Replicas for Envoy xDS Controller to 0.

```bash
kubectl scale deployment -n envoy-xds-controller envoy-envoy-xds-controller --replicas 0
```

After this, create dir for local certificates for Webhook Server:

```bash
mkdir -p /tmp/k8s-webhook-server/serving-certs
```

Copy generated certificate and key for Webhook Server:

```bash
kubectl get secrets -n envoy-xds-controller envoy-xds-controller-tls -o jsonpath='{.data.tls\.crt}' | base64 -D > /tmp/k8s-webhook-server/serving-certs/tls.crt
kubectl get secrets -n envoy-xds-controller envoy-xds-controller-tls -o jsonpath='{.data.tls\.key}' | base64 -D > /tmp/k8s-webhook-server/serving-certs/tls.key
```

Delete service for Werhook

```bash
kubectl delete service -n envoy-xds-controller envoy-xds-controller-webhook-service
```

Apply new service. Insert you IP to <WORKSTATION_IP>:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: envoy-xds-controller-webhook-service
  namespace: envoy-xds-controller
spec:
  ports:
    - protocol: TCP
      port: 443
      targetPort: 9443
---
apiVersion: v1
kind: Endpoints
metadata:
  name: envoy-xds-controller-webhook-service
  namespace: envoy-xds-controller
subsets:
  - addresses:
      - ip: 172.28.128.20
    ports:
      - port: 9443
```

