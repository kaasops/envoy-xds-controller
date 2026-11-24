package xds

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"unicode"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/go-logr/logr"
	"github.com/kaasops/envoy-xds-controller/internal/xds/clients"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	// Log messages stay constant: operators search for them, and the zap
	// sampler groups lines by level and message.
	nackMessage        = "Envoy NACKed xDS response"
	unknownNackMessage = "Envoy NACKed an xDS response unknown to this stream"
	clearedMessage     = "xDS rejection cleared"

	// After a restart one response carries every resource of a type, and an
	// unsubscribe list is as long as the client makes it.
	maxLoggedResources = 20
	// A client chooses these two itself, up to the gRPC message limit.
	maxLoggedTypeURLLen = 256
	maxLoggedNonceLen   = 64
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

func (cb *Callbacks) OnStreamOpen(ctx context.Context, id int64, typ string) error {
	address := peerAddress(ctx)
	cb.connectedClients.Set(clients.Key{Protocol: clients.SotW, ID: id}, &clients.Info{
		Address: address, ForwardedAddress: forwardedAddress(ctx),
	})
	cb.log.Info("stream open", "id", id, "typ", typ, "address", address)
	return nil
}
func (cb *Callbacks) OnStreamClosed(id int64, node *core.Node) {
	cb.log.Info("stream closed", "id", id, "nodeId", node.GetId())
	cb.connectedClients.Delete(clients.Key{Protocol: clients.SotW, ID: id})
}

func (cb *Callbacks) OnDeltaStreamOpen(ctx context.Context, id int64, typ string) error {
	address := peerAddress(ctx)
	cb.connectedClients.Set(clients.Key{Protocol: clients.Delta, ID: id}, &clients.Info{
		Address: address, ForwardedAddress: forwardedAddress(ctx),
	})
	cb.log.Info("delta stream open", "id", id, "typ", typ, "address", address)
	return nil
}
func (cb *Callbacks) OnDeltaStreamClosed(id int64, node *core.Node) {
	cb.log.Info("delta stream closed", "id", id, "nodeId", node.GetId())
	cb.connectedClients.Delete(clients.Key{Protocol: clients.Delta, ID: id})
}

func (cb *Callbacks) OnStreamRequest(id int64, req *discovery.DiscoveryRequest) error {
	cb.countRequest(&cb.Requests)
	node := req.GetNode()
	cb.log.Info("received stream request",
		"typeUrl", req.GetTypeUrl(),
		"id", id,
		"versionInfo", req.GetVersionInfo(),
		"resourceNames", req.GetResourceNames(),
		"nodeId", node.GetId(),
	)
	key := clients.Key{Protocol: clients.SotW, ID: id}
	// The node is only guaranteed on the first request of a stream, so keep the
	// info stored back then instead of overwriting it with empty values.
	if node != nil {
		cb.connectedClients.Update(key, clientInfo(node))
	}
	cb.report(req.GetTypeUrl(), req.GetResponseNonce(), cb.connectedClients.RequestReceived(key, clients.Request{
		TypeURL:  req.GetTypeUrl(),
		Nonce:    req.GetResponseNonce(),
		Version:  req.GetVersionInfo(),
		Rejected: req.GetErrorDetail() != nil,
		Error:    req.GetErrorDetail().GetMessage(),
	}))
	return nil
}

// The request passed to a response callback is the one that opened the watch,
// possibly an old NACK, so only the response is recorded.
func (cb *Callbacks) OnStreamResponse(
	_ context.Context,
	id int64,
	req *discovery.DiscoveryRequest,
	resp *discovery.DiscoveryResponse,
) {
	cb.mu.Lock()
	cb.Responses++
	cb.mu.Unlock()
	cb.log.Info("responding to stream request", "typeUrl", req.GetTypeUrl(), "id", id, "nodeId", req.GetNode().GetId())
	if !isKnownType(resp.GetTypeUrl()) {
		return
	}
	cb.connectedClients.ResponseSent(clients.Key{Protocol: clients.SotW, ID: id}, clients.Response{
		TypeURL: resp.GetTypeUrl(),
		Nonce:   resp.GetNonce(),
		Version: resp.GetVersionInfo(),
	})
}

func (cb *Callbacks) OnStreamDeltaResponse(
	id int64,
	req *discovery.DeltaDiscoveryRequest,
	resp *discovery.DeltaDiscoveryResponse,
) {
	cb.mu.Lock()
	cb.DeltaResponses++
	cb.mu.Unlock()
	cb.log.Info("responding to stream delta request", "typeUrl", req.GetTypeUrl(), "id", id)
	if !isKnownType(resp.GetTypeUrl()) {
		return
	}
	// The response object is shared with the cache: copy what is kept.
	sent := clients.Response{
		TypeURL:   resp.GetTypeUrl(),
		Nonce:     resp.GetNonce(),
		Version:   resp.GetSystemVersionInfo(),
		Resources: make([]clients.ResourceRef, 0, len(resp.GetResources())),
		Removed:   slices.Clone(resp.GetRemovedResources()),
	}
	for _, res := range resp.GetResources() {
		sent.Resources = append(sent.Resources, clients.ResourceRef{Name: res.GetName(), Version: res.GetVersion()})
	}
	cb.connectedClients.ResponseSent(clients.Key{Protocol: clients.Delta, ID: id}, sent)
}

