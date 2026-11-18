package clients

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	listenerTypeURL = "type.googleapis.com/envoy.config.listener.v3.Listener"
	clusterTypeURL  = "type.googleapis.com/envoy.config.cluster.v3.Cluster"

	// Envoy cuts its error text at 4096 characters and appends this mark.
	maxErrorLen   = 4096
	truncatedMark = "...(truncated)"
	// maxRawErrorLen bounds the work spent on a text no Envoy would send.
	maxRawErrorLen = 2 * maxErrorLen

	validationMark = "Proto constraint validation failed ("
)

// sanitizeError removes message dumps from a client's error text before the
// text is logged and served.
//
// Envoy builds the text from an exception and, for a message that fails
// validation or cannot be unpacked, puts the whole message into it without
// redacting anything: a rejected secret carries its key. No field name or
// format can be relied on, so everything from the first double quote or PEM
// header on is dropped. What Envoy echoes outside double quotes stays, such as
// a file path or a header value it could not parse. The full text is always in
// Envoy's own log.
func sanitizeError(raw string) string {
	text := raw
	if len(text) > maxRawErrorLen {
		text = text[:maxRawErrorLen]
	}
	cut := strings.IndexByte(text, '"')
	if i := strings.Index(text, "-----BEGIN"); i >= 0 && (cut < 0 || i < cut) {
		cut = i
	}
	if cut >= 0 {
		removed, reason := text[cut:], ""
		// Envoy states why validation failed after the message; keep that
		// unless it quotes data itself.
		if i := strings.LastIndex(removed, validationMark); i >= 0 &&
			!strings.ContainsRune(removed[i:], '"') && !strings.Contains(removed[i:], "-----BEGIN") {
			reason = " " + removed[i:]
		}
		text = fmt.Sprintf("%s[%d bytes removed: quoted data]%s", text[:cut], len(removed)-len(reason), reason)
	}
	text = strings.TrimRightFunc(text, unicode.IsSpace)
	if len(text) > maxErrorLen {
		text = text[:maxErrorLen] + truncatedMark
	}
	return strings.ToValidUTF8(text, "")
}

// splitRejected tells which resources of a NACKed Delta response the client
// refused and which it took.
//
// Envoy applies the good listeners and clusters of a response, skips the bad
// ones and lists the bad ones by name in the error. For every other type, and
// whenever the text does not have that form, it may have applied nothing, so
// the whole response is held rejected.
//
// The texts come from LdsApiImpl::onConfigUpdate (listener_manager/lds_api.cc)
// and CdsApiImpl::onConfigUpdate (upstream/cds_api_impl.cc) and are the same
// in Envoy 1.34 and 1.40. They are log text, not an API: check them again when
// the supported Envoy versions change.
func splitRejected(typeURL, raw string, resources []ResourceRef) (rejected, rest []ResourceRef) {
	var prefix, separator string
	switch typeURL {
	case listenerTypeURL:
		prefix, separator = "Error adding/updating listener(s) ", "\n"
	case clusterTypeURL:
		prefix, separator = "Error adding/updating cluster(s) ", ", "
	default:
		return resources, nil
	}
	list, ok := strings.CutPrefix(raw, prefix)
	if !ok || strings.HasSuffix(raw, truncatedMark) {
		return resources, nil
	}
	list = separator + list
	for _, res := range resources {
		if strings.Contains(list, separator+res.Name+": ") {
			rejected = append(rejected, res)
		} else {
			rest = append(rest, res)
		}
	}
	if len(rejected) == 0 {
		return resources, nil
	}
	return rejected, rest
}
