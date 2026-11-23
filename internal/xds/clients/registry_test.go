package clients

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	listenerType = "type.googleapis.com/envoy.config.listener.v3.Listener"
	clusterType  = "type.googleapis.com/envoy.config.cluster.v3.Cluster"
	routeType    = "type.googleapis.com/envoy.config.route.v3.RouteConfiguration"

	badListenerError = "Error adding/updating listener(s) default/bad: " +
		"unable to open file '/tmp/nodir/access.log': No such file or directory\n"
)

var (
	t0       = time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	deltaKey = Key{Protocol: Delta, ID: 1}
	sotwKey  = Key{Protocol: SotW, ID: 1}
)

type testClock struct{ now time.Time }

func (c *testClock) advance(d time.Duration) time.Time {
	c.now = c.now.Add(d)
	return c.now
}

// newTestRegistry returns a registry with one open stream per protocol and a
// clock the test moves by hand.
func newTestRegistry(t *testing.T) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: t0}
	r := NewRegistry()
	r.now = func() time.Time { return clock.now }
	for _, key := range []Key{deltaKey, sotwKey} {
		r.Set(key, &Info{Address: "10.0.0.7:41000"})
		r.Update(key, &Info{NodeID: "one", Version: "v1.40.2"})
	}
	return r, clock
}

// refs builds resource references from "name@version" pairs.
func refs(pairs ...string) []ResourceRef {
	out := make([]ResourceRef, 0, len(pairs))
	for _, pair := range pairs {
		name, version, _ := strings.Cut(pair, "@")
		out = append(out, ResourceRef{Name: name, Version: version})
	}
	return out
}

func clientOf(t *testing.T, r *Registry, key Key) Info {
	t.Helper()
	for _, c := range r.List() {
		if c.ID == key.ID && c.Protocol == key.Protocol {
			return c
		}
	}
	require.Failf(t, "stream not found", "%+v", key)
	return Info{}
}

func statusOf(t *testing.T, r *Registry, key Key, typeURL string) TypeStatus {
	t.Helper()
	for _, s := range clientOf(t, r, key).Types {
		if s.TypeURL == typeURL {
			return s
		}
	}
	require.Failf(t, "type not tracked", "%s on %+v", typeURL, key)
	return TypeStatus{}
}

func rejectedNames(s TypeStatus) []string {
	names := []string{}
	for _, rejection := range s.Rejections {
		for _, res := range rejection.Resources {
			names = append(names, res.Name)
		}
	}
	return names
}

func nack(typeURL, nonce, text string) Request {
	return Request{TypeURL: typeURL, Nonce: nonce, Rejected: true, Error: text}
}

// sotwNack rejects a listener response, sent by a client whose last accepted
// version is "6".
func sotwNack(nonce, text string) Request {
	return Request{TypeURL: listenerType, Nonce: nonce, Version: "6", Rejected: true, Error: text}
}

func cleared(typeURL, reason string, resourceNames ...string) []Cleared {
	return []Cleared{{TypeURL: typeURL, Reason: reason, Resources: resourceNames}}
}

func TestDeltaAckRecordsTheAcceptedVersion(t *testing.T) {
	r, clock := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/https@1")})

	sent := statusOf(t, r, deltaKey, listenerType)
	assert.Equal(t, t0, sent.AwaitingAckSince)
	assert.Empty(t, sent.AckedVersion)
	assert.True(t, sent.LastAckAt.IsZero())

	ackedAt := clock.advance(5 * time.Second)
	out := r.RequestReceived(deltaKey, Request{TypeURL: listenerType, Nonce: "1"})

	acked := statusOf(t, r, deltaKey, listenerType)
	assert.Equal(t, "7", acked.AckedVersion)
	assert.Equal(t, ackedAt, acked.LastAckAt)
	assert.True(t, acked.AwaitingAckSince.IsZero())
	assert.Empty(t, acked.Rejections)
	assert.Nil(t, out.Rejection)
	assert.Empty(t, out.Cleared)
}

