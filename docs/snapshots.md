# Node snapshots

Each Envoy node receives a snapshot over xDS. In the console, **Updates** is **Active** while the controller pushes changes and **Paused** while it holds them. The node page shows these snapshots.

| Snapshot | Contents |
| --- | --- |
| Served | What this controller replica currently offers the node over xDS. A stream's ACK or NACK is recorded separately. |
| Pending | What the current Kubernetes objects build for a paused node. A failed build reports its cause in `pending.error`. An active node has `pending: null`. |
| History | Up to ten previously served snapshots, newest first. History is held in memory when the cache API and writes are enabled. |

The snapshot ID is the first 12 hexadecimal digits of a SHA-256 over the type, name and content hash of each resource. An ID identifies content. Rollback can serve the same content with higher xDS versions.

## Configuration and persistence

Reading requires the cache API. Writes are enabled by default with `snapshotControl.enabled: true` (`API_SNAPSHOT_CONTROL`), also on upgrades with older values. `false` disables control operations and history; existing pauses stay in effect.

Pause records and the version floor are kept in a Secret in the controller namespace, selected by `SNAPSHOT_PAUSE_SECRET` (the chart uses `<fullname>-frozen-nodes`). The controller loads it on startup, also with writes disabled. A missing Secret means no pauses. A Secret that cannot be read or parsed stops startup: fix or delete it, then restart the controller.

With OIDC, writes are limited to nodes in the user's access groups and record the operator. Without OIDC, anyone who can reach the cache API can control nodes. Use one controller replica: with several, a request changes only the replica that receives it (`controllerPod` in the response).

## API

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

You need one writable controller replica, a node with a served snapshot and access to every VirtualService and template in the change. Keep the change in one Argo CD sync wave: a degraded VirtualService in an earlier wave blocks later waves.

1. Pause updates to the node with a reason. Node updates show **Paused** and the affected VirtualServices report the pause in `status.message`.
2. Apply all affected objects. The node keeps the previous snapshot. The webhook allows the temporary listener address conflict on the paused node with a warning; other errors still block admission. With `webhook.lightDryRun: true`, a VirtualService targeting a paused node also gets a full dry run within `webhook.dryRunTimeoutMS`.
3. Compare Served with Pending. Resolve any `pending.error`, check the additions, removals and changes, and keep the pending ID from this comparison.
4. Resume with that ID. Check **Sync status** of the node's streams and the traffic path.

Rollback serves the selected historical snapshot at once and pauses updates. The objects in Git are unchanged: before resuming, bring them to the configuration you want and review Pending again.

## Restart and cost

Pause records survive a controller restart; served snapshots and history stay in memory only, because they contain secrets. After a restart a paused node has Pending but no Served snapshot. Running Envoys keep their configuration, a newly started Envoy gets none from that replica until Resume. The console compares Pending against an empty baseline.

Reading a paused node or comparing with Pending runs a dry build of all nodes, so the console does it only when a node page is opened or refreshed. The node list does not.
