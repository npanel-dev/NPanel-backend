package routing

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	publicrouting "github.com/npanel-dev/NPanel-backend/internal/biz/public/routing"
)

type fakeRoutingRepo struct {
	profiles      []*RouteProfile
	rules         []*RouteRule
	healthReports []*RoutingHealthReport
	routeEvents   []*RoutingRouteEvent
	grayReleases  []*RoutingGrayRelease
	tokenScope    ScopeContext
}

func (r fakeRoutingRepo) SaveProfile(context.Context, *RouteProfile) (*RouteProfile, error) {
	panic("not used")
}
func (r fakeRoutingRepo) UpdateProfile(context.Context, *RouteProfile) (*RouteProfile, error) {
	panic("not used")
}
func (r fakeRoutingRepo) FindProfileByID(context.Context, int64) (*RouteProfile, error) {
	panic("not used")
}
func (r fakeRoutingRepo) ListProfiles(context.Context, int, int, string, *bool) ([]*RouteProfile, int32, error) {
	return r.profiles, int32(len(r.profiles)), nil
}
func (r fakeRoutingRepo) DeleteProfile(context.Context, int64) error { panic("not used") }
func (r fakeRoutingRepo) SaveRule(context.Context, *RouteRule) (*RouteRule, error) {
	panic("not used")
}
func (r fakeRoutingRepo) UpdateRule(context.Context, *RouteRule) (*RouteRule, error) {
	panic("not used")
}
func (r fakeRoutingRepo) FindRuleByID(context.Context, int64) (*RouteRule, error) {
	panic("not used")
}
func (r fakeRoutingRepo) ListRules(context.Context, int, int, int64, string, *bool) ([]*RouteRule, int32, error) {
	return r.rules, int32(len(r.rules)), nil
}
func (r fakeRoutingRepo) DeleteRule(context.Context, int64) error { panic("not used") }
func (r fakeRoutingRepo) SaveDNSResolver(context.Context, *DNSResolver) (*DNSResolver, error) {
	panic("not used")
}
func (r fakeRoutingRepo) UpdateDNSResolver(context.Context, *DNSResolver) (*DNSResolver, error) {
	panic("not used")
}
func (r fakeRoutingRepo) FindDNSResolverByID(context.Context, int64) (*DNSResolver, error) {
	panic("not used")
}
func (r fakeRoutingRepo) ListDNSResolvers(context.Context, int, int, string, *bool) ([]*DNSResolver, int32, error) {
	return nil, 0, nil
}
func (r fakeRoutingRepo) DeleteDNSResolver(context.Context, int64) error { panic("not used") }
func (r fakeRoutingRepo) SaveOutbound(context.Context, *RouteOutbound) (*RouteOutbound, error) {
	panic("not used")
}
func (r fakeRoutingRepo) UpdateOutbound(context.Context, *RouteOutbound) (*RouteOutbound, error) {
	panic("not used")
}
func (r fakeRoutingRepo) FindOutboundByID(context.Context, int64) (*RouteOutbound, error) {
	panic("not used")
}
func (r fakeRoutingRepo) ListOutbounds(context.Context, int, int, string, *bool) ([]*RouteOutbound, int32, error) {
	return nil, 0, nil
}
func (r fakeRoutingRepo) DeleteOutbound(context.Context, int64) error { panic("not used") }
func (r fakeRoutingRepo) SaveUnlockService(context.Context, *UnlockService) (*UnlockService, error) {
	panic("not used")
}
func (r fakeRoutingRepo) UpdateUnlockService(context.Context, *UnlockService) (*UnlockService, error) {
	panic("not used")
}
func (r fakeRoutingRepo) FindUnlockServiceByID(context.Context, int64) (*UnlockService, error) {
	panic("not used")
}
func (r fakeRoutingRepo) ListUnlockServices(context.Context, int, int, string, *bool) ([]*UnlockService, int32, error) {
	return nil, 0, nil
}
func (r fakeRoutingRepo) DeleteUnlockService(context.Context, int64) error { panic("not used") }
func (r fakeRoutingRepo) ResolveScopeBySubscribeToken(context.Context, string) (ScopeContext, error) {
	return r.tokenScope, nil
}
func (r fakeRoutingRepo) SaveHealthReports(context.Context, []*RoutingHealthReport) error {
	return nil
}
func (r fakeRoutingRepo) ListHealthReports(context.Context, int, int, string, string, string) ([]*RoutingHealthReport, int32, error) {
	return r.healthReports, int32(len(r.healthReports)), nil
}
func (r fakeRoutingRepo) SaveRouteEvents(context.Context, []*RoutingRouteEvent) error {
	return nil
}
func (r fakeRoutingRepo) ListRouteEvents(context.Context, int, int, string, string, string) ([]*RoutingRouteEvent, int32, error) {
	return r.routeEvents, int32(len(r.routeEvents)), nil
}
func (r fakeRoutingRepo) SaveGrayRelease(_ context.Context, item *RoutingGrayRelease) (*RoutingGrayRelease, error) {
	item.ID = 1
	return item, nil
}
func (r fakeRoutingRepo) UpdateGrayRelease(_ context.Context, item *RoutingGrayRelease) (*RoutingGrayRelease, error) {
	return item, nil
}
func (r fakeRoutingRepo) FindGrayReleaseByID(_ context.Context, id int64) (*RoutingGrayRelease, error) {
	for _, item := range r.grayReleases {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, nil
}
func (r fakeRoutingRepo) ListGrayReleases(context.Context, int, int, string, string) ([]*RoutingGrayRelease, int32, error) {
	return r.grayReleases, int32(len(r.grayReleases)), nil
}
func (r fakeRoutingRepo) DeleteGrayRelease(context.Context, int64) error { return nil }

func TestRecordRouteEventAcceptsRouteDecision(t *testing.T) {
	uc := NewRoutingUsecase(fakeRoutingRepo{}, log.DefaultLogger)

	err := uc.RecordRouteEvent(context.Background(), publicrouting.RouteEventRequest{
		ReporterType: "client",
		ReporterID:   "device-1",
		ProfileCode:  "user_profile",
		RoutingHash:  "hash-1",
		Events: []publicrouting.RouteEventItem{
			{
				EventType:   "route_decision",
				Subject:     "openai.com",
				RuleID:      "rule_openai",
				ActionType:  "outbound",
				OutboundTag: "unlock:openai:us",
				Status:      "matched",
			},
		},
	})
	if err != nil {
		t.Fatalf("RecordRouteEvent() error = %v", err)
	}
}

func TestActGrayReleaseAdvanceIncrementsBatch(t *testing.T) {
	uc := NewRoutingUsecase(fakeRoutingRepo{
		grayReleases: []*RoutingGrayRelease{
			{
				ID:            1,
				ProfileCode:   "p_user_1",
				Name:          "user 1 gray",
				Status:        "draft",
				TargetType:    "user",
				TargetIDsJSON: `[1]`,
				ReleaseJSON:   `{}`,
			},
		},
	}, log.DefaultLogger)

	release, err := uc.ActGrayRelease(context.Background(), 1, "advance", "admin", "")
	if err != nil {
		t.Fatalf("ActGrayRelease() error = %v", err)
	}
	if release.Status != "running" {
		t.Fatalf("Status = %q, want running", release.Status)
	}
	if release.BatchNo != 1 {
		t.Fatalf("BatchNo = %d, want 1", release.BatchNo)
	}
	if release.StartedAt.IsZero() {
		t.Fatal("StartedAt is zero, want action timestamp")
	}
}

func TestRoutingAnalyticsAggregatesFallbackAndHealthFailures(t *testing.T) {
	now := time.Now()
	uc := NewRoutingUsecase(fakeRoutingRepo{
		routeEvents: []*RoutingRouteEvent{
			{
				ReporterID:  "device-1",
				ProfileCode: "p_user_1",
				RoutingHash: "hash-1",
				EventType:   "route_decision",
				Status:      "matched",
				EventAt:     now.Add(-time.Minute),
			},
			{
				ReporterID:  "device-1",
				ProfileCode: "p_user_1",
				RoutingHash: "hash-1",
				EventType:   "route_fallback",
				Status:      "fallback",
				Error:       "outbound failed",
				EventAt:     now.Add(-30 * time.Second),
			},
		},
		healthReports: []*RoutingHealthReport{
			{
				ReporterID:  "device-1",
				ProfileCode: "p_user_1",
				RoutingHash: "hash-1",
				SubjectType: "dns_resolver",
				SubjectKey:  "dns:cloudflare-doh",
				Status:      "failed",
				LastError:   "dns timeout",
				CheckedAt:   now.Add(-20 * time.Second),
			},
		},
	}, log.DefaultLogger)

	analytics, err := uc.Analytics(context.Background(), "p_user_1", "hash-1", 60)
	if err != nil {
		t.Fatalf("Analytics() error = %v", err)
	}
	if analytics.TotalRouteEvents != 2 {
		t.Fatalf("TotalRouteEvents = %d, want 2", analytics.TotalRouteEvents)
	}
	if analytics.FallbackRateBP != 10_000 {
		t.Fatalf("FallbackRateBP = %d, want 10000", analytics.FallbackRateBP)
	}
	if analytics.DNSFailRateBP != 10_000 {
		t.Fatalf("DNSFailRateBP = %d, want 10000", analytics.DNSFailRateBP)
	}
	if len(analytics.TopErrors) == 0 {
		t.Fatal("TopErrors is empty, want aggregated errors")
	}
}

func TestReleaseGateBlocksGlobalProfileAndHighFallback(t *testing.T) {
	now := time.Now()
	uc := NewRoutingUsecase(fakeRoutingRepo{
		profiles: []*RouteProfile{
			{
				ID:          1,
				Code:        "global_profile",
				Name:        "Global Profile",
				ScopeType:   "global",
				ScopeID:     "default",
				Enabled:     true,
				ProfileJSON: `{}`,
			},
		},
		grayReleases: []*RoutingGrayRelease{
			{
				ID:            1,
				ProfileCode:   "global_profile",
				Name:          "global gray",
				Status:        "running",
				BatchNo:       1,
				TargetType:    "user",
				TargetIDsJSON: `[1]`,
				ReleaseJSON:   `{}`,
			},
		},
		routeEvents: []*RoutingRouteEvent{
			{
				ReporterID:  "device-1",
				ProfileCode: "global_profile",
				RoutingHash: "hash-1",
				EventType:   "route_decision",
				Status:      "matched",
				EventAt:     now.Add(-time.Minute),
			},
			{
				ReporterID:  "device-1",
				ProfileCode: "global_profile",
				RoutingHash: "hash-1",
				EventType:   "route_fallback",
				Status:      "fallback",
				EventAt:     now.Add(-30 * time.Second),
			},
		},
	}, log.DefaultLogger)

	gate, err := uc.ReleaseGate(context.Background(), "global_profile", "hash-1", 60)
	if err != nil {
		t.Fatalf("ReleaseGate() error = %v", err)
	}
	if gate.Allowed {
		t.Fatal("Allowed = true, want blocked for global/high fallback")
	}
	if !hasGateCheck(gate.Checks, "profile_not_global", false) {
		t.Fatal("profile_not_global check did not block")
	}
	if !hasGateCheck(gate.Checks, "fallback_rate_ok", false) {
		t.Fatal("fallback_rate_ok check did not block")
	}
}

func TestReleaseGateUsesConfigurableThresholds(t *testing.T) {
	now := time.Now()
	uc := NewRoutingUsecase(fakeRoutingRepo{
		profiles: []*RouteProfile{
			{
				ID:          1,
				Code:        "p_user_1",
				Name:        "User Profile",
				ScopeType:   "user",
				ScopeID:     "1",
				Enabled:     true,
				ProfileJSON: `{}`,
			},
		},
		grayReleases: []*RoutingGrayRelease{
			{
				ID:            1,
				ProfileCode:   "p_user_1",
				Name:          "user gray",
				Status:        "running",
				BatchNo:       1,
				TargetType:    "user",
				TargetIDsJSON: `[1]`,
				ReleaseJSON:   `{}`,
			},
		},
		routeEvents: []*RoutingRouteEvent{
			{ReporterID: "device-1", ProfileCode: "p_user_1", RoutingHash: "hash-1", EventType: "route_decision", Status: "matched", EventAt: now.Add(-time.Minute)},
			{ReporterID: "device-1", ProfileCode: "p_user_1", RoutingHash: "hash-1", EventType: "route_fallback", Status: "fallback", EventAt: now.Add(-30 * time.Second)},
		},
		healthReports: []*RoutingHealthReport{
			{ReporterID: "device-1", ProfileCode: "p_user_1", RoutingHash: "hash-1", SubjectType: "outbound", SubjectKey: "unlock:openai:us", Status: "healthy", CheckedAt: now.Add(-20 * time.Second)},
		},
	}, log.DefaultLogger)

	blocked, err := uc.ReleaseGate(context.Background(), "p_user_1", "hash-1", 60)
	if err != nil {
		t.Fatalf("ReleaseGate() error = %v", err)
	}
	if blocked.Allowed {
		t.Fatal("ReleaseGate() allowed with default fallback threshold, want blocked")
	}

	allowed, err := uc.ReleaseGate(context.Background(), "p_user_1", "hash-1", 60, RoutingReleaseThresholds{
		FallbackRateBP:     10_000,
		DNSFailRateBP:      500,
		OutboundFailRateBP: 500,
		MinRouteEvents:     1,
		MinHealthReports:   1,
	})
	if err != nil {
		t.Fatalf("ReleaseGate() with thresholds error = %v", err)
	}
	if !allowed.Allowed {
		t.Fatalf("ReleaseGate() blocked with relaxed threshold: %+v", allowed.Checks)
	}
}

func TestSnapshotReleaseAuditPersistsReleaseJSON(t *testing.T) {
	now := time.Now()
	release := &RoutingGrayRelease{
		ID:            1,
		ProfileCode:   "p_user_1",
		Name:          "user gray",
		Status:        "running",
		BatchNo:       1,
		TargetType:    "user",
		TargetIDsJSON: `[1]`,
		ReleaseJSON:   `{}`,
	}
	uc := NewRoutingUsecase(fakeRoutingRepo{
		profiles: []*RouteProfile{
			{ID: 1, Code: "p_user_1", Name: "User Profile", ScopeType: "user", ScopeID: "1", Enabled: true, ProfileJSON: `{}`},
		},
		grayReleases: []*RoutingGrayRelease{release},
		routeEvents: []*RoutingRouteEvent{
			{ReporterID: "device-1", ProfileCode: "p_user_1", RoutingHash: "hash-1", EventType: "route_decision", Status: "matched", EventAt: now.Add(-time.Minute)},
		},
		healthReports: []*RoutingHealthReport{
			{ReporterID: "device-1", ProfileCode: "p_user_1", RoutingHash: "hash-1", SubjectType: "outbound", SubjectKey: "unlock:openai:us", Status: "healthy", CheckedAt: now.Add(-20 * time.Second)},
		},
	}, log.DefaultLogger)

	snapshot, err := uc.SnapshotReleaseAudit(context.Background(), 1, "p_user_1", "hash-1", 60, "admin", RoutingReleaseThresholds{})
	if err != nil {
		t.Fatalf("SnapshotReleaseAudit() error = %v", err)
	}
	if snapshot.ID == "" {
		t.Fatal("SnapshotReleaseAudit() snapshot ID is empty")
	}
	if !strings.Contains(release.ReleaseJSON, "audit_snapshots") {
		t.Fatalf("ReleaseJSON = %s, want audit_snapshots", release.ReleaseJSON)
	}
	if !strings.Contains(release.ReleaseJSON, "thresholds") {
		t.Fatalf("ReleaseJSON = %s, want thresholds", release.ReleaseJSON)
	}
}

func TestCapabilityMatrixKeepsNonPpanelOutOfEnforce(t *testing.T) {
	uc := NewRoutingUsecase(fakeRoutingRepo{}, log.DefaultLogger)
	matrix := uc.CapabilityMatrix(context.Background())
	foundLegacyPanel := false
	for _, item := range matrix.Items {
		if item.Panel == "xboard/xiaov2board/v2board/sspanel" {
			foundLegacyPanel = true
			if item.EnforceCandidate {
				t.Fatal("non-ppanel matrix item is enforce candidate")
			}
		}
	}
	if !foundLegacyPanel {
		t.Fatal("non-ppanel capability matrix item not found")
	}
}

func hasGateCheck(checks []RoutingReleaseGateCheck, key string, passed bool) bool {
	for _, check := range checks {
		if check.Key == key && check.Passed == passed {
			return true
		}
	}
	return false
}

func TestBuildConfigFallsBackToFixtureWhenStoreIsEmpty(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	uc := NewRoutingUsecase(fakeRoutingRepo{}, log.DefaultLogger)

	cfg, err := uc.BuildConfig(context.Background(), now)
	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	if cfg.Profile.Code != "p0_default_smart" {
		t.Fatalf("Profile.Code = %q, want p0_default_smart", cfg.Profile.Code)
	}
}

func TestBuildConfigKeepsPreviewDefaultsWhenProfileHasNoResources(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	uc := NewRoutingUsecase(fakeRoutingRepo{
		profiles: []*RouteProfile{
			{
				ID:          1,
				Code:        "db_profile",
				Name:        "DB Profile",
				ScopeType:   "global",
				ScopeID:     "default",
				Mode:        publicrouting.ModeObserve,
				Enabled:     true,
				ProfileJSON: `{"default_action":{"type":"proxy"},"default_dns_resolver_tag":"dns:cloudflare-doh","default_fallback_policy":"fallback_default"}`,
			},
		},
	}, log.DefaultLogger)

	cfg, err := uc.BuildConfig(context.Background(), now)
	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	if cfg.Profile.Code != "db_profile" {
		t.Fatalf("Profile.Code = %q, want db_profile", cfg.Profile.Code)
	}
	if len(cfg.DNSResolvers) == 0 {
		t.Fatal("DNSResolvers is empty, want preview defaults")
	}
	if len(cfg.Outbounds) == 0 {
		t.Fatal("Outbounds is empty, want preview defaults")
	}
}

func TestBuildConfigSelectsUserProfileBeforeGlobal(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	uc := NewRoutingUsecase(fakeRoutingRepo{
		profiles: []*RouteProfile{
			{
				ID:          1,
				Code:        "global_profile",
				Name:        "Global Profile",
				ScopeType:   "global",
				ScopeID:     "default",
				Mode:        publicrouting.ModeObserve,
				Enabled:     true,
				ProfileJSON: `{"default_action":{"type":"proxy"},"default_dns_resolver_tag":"dns:system","default_fallback_policy":"fallback_default"}`,
			},
			{
				ID:          2,
				Code:        "user_profile",
				Name:        "User Profile",
				ScopeType:   "user",
				ScopeID:     "10001",
				Mode:        publicrouting.ModeObserve,
				Enabled:     true,
				ProfileJSON: `{"default_action":{"type":"proxy"},"default_dns_resolver_tag":"dns:system","default_fallback_policy":"fallback_default"}`,
			},
		},
	}, log.DefaultLogger)

	cfg, err := uc.BuildConfig(context.Background(), now, publicrouting.ConfigOptions{UserID: 10001})
	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	if cfg.Profile.Code != "user_profile" {
		t.Fatalf("Profile.Code = %q, want user_profile", cfg.Profile.Code)
	}
}

func TestBuildConfigResolveScopeFromSubscribeToken(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	uc := NewRoutingUsecase(fakeRoutingRepo{
		tokenScope: ScopeContext{UserID: 10001, SubscribeID: 7, UserSubscribeID: 88},
		profiles: []*RouteProfile{
			{
				ID:          1,
				Code:        "global_profile",
				Name:        "Global Profile",
				ScopeType:   "global",
				ScopeID:     "default",
				Mode:        publicrouting.ModeObserve,
				Enabled:     true,
				ProfileJSON: `{"default_action":{"type":"proxy"},"default_dns_resolver_tag":"dns:system","default_fallback_policy":"fallback_default"}`,
			},
			{
				ID:          2,
				Code:        "subscription_instance_profile",
				Name:        "Subscription Instance Profile",
				ScopeType:   "user_subscribe",
				ScopeID:     "88",
				Mode:        publicrouting.ModeObserve,
				Enabled:     true,
				ProfileJSON: `{"default_action":{"type":"proxy"},"default_dns_resolver_tag":"dns:system","default_fallback_policy":"fallback_default"}`,
			},
		},
	}, log.DefaultLogger)

	cfg, err := uc.BuildConfig(context.Background(), now, publicrouting.ConfigOptions{SubscribeToken: "sub-token"})
	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	if cfg.Profile.Code != "subscription_instance_profile" {
		t.Fatalf("Profile.Code = %q, want subscription_instance_profile", cfg.Profile.Code)
	}
}

func TestBuildConfigMergesFreshHealthReports(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	uc := NewRoutingUsecase(fakeRoutingRepo{
		profiles: []*RouteProfile{
			{
				ID:          1,
				Code:        "db_profile",
				Name:        "DB Profile",
				ScopeType:   "user",
				ScopeID:     "10001",
				Mode:        "enforce",
				Enabled:     true,
				ProfileJSON: `{"default_action":{"type":"proxy"},"default_dns_resolver_tag":"dns:system","default_fallback_policy":"fallback_default"}`,
			},
		},
		healthReports: []*RoutingHealthReport{
			{
				SubjectType: "outbound",
				SubjectKey:  "unlock:openai:us",
				Status:      "healthy",
				Source:      "client_health_report",
				RTTMS:       42,
				CheckedAt:   now.Add(-time.Minute),
			},
			{
				SubjectType: "dns_resolver",
				SubjectKey:  "dns:cloudflare-doh",
				Status:      "healthy",
				Source:      "client_health_report",
				CheckedAt:   now.Add(-time.Minute),
			},
			{
				SubjectType: "service",
				SubjectKey:  "openai",
				Status:      "healthy",
				Source:      "client_health_report",
				CheckedAt:   now.Add(-time.Minute),
			},
		},
	}, log.DefaultLogger)

	cfg, err := uc.BuildConfig(context.Background(), now, publicrouting.ConfigOptions{UserID: 10001})
	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	result := publicrouting.PreviewRouteConfig(cfg, publicrouting.PreviewRequest{
		Domain:            "example.com",
		SupportedFeatures: []string{"route_outbound", "route_dns_resolver", "doh"},
	})
	if !result.ExecutionEnabled {
		t.Fatal("ExecutionEnabled = false, want true for enforce gray scope with healthy reports")
	}
	if cfg.HealthSnapshot.Outbounds[0].Status != "healthy" {
		t.Fatalf("outbound status = %q, want healthy", cfg.HealthSnapshot.Outbounds[0].Status)
	}
}

func TestBuildConfigDoesNotLeakUserProfileWithoutScope(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	uc := NewRoutingUsecase(fakeRoutingRepo{
		profiles: []*RouteProfile{
			{
				ID:          1,
				Code:        "user_profile",
				Name:        "User Profile",
				ScopeType:   "user",
				ScopeID:     "10001",
				Mode:        publicrouting.ModeObserve,
				Enabled:     true,
				ProfileJSON: `{"default_action":{"type":"proxy"},"default_dns_resolver_tag":"dns:system","default_fallback_policy":"fallback_default"}`,
			},
		},
	}, log.DefaultLogger)

	cfg, err := uc.BuildConfig(context.Background(), now)
	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	if cfg.Profile.Code != "p0_default_smart" {
		t.Fatalf("Profile.Code = %q, want p0_default_smart", cfg.Profile.Code)
	}
}

func TestBuildConfigFallsBackWhenRuleReferencesMissingOutbound(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	uc := NewRoutingUsecase(fakeRoutingRepo{
		profiles: []*RouteProfile{
			{
				ID:          1,
				Code:        "db_profile",
				Name:        "DB Profile",
				ScopeType:   "global",
				ScopeID:     "default",
				Mode:        publicrouting.ModeObserve,
				Enabled:     true,
				ProfileJSON: `{"default_action":{"type":"proxy"},"default_dns_resolver_tag":"dns:system","default_fallback_policy":"fallback_default"}`,
			},
		},
		rules: []*RouteRule{
			{
				ID:          1,
				ProfileID:   1,
				Name:        "Broken outbound",
				Priority:    100,
				Enabled:     true,
				MatcherJSON: `{"type":"domain_suffix","value":"openai.com"}`,
				ActionJSON:  `{"type":"outbound","outbound_tag":"missing:outbound","fail_policy":"fallback_default"}`,
			},
		},
	}, log.DefaultLogger)

	cfg, err := uc.BuildConfig(context.Background(), now)
	if err == nil {
		t.Fatal("BuildConfig() error = nil, want missing outbound error")
	}
	if cfg.Profile.Code != "p0_default_smart" {
		t.Fatalf("Profile.Code = %q, want p0_default_smart", cfg.Profile.Code)
	}
}
