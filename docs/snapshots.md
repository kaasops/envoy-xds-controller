# Node Snapshots: Inspect, Freeze, Roll Back

The controller builds a snapshot per node from the cluster objects and serves it to the node's Envoys over xDS. The cache API shows, for each node, what is served and what the current objects would build, compares the two resource by resource, and can freeze a node or roll it back to a snapshot it was served before.

| Term | Meaning |
| --- | --- |
| served | the snapshot the xDS server gives the node's Envoys |
| built | what the current objects build for the node now, computed on request without publishing anything |
| history | snapshots the node was served before, newest first, up to 10, kept in memory while the cache API can write |
| frozen | builds go on, but the node is served what it has until it is unfrozen |

A snapshot ID is the first 12 hex digits of a sha256 over the type, name and content hash of every resource. The builder gives the same bytes for the same objects, so the same content has the same ID in any build. Versions are not part of it: a rollback serves old content under new versions, so the ID stays and the versions grow.

## Enabling

Reading is available whenever the cache API is enabled. Writing is on by default: `snapshotControl.enabled: true` in the chart, environment variable `API_SNAPSHOT_CONTROL`. An upgrade with `helm upgrade --reuse-values` from a chart without the setting gets it on as well. Set it to `false` to keep the API read only: nodes frozen before stay frozen and can no longer be unfrozen through the API, and no history is kept. Frozen nodes and the last version a freeze or rollback served are kept in the Secret `<fullname>-frozen-nodes` of the controller namespace (`SNAPSHOT_FREEZE_SECRET`), where `<fullname>` is the full name of the release as the chart builds it, `<release>-envoy-xds-controller` unless the release name contains the chart name. A Secret, because what it holds decides what a node is served after a restart and names the operators; the controller manages Secrets of its namespace already, so no new permissions are needed. The Secret has no `envoy.kaasops.io/secret-type` label and is never served to Envoy. The controller reads it at every start, whatever the settings, and does not start if it cannot read it: starting as if nothing were frozen would publish what a frozen node holds back. A missing Secret means nothing is frozen.

With OIDC enabled, a user reaches only the nodes of their access groups, and every write records the user (email, preferred user name or subject). There is no separate permission to write: whoever reaches a node can freeze and roll it back. Without OIDC, anyone who reaches the cache API can freeze and roll back nodes; the controller logs a warning at start.

## API

| Request | Result |
| --- | --- |
| `GET /api/v1/frozenNodes` | frozen nodes with who froze them, when and why |
| `GET /api/v1/nodes/{nodeId}/snapshots` | `served`, `built`, `history`, `frozen`, `freeze`, `canControl` |
| `GET /api/v1/nodes/{nodeId}/snapshots/diff?from=served&to=built` | per resource: `same` (equal bytes), `sameDecoded` (different bytes, equal messages, e.g. field order inside `Any`), `changed`, `added`, `removed`, with sha256 of both sides. `from` and `to` take `served`, `built` or a snapshot ID from the history. Add `all=true` to list unchanged resources too |
| `GET /api/v1/nodes/{nodeId}/snapshots/resource?snapshot=served&type=…&name=…` | the resource as JSON. A secret comes as its sha256 only |
| `POST /api/v1/nodes/{nodeId}/freeze` `{"reason": "…"}` | stop publishing to the node |
| `POST /api/v1/nodes/{nodeId}/unfreeze` `{"expectBuilt": "<id>"}` | serve the built snapshot of that ID, then follow the objects again |
| `POST /api/v1/nodes/{nodeId}/rollback` `{"to": "<id>", "expectServed": "<id>"}` | serve a snapshot from the history and freeze the node there |

Writes answer with the state of the node, like the `GET` of its snapshots. `409 Conflict` means: the node is frozen already or not frozen; the build for the node fails, so unfreezing would not publish it; or `expectBuilt` / `expectServed` no longer match, because the objects or the served snapshot changed since the caller read them. Read again and decide again.

## Behaviour

- **Freeze** keeps the node on its served snapshot. Builds go on, other nodes are published as usual. An Envoy of the node that starts or reconnects while it is frozen gets the served snapshot. The services of the node say in `status.message` since when and why it is frozen; who froze it is in `GET /api/v1/frozenNodes`.
- **Unfreeze** serves exactly the built snapshot the caller compared, named by `expectBuilt`, then lets every build publish to the node again. It is refused while the build fails or when the build differs from `expectBuilt`.
- **Rollback** is instant and does not depend on Git, Argo CD or the webhook. With `expectServed`, the check and the rollback are one step. Git still holds the newer objects: unfreezing publishes them again, so revert them in Git before unfreezing.
- **Restart.** The frozen nodes survive a controller restart, the served snapshots and the history do not: they hold TLS keys and stay in memory. After a restart a frozen node is served nothing: running Envoys keep the configuration they have, an Envoy of that node started later gets nothing until the node is unfrozen.

## Switching a node to another listener

A node cannot have two listeners on one address. Moving a node from one listener to another on the same address, for example `https` to `https-next` on `0.0.0.0:443`, changes every VirtualService and template of the node that refers to the listener, and every object in between would give the node both listeners. Freeze the node, apply the change, compare, unfreeze:

1. `POST .../freeze` with a reason. The services of the node report the freeze in `status.message`.
2. Apply the objects. While the node is frozen, the webhooks accept a listener address conflict on it and return a warning per node, for example `node "edge" gets no new configuration while listeners default/https and default/https-next share 0.0.0.0:443`; listener names from another namespace are not shown. Any other error is rejected as before. With `webhook.lightDryRun: true`, a VirtualService that targets a frozen node before or after the change, `nodeIDs: ["*"]` included, gets the listener checks of the light mode and then the full dry run within `webhook.dryRunTimeoutMS`: the light check does not see listeners that come from templates.
3. Compare `served` with `built`: only the listeners should differ. While the conflict lasts, `built.error` names it; unfreezing is refused until the last reference is moved, so a forgotten one shows before anything is served.
4. `POST .../unfreeze` with the `built` ID you compared. The node gets the new listener in one update.

Keep all objects of the change in one Argo CD sync wave: with a health check for VirtualService installed, a `Degraded` VirtualService in an earlier wave stops the next one. While a node is held by a listener address conflict, the domain index of `webhook.validationIndices` and the secrets protected from deletion cover both the configuration the node serves and the one it waits for.

## Cost

`GET .../snapshots` and a diff against `built` build the snapshots of every node from the current objects, as a full dry run does: hundreds of milliseconds of controller CPU per request on a large fleet. Read them when a node is opened or refreshed, not on a timer and not for a list of nodes; `GET /api/v1/frozenNodes` and `GET /api/v1/clients` are cheap.

## Limits

- One controller replica. With more, a write applies to the replica that received it; `controllerPod` in the state tells which.
- The status of a VirtualService on a frozen node shows the result of the build and the note that the node is frozen, not what the node is served.