// Envoy stops applying an RDS, SDS or EDS response at the first error and names
// no resource, so nothing in the response can be assumed applied.
func TestDeltaNackOfRoutesHoldsEveryResourceOfTheResponse(t *testing.T) {
	r, clock := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "1", Version: "3", Resources: refs("r1@x", "r2@y")})

	rejectedAt := clock.advance(time.Second)
	out := r.RequestReceived(deltaKey, nack(routeType, "1", "Only unique values for domains are permitted"))

	want := Rejection{
		Nonce:           "1",
		ResponseVersion: "3",
		RejectedAt:      rejectedAt,
		Count:           1,
		Error:           "Only unique values for domains are permitted",
		Resources:       refs("r1@x", "r2@y"),
	}
	require.NotNil(t, out.Rejection)
	assert.Equal(t, want, *out.Rejection)
	assert.Equal(t, routeType, out.TypeURL)
	assert.Equal(t, "one", out.Client.NodeID)
	assert.Equal(t, "10.0.0.7:41000", out.Client.Address)
	assert.Equal(t, "v1.40.2", out.Client.Version)
	assert.Equal(t, Delta, out.Client.Protocol)

	status := statusOf(t, r, deltaKey, routeType)
	assert.Equal(t, []Rejection{want}, status.Rejections)
	assert.Empty(t, status.AckedVersion)
	assert.True(t, status.AwaitingAckSince.IsZero())
}

// Envoy applies the good listeners of a response it NACKs and names the bad
// ones. The server never resends the good ones, so blaming them would leave a
// rejection that no fix can clear.
func TestDeltaNackOfListenersBlamesOnlyTheOnesEnvoyNamed(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{
		TypeURL: listenerType, Nonce: "1", Version: "7",
		Resources: refs("default/good@1", "default/bad@2", "default/https@3"),
	})

	out := r.RequestReceived(deltaKey, nack(listenerType, "1", badListenerError))

	require.NotNil(t, out.Rejection)
	assert.Equal(t, refs("default/bad@2"), out.Rejection.Resources)
	assert.Equal(t, []string{"default/bad"}, rejectedNames(statusOf(t, r, deltaKey, listenerType)))
}

func TestDeltaRejectionSurvivesAckOfAnotherResource(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/bad@2")})
	r.RequestReceived(deltaKey, nack(listenerType, "1", badListenerError))

	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "2", Version: "8", Resources: refs("default/good@1b")})
	out := r.RequestReceived(deltaKey, Request{TypeURL: listenerType, Nonce: "2"})

	status := statusOf(t, r, deltaKey, listenerType)
	assert.Equal(t, []string{"default/bad"}, rejectedNames(status))
	assert.Equal(t, "8", status.AckedVersion)
	assert.Empty(t, out.Cleared)
}

func TestDeltaAckOfRedeliveredResourceClearsItsRejection(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/bad@2")})
	r.RequestReceived(deltaKey, nack(listenerType, "1", badListenerError))

	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "2", Version: "8", Resources: refs("default/bad@2b")})
	out := r.RequestReceived(deltaKey, Request{TypeURL: listenerType, Nonce: "2"})

	assert.Empty(t, statusOf(t, r, deltaKey, listenerType).Rejections)
	assert.Equal(t, cleared(listenerType, ReasonAccepted, "default/bad"), out.Cleared)
}

// A listener that Envoy applied out of a NACKed response counts as accepted.
func TestDeltaNackClearsRejectionOfListenersItDidNotName(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/bad@2")})
	r.RequestReceived(deltaKey, nack(listenerType, "1", badListenerError))

	r.ResponseSent(deltaKey, Response{
		TypeURL: listenerType, Nonce: "2", Version: "8",
		Resources: refs("default/bad@2b", "default/worse@1"),
	})
	out := r.RequestReceived(deltaKey, nack(listenerType, "2",
		"Error adding/updating listener(s) default/worse: unknown filter\n"))

	assert.Equal(t, []string{"default/worse"}, rejectedNames(statusOf(t, r, deltaKey, listenerType)))
	assert.Equal(t, cleared(listenerType, ReasonAccepted, "default/bad"), out.Cleared)
}

