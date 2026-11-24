# Node snapshots

Each Envoy node receives a snapshot over xDS. In the console, **Updates** is **Active** while the controller pushes changes and **Paused** while it holds them. The node page shows these snapshots.

| Snapshot | Contents |
| --- | --- |
| Served | What this controller replica currently offers the node over xDS. A stream's ACK or NACK is recorded separately. |
| Pending | What the current Kubernetes objects build for a paused node. A failed build reports its cause in `pending.error`. An active node has `pending: null`. |
| History | Up to ten previously served snapshots, newest first. History is held in memory when the cache API and writes are enabled. |

The snapshot ID is the first 12 hexadecimal digits of a SHA-256 over the type, name and content hash of each resource. An ID identifies content. Rollback can serve the same content with higher xDS versions.

## Configuration and persistence

Enable the cache API for read access. The chart enables writes by default with `snapshotControl.enabled: true` (`API_SNAPSHOT_CONTROL`). This default also applies when upgrading with values from a chart that predates this setting. Setting it to `false` disables control operations and history. Existing pauses remain effective.

The controller stores pause records and the version floor in a Kubernetes Secret in its namespace. `SNAPSHOT_PAUSE_SECRET` selects the Secret. The chart retains the physical name `<fullname>-frozen-nodes` and the existing storage format so pauses survive an upgrade from an earlier release candidate. It has no xDS secret-type label. On startup the controller loads this Secret even when writes are disabled. A missing Secret means there are no persisted pauses. Other read errors prevent startup.

With OIDC, writes are allowed on nodes in the user's access groups and record the operator identity. Without OIDC, anyone with access to the writable cache API can control nodes. Use one controller replica for snapshot control. With multiple replicas a request changes the replica that receives it. The response identifies it in `controllerPod`.

## API

Earlier v0.19.0 release candidates used Freeze/Unfreeze and Built. In rc.3 the corresponding API terms are Pause/Resume and Pending.

All paths below are relative to `/api/v1`. Node responses include `served`, `pending`, `history`, `paused`, `pause` and `canControl`. The pause record contains `by`, `at` and `reason`.

| Request | Result |
| --- | --- |
| `GET /pausedNodes` | Returns `pausedNodes`, keyed by node ID, including paused nodes with no served snapshot after a restart. |
| `GET /nodes/{nodeId}/snapshots` | Returns the node state. While paused, the controller computes Pending on request. |
| `GET /nodes/{nodeId}/snapshots/diff?from=served&to=pending` | Compares snapshots resource by resource. Each side accepts `served`, `pending` or a snapshot ID. Add `all=true` to include unchanged resources. |
| `GET /nodes/{nodeId}/snapshots/resource?snapshot=served&type=…&name=…` | Returns a decoded resource. For a Secret, only its type, name and SHA-256 are exposed. |
| `POST /nodes/{nodeId}/pause` with `{"reason":"…"}` | Holds the node on its served snapshot. |
| `POST /nodes/{nodeId}/resume` with `{"expectPending":"<id>"}` | Pushes the reviewed pending snapshot and resumes automatic updates. |
| `POST /nodes/{nodeId}/rollback` with `{"to":"<id>","expectServed":"<id>"}` | Checks the served ID, serves the historical snapshot and pauses updates. The reason is `rollback to <id>`. Pending reflects the current objects. |

Diff groups resources as `same` (equal bytes), `sameDecoded` (different bytes that decode to equal messages), `changed`, `added` and `removed`. Secrets are compared by bytes only. The default comparison is Served to Pending and requires a paused node. On an active node specify the two references explicitly, for example a historical ID and `served`.

A successful write returns the current node state. Resume requires the ID from the comparison. A missing ID returns HTTP 400. HTTP 409 means the operation conflicts with the current state, for example the node is already paused, Pending failed to build, or the reviewed ID changed. Refresh the state and compare again. A failed comparison or resume leaves the pause in effect.

## Switching a node to another listener

Start with one writable controller replica, a node with a served snapshot and access to all the VirtualServices and templates in the change. When moving a node between two listeners on the same address, keep the change in one Argo CD sync wave. A degraded VirtualService in an earlier wave can block subsequent waves.

1. Pause updates to the node with a reason. Node updates show **Paused** and the affected VirtualServices report the pause in `status.message`.
2. Apply all affected objects. The controller continues building while serving the previous snapshot. The webhook allows the temporary listener address conflict on the paused node and returns a warning. Other errors can still block admission. With `webhook.lightDryRun: true`, a VirtualService that targets a paused node before or after the change also gets a full dry run within `webhook.dryRunTimeoutMS`. During a listener conflict the domain index and secret deletion checks account for served and pending resources.
3. Compare Served with Pending. Resolve any error in `pending.error`. Check the intended additions, removals and changes and retain the pending ID from this comparison.
4. Resume with that ID. Envoy's response appears separately under **Sync status**. Check the affected streams and the traffic path.

Rollback serves the selected historical snapshot immediately and pauses updates. Git still contains the desired objects. Before resuming, reconcile those objects with the configuration you want to serve, then review Pending again.

## Restart and cost

Pause records survive a controller restart. Served snapshots and history contain sensitive resources and stay in memory. After a restart, a paused node can therefore have a Pending snapshot and no Served snapshot. Running Envoys retain their previous configuration. A cold Envoy receives no configuration from that replica until updates resume. The console compares the pending resources against an empty baseline so the operator can review and resume.

Reading a paused node's state or comparing with Pending triggers a dry build of all nodes. Request these views when opening or explicitly refreshing a node. The node list uses `GET /pausedNodes` and `GET /clients`.