func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discovery.DeltaDiscoveryRequest) error {
	cb.countRequest(&cb.DeltaRequests)
	node := req.GetNode()
	cb.log.Info("received stream delta request",
		"typeUrl", req.GetTypeUrl(),
		"id", id,
		"nodeId", node.GetId(),
	)
	key := clients.Key{Protocol: clients.Delta, ID: id}
	// Envoy sends the node only on the first request of a delta stream, and
	// go-control-plane restores it only after this callback returns, so a nil
	// node here is expected - keep the info stored on the first request.
	if node != nil {
		cb.connectedClients.Update(key, clientInfo(node))
	}
	cb.report(req.GetTypeUrl(), req.GetResponseNonce(), cb.connectedClients.RequestReceived(key, clients.Request{
		TypeURL:      req.GetTypeUrl(),
		Nonce:        req.GetResponseNonce(),
		Rejected:     req.GetErrorDetail() != nil,
		Error:        req.GetErrorDetail().GetMessage(),
		Unsubscribed: req.GetResourceNamesUnsubscribe(),
	}))
	return nil
}

// countRequest runs for every request of every stream, so it holds the lock
// for the counter only.
func (cb *Callbacks) countRequest(counter *int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	*counter++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
}

// report logs what a request changed in the state of its stream. The request
// seldom carries the node, so the stream is named from what the registry holds.
func (cb *Callbacks) report(typeURL, nonce string, out clients.Outcome) {
	// Nearly every request changes nothing worth a line.
	if out.Rejection == nil && !out.UnknownNack && len(out.Cleared) == 0 {
		return
	}
	client := out.Client
	log := cb.log.WithValues("nodeId", client.NodeID, "id", client.ID, "protocol", client.Protocol)
	for _, cleared := range out.Cleared {
		log.Info(clearedMessage,
			"typeUrl", cleared.TypeURL,
			"reason", cleared.Reason,
			"resources", cleared.Resources[:min(len(cleared.Resources), maxLoggedResources)],
			"resourceCount", len(cleared.Resources),
		)
	}
	if out.UnknownNack {
		log.Info(unknownNackMessage,
			"typeUrl", truncate(typeURL, maxLoggedTypeURLLen),
			"nonce", truncate(nonce, maxLoggedNonceLen),
			"error", out.UnknownError,
		)
	}
	if rejection := out.Rejection; rejection != nil {
		resources := make([]string, 0, maxLoggedResources)
		for _, res := range rejection.Resources[:min(len(rejection.Resources), maxLoggedResources)] {
			resources = append(resources, res.Name)
		}
		log.Error(errors.New(rejection.Error), nackMessage,
			"typeUrl", out.TypeURL,
			"address", client.Address,
			"envoyVersion", client.Version,
			"nonce", rejection.Nonce,
			"responseVersion", rejection.ResponseVersion,
			"resources", resources,
			"resourceCount", len(rejection.Resources),
		)
	}
}

// isKnownType reports whether the cache serves the type. A client may ask for
// any type URL, and the Delta cache answers a wildcard request for each.
func isKnownType(typeURL string) bool {
	return cache.GetResponseType(typeURL) != types.UnknownType
}

func truncate(s string, n int) string {
	return s[:min(len(s), n)]
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

// peerAddress returns the remote address of the stream, or "" when the context
// carries none. Behind a proxy this is the proxy, not Envoy.
func peerAddress(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok && p != nil && p.Addr != nil {
		return p.Addr.String()
	}
	return ""
}

// The first X-Forwarded-For entry is the reported originating address. It is a display hint.
func forwardedAddress(ctx context.Context) string {
	values := metadata.ValueFromIncomingContext(ctx, "x-forwarded-for")
	if len(values) == 0 {
		return ""
	}
	first, _, _ := strings.Cut(values[0], ",")
	hop := strings.Trim(first, " \t")
	address, err := netip.ParseAddr(hop)
	if err != nil || address.Zone() != "" {
		return ""
	}
	return address.String()
}

func clientInfo(node *core.Node) *clients.Info {
	var host string
	fields := node.GetMetadata().GetFields()
	for _, key := range []string{"instance", "hostname", "HOSTNAME"} {
		host = nodeLabel(fields[key].GetStringValue())
		if host != "" {
			break
		}
	}
	locality := node.GetLocality()
	return &clients.Info{
		NodeID: node.GetId(),
		Host:   host,
		Locality: clients.Locality{
			Region:  nodeLabel(locality.GetRegion()),
			Zone:    nodeLabel(locality.GetZone()),
			SubZone: nodeLabel(locality.GetSubZone()),
		},
		Version: semver(node.GetUserAgentBuildVersion().GetVersion()),
	}
}

// Node metadata is supplied by the client. Skip unusable labels instead of
// truncating two different instance names into the same one.
func nodeLabel(value string) string {
	const maxBytes = 255
	if len(value) > maxBytes {
		return ""
	}
	for _, r := range value {
		if !unicode.IsPrint(r) {
			return ""
		}
	}
	return strings.TrimSpace(value)
}

func semver(ver *v3.SemanticVersion) string {
	if ver == nil {
		return "-"
	}
	return fmt.Sprintf("v%d.%d.%d", ver.MajorNumber, ver.MinorNumber, ver.Patch)
}