func TestDeltaRenewedRejectionReplacesTheEarlierOne(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/bad@2")})
	r.RequestReceived(deltaKey, nack(listenerType, "1", badListenerError))

	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "2", Version: "8", Resources: refs("default/bad@2b")})
	out := r.RequestReceived(deltaKey, nack(listenerType, "2", badListenerError))

	status := statusOf(t, r, deltaKey, listenerType)
	require.Len(t, status.Rejections, 1)
	assert.Equal(t, "2", status.Rejections[0].Nonce)
	assert.Equal(t, refs("default/bad@2b"), status.Rejections[0].Resources)
	require.NotNil(t, out.Rejection)
	assert.Empty(t, out.Cleared)
}

func TestDeltaRemovalClearsRejection(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/bad@2")})
	r.RequestReceived(deltaKey, nack(listenerType, "1", badListenerError))

	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "2", Version: "8", Removed: []string{"default/bad"}})
	out := r.RequestReceived(deltaKey, Request{TypeURL: listenerType, Nonce: "2"})

	assert.Empty(t, statusOf(t, r, deltaKey, listenerType).Rejections)
	assert.Equal(t, cleared(listenerType, ReasonRemoved, "default/bad"), out.Cleared)
}

// The server forgets an unsubscribed resource and sends nothing, so no later
// verdict would ever clear its rejection.
func TestDeltaUnsubscribeClearsRejection(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "1", Version: "3", Resources: refs("r1@x")})
	r.RequestReceived(deltaKey, nack(routeType, "1", "bad route"))

	out := r.RequestReceived(deltaKey, Request{TypeURL: routeType, Unsubscribed: []string{"r1"}})

	assert.Empty(t, statusOf(t, r, deltaKey, routeType).Rejections)
	assert.Equal(t, cleared(routeType, ReasonUnsubscribed, "r1"), out.Cleared)
}

// A subscription change can make the server send the next response of a type
// before the client has answered the previous one.
func TestDeltaVerdictReachesAnEarlierResponseAfterALaterOneWasSent(t *testing.T) {
	r, clock := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "1", Version: "3", Resources: refs("r1@x")})
	secondSentAt := clock.advance(time.Second)
	r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "2", Version: "3", Resources: refs("r2@y")})

	r.RequestReceived(deltaKey, nack(routeType, "1", "bad route"))
	afterNack := statusOf(t, r, deltaKey, routeType)
	assert.Equal(t, []string{"r1"}, rejectedNames(afterNack))
	assert.Equal(t, secondSentAt, afterNack.AwaitingAckSince)

	r.RequestReceived(deltaKey, Request{TypeURL: routeType, Nonce: "2"})
	afterAck := statusOf(t, r, deltaKey, routeType)
	assert.Equal(t, []string{"r1"}, rejectedNames(afterAck))
	assert.Equal(t, "3", afterAck.AckedVersion)
	assert.True(t, afterAck.AwaitingAckSince.IsZero())
}

func TestDeltaRejectionWithoutResourcesIsClearedByTheNextAck(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Removed: []string{"default/gone"}})
	r.RequestReceived(deltaKey, nack(listenerType, "1", "cannot remove"))
	require.Len(t, statusOf(t, r, deltaKey, listenerType).Rejections, 1)

	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "2", Version: "8", Resources: refs("default/https@3")})
	r.RequestReceived(deltaKey, Request{TypeURL: listenerType, Nonce: "2"})

	assert.Empty(t, statusOf(t, r, deltaKey, listenerType).Rejections)
}

// Envoy removes listeners before it adds any, so the removals of a response
// take effect even when the response is NACKed.
func TestDeltaNackStillAppliesItsRemovals(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/bad@2")})
	r.RequestReceived(deltaKey, nack(listenerType, "1", badListenerError))

	r.ResponseSent(deltaKey, Response{
		TypeURL: listenerType, Nonce: "2", Version: "8",
		Resources: refs("default/worse@1"), Removed: []string{"default/bad"},
	})
	out := r.RequestReceived(deltaKey, nack(listenerType, "2",
		"Error adding/updating listener(s) default/worse: unknown filter\n"))

	assert.Equal(t, []string{"default/worse"}, rejectedNames(statusOf(t, r, deltaKey, listenerType)))
	assert.Equal(t, cleared(listenerType, ReasonRemoved, "default/bad"), out.Cleared)
}

