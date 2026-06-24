package subscription

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func TestBuildOmnxtProtocolLinksIncludesOnlySimnetClientBackpressure(t *testing.T) {
	proxies := []map[string]interface{}{
		{
			"Type":                                 "simnet",
			"Name":                                 "simnet-node",
			"Server":                               "edge.example.com",
			"Port":                                 443,
			"SimnetPath":                           "/simnet/session",
			"SimnetCarrier":                        "h2",
			"SimnetClientMaxConcurrentStreams":     48,
			"SimnetClientMaxStreamsPerSession":     768,
			"SimnetClientSessionIdleTimeoutSecs":   120,
			"SimnetClientMaxUDPSessions":           24,
			"SimnetInboundMaxStreamsPerSession":    128,
			"SimnetInboundMaxUDPStreamsPerSession": 64,
			"SimnetTargetDialTimeoutMs":            12000,
			"SimnetSendWindow":                     4 * 1024 * 1024,
		},
		{
			"Type":   "vless",
			"Name":   "vless-node",
			"Server": "vless.example.com",
			"Port":   8443,
		},
	}

	links := buildOmnxtProtocolLinks(proxies, UserInfo{SubscribeID: 7, Password: "user-secret"}, map[string]string{})
	if len(links) != 1 {
		t.Fatalf("expected exactly one simnet link, got %d: %#v", len(links), links)
	}

	values := decodeSimnetLinkPayload(t, links[0])
	assertQueryValue(t, values, "simnet_client_max_concurrent_streams", "48")
	assertQueryValue(t, values, "simnet_client_max_streams_per_session", "768")
	assertQueryValue(t, values, "simnet_client_session_idle_timeout_secs", "120")
	assertQueryValue(t, values, "simnet_client_max_udp_sessions", "24")

	for _, nodeOnlyKey := range []string{
		"simnet_inbound_max_streams_per_session",
		"simnet_inbound_max_udp_streams_per_session",
		"simnet_target_dial_timeout_ms",
		"simnet_send_window",
	} {
		if values.Has(nodeOnlyKey) {
			t.Fatalf("node-only key %q leaked into client subscription payload: %s", nodeOnlyKey, values.Encode())
		}
	}
}

func TestBuildOmnxtProtocolLinksDefaultsSimnetClientBackpressure(t *testing.T) {
	links := buildOmnxtProtocolLinks([]map[string]interface{}{
		{
			"Type":   "simnet",
			"Name":   "simnet-node",
			"Server": "edge.example.com",
			"Port":   443,
		},
	}, UserInfo{SubscribeID: 8, Password: "user-secret"}, map[string]string{})
	if len(links) != 1 {
		t.Fatalf("expected one simnet link, got %d", len(links))
	}

	values := decodeSimnetLinkPayload(t, links[0])
	assertQueryValue(t, values, "simnet_client_max_concurrent_streams", "32")
	assertQueryValue(t, values, "simnet_client_max_streams_per_session", "512")
	assertQueryValue(t, values, "simnet_client_session_idle_timeout_secs", "90")
	assertQueryValue(t, values, "simnet_client_max_udp_sessions", "64")
}

func decodeSimnetLinkPayload(t *testing.T, link string) url.Values {
	t.Helper()
	const prefix = "simnet://"
	if !strings.HasPrefix(link, prefix) {
		t.Fatalf("unexpected link scheme: %s", link)
	}
	encoded := strings.TrimPrefix(link, prefix)
	if hash := strings.IndexByte(encoded, '#'); hash >= 0 {
		encoded = encoded[:hash]
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	values, err := url.ParseQuery(string(raw))
	if err != nil {
		t.Fatalf("parse payload query %q: %v", raw, err)
	}
	return values
}

func assertQueryValue(t *testing.T, values url.Values, key, want string) {
	t.Helper()
	if got := values.Get(key); got != want {
		t.Fatalf("%s = %q, want %q (payload=%s)", key, got, want, values.Encode())
	}
}
