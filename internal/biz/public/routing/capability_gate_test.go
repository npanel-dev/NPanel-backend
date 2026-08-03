package routing

import (
	"encoding/json"
	"testing"
)

func TestApplyClientCapabilitiesNormalizesRequiredSequences(t *testing.T) {
	envelope := Envelope{
		Outbounds:      []RouteOutbound{{Tag: "proxy:default"}},
		DNSResolvers:   []DNSResolver{{Tag: "dns:system"}},
		UnlockServices: []UnlockService{{Code: "openai"}},
	}

	normalized := ApplyClientCapabilities(envelope, nil)

	if normalized.CapabilityRequirements.RequiredFeatures == nil ||
		normalized.CapabilityRequirements.OptionalFeatures == nil ||
		normalized.CapabilityRequirements.NodeRequiredFeatures == nil {
		t.Fatal("capability requirement sequences must be non-nil")
	}
	if normalized.DNSResolvers[0].Bootstrap == nil {
		t.Fatal("dns_resolvers[].bootstrap must be non-nil")
	}
	if normalized.Outbounds[0].ServiceTags == nil || normalized.Outbounds[0].FallbackPoolTags == nil {
		t.Fatal("outbound sequences must be non-nil")
	}
	if normalized.UnlockServices[0].Regions == nil {
		t.Fatal("unlock_services[].regions must be non-nil")
	}
	if normalized.Rules == nil || normalized.HealthSnapshot.Outbounds == nil ||
		normalized.HealthSnapshot.DNSResolvers == nil || normalized.HealthSnapshot.Services == nil ||
		normalized.Compat.LegacyDNS == nil || normalized.Compat.LegacyOutbound == nil {
		t.Fatal("top-level client sequences must be non-nil")
	}
	if normalized.RoutingHash == "" || normalized.RoutingHash != StableHash(normalized) {
		t.Fatalf("routing hash was not recalculated after normalization: %q", normalized.RoutingHash)
	}

	payload, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("marshal normalized envelope: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode normalized envelope: %v", err)
	}
	for _, key := range []string{"dns_resolvers", "outbounds", "unlock_services", "rules"} {
		if _, ok := decoded[key].([]any); !ok {
			t.Fatalf("%s = %#v, want JSON array", key, decoded[key])
		}
	}
}

func TestApplyClientCapabilitiesNormalizesPrunedSequences(t *testing.T) {
	envelope := Envelope{
		CapabilityRequirements: CapabilityRequirements{
			RequiredFeatures: []string{"route_outbound", "external_wireguard"},
		},
		Outbounds: []RouteOutbound{{
			Tag:              "external:wireguard",
			Type:             "external",
			External:         &ExternalOutbound{Protocol: "wireguard"},
			FallbackPoolTags: []string{"proxy:default"},
		}},
	}

	normalized := ApplyClientCapabilities(envelope, []string{"route_outbound"})

	if normalized.Outbounds == nil {
		t.Fatal("pruned outbounds must encode as [] instead of null")
	}
	if len(normalized.Outbounds) != 0 {
		t.Fatalf("Outbounds = %#v, want unsupported outbounds pruned", normalized.Outbounds)
	}
	if normalized.Mode != ModeObserve {
		t.Fatalf("Mode = %q, want observe", normalized.Mode)
	}
	if normalized.RoutingHash != StableHash(normalized) {
		t.Fatal("routing hash must describe the normalized response")
	}
}