func TestDeltaRejectionWithoutResourcesIsReplacedAndOutlivesAnUnrelatedRelease(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Removed: []string{"default/gone"}})
	r.RequestReceived(deltaKey, nack(listenerType, "1", "cannot remove"))
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "2", Version: "8", Removed: []string{"default/gone2"}})
	r.RequestReceived(deltaKey, nack(listenerType, "2", "cannot remove"))
	require.Len(t, statusOf(t, r, deltaKey, listenerType).Rejections, 1)

	out := r.RequestReceived(deltaKey, Request{TypeURL: listenerType, Unsubscribed: []string{"default/other"}})

	require.Len(t, statusOf(t, r, deltaKey, listenerType).Rejections, 1)
	assert.Empty(t, out.Cleared)
}

// Envoy names the listeners it skipped in the text as it sent it. A listener
// named after a quoted value would be lost if the names were looked up in the
// sanitized text, and would then count as accepted.
func TestDeltaNackFindsNamesInTheTextAsEnvoySentIt(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{
		TypeURL: listenerType, Nonce: "1", Version: "7",
		Resources: refs("default/a@1", "default/b@1", "default/c@1"),
	})

	out := r.RequestReceived(deltaKey, nack(listenerType, "1",
		"Error adding/updating listener(s) default/a: invalid character '\"' in regex\ndefault/c: unknown filter\n"))

	require.NotNil(t, out.Rejection)
	assert.Equal(t, refs("default/a@1", "default/c@1"), out.Rejection.Resources)
	assert.NotContains(t, out.Rejection.Error, "default/c")
}

// The server sends nothing more about a resource the client unsubscribed
// from, so holding it rejected would keep the rejection for good.
func TestDeltaNackDoesNotHoldWhatTheClientUnsubscribedFrom(t *testing.T) {
	for _, tc := range []struct {
		name string
		nack func(r *Registry) Outcome
	}{
		{
			name: "in the request that rejects the response",
			nack: func(r *Registry) Outcome {
				return r.RequestReceived(deltaKey, Request{
					TypeURL: routeType, Nonce: "1", Rejected: true, Error: "bad route", Unsubscribed: []string{"r1"},
				})
			},
		},
		{
			name: "in a request sent before the verdict",
			nack: func(r *Registry) Outcome {
				r.RequestReceived(deltaKey, Request{TypeURL: routeType, Unsubscribed: []string{"r1"}})
				return r.RequestReceived(deltaKey, nack(routeType, "1", "bad route"))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := newTestRegistry(t)
			r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "1", Version: "3", Resources: refs("r1@x", "r2@y")})

			out := tc.nack(r)

			require.NotNil(t, out.Rejection)
			assert.Equal(t, refs("r2@y"), out.Rejection.Resources)
			assert.Equal(t, []string{"r2"}, rejectedNames(statusOf(t, r, deltaKey, routeType)))
			assert.Empty(t, out.Cleared)
		})
	}
}

func TestDeltaNackOfAResponseTheClientNoLongerWantsHoldsNothing(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "1", Version: "3", Resources: refs("r1@x")})

	out := r.RequestReceived(deltaKey, Request{
		TypeURL: routeType, Nonce: "1", Rejected: true, Error: "bad route", Unsubscribed: []string{"r1"},
	})

	assert.Nil(t, out.Rejection)
	assert.Empty(t, statusOf(t, r, deltaKey, routeType).Rejections)
}

// A route, a secret and a listener are all named after their namespace/name.
func TestDeltaUnsubscribeMarksOnlyResponsesOfItsType(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: clusterType, Nonce: "1", Version: "3", Resources: refs("shared@1")})
	r.RequestReceived(deltaKey, Request{TypeURL: routeType, Unsubscribed: []string{"shared"}})

	out := r.RequestReceived(deltaKey, nack(clusterType, "1", "bad cluster"))

	require.NotNil(t, out.Rejection)
	assert.Equal(t, refs("shared@1"), out.Rejection.Resources)
}

func TestDeltaNackThatHoldsNothingStillReportsWhatItCleared(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "1", Version: "3", Resources: refs("old@1")})
	r.RequestReceived(deltaKey, nack(routeType, "1", "bad route"))
	r.ResponseSent(deltaKey, Response{
		TypeURL: routeType, Nonce: "2", Version: "4", Resources: refs("r1@x"), Removed: []string{"old"},
	})

	out := r.RequestReceived(deltaKey, Request{
		TypeURL: routeType, Nonce: "2", Rejected: true, Error: "bad route", Unsubscribed: []string{"r1"},
	})

	assert.Nil(t, out.Rejection)
	assert.Equal(t, cleared(routeType, ReasonRemoved, "old"), out.Cleared)
	assert.Equal(t, "one", out.Client.NodeID)
	assert.True(t, statusOf(t, r, deltaKey, routeType).AwaitingAckSince.IsZero())
}

