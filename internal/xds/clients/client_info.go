package clients

import "time"

// Protocol is the xDS transport variant a stream uses.
type Protocol string

const (
	SotW  Protocol = "sotw"
	Delta Protocol = "delta"
)

// Key identifies an xDS stream. The SotW and Delta servers count their streams
// independently, so a stream id alone is ambiguous.
type Key struct {
	Protocol Protocol
	ID       int64
}

// Info describes one connected xDS stream. Several Envoy instances may share a
// node id, so the stream is the only unit the server can tell apart.
type Info struct {
	ID       int64    `json:"streamId"`
	Protocol Protocol `json:"protocol"`
	NodeID   string   `json:"nodeId"`
	Address  string   `json:"address"`
	// ForwardedAddress is the reported originating IP in the stream's X-Forwarded-For header.
	ForwardedAddress string `json:"forwardedAddress,omitempty"`
	// Host is the instance or hostname reported in node metadata.
	Host        string    `json:"host,omitempty"`
	Locality    Locality  `json:"locality,omitzero"`
	Version     string    `json:"envoyVersion"`
	ConnectedAt time.Time `json:"connectedAt,omitzero"`
	// UnknownNacks counts NACKs that named no response of this stream.
	UnknownNacks int          `json:"unknownNacks,omitempty"`
	Types        []TypeStatus `json:"types"`
}

// Locality is the location reported by the xDS client.
type Locality struct {
	Region  string `json:"region,omitempty"`
	Zone    string `json:"zone,omitempty"`
	SubZone string `json:"subZone,omitempty"`
}

// TypeStatus is what the stream has told the server about one resource type.
type TypeStatus struct {
	TypeURL string `json:"typeUrl"`
	// AckedVersion is the type version carried by the last response the client
	// ACKed. An ACK means the client found the resources valid, not that they
	// already serve traffic.
	AckedVersion string    `json:"ackedVersion,omitempty"`
	LastAckAt    time.Time `json:"lastAckAt,omitzero"`
	// AwaitingAckSince is when the oldest response without a verdict was sent.
	AwaitingAckSince time.Time `json:"awaitingAckSince,omitzero"`
	// Rejections lists what the client NACKed and has not accepted since.
	Rejections        []Rejection `json:"rejections"`
	DroppedRejections int         `json:"droppedRejections,omitempty"`
}

// Rejection is a NACKed response whose resources the client has not accepted since.
type Rejection struct {
	Nonce           string    `json:"nonce"`
	ResponseVersion string    `json:"responseVersion"`
	RejectedAt      time.Time `json:"rejectedAt"`
	// Count is how many times the client NACKed this version. It grows only in
	// SotW, where the server answers every NACK with the same snapshot.
	Count int `json:"count"`
	// Error is the client's error text with quoted data removed.
	Error string `json:"error"`
	// Resources are the listeners or clusters the client named in its error; for
	// other types, every resource of the response, because Envoy stops applying a
	// response at the first error. Empty in SotW: the whole type version is rejected.
	Resources []ResourceRef `json:"resources"`
}

type ResourceRef struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Response is an xDS response as the server hands it to a stream.
type Response struct {
	TypeURL string
	Nonce   string
	Version string
	// Resources and Removed are set in Delta only.
	Resources []ResourceRef
	Removed   []string
}

// Request is the part of a client request that can answer an earlier response.
type Request struct {
	TypeURL string
	Nonce   string
	// Version is the SotW version_info: the last version the client accepted.
	Version string
	// Rejected reports that the request carries error_detail, Error is its text.
	Rejected bool
	Error    string
	// Unsubscribed is the Delta resource_names_unsubscribe list.
	Unsubscribed []string
}

// Why a recorded rejection no longer stands.
const (
	ReasonAccepted     = "accepted"
	ReasonRemoved      = "removed"
	ReasonUnsubscribed = "unsubscribed"
)

// Cleared names rejections that a request made obsolete.
type Cleared struct {
	TypeURL   string
	Reason    string
	Resources []string
}

// Outcome is what a request changed, for the caller to log.
type Outcome struct {
	// Client identifies the stream; its Types are not filled.
	Client Info
	// TypeURL and Rejection are set when the request recorded a new rejection.
	TypeURL   string
	Rejection *Rejection
	// UnknownNack is set on the first NACK of a stream that matched no response
	// of it, UnknownError is the text of that NACK with quoted data removed.
	UnknownNack  bool
	UnknownError string
	Cleared      []Cleared
}
