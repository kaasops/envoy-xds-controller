package clients

import (
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

const (
	// A well-behaved client leaves one response per type unanswered at most.
	// The bounds keep a misbehaving one from growing the registry.
	maxPending    = 64
	maxRejections = 64
)

// Registry keeps the connected xDS streams and what each of them has
// acknowledged or rejected.
//
// Every stream has a lock of its own. A client decides how much work its
// stream takes, by leaving responses unanswered and by the length of its
// lists, and with one lock for all it would decide that for every other
// stream too: the xDS port takes any client. The registry lock guards the map
// only and is never taken while a stream is locked.
type Registry struct {
	mu   sync.RWMutex
	data map[Key]*stream
	now  func() time.Time
}

type stream struct {
	mu           sync.Mutex
	info         Info
	nodeReported bool
	// Single-type RPCs can omit the type URL on the wire. ADS has no default.
	defaultTypeURL string
	// types holds known types observed through a subscription or a response.
	types map[string]*typeState
	// pending holds responses without a verdict, by nonce. Nonces are unique
	// within a stream across types.
	pending map[string]sentResponse
}

type typeState struct {
	awaitingFirstResponse bool
	ackedVersion          string
	lastAckAt             time.Time
	rejections            []Rejection
	dropped               int
}

type sentResponse struct {
	Response
	at time.Time
	// unsubscribed names the resources of the response the client has
	// unsubscribed from since it was sent.
	unsubscribed map[string]struct{}
}

func NewRegistry() *Registry {
	return &Registry{
		data: make(map[Key]*stream),
		now:  time.Now,
	}
}

func (r *Registry) newStream(key Key, ci *Info) *stream {
	s := &stream{
		info:    *ci,
		types:   make(map[string]*typeState),
		pending: make(map[string]sentResponse),
	}
	s.info.ID, s.info.Protocol = key.ID, key.Protocol
	s.info.ConnectedAt = r.now()
	s.info.Types = nil
	return s
}

func (r *Registry) Open(key Key, defaultTypeURL string, ci *Info) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.newStream(key, ci)
	s.defaultTypeURL = defaultTypeURL
	r.data[key] = s
}

func (r *Registry) Update(key Key, ci *Info) {
	r.mu.Lock()
	prev, ok := r.data[key]
	if !ok {
		prev = r.newStream(key, ci)
		prev.nodeReported = true
		r.data[key] = prev
	}
	r.mu.Unlock()
	if !ok {
		return
	}
	prev.mu.Lock()
	defer prev.mu.Unlock()
	// Subsequent requests for the same node can omit its original metadata.
	if prev.nodeReported && prev.info.NodeID == ci.NodeID {
		return
	}
	prev.nodeReported = true
	prev.info.NodeID = ci.NodeID
	prev.info.Host = ci.Host
	prev.info.Locality = ci.Locality
	prev.info.Version = ci.Version
}

func (r *Registry) get(key Key) *stream {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.data[key]
}

func (r *Registry) Delete(key Key) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, key)
}

// List returns a copy of every stream, ordered by protocol and stream id.
func (r *Registry) List() []Info {
	r.mu.RLock()
	streams := make([]*stream, 0, len(r.data))
	for _, s := range r.data {
		streams = append(streams, s)
	}
	r.mu.RUnlock()
	result := make([]Info, 0, len(streams))
	for _, s := range streams {
		result = append(result, s.snapshot())
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Protocol != result[j].Protocol {
			return result[i].Protocol < result[j].Protocol
		}
		return result[i].ID < result[j].ID
	})
	return result
}

// ResponseSent records a response handed to the stream, so that the client's
// answer can be matched to it by nonce. The registry keeps resp's slices.
func (r *Registry) ResponseSent(key Key, resp Response) {
	if resp.Nonce == "" {
		return
	}
	s := r.get(key)
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if t := s.types[resp.TypeURL]; t != nil {
		t.awaitingFirstResponse = false
	} else {
		s.types[resp.TypeURL] = &typeState{}
	}
	if len(s.pending) >= maxPending {
		delete(s.pending, s.oldestPending())
	}
	s.pending[resp.Nonce] = sentResponse{Response: resp, at: r.now()}
}

