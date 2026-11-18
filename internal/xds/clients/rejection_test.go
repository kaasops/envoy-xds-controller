package clients

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

const (
	keyBody = "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7"
	// How protobuf prints a private key inside a message dump: one quoted
	// literal with the line breaks escaped.
	keyLiteral = `"-----BEGIN PRIVATE KEY-----\n` + keyBody + `\n-----END PRIVATE KEY-----\n"`
	// Envoy 1.40 starts a message dump with a protobuf marker, 1.34 does not.
	debugMarker = "goo.gle/debugonly  "
)

func TestSanitizeErrorKeepsOrdinaryRejectionsWhole(t *testing.T) {
	for _, tc := range []struct{ name, raw, want string }{
		{
			name: "missing log directory",
			raw: "Error adding/updating listener(s) default/https: " +
				"unable to open file '/tmp/nodir/access.log': No such file or directory\n",
			want: "Error adding/updating listener(s) default/https: " +
				"unable to open file '/tmp/nodir/access.log': No such file or directory",
		},
		{
			name: "missing module file",
			raw: "Error adding/updating listener(s) https: Failed to load dynamic module: /usr/lib/envoy/libwaf.so : " +
				"/usr/lib/envoy/libwaf.so: cannot open shared object file: No such file or directory\n",
			want: "Error adding/updating listener(s) https: Failed to load dynamic module: /usr/lib/envoy/libwaf.so : " +
				"/usr/lib/envoy/libwaf.so: cannot open shared object file: No such file or directory",
		},
		{
			name: "unknown extension",
			raw: "Error adding/updating listener(s) https: Didn't find a registered implementation for " +
				"'envoy.filters.http.golang' with type URL: 'envoy.extensions.filters.http.golang.v3alpha.Config'\n",
			want: "Error adding/updating listener(s) https: Didn't find a registered implementation for " +
				"'envoy.filters.http.golang' with type URL: 'envoy.extensions.filters.http.golang.v3alpha.Config'",
		},
		{
			name: "two listeners",
			raw:  "Error adding/updating listener(s) a: unable to open file '/var/log/a.log'\nb: duplicate address\n",
			want: "Error adding/updating listener(s) a: unable to open file '/var/log/a.log'\nb: duplicate address",
		},
		{name: "empty", raw: "", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sanitizeError(tc.raw))
		})
	}
}

// Envoy puts the whole offending message into the error text and redacts
// nothing, so a rejected secret would carry its key into logs and the API.
func TestSanitizeErrorRemovesQuotedData(t *testing.T) {
	const reason = "Proto constraint validation failed " +
		"(SecretValidationError.Name: value length must be at least 1 characters)"
	for _, tc := range []struct {
		name   string
		raw    string
		secret string
		keeps  string
	}{
		{
			name: "whole key in a message dump",
			raw: "name: \"prod/tls\"\ntls_certificate {\n  private_key {\n    inline_bytes: " + keyLiteral +
				"\n  }\n}\n: " + reason,
			secret: keyBody,
			keeps:  reason,
		},
		{
			name: "dump cut by Envoy in the middle of the key",
			raw: debugMarker + "name: \"prod/tls\"\ntls_certificate {\n  private_key {\n    inline_bytes: " +
				`"-----BEGIN PRIVATE KEY-----\n` + keyBody + "...(truncated)",
			secret: keyBody,
		},
		{
			name: "secret in a field no rule knows by name",
			raw: debugMarker + "name: \"https\"\nrequest_headers_to_add {\n  header {\n    key: \"authorization\"\n" +
				"    value: \"Bearer eyJhbGciOiJSUzI1NiJ9.payload.signature\"\n  }\n}\n: " + reason,
			secret: "eyJhbGciOiJSUzI1NiJ9",
			keeps:  reason,
		},
		{
			name: "a stray quote before the dump",
			raw: "Error adding/updating listener(s) a: invalid character '\"' in regex\nb: wrong transport socket config: " +
				"name: \"tls\"\ninline_string: " + keyLiteral,
			secret: keyBody,
			keeps:  "Error adding/updating listener(s) a: invalid character '",
		},
		{
			name:   "key after the validation reason",
			raw:    "name: \"x\"\n: Proto constraint validation failed (y) -----BEGIN PRIVATE KEY-----\n" + keyBody,
			secret: keyBody,
		},
		{
			name:   "key outside quotes",
			raw:    "cannot parse -----BEGIN PRIVATE KEY-----\n" + keyBody + "\n-----END PRIVATE KEY-----",
			secret: keyBody,
			keeps:  "cannot parse ",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeError(tc.raw)

			assert.NotContains(t, got, tc.secret)
			assert.Contains(t, got, "bytes removed: quoted data]")
			if tc.keeps != "" {
				assert.Contains(t, got, tc.keeps)
			}
		})
	}
}