// Envoy named one listener and the same request unsubscribes from it: nothing
// is left to hold, yet the listeners it did not name were applied.
func TestDeltaNackOfAnUnsubscribedListenerStillAcceptsTheOthers(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/good@1")})
	r.RequestReceived(deltaKey, nack(listenerType, "1", "Error adding/updating listener(s) default/good: no file\n"))
	r.ResponseSent(deltaKey, Response{
		TypeURL: listenerType, Nonce: "2", Version: "8",
		Resources: refs("default/good@2", "default/bad@2"),
	})

	out := r.RequestReceived(deltaKey, Request{
		TypeURL: listenerType, Nonce: "2", Rejected: true, Error: badListenerError,
		Unsubscribed: []string{"default/bad"},
	})

	assert.Nil(t, out.Rejection)
	assert.Equal(t, cleared(listenerType, ReasonAccepted, "default/good"), out.Cleared)
	assert.Empty(t, statusOf(t, r, deltaKey, listenerType).Rejections)
}

// A client decides how much work its own stream takes: how many responses it
// leaves unanswered, how long its lists are. The others must not queue behind
// it, nor behind a List that waits for it.
func TestStreamsDoNotWaitForEachOther(t *testing.T) {
	for _, tc := range []struct {
		name string
		work func(r *Registry)
	}{
		{name: "while a response is recorded", work: func(r *Registry) {
			r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "2", Version: "8"})
		}},
		{name: "while a request is applied", work: func(r *Registry) {
			r.RequestReceived(deltaKey, Request{TypeURL: listenerType, Nonce: "1"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			r.Set(deltaKey, &Info{})
			r.Set(sotwKey, &Info{})
			r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7"})
			entered, release := make(chan struct{}), make(chan struct{})
			var readings atomic.Int32
			// The clock is read in the middle of the work on a stream: the first
			// reading stops there until released.
			r.now = func() time.Time {
				if readings.Add(1) == 1 {
					close(entered)
					<-release
				}
				return t0
			}
			busy := make(chan struct{})
			go func() {
				defer close(busy)
				tc.work(r)
			}()
			<-entered

			// List has to wait for the busy stream. If it does not, the clock is no
			// longer read while the stream is locked and the rest proves nothing.
			listed := make(chan struct{})
			go func() {
				defer close(listed)
				r.List()
			}()
			select {
			case <-listed:
				t.Error("List did not wait for the stream that is being worked on")
			case <-time.After(10 * time.Millisecond):
			}

			other := make(chan struct{})
			go func() {
				defer close(other)
				// What the callbacks call for a request and a response of a stream.
				r.Update(sotwKey, &Info{NodeID: "two"})
				r.ResponseSent(sotwKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7"})
				r.RequestReceived(sotwKey, Request{TypeURL: listenerType, Nonce: "1", Version: "7"})
			}()
			select {
			case <-other:
			case <-time.After(5 * time.Second):
				t.Error("a stream waited for the work on another stream")
			}
			close(release)
			<-busy
			<-other
			<-listed
		})
	}
}

// On a stream that serves one type the client may leave the type URL out.
func TestRequestWithoutTypeURLIsMatchedByNonce(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/https@3")})

	r.RequestReceived(deltaKey, Request{Nonce: "1"})

	assert.Equal(t, "7", statusOf(t, r, deltaKey, listenerType).AckedVersion)
}

// In SotW every request after the first response repeats its nonce, and a
// request that carries the old version only reports what the client still has.
func TestSotWAckNeedsTheVersionOfTheResponse(t *testing.T) {
	r, clock := newTestRegistry(t)
	r.ResponseSent(sotwKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7"})
	r.RequestReceived(sotwKey, Request{TypeURL: listenerType, Nonce: "1", Version: "6"})

	ignored := statusOf(t, r, sotwKey, listenerType)
	assert.Empty(t, ignored.AckedVersion)
	assert.True(t, ignored.LastAckAt.IsZero())
	assert.True(t, ignored.AwaitingAckSince.IsZero())

	r.ResponseSent(sotwKey, Response{TypeURL: listenerType, Nonce: "2", Version: "7"})
	ackedAt := clock.advance(time.Second)
	r.RequestReceived(sotwKey, Request{TypeURL: listenerType, Nonce: "2", Version: "7"})

	acked := statusOf(t, r, sotwKey, listenerType)
	assert.Equal(t, "7", acked.AckedVersion)
	assert.Equal(t, ackedAt, acked.LastAckAt)
}

// Envoy drops error_detail after sending it once but keeps the nonce, so the
// next subscription change looks like an ACK of the response it has rejected.
func TestSotWRejectionSurvivesRequestThatRepeatsTheNonce(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(sotwKey, Response{TypeURL: routeType, Nonce: "1", Version: "3"})
	r.RequestReceived(sotwKey, Request{TypeURL: routeType, Nonce: "1", Version: "2", Rejected: true, Error: "bad route"})

	r.RequestReceived(sotwKey, Request{TypeURL: routeType, Nonce: "1", Version: "2"})

	status := statusOf(t, r, sotwKey, routeType)
	require.Len(t, status.Rejections, 1)
	assert.Equal(t, "3", status.Rejections[0].ResponseVersion)
	assert.Empty(t, status.Rejections[0].Resources)
}

// The server answers every SotW NACK with the same snapshot, a thousand times
// a second, and the error text changes with the order of resources in it.
func TestSotWRepeatedNackOfAVersionIsCountedNotReported(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(sotwKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7"})
	first := r.RequestReceived(sotwKey, sotwNack("1", "a: x\nb: y\n"))
	require.NotNil(t, first.Rejection)

	r.ResponseSent(sotwKey, Response{TypeURL: listenerType, Nonce: "2", Version: "7"})
	repeat := r.RequestReceived(sotwKey, sotwNack("2", "b: y\na: x\n"))

	assert.Nil(t, repeat.Rejection)
	status := statusOf(t, r, sotwKey, listenerType)
	require.Len(t, status.Rejections, 1)
	assert.Equal(t, 2, status.Rejections[0].Count)
	assert.Equal(t, "a: x\nb: y", status.Rejections[0].Error)

	r.ResponseSent(sotwKey, Response{TypeURL: listenerType, Nonce: "3", Version: "8"})
	next := r.RequestReceived(sotwKey, sotwNack("3", "a: x\n"))

	require.NotNil(t, next.Rejection)
	status = statusOf(t, r, sotwKey, listenerType)
	require.Len(t, status.Rejections, 1)
	assert.Equal(t, "8", status.Rejections[0].ResponseVersion)
	assert.Equal(t, 1, status.Rejections[0].Count)
}

func TestSotWAckClearsRejection(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(sotwKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7"})
	r.RequestReceived(sotwKey, sotwNack("1", "bad"))

	r.ResponseSent(sotwKey, Response{TypeURL: listenerType, Nonce: "2", Version: "8"})
	out := r.RequestReceived(sotwKey, Request{TypeURL: listenerType, Nonce: "2", Version: "8"})

	assert.Empty(t, statusOf(t, r, sotwKey, listenerType).Rejections)
	assert.Equal(t, []Cleared{{TypeURL: listenerType, Reason: ReasonAccepted}}, out.Cleared)
}

func TestNackThatMatchesNoResponseIsCountedAndReportedOnce(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/https@3")})

	unknownNonce := r.RequestReceived(deltaKey, nack(listenerType, "9", "x"))
	withoutNonce := r.RequestReceived(deltaKey, nack(listenerType, "", "x"))
	otherType := r.RequestReceived(deltaKey, nack(clusterType, "1", "x"))

	assert.True(t, unknownNonce.UnknownNack)
	assert.Equal(t, "x", unknownNonce.UnknownError)
	assert.Equal(t, "one", unknownNonce.Client.NodeID)
	assert.False(t, withoutNonce.UnknownNack)
	assert.False(t, otherType.UnknownNack)
	for _, out := range []Outcome{unknownNonce, withoutNonce, otherType} {
		assert.Nil(t, out.Rejection)
	}

	client := clientOf(t, r, deltaKey)
	assert.Equal(t, 3, client.UnknownNacks)
	listeners := statusOf(t, r, deltaKey, listenerType)
	assert.Empty(t, listeners.Rejections)
	assert.Equal(t, t0, listeners.AwaitingAckSince)
}

func TestStreamThatWasNeverOpenedIsNotTracked(t *testing.T) {
	r := NewRegistry()
	key := Key{Protocol: Delta, ID: 42}

	r.ResponseSent(key, Response{TypeURL: listenerType, Nonce: "1", Version: "7"})
	out := r.RequestReceived(key, nack(listenerType, "1", "x"))

	assert.Empty(t, r.List())
	assert.Equal(t, Outcome{}, out)
}

func TestResponseWithoutNonceIsNotTracked(t *testing.T) {
	r, _ := newTestRegistry(t)

	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Version: "7"})

	assert.Empty(t, clientOf(t, r, deltaKey).Types)
}

func TestUnansweredResponsesAreBounded(t *testing.T) {
	r, clock := newTestRegistry(t)
	sentAt := map[string]time.Time{}
	for i := 1; i <= 70; i++ {
		nonce := strconv.Itoa(i)
		sentAt[nonce] = clock.advance(time.Second)
		r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: nonce, Version: "3", Resources: refs("r" + nonce)})
	}

	assert.Equal(t, sentAt["7"], statusOf(t, r, deltaKey, routeType).AwaitingAckSince)

	forgotten := r.RequestReceived(deltaKey, nack(routeType, "6", "x"))
	assert.Nil(t, forgotten.Rejection)
	assert.True(t, forgotten.UnknownNack)

	kept := r.RequestReceived(deltaKey, nack(routeType, "7", "x"))
	require.NotNil(t, kept.Rejection)
}

