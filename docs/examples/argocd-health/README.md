# VirtualService health in Argo CD

This health check reports the snapshot publication result for the current VirtualService generation. When publication fails, Argo CD displays the reason from `status.message`. Check xDS acknowledgements separately to see what Envoy has accepted. If several replicas build snapshots and write status for the same VirtualService, Kubernetes shows the most recent write. Check each replica's logs for its publication result.

| VirtualService status | Argo CD health |
| --- | --- |
| The controller has not recorded a result for the current `metadata.generation` | `Progressing` |
| `status.observedGeneration` matches `metadata.generation` and `status.invalid` is `true` | `Degraded` |
| The generations match and `status.invalid` is `false` | `Healthy` |

## Installation

Install the CRD and controller versions that support `status.observedGeneration`. You need permission to patch the `argocd-cm` ConfigMap. Run the following command from the repository root with your Kubernetes context set to the target cluster. Replace `argocd` if Argo CD runs in a different namespace.

```sh
kubectl -n argocd patch configmap argocd-cm --type=merge \
  --patch-file docs/examples/argocd-health/argocd-cm.patch.yaml
```

The patch updates one key in `data` and preserves the other settings. If Git manages your Argo CD configuration, add that key to the source manifest for `argocd-cm`. [Argo CD custom health checks](https://argo-cd.readthedocs.io/en/stable/operator-manual/health/#configuring-custom-health-checks)

For Kustomize, copy `argocd-cm.patch.yaml` next to `kustomization.yaml` and add the following entry to its `patches` list. Set `target.namespace` to the namespace in the source ConfigMap manifest. If that manifest omits the namespace, omit `target.namespace` as well.

```yaml
patches:
  - path: argocd-cm.patch.yaml
    target: {version: v1, kind: ConfigMap, name: argocd-cm, namespace: argocd}
```

## Verification

From the repository root, run the test with Python 3, `kubectl` with Kustomize support, and the `argocd` CLI installed. The test renders the example configuration and runs the Argo CD health check against VirtualServices with different publication outcomes.

```sh
python3 test/argocd-health/test_health.py
```

In a test application, apply a dependency change that passes admission but prevents snapshot publication. The VirtualService should become `Degraded` and display the cause. Sync the fix and verify that the VirtualService returns to `Healthy`. If you use sync waves, make the fix available in the same wave or an earlier one. [Argo CD sync waves and hooks](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/#combining-sync-waves-and-hooks)
