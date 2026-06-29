package data

import "testing"

func TestServerNodeFirstBoolReadsProxyProtocolAliases(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]interface{}
	}{
		{
			name:   "snake",
			values: map[string]interface{}{"proxy_protocol": true},
		},
		{
			name:   "camel",
			values: map[string]interface{}{"proxyProtocol": true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !serverNodeFirstBool(tc.values, "proxy_protocol", "proxyProtocol") {
				t.Fatalf("serverNodeFirstBool(%v) = false, want true", tc.values)
			}
		})
	}
}
