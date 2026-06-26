package routing

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	publicrouting "github.com/npanel-dev/NPanel-backend/internal/biz/public/routing"
)

type fakeRoutingRepo struct {
	profiles []*RouteProfile
	rules    []*RouteRule
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