// A validation reason that quotes a value cannot be told from data.
func TestSanitizeErrorDropsValidationReasonThatQuotesData(t *testing.T) {
	got := sanitizeError("name: \"x\"\n: Proto constraint validation failed " +
		"(ListenerValidationError.Name: value does not match regex pattern \"^token-[a-z]+$\")")

	assert.NotContains(t, got, "token-")
	assert.NotContains(t, got, "Proto constraint validation failed")
}

// Envoy cuts the text at 4096 characters. Anything longer comes from another
// client, which may send megabytes.
func TestSanitizeErrorBoundsTheLength(t *testing.T) {
	for _, tc := range []struct{ name, raw string }{
		{"ascii", strings.Repeat("a", 4<<20)},
		// Three bytes a rune: neither cut falls on a rune boundary.
		{"multibyte", strings.Repeat("€", 2<<20)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeError(tc.raw)

			assert.LessOrEqual(t, len(got), 4096+len("...(truncated)"))
			assert.True(t, strings.HasSuffix(got, "...(truncated)"))
			assert.True(t, utf8.ValidString(got))
		})
	}
}

func TestSplitRejected(t *testing.T) {
	all := refs("default/good@1", "default/https@2", "default/https-module@3")
	for _, tc := range []struct {
		name         string
		typeURL, raw string
		resources    []ResourceRef
		wantRejected []ResourceRef
		wantRest     []ResourceRef
	}{
		{
			name:    "one listener named",
			typeURL: listenerType,
			raw: "Error adding/updating listener(s) default/https-module: Failed to load dynamic module: " +
				"/usr/lib/envoy/libexample.so: cannot open shared object file\n",
			resources:    all,
			wantRejected: refs("default/https-module@3"),
			wantRest:     refs("default/good@1", "default/https@2"),
		},
		{
			name:         "two listeners named",
			typeURL:      listenerType,
			raw:          "Error adding/updating listener(s) default/https: no file\ndefault/good: duplicate address\n",
			resources:    all,
			wantRejected: refs("default/good@1", "default/https@2"),
			wantRest:     refs("default/https-module@3"),
		},
		{
			name:         "a name inside another name is not a match",
			typeURL:      listenerType,
			raw:          "Error adding/updating listener(s) default/https: no file\n",
			resources:    refs("https@1", "default/https@2"),
			wantRejected: refs("default/https@2"),
			wantRest:     refs("https@1"),
		},
		{
			name:         "clusters are separated by commas",
			typeURL:      clusterType,
			raw:          "Error adding/updating cluster(s) c1: bad lb policy, c3: no endpoints",
			resources:    refs("c1@1", "c2@2", "c3@3"),
			wantRejected: refs("c1@1", "c3@3"),
			wantRest:     refs("c2@2"),
		},
		{
			name:         "text cut by Envoy may miss names",
			typeURL:      listenerType,
			raw:          "Error adding/updating listener(s) default/https: unable to open file...(truncated)",
			resources:    all,
			wantRejected: all,
		},
		{
			name:         "message dump: nothing in the response was applied",
			typeURL:      listenerType,
			raw:          "name: \"default/https\"\n: Proto constraint validation failed (value is required)",
			resources:    all,
			wantRejected: all,
		},
		{
			name:         "no resource of the response is named",
			typeURL:      listenerType,
			raw:          "Error adding/updating listener(s) other/listener: unable to open file\n",
			resources:    all,
			wantRejected: all,
		},
		{
			name:         "routes are never named",
			typeURL:      routeType,
			raw:          "Error adding/updating listener(s) r1: looks like a listener error\n",
			resources:    refs("r1@1", "r2@2"),
			wantRejected: refs("r1@1", "r2@2"),
		},
		{
			name:    "response without resources",
			typeURL: listenerType,
			raw:     "Error adding/updating listener(s) default/https: unable to open file\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rejected, rest := splitRejected(tc.typeURL, tc.raw, tc.resources)

			assert.ElementsMatch(t, tc.wantRejected, rejected)
			assert.ElementsMatch(t, tc.wantRest, rest)
		})
	}
}
