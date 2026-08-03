package routing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

const (
	SchemaV1    = "routing_profile.v1"
	ModeObserve = "observe"
)

func BuildPreviewConfig(now time.Time, opts ConfigOptions) Envelope {
	generatedAt := now.UTC().Format(time.RFC3339)
	envelope := Envelope{
		Schema:            SchemaV1,
		RoutingVersion:    1,
		GeneratedAt:       generatedAt,
		ExpiresAt:         now.UTC().Add(10 * time.Minute).Format(time.RFC3339),
		Mode:              ModeObserve,
		ManagedBy:         "backend",
		AllowUserOverride: false,
		Profile: Profile{
			ID:          "profile_p0_default",
			Code:        "p0_default_smart",
			Name:        "P0 默认智能分流",
			Description: "P0 read-only preview profile; it does not change live traffic.",
			Scope: ProfileScope{
				Type: "global",
				ID:   "default",
			},
			Priority: 100,
			DefaultAction: RouteAction{
				Type: "proxy",
			},
			DefaultDNSResolverTag: "dns:system",
			DefaultFallbackPolicy: "fallback_default",
			Enabled:               true,
		},
		CapabilityRequirements: CapabilityRequirements{
			MinSchema:        SchemaV1,
			MinOmnxtSDK:      "0.1.0",
			MinFlutterPlugin: "0.1.0",
			RequiredFeatures: []string{
				"routing_profile_v1",
				"route_outbound",
				"route_dns_resolver",
				"route_fail_policy",
				"route_fallback",
				"doh",
			},
			OptionalFeatures: []string{
				"route_events",
				"client_health_report",
				"node_health_report_v1",
			},
			NodeRequiredFeatures: []string{
				"node_routing_profile_v1",
				"node_health_report_v1",
			},
		},
		DNSResolvers: []DNSResolver{
			{
				Tag:        "dns:cloudflare-doh",
				Name:       "Cloudflare DoH",
				Proto:      "doh",
				Address:    "https://cloudflare-dns.com/dns-query",
				Port:       443,
				Path:       "/dns-query",
				ServerName: "cloudflare-dns.com",
				Bootstrap:  []string{"1.1.1.1", "1.0.0.1"},
				Detour: ResolverDetour{
					Type: "proxy",
				},
				TimeoutMS:       3000,
				Retry:           2,
				CacheTTLSeconds: 300,
				HealthCheck: HealthCheckSpec{
					Enabled:         true,
					Domain:          "www.cloudflare.com",
					IntervalSeconds: 60,
				},
				Enabled: true,
			},
		},
		Outbounds: []RouteOutbound{
			{
				Tag:             "unlock:openai:us",
				Name:            "OpenAI US Preview",
				Type:            "node_group",
				NodeGroupID:     "preview",
				Region:          "US",
				ServiceTags:     []string{"openai"},
				SelectionPolicy: "health_first",
				FailPolicy:      "fallback_default",
				FallbackPoolTags: []string{
					"proxy:default",
				},
				HealthCheck: HealthCheckSpec{
					Enabled:         true,
					URL:             "https://chat.openai.com/cdn-cgi/trace",
					IntervalSeconds: 60,
					TimeoutMS:       5000,
				},
				Enabled: true,
			},
		},
		UnlockServices: []UnlockService{
			{
				Code:                  "openai",
				Name:                  "OpenAI",
				Category:              "ai",
				Regions:               []string{"US"},
				DefaultRegion:         "US",
				DefaultOutboundTag:    "unlock:openai:us",
				DefaultDNSResolverTag: "dns:cloudflare-doh",
				DefaultFailPolicy:     "fallback_default",
				HealthCheckURL:        "https://chat.openai.com/cdn-cgi/trace",
				Enabled:               true,
			},
		},
		Rules: []Rule{
			{
				ID:          "rule_openai_suffix",
				Name:        "OpenAI preview route",
				Priority:    200,
				Enabled:     true,
				ServiceCode: "openai",
				Matcher: Matcher{
					Type:  "domain_suffix",
					Value: "openai.com",
				},
				Action: RouteAction{
					Type:           "outbound",
					OutboundTag:    "unlock:openai:us",
					DNSResolverTag: "dns:cloudflare-doh",
					FailPolicy:     "fallback_default",
				},
			},
		},
		HealthSnapshot: HealthSnapshot{
			GeneratedAt: generatedAt,
			Outbounds: []HealthStatus{
				{
					Tag:       "unlock:openai:us",
					Status:    "unknown",
					Source:    "backend_global",
					CheckedAt: generatedAt,
				},
			},
			DNSResolvers: []HealthStatus{
				{
					Tag:       "dns:cloudflare-doh",
					Status:    "unknown",
					Source:    "backend_global",
					CheckedAt: generatedAt,
				},
			},
			Services: []HealthStatus{
				{
					Code:           "openai",
					Region:         "US",
					Status:         "unknown",
					Source:         "backend_global",
					CheckedAt:      generatedAt,
					OutboundTag:    "unlock:openai:us",
					DNSResolverTag: "dns:cloudflare-doh",
				},
			},
		},
		Compat: Compat{
			LegacyDNS:      []string{},
			LegacyOutbound: []string{},
		},
	}
	envelope.RoutingHash = StableHash(envelope)
	return envelope
}