func TestRejectionsOfATypeAreBounded(t *testing.T) {
	r, _ := newTestRegistry(t)
	for i := 1; i <= 70; i++ {
		nonce := strconv.Itoa(i)
		r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: nonce, Version: "3", Resources: refs("r" + nonce)})
		r.RequestReceived(deltaKey, nack(routeType, nonce, "bad route"))
	}

	status := statusOf(t, r, deltaKey, routeType)
	names := rejectedNames(status)
	require.Len(t, names, 64)
	assert.Equal(t, "r7", names[0])
	assert.Equal(t, "r70", names[63])
	assert.Equal(t, 6, status.DroppedRejections)
}

func TestListIsOrderedByProtocolAndStream(t *testing.T) {
	r := NewRegistry()
	for _, key := range []Key{{Delta, 2}, {SotW, 1}, {Delta, 1}} {
		r.Set(key, &Info{})
	}

	keys := []Key{}
	for _, c := range r.List() {
		keys = append(keys, Key{Protocol: c.Protocol, ID: c.ID})
	}
	assert.Equal(t, []Key{{Delta, 1}, {Delta, 2}, {SotW, 1}}, keys)
}

func TestListReturnsCopiesTheCallerMayChange(t *testing.T) {
	r, _ := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "1", Version: "3", Resources: refs("r1@x")})
	r.RequestReceived(deltaKey, nack(routeType, "1", "bad route"))

	list := r.List()
	require.Len(t, list[0].Types, 1)
	require.Len(t, list[0].Types[0].Rejections, 1)
	list[0].NodeID = "changed"
	list[0].Types[0].Rejections[0].Resources[0].Name = "changed"
	list[0].Types[0].Rejections = nil

	assert.Equal(t, "one", clientOf(t, r, deltaKey).NodeID)
	assert.Equal(t, []string{"r1"}, rejectedNames(statusOf(t, r, deltaKey, routeType)))
}

