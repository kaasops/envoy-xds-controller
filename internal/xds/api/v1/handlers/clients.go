package handlers

import (
	"net/http"
	"sort"

	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/gin-gonic/gin"
	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/middlewares"
	"github.com/kaasops/envoy-xds-controller/internal/xds/clients"
)

type clientsResponse struct {
	// ControllerPod is the pod that answered. Each pod keeps the streams
	// connected to it, so with several replicas every pod has to be asked.
	ControllerPod string `json:"controllerPod"`
	// SnapshotVersions holds, for every node of the listed clients that has a
	// snapshot, the version of each resource type by type URL. A version is a
	// counter kept by one pod and starts over when the pod restarts, so it
	// compares only with ackedVersion of the streams in the same response.
	// Only listeners stay level with it: a stream is sent nothing while no
	// resource it holds changes, and the counter of a type may still move.
	SnapshotVersions map[string]map[string]string `json:"snapshotVersions"`
	Clients          []clients.Info               `json:"clients"`
}

// getClients lists the xDS streams connected to this controller pod.
// @Summary List connected xDS clients
// @Description One entry per xDS stream with the node, peer address, Envoy version, optional forwarded address,
// @Description and optional client-reported host and locality. Each resource type lists the last acknowledged version
// @Description and the rejections still standing.
// @Description Several Envoy instances may share a node ID. An empty list means that no stream
// @Description of that node is connected to the pod that answered, or that the node is outside
// @Description the node IDs the caller may access.
// @Tags clients
// @Accept json
// @Produce json
// @Param node_id query string false "Node ID"
// @Success 200 {object} clientsResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/clients [get]
func (h *handler) getClients(ctx *gin.Context) {
	nodeID, err := h.getNotRequiredOnlyOneParam(ctx.Request.URL.Query(), nodeIDParamName)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var allowed map[string]struct{}
	v, restricted := ctx.Get(middlewares.AvailableNodeIDs)
	if restricted {
		allowed, _ = v.(map[string]struct{})
	}
	list := filterClients(h.clients.List(), nodeID, allowed, restricted)
	ctx.JSON(http.StatusOK, clientsResponse{
		ControllerPod:    h.podName,
		SnapshotVersions: h.snapshotVersions(list),
		Clients:          list,
	})
}

// filterClients keeps the streams the caller may see and orders them by node.
//
// It deliberately does not go through the snapshot cache, unlike the other
// handlers: an Envoy that connected with a node ID nobody configured has no
// snapshot, and that is exactly the stream an operator needs to find. A
// restricted caller never sees a stream that has not named its node yet.
func filterClients(list []clients.Info, nodeID string, allowed map[string]struct{}, restricted bool) []clients.Info {
	filtered := make([]clients.Info, 0, len(list))
	for _, client := range list {
		if nodeID != "" && client.NodeID != nodeID {
			continue
		}
		if _, ok := allowed[client.NodeID]; restricted && !ok {
			continue
		}
		filtered = append(filtered, client)
	}
	sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].NodeID < filtered[j].NodeID })
	return filtered
}

// snapshotVersions reads the versions the streams are to be compared with
// from the same pod and at the same moment as the streams themselves.
func (h *handler) snapshotVersions(list []clients.Info) map[string]map[string]string {
	versions := make(map[string]map[string]string)
	for _, client := range list {
		if _, done := versions[client.NodeID]; done {
			continue
		}
		snapshot, err := h.cache.GetSnapshot(client.NodeID)
		if err != nil {
			continue
		}
		byType := make(map[string]string, 4)
		for _, typeURL := range []string{
			resourcev3.ClusterType, resourcev3.ListenerType, resourcev3.RouteType, resourcev3.SecretType,
		} {
			if version := snapshot.GetVersion(typeURL); version != "" {
				byType[typeURL] = version
			}
		}
		versions[client.NodeID] = byType
	}
	return versions
}