func PreviewRouteConfig(envelope Envelope, req PreviewRequest) PreviewResult {
	unsupported := MissingRequiredFeatures(
		envelope.CapabilityRequirements.RequiredFeatures,
		req.SupportedFeatures,
	)
	result := PreviewResult{
		RoutingHash:      envelope.RoutingHash,
		Profile:          envelope.Profile,
		ScopeType:        envelope.Profile.Scope.Type,
		ScopeID:          envelope.Profile.Scope.ID,
		Action:           envelope.Profile.DefaultAction,
		DNSResolverTag:   envelope.Profile.DefaultDNSResolverTag,
		FallbackPolicy:   envelope.Profile.DefaultFallbackPolicy,
		Unsupported:      cloneRoutingSlice(unsupported),
		EffectiveMode:    envelope.Mode,
		ExecutionEnabled: envelope.Mode == "enforce" && envelope.Profile.Scope.Type != "" && envelope.Profile.Scope.Type != "global" && len(unsupported) == 0 && healthSnapshotOK(envelope.HealthSnapshot),
	}
	if result.Action.Type == "proxy" {
		result.OutboundTag = "proxy:default"
	}

	for _, rule := range envelope.Rules {
		if !rule.Enabled || !matches(rule.Matcher, req) {
			continue
		}
		matched := rule
		result.Matched = true
		result.Rule = &matched
		result.Action = rule.Action
		result.DNSResolverTag = firstNonEmpty(rule.Action.DNSResolverTag, envelope.Profile.DefaultDNSResolverTag)
		result.OutboundTag = firstNonEmpty(rule.Action.OutboundTag, "proxy:default")
		result.FallbackPolicy = firstNonEmpty(rule.Action.FailPolicy, envelope.Profile.DefaultFallbackPolicy)
		break
	}

	return result
}

func healthSnapshotOK(snapshot HealthSnapshot) bool {
	total := len(snapshot.Outbounds) + len(snapshot.DNSResolvers) + len(snapshot.Services)
	if total == 0 {
		return false
	}
	for _, item := range append(append([]HealthStatus{}, snapshot.Outbounds...), append(snapshot.DNSResolvers, snapshot.Services...)...) {
		if item.Status != "healthy" && item.Status != "ok" && item.Status != "disabled" {
			return false
		}
	}
	return true
}

func MissingRequiredFeatures(required, supported []string) []string {
	if len(required) == 0 {
		return nil
	}
	if len(supported) == 0 {
		return append([]string{}, required...)
	}
	seen := make(map[string]struct{}, len(supported))
	for _, feature := range supported {
		normalized := strings.TrimSpace(feature)
		if normalized != "" {
			seen[normalized] = struct{}{}
		}
	}
	var missing []string
	for _, feature := range required {
		if _, ok := seen[feature]; !ok {
			missing = append(missing, feature)
		}
	}
	return missing
}

func StableHash(envelope Envelope) string {
	payload := struct {
		Schema                 string                 `json:"schema"`
		RoutingVersion         int                    `json:"routing_version"`
		Mode                   string                 `json:"mode"`
		ManagedBy              string                 `json:"managed_by"`
		AllowUserOverride      bool                   `json:"allow_user_override"`
		Profile                Profile                `json:"profile"`
		CapabilityRequirements CapabilityRequirements `json:"capability_requirements"`
		DNSResolvers           []DNSResolver          `json:"dns_resolvers"`
		Outbounds              []RouteOutbound        `json:"outbounds"`
		UnlockServices         []UnlockService        `json:"unlock_services"`
		Rules                  []Rule                 `json:"rules"`
		Compat                 Compat                 `json:"compat"`
	}{
		Schema:                 envelope.Schema,
		RoutingVersion:         envelope.RoutingVersion,
		Mode:                   envelope.Mode,
		ManagedBy:              envelope.ManagedBy,
		AllowUserOverride:      envelope.AllowUserOverride,
		Profile:                envelope.Profile,
		CapabilityRequirements: envelope.CapabilityRequirements,
		DNSResolvers:           envelope.DNSResolvers,
		Outbounds:              envelope.Outbounds,
		UnlockServices:         envelope.UnlockServices,
		Rules:                  envelope.Rules,
		Compat:                 envelope.Compat,
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func ParseFeatureList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	features := make([]string, 0, len(parts))
	for _, part := range parts {
		feature := strings.TrimSpace(part)
		if feature != "" {
			features = append(features, feature)
		}
	}
	sort.Strings(features)
	return features
}

func matches(matcher Matcher, req PreviewRequest) bool {
	switch matcher.Type {
	case "domain_suffix":
		value, ok := matcher.Value.(string)
		if !ok {
			return false
		}
		domain := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(req.Domain)), ".")
		suffix := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
		return domain == suffix || strings.HasSuffix(domain, "."+suffix)
	case "domain_full":
		value, ok := matcher.Value.(string)
		return ok && strings.EqualFold(strings.TrimSpace(req.Domain), strings.TrimSpace(value))
	case "domain_keyword":
		value, ok := matcher.Value.(string)
		return ok && strings.Contains(strings.ToLower(req.Domain), strings.ToLower(value))
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