// The JSON shape is what /debug/connected-clients serves.
func TestListJSON(t *testing.T) {
	clock := &testClock{now: t0}
	r := NewRegistry()
	r.now = func() time.Time { return clock.now }
	r.Set(deltaKey, &Info{Address: "10.0.0.7:41000"})
	r.Update(deltaKey, &Info{NodeID: "one", Version: "v1.40.2"})
	r.Set(Key{Protocol: Delta, ID: 2}, &Info{})
	r.ResponseSent(deltaKey, Response{TypeURL: listenerType, Nonce: "1", Version: "7", Resources: refs("default/bad@2")})
	clock.advance(time.Second)
	r.RequestReceived(deltaKey, nack(listenerType, "1", badListenerError))

	got, err := json.Marshal(r.List())
	require.NoError(t, err)

	assert.JSONEq(t, `[
		{
			"streamId": 1, "protocol": "delta", "nodeId": "one", "address": "10.0.0.7:41000",
			"envoyVersion": "v1.40.2", "connectedAt": "2026-11-18T10:00:00Z",
			"types": [{
				"typeUrl": "type.googleapis.com/envoy.config.listener.v3.Listener",
				"rejections": [{
					"nonce": "1", "responseVersion": "7", "rejectedAt": "2026-11-18T10:00:01Z", "count": 1,
					"error": "Error adding/updating listener(s) default/bad: unable to open file '/tmp/nodir/access.log'`+
		`: No such file or directory",
					"resources": [{"name": "default/bad", "version": "2"}]
				}]
			}]
		},
		{
			"streamId": 2, "protocol": "delta", "nodeId": "", "address": "", "envoyVersion": "",
			"connectedAt": "2026-11-18T10:00:00Z", "types": []
		}
	]`, string(got))
}

