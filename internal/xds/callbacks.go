package xds

import (
	"context"
	"fmt"
	"sync"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/go-logr/logr"
	"github.com/kaasops/envoy-xds-controller/internal/xds/clients"
	"google.golang.org/grpc/peer"
)

type Callbacks struct {
	log              logr.Logger
	Signal           chan struct{}
	Fetches          int
	Requests         int
	Responses        int
	DeltaRequests    int
	DeltaResponses   int
	mu               sync.Mutex
	connectedClients *clients.Registry
}

func NewCallbacks(logger logr.Logger, registry *clients.Registry) *Callbacks {
	return &Callbacks{
		log:              logger,
		connectedClients: registry,
	}
}

var _ server.Callbacks = &Callbacks{}

func (cb *Callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.log.Info("server callbacks",
		"fetches", cb.Fetches,
		"requests", cb.Requests,
		"responses", cb.Responses,
	)
}

func (cb *Callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	cb.log.Info("stream open", "id", id, "typ", typ)
	return nil
}
func (cb *Callbacks) OnStreamClosed(id int64, node *core.Node) {
	cb.log.Info("stream closed", "id", id, "nodeId", node.Id)
	cb.connectedClients.Delete(id)
}

func (cb *Callbacks) OnDeltaStreamOpen(ctx context.Context, id int64, typ string) error {
	p, ok := peer.FromContext(ctx)
	clientInfo := &clients.Info{ID: id}
	if ok && p != nil && p.Addr != nil {
		clientInfo.Address = p.Addr.String()
	}
	cb.connectedClients.Set(id, clientInfo)
	cb.log.Info("delta stream open", "id", id, "typ", typ)
	return nil
}
func (cb *Callbacks) OnDeltaStreamClosed(id int64, node *core.Node) {
	cb.log.Info("delta stream closed", "id", id, "nodeId", node.Id)
	cb.connectedClients.Delete(id)
}

func (cb *Callbacks) OnStreamRequest(id int64, req *discovery.DiscoveryRequest) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Requests++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	cb.log.Info("received stream request",
		"typeUrl", req.GetTypeUrl(),
		"id", id,
		"versionInfo", req.VersionInfo,
		"resourceNames", req.ResourceNames,
		"nodeId", req.Node.Id,
	)
	cb.connectedClients.Update(id, &clients.Info{
		ID:      id,
		NodeID:  req.Node.Id,
		Version: semver(req.Node.GetUserAgentBuildVersion().GetVersion()),
	})
	return nil
}

func (cb *Callbacks) OnStreamResponse(_ context.Context, id int64, req *discovery.DiscoveryRequest, _ *discovery.DiscoveryResponse) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Responses++
	cb.log.Info("responding to stream request", "typeUrl", req.GetTypeUrl(), "id", id, "nodeId", req.Node.Id)
}

func (cb *Callbacks) OnStreamDeltaResponse(id int64, req *discovery.DeltaDiscoveryRequest, res *discovery.DeltaDiscoveryResponse) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.DeltaResponses++
	cb.log.Info("responding to stream delta request", "typeUrl", req.GetTypeUrl(), "id", id)
}
func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discovery.DeltaDiscoveryRequest) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.DeltaRequests++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	cb.log.Info("received stream delta request",
		"typeUrl", req.GetTypeUrl(),
		"id", id,
		"nodeId", req.Node.Id,
	)
	cb.connectedClients.Update(id, &clients.Info{
		ID:      id,
		NodeID:  req.Node.Id,
		Version: semver(req.Node.GetUserAgentBuildVersion().GetVersion()),
	})
	return nil
}
func (cb *Callbacks) OnFetchRequest(_ context.Context, req *discovery.DiscoveryRequest) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Fetches++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	cb.log.Info("received fetch request", "typeUrl", req.GetTypeUrl())
	return nil
}
func (cb *Callbacks) OnFetchResponse(*discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.log.Info("responding to fetch request")
}

func semver(ver *v3.SemanticVersion) string {
	if ver == nil {
		return "-"
	}
	return fmt.Sprintf("v%d.%d.%d", ver.MajorNumber, ver.MinorNumber, ver.Patch)
}