// RequestReceived applies a client request to the state of the stream and
// reports what changed.
//
// A request is a verdict only if it names, by nonce, a response of the same
// type that has none yet. Envoy repeats the nonce in later SotW requests, and
// those say nothing about the response.
func (r *Registry) RequestReceived(key Key, req Request) Outcome {
	// The error text and the unsubscribe list are as long as the client makes
	// them, so both are prepared before the stream is locked: List waits for it.
	var text, raw string
	if req.Rejected {
		text = sanitizeError(req.Error)
		if len(req.Error) <= maxRawErrorLen {
			raw = req.Error
		}
	}
	var unsubscribed map[string]struct{}
	if key.Protocol == Delta {
		unsubscribed = nameSet(req.Unsubscribed)
	}

	s := r.get(key)
	if s == nil {
		return Outcome{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.TypeURL == "" {
		req.TypeURL = s.defaultTypeURL
	}
	if s.types[req.TypeURL] == nil && req.isSubscription() && IsKnownType(req.TypeURL) {
		s.types[req.TypeURL] = &typeState{awaitingFirstResponse: true}
	}
	var out Outcome
	if len(unsubscribed) > 0 {
		// The server sends nothing more about a resource the client unsubscribed
		// from, so no verdict, past or still to come, may hold it rejected.
		if t := s.types[req.TypeURL]; t != nil {
			out.clear(req.TypeURL, ReasonUnsubscribed, t.release(unsubscribed))
		}
		s.forget(req.TypeURL, unsubscribed)
	}

	sent, ok := s.pending[req.Nonce]
	if !ok || (req.TypeURL != "" && req.TypeURL != sent.TypeURL) {
		if req.Rejected {
			s.info.UnknownNacks++
			if s.info.UnknownNacks == 1 {
				out.UnknownNack, out.UnknownError = true, text
			}
		}
		out.Client = s.info
		return out
	}
	delete(s.pending, req.Nonce)
	out.Client = s.info
	t := s.types[sent.TypeURL]
	now := r.now()

	switch {
	case req.Rejected && key.Protocol == SotW:
		// The server answers a SotW NACK with the same snapshot at once, so the
		// same version comes back rejected over and over.
		if len(t.rejections) == 1 && t.rejections[0].ResponseVersion == sent.Version {
			t.rejections[0].Count++
			return out
		}
		t.rejections = nil
		out.reject(sent.TypeURL, t.reject(sent, nil, text, now))
	case req.Rejected:
		rejected, rest := splitRejected(sent.TypeURL, raw, sent.Resources)
		rejected = slices.DeleteFunc(rejected, func(res ResourceRef) bool {
			_, dropped := sent.unsubscribed[res.Name]
			return dropped
		})
		out.clear(sent.TypeURL, ReasonAccepted, t.release(refSet(rest)))
		out.clear(sent.TypeURL, ReasonRemoved, t.release(nameSet(sent.Removed)))
		t.release(refSet(rejected))
		switch {
		case len(sent.Resources) == 0:
			t.releaseUnnamed()
		case len(rejected) == 0:
			// The client wants none of what the response carried any more.
			return out
		}
		out.reject(sent.TypeURL, t.reject(sent, rejected, text, now))
	case key.Protocol == SotW:
		// version_info moves only when the client accepts a response.
		if req.Version != sent.Version {
			return out
		}
		t.ackedVersion, t.lastAckAt = sent.Version, now
		if len(t.rejections) > 0 {
			t.rejections = nil
			out.Cleared = append(out.Cleared, Cleared{TypeURL: sent.TypeURL, Reason: ReasonAccepted})
		}
	default:
		t.ackedVersion, t.lastAckAt = sent.Version, now
		accepted := t.release(refSet(sent.Resources))
		if t.releaseUnnamed() || len(accepted) > 0 {
			out.Cleared = append(out.Cleared, Cleared{TypeURL: sent.TypeURL, Reason: ReasonAccepted, Resources: accepted})
		}
		out.clear(sent.TypeURL, ReasonRemoved, t.release(nameSet(sent.Removed)))
	}
	return out
}

func (s *stream) oldestPending() string {
	var oldest string
	for nonce, sent := range s.pending {
		if oldest == "" || sent.at.Before(s.pending[oldest].at) {
			oldest = nonce
		}
	}
	return oldest
}

// forget marks the named resources in the unanswered responses of the type.
// It walks every resource of those responses, however short the list is.
func (s *stream) forget(typeURL string, unsubscribed map[string]struct{}) {
	for nonce, sent := range s.pending {
		if sent.TypeURL != typeURL {
			continue
		}
		for _, res := range sent.Resources {
			if _, ok := unsubscribed[res.Name]; !ok {
				continue
			}
			if sent.unsubscribed == nil {
				sent.unsubscribed = make(map[string]struct{})
			}
			sent.unsubscribed[res.Name] = struct{}{}
		}
		s.pending[nonce] = sent
	}
}

func (s *stream) snapshot() Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	info := s.info
	info.Types = make([]TypeStatus, 0, len(s.types))
	for typeURL, t := range s.types {
		status := TypeStatus{
			TypeURL:               typeURL,
			AwaitingFirstResponse: t.awaitingFirstResponse,
			AckedVersion:          t.ackedVersion,
			LastAckAt:             t.lastAckAt,
			Rejections:            make([]Rejection, 0, len(t.rejections)),
			DroppedRejections:     t.dropped,
		}
		for _, rejection := range t.rejections {
			status.Rejections = append(status.Rejections, rejection.clone())
		}
		for _, sent := range s.pending {
			if sent.TypeURL == typeURL && (status.AwaitingAckSince.IsZero() || sent.at.Before(status.AwaitingAckSince)) {
				status.AwaitingAckSince = sent.at
			}
		}
		info.Types = append(info.Types, status)
	}
	sort.Slice(info.Types, func(i, j int) bool { return info.Types[i].TypeURL < info.Types[j].TypeURL })
	return info
}

// reject records a rejection of the response and returns a copy of it.
func (t *typeState) reject(sent sentResponse, resources []ResourceRef, text string, now time.Time) Rejection {
	rejection := Rejection{
		Nonce:           sent.Nonce,
		ResponseVersion: sent.Version,
		RejectedAt:      now,
		Count:           1,
		Error:           text,
		Resources:       resources,
	}
	if len(t.rejections) >= maxRejections {
		t.rejections = t.rejections[1:]
		t.dropped++
	}
	t.rejections = append(t.rejections, rejection)
	return rejection.clone()
}

// release drops the named resources from the rejections that hold them, and a
// rejection with it once it holds none. It returns the names it found.
//
// The names are a set: the list may come from the client, and looking each
// held resource up in a slice would let one request keep the stream locked,
// and List waiting for it, for seconds.
func (t *typeState) release(resourceNames map[string]struct{}) []string {
	if len(resourceNames) == 0 || len(t.rejections) == 0 {
		return nil
	}
	var released []string
	kept := t.rejections[:0]
	for _, rejection := range t.rejections {
		if len(rejection.Resources) == 0 {
			kept = append(kept, rejection)
			continue
		}
		rejection.Resources = slices.DeleteFunc(rejection.Resources, func(res ResourceRef) bool {
			if _, named := resourceNames[res.Name]; !named {
				return false
			}
			released = append(released, res.Name)
			return true
		})
		if len(rejection.Resources) > 0 {
			kept = append(kept, rejection)
		}
	}
	t.rejections = kept
	return released
}

// releaseUnnamed drops rejections of responses that carried no resources:
// nothing names them, so the next verdict on the type stands for them.
func (t *typeState) releaseUnnamed() bool {
	before := len(t.rejections)
	t.rejections = slices.DeleteFunc(t.rejections, func(rejection Rejection) bool {
		return len(rejection.Resources) == 0
	})
	return len(t.rejections) < before
}

func (r Rejection) clone() Rejection {
	r.Resources = append([]ResourceRef{}, r.Resources...)
	return r
}

func (o *Outcome) reject(typeURL string, rejection Rejection) {
	o.TypeURL, o.Rejection = typeURL, &rejection
}

func (o *Outcome) clear(typeURL, reason string, resourceNames []string) {
	if len(resourceNames) > 0 {
		o.Cleared = append(o.Cleared, Cleared{TypeURL: typeURL, Reason: reason, Resources: resourceNames})
	}
}

func nameSet(resourceNames []string) map[string]struct{} {
	set := make(map[string]struct{}, len(resourceNames))
	for _, name := range resourceNames {
		set[name] = struct{}{}
	}
	return set
}

func refSet(resources []ResourceRef) map[string]struct{} {
	set := make(map[string]struct{}, len(resources))
	for _, res := range resources {
		set[res.Name] = struct{}{}
	}
	return set
}

// IsKnownType reports whether the cache serves the type. Unrecognized client
// type URLs must not grow the per-stream registry.
func IsKnownType(typeURL string) bool {
	return cache.GetResponseType(typeURL) != types.UnknownType
}