func TestListJSONOfAnAcknowledgedType(t *testing.T) {
	r, clock := newTestRegistry(t)
	r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "1", Version: "7", Resources: refs("r1@3")})
	clock.advance(time.Second)
	r.RequestReceived(deltaKey, Request{TypeURL: routeType, Nonce: "1"})
	clock.advance(time.Second)
	r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: "2", Version: "8", Resources: refs("r1@4")})
	r.RequestReceived(deltaKey, nack(routeType, "9", "x"))

	got, err := json.Marshal(clientOf(t, r, deltaKey))
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"streamId": 1, "protocol": "delta", "nodeId": "one", "address": "10.0.0.7:41000",
		"envoyVersion": "v1.40.2", "connectedAt": "2026-11-18T10:00:00Z", "unknownNacks": 1,
		"types": [{
			"typeUrl": "type.googleapis.com/envoy.config.route.v3.RouteConfiguration",
			"ackedVersion": "7", "lastAckAt": "2026-11-18T10:00:01Z",
			"awaitingAckSince": "2026-11-18T10:00:02Z", "rejections": []
		}]
	}`, string(got))
}

func TestListJSONCountsDroppedRejections(t *testing.T) {
	r, _ := newTestRegistry(t)
	for i := 1; i <= 65; i++ {
		nonce := strconv.Itoa(i)
		r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: nonce, Version: "3", Resources: refs("r" + nonce)})
		r.RequestReceived(deltaKey, nack(routeType, nonce, "bad route"))
	}

	got, err := json.Marshal(statusOf(t, r, deltaKey, routeType))
	require.NoError(t, err)

	assert.Contains(t, string(got), `"droppedRejections":1`)
}

// Run with -race: readers serialise the list while one stream keeps changing
// and others open and close.
func TestListIsSafeWhileStreamsChange(t *testing.T) {
	r, _ := newTestRegistry(t)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 1; i <= 500; i++ {
			nonce := strconv.Itoa(i)
			r.Update(deltaKey, &Info{NodeID: "one", Version: "v1.40." + nonce})
			r.ResponseSent(deltaKey, Response{TypeURL: routeType, Nonce: nonce, Version: "3", Resources: refs("r1@" + nonce)})
			if i%2 == 0 {
				r.RequestReceived(deltaKey, Request{TypeURL: routeType, Nonce: nonce})
			} else {
				r.RequestReceived(deltaKey, nack(routeType, nonce, "bad route"))
			}
		}
	}()
	go func() {
		defer wg.Done()
		for id := int64(2); id <= 501; id++ {
			key := Key{Protocol: Delta, ID: id}
			r.Set(key, &Info{})
			r.Update(key, &Info{NodeID: "two"})
			r.Delete(key)
		}
	}()
	go func() {
		defer wg.Done()
		for range 500 {
			_, err := json.Marshal(r.List())
			assert.NoError(t, err)
		}
	}()
	wg.Wait()
}
