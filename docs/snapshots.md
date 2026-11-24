# Node snapshots

Each Envoy node receives a snapshot over xDS. In the console, **Updates** is **Active** while the controller pushes changes and **Paused** while it holds them. The node page shows these snapshots.

| Snapshot | Contents |
| --- | --- |
| Served | What this controller replica currently offers the node over xDS. A stream's ACK or NACK is recorded separately. |
| Pending | What the current Kubernetes objects build for a paused node. A failed build reports its cause in `pending.error`. An active node has `pending: null`. |
| History | Up to ten previously served snapshots, newest first. History is held in memory when the cache API and writes are enabled. |

The snapshot ID is the first 12 hexadecimal digits of a SHA-256 over the type, name and content hash of each resource. An ID identifies content. Rollback can serve the same content with higher xDS versions.

## Configuration and persistence

Enable the cache API to read snapshots. Writes default to `snapshotControl.enabled: true` (`API_SNAPSHOT_CONTROL`), including upgrades from charts without this setting. Setting it to `false` disables writes and history. Existing pauses stay in effect.

On startup the controller loads pause records and the version floor from a Secret in its namespace, even with writes disabled. `SNAPSHOT_PAUSE_SECRET` selects the Secret (the chart uses `<fullname>-frozen-nodes`). An unreadable or malformed Secret stops startup. Restore access or recover its data from a backup before restarting. Without the Secret, the controller starts with no pauses and a zero version floor.

With writes enabled, OIDC identifies the operator. A configured `auth.acl.nodeIdsByGroup` limits access to the nodes assigned to the user's groups. If the mapping is empty, any authenticated user can control any node. Without OIDC, anyone who can reach the writable cache API can control nodes.

Use one controller replica for snapshot control. A request changes the replica named in the response's `controllerPod` field. Other running replicas keep their own state.

## API

All paths below are relative to `/api/v1`. Node responses include `served`, `pending`, `history`, `paused`, `pause` and `canControl`. The pause record contains `by`, `at` and `reason`.

| Request | Result |
| --- | --- |
| `GET /pausedNodes` | Returns `pausedNodes`, keyed by node ID, including paused nodes with no served snapshot after a restart. |
| `GET /nodes/{nodeId}/snapshots` | Returns the node state. While paused, the controller computes Pending on request. |
| `GET /nodes/{nodeId}/snapshots/diff?from=served&to=pending` | Compares snapshots resource by resource. Each side accepts `served`, `pending` or an ID from history. Add `all=true` to include unchanged resources. |
| `GET /nodes/{nodeId}/snapshots/resource?snapshot=served&type=…&name=…` | Returns a decoded resource. For a Secret, only its type, name and SHA-256 are exposed. |
| `POST /nodes/{nodeId}/pause` with `{"reason":"…"}` | Holds the node on its served snapshot. |
| `POST /nodes/{nodeId}/resume` with `{"expectPending":"<id>"}` | Pushes the reviewed pending snapshot and resumes automatic updates. |
| `POST /nodes/{nodeId}/rollback` with `{"to":"<id>","expectServed":"<id>"}` | Checks the served ID, serves the historical snapshot and pauses updates. The reason is `rollback to <id>`. Pending reflects the current objects. |

Diff groups resources as `same` (equal bytes), `sameDecoded` (different bytes that decode to equal messages), `changed`, `added` and `removed`. Secrets are compared by bytes only. The default comparison is Served to Pending and requires a paused node. On an active node specify the two references explicitly, for example a historical ID and `served`.

A successful write returns the current node state. Resume requires the ID from the comparison. A missing ID returns HTTP 400. HTTP 409 means the operation conflicts with the current state, for example the node is already paused, Pending failed to build, or the reviewed ID changed. Refresh the state and compare again. A failed comparison or resume leaves the pause in effect.

## Switching a node to another listener

You need one writable controller replica, a node with a served snapshot and access to every VirtualService and template in the change. With the VirtualService health check installed, keep the changes in one Argo CD sync wave. A VirtualService marked `Degraded` can block later waves. [Argo CD health check](examples/argocd-health/README.md)

1. Pause updates to the node with a reason. Node updates show **Paused** and the affected VirtualServices report the pause in `status.message`.
2. Apply the affected objects. The node keeps its served snapshot. The webhook admits temporary listener address conflicts on paused nodes with a warning. Other validation errors still block admission. With `webhook.lightDryRun: true`, a VirtualService targeting a paused node before or after the change gets a full dry run within `webhook.dryRunTimeoutMS`.
3. Compare Served with Pending. Resolve any `pending.error`, check the additions, removals and changes, and keep the pending ID from this comparison.
4. Resume with that ID. Check **Sync status** of the node's streams and the traffic path.

While listener address conflicts are the only build errors, the Secret deletion check protects both previously used and newly referenced Secrets. With `webhook.validationIndices: true`, the domain index also keeps the previous and newly built domains for each node with a conflict.

Rollback immediately serves the historical snapshot and pauses updates. Pending still follows the current Kubernetes objects. To keep the rolled-back configuration, revert or correct the change in Git, sync it and review Pending before Resume.

## Restart and cost

Pause records survive a controller restart. Served snapshots and history stay in memory because they can contain secrets. A restarted replica has no Served snapshot for a paused node. Running Envoys keep their configuration. A newly started Envoy connecting to that replica waits for Resume. The console can review a successful Pending build against an empty baseline.

Reading a paused node's state, comparing with Pending or opening a resource from Pending triggers a dry build of all nodes. Loading the node list skips this build.
