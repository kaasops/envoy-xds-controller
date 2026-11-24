package handlers

import (
	"errors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kaasops/envoy-xds-controller/internal/xds/api/v1/middlewares"
	xdscache "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/snapshots"
)

type snapshotHandler struct {
	control *snapshots.Control
	writes  bool
	podName string
	logger  *zap.Logger
}

type snapshotStateResponse struct {
	snapshots.State
	ControllerPod string `json:"controllerPod"`
	// CanControl tells the caller whether it may pause, resume and roll back this node.
	CanControl bool `json:"canControl"`
}

// RegisterSnapshotRoutes adds the routes that show what nodes are served and what is built for them and,
// when writes is true, pause, resume and roll back nodes.
func RegisterSnapshotRoutes(r *gin.Engine, control *snapshots.Control, writes bool, logger *zap.Logger) {
	// In Kubernetes the hostname is the pod name.
	podName, _ := os.Hostname()
	h := &snapshotHandler{control: control, writes: writes, podName: podName, logger: logger}
	routes := r.Group(version)
	routes.GET("/pausedNodes", h.getPausedNodes)
	routes.GET("/nodes/:nodeId/snapshots", h.getState)
	routes.GET("/nodes/:nodeId/snapshots/diff", h.getDiff)
	routes.GET("/nodes/:nodeId/snapshots/resource", h.getResource)
	routes.POST("/nodes/:nodeId/pause", h.pause)
	routes.POST("/nodes/:nodeId/resume", h.resume)
	routes.POST("/nodes/:nodeId/rollback", h.rollback)
}

// getPausedNodes lists the paused nodes the caller may see, also those served nothing after a restart.
func (h *snapshotHandler) getPausedNodes(ctx *gin.Context) {
	paused := h.control.PausedNodes()
	if v, restricted := ctx.Get(middlewares.AvailableNodeIDs); restricted {
		allowed, _ := v.(map[string]struct{})
		for nodeID := range paused {
			if _, ok := allowed[nodeID]; !ok {
				delete(paused, nodeID)
			}
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"pausedNodes": paused})
}

// getState returns what the node is served, what is built for it and what it was served before.
func (h *snapshotHandler) getState(ctx *gin.Context) {
	nodeID, ok := h.node(ctx)
	if !ok {
		return
	}
	state, err := h.control.State(ctx.Request.Context(), nodeID)
	h.reply(ctx, state, err)
}

// getDiff compares two snapshots of the node: from and to are served, pending or a snapshot ID.
func (h *snapshotHandler) getDiff(ctx *gin.Context) {
	nodeID, ok := h.node(ctx)
	if !ok {
		return
	}
	from, to := ctx.DefaultQuery("from", snapshots.Served), ctx.DefaultQuery("to", snapshots.Pending)
	diff, err := h.control.Diff(ctx.Request.Context(), nodeID, from, to, ctx.Query("all") == "true")
	if err != nil {
		h.fail(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, diff)
}

// getResource returns one resource of a snapshot of the node; secrets come as a hash only.
func (h *snapshotHandler) getResource(ctx *gin.Context) {
	nodeID, ok := h.node(ctx)
	if !ok {
		return
	}
	res, err := h.control.Resource(ctx.Request.Context(), nodeID,
		ctx.DefaultQuery("snapshot", snapshots.Served), ctx.Query("type"), ctx.Query("name"))
	if err != nil {
		h.fail(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, res)
}

func (h *snapshotHandler) pause(ctx *gin.Context) {
	var body struct {
		Reason string `json:"reason"`
	}
	nodeID, by, ok := h.write(ctx, &body)
	if !ok {
		return
	}
	state, err := h.control.Pause(ctx.Request.Context(), nodeID, by, body.Reason)
	h.audit("pause", nodeID, by, body.Reason, state, err)
	h.reply(ctx, state, err)
}

func (h *snapshotHandler) resume(ctx *gin.Context) {
	var body struct {
		ExpectPending string `json:"expectPending"`
	}
	nodeID, by, ok := h.write(ctx, &body)
	if !ok {
		return
	}
	state, err := h.control.Resume(ctx.Request.Context(), nodeID, by, body.ExpectPending)
	h.audit("resume", nodeID, by, body.ExpectPending, state, err)
	h.reply(ctx, state, err)
}

func (h *snapshotHandler) rollback(ctx *gin.Context) {
	var body struct {
		To           string `json:"to"`
		ExpectServed string `json:"expectServed"`
	}
	nodeID, by, ok := h.write(ctx, &body)
	if !ok {
		return
	}
	state, err := h.control.Rollback(ctx.Request.Context(), nodeID, by, body.To, body.ExpectServed)
	h.audit("rollback", nodeID, by, body.To, state, err)
	h.reply(ctx, state, err)
}

// node returns the node of the request when the caller may see it.
func (h *snapshotHandler) node(ctx *gin.Context) (string, bool) {
	nodeID := ctx.Param("nodeId")
	if v, restricted := ctx.Get(middlewares.AvailableNodeIDs); restricted {
		allowed, _ := v.(map[string]struct{})
		if _, ok := allowed[nodeID]; !ok {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "node is not available to you"})
			return "", false
		}
	}
	return nodeID, true
}

// write checks that writes are on and the caller may change the node, and reads the body.
func (h *snapshotHandler) write(ctx *gin.Context, body any) (nodeID, by string, ok bool) {
	if !h.writes {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "snapshot control is turned off on this controller"})
		return "", "", false
	}
	if nodeID, ok = h.node(ctx); !ok {
		return "", "", false
	}
	if ctx.Request.ContentLength != 0 {
		if err := ctx.ShouldBindJSON(body); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return "", "", false
		}
	}
	by = ctx.GetString(middlewares.UserName)
	if by == "" {
		by = "anonymous " + ctx.ClientIP()
	}
	return nodeID, by, true
}

func (h *snapshotHandler) audit(op, nodeID, by, arg string, state snapshots.State, err error) {
	served := ""
	if state.Served != nil {
		served = state.Served.ID
	}
	h.logger.Info("snapshot control", zap.String("operation", op), zap.String("nodeId", nodeID),
		zap.String("by", by), zap.String("argument", arg), zap.String("servedId", served), zap.Error(err))
}

func (h *snapshotHandler) reply(ctx *gin.Context, state snapshots.State, err error) {
	if err != nil {
		h.fail(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, snapshotStateResponse{State: state, ControllerPod: h.podName, CanControl: h.writes})
}

func (h *snapshotHandler) fail(ctx *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, snapshots.ErrUnknownNode), errors.Is(err, snapshots.ErrUnknownResource),
		errors.Is(err, xdscache.ErrUnknownSnapshot):
		status = http.StatusNotFound
	case errors.Is(err, snapshots.ErrAlreadyPaused), errors.Is(err, snapshots.ErrNotPaused),
		errors.Is(err, snapshots.ErrStateChanged), errors.Is(err, snapshots.ErrBuildFails):
		status = http.StatusConflict
	case errors.Is(err, snapshots.ErrNoReason), errors.Is(err, snapshots.ErrNoPendingID):
		status = http.StatusBadRequest
	}
	ctx.JSON(status, gin.H{"error": err.Error()})
}
