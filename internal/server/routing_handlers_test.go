package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	adminroutingbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/routing"
	publicrouting "github.com/npanel-dev/NPanel-backend/internal/biz/public/routing"
	pkgmiddleware "github.com/npanel-dev/NPanel-backend/internal/pkg/middleware"
	adminroutingservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/routing"
)

type routingHandlerFakeRepo struct {
	profiles     []*adminroutingbiz.RouteProfile
	rules        []*adminroutingbiz.RouteRule
	dnsResolvers []*adminroutingbiz.DNSResolver
	outbounds    []*adminroutingbiz.RouteOutbound
	services     []*adminroutingbiz.UnlockService
}

func (r routingHandlerFakeRepo) SaveProfile(context.Context, *adminroutingbiz.RouteProfile) (*adminroutingbiz.RouteProfile, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) UpdateProfile(context.Context, *adminroutingbiz.RouteProfile) (*adminroutingbiz.RouteProfile, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) FindProfileByID(context.Context, int64) (*adminroutingbiz.RouteProfile, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) ListProfiles(context.Context, int, int, string, *bool) ([]*adminroutingbiz.RouteProfile, int32, error) {
	return r.profiles, int32(len(r.profiles)), nil
}
func (r routingHandlerFakeRepo) DeleteProfile(context.Context, int64) error { panic("not used") }
func (r routingHandlerFakeRepo) SaveRule(context.Context, *adminroutingbiz.RouteRule) (*adminroutingbiz.RouteRule, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) UpdateRule(context.Context, *adminroutingbiz.RouteRule) (*adminroutingbiz.RouteRule, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) FindRuleByID(context.Context, int64) (*adminroutingbiz.RouteRule, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) ListRules(context.Context, int, int, int64, string, *bool) ([]*adminroutingbiz.RouteRule, int32, error) {
	return r.rules, int32(len(r.rules)), nil
}
func (r routingHandlerFakeRepo) DeleteRule(context.Context, int64) error { panic("not used") }
func (r routingHandlerFakeRepo) SaveDNSResolver(context.Context, *adminroutingbiz.DNSResolver) (*adminroutingbiz.DNSResolver, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) UpdateDNSResolver(context.Context, *adminroutingbiz.DNSResolver) (*adminroutingbiz.DNSResolver, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) FindDNSResolverByID(context.Context, int64) (*adminroutingbiz.DNSResolver, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) ListDNSResolvers(context.Context, int, int, string, *bool) ([]*adminroutingbiz.DNSResolver, int32, error) {
	return r.dnsResolvers, int32(len(r.dnsResolvers)), nil
}
func (r routingHandlerFakeRepo) DeleteDNSResolver(context.Context, int64) error { panic("not used") }
func (r routingHandlerFakeRepo) SaveOutbound(context.Context, *adminroutingbiz.RouteOutbound) (*adminroutingbiz.RouteOutbound, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) UpdateOutbound(context.Context, *adminroutingbiz.RouteOutbound) (*adminroutingbiz.RouteOutbound, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) FindOutboundByID(context.Context, int64) (*adminroutingbiz.RouteOutbound, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) ListOutbounds(context.Context, int, int, string, *bool) ([]*adminroutingbiz.RouteOutbound, int32, error) {
	return r.outbounds, int32(len(r.outbounds)), nil
}
func (r routingHandlerFakeRepo) DeleteOutbound(context.Context, int64) error { panic("not used") }
func (r routingHandlerFakeRepo) SaveUnlockService(context.Context, *adminroutingbiz.UnlockService) (*adminroutingbiz.UnlockService, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) UpdateUnlockService(context.Context, *adminroutingbiz.UnlockService) (*adminroutingbiz.UnlockService, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) FindUnlockServiceByID(context.Context, int64) (*adminroutingbiz.UnlockService, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) ListUnlockServices(context.Context, int, int, string, *bool) ([]*adminroutingbiz.UnlockService, int32, error) {
	return r.services, int32(len(r.services)), nil
}
func (r routingHandlerFakeRepo) DeleteUnlockService(context.Context, int64) error { panic("not used") }
func (r routingHandlerFakeRepo) ResolveScopeBySubscribeToken(context.Context, string) (adminroutingbiz.ScopeContext, error) {
	return adminroutingbiz.ScopeContext{}, nil
}
func (r routingHandlerFakeRepo) SaveHealthReports(context.Context, []*adminroutingbiz.RoutingHealthReport) error {
	return nil
}
func (r routingHandlerFakeRepo) ListHealthReports(context.Context, int, int, string, string, string) ([]*adminroutingbiz.RoutingHealthReport, int32, error) {
	return nil, 0, nil
}
func (r routingHandlerFakeRepo) SaveRouteEvents(context.Context, []*adminroutingbiz.RoutingRouteEvent) error {
	return nil
}
func (r routingHandlerFakeRepo) ListRouteEvents(context.Context, int, int, string, string, string) ([]*adminroutingbiz.RoutingRouteEvent, int32, error) {
	return nil, 0, nil
}
func (r routingHandlerFakeRepo) SaveGrayRelease(context.Context, *adminroutingbiz.RoutingGrayRelease) (*adminroutingbiz.RoutingGrayRelease, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) UpdateGrayRelease(context.Context, *adminroutingbiz.RoutingGrayRelease) (*adminroutingbiz.RoutingGrayRelease, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) FindGrayReleaseByID(context.Context, int64) (*adminroutingbiz.RoutingGrayRelease, error) {
	panic("not used")
}
func (r routingHandlerFakeRepo) ListGrayReleases(context.Context, int, int, string, string) ([]*adminroutingbiz.RoutingGrayRelease, int32, error) {
	return nil, 0, nil
}
func (r routingHandlerFakeRepo) DeleteGrayRelease(context.Context, int64) error { panic("not used") }

func TestRoutingConfigHeadersBlockUnsupportedDot(t *testing.T) {
	handler, cache := routingConfigHandlerForTest()
	req := httptest.NewRequest(http.MethodGet, "/v1/public/routing/config?user_id=10001", nil)
	req = req.WithContext(pkgmiddleware.WithUserID(req.Context(), 10001))
	req.Header.Set("X-Routing-Features", strings.Join(routingHandlerFeatures(), ","))
	rec := httptest.NewRecorder()

	handler(handleRoutingConfig(routingServiceForTest(), cache)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Routing-Execution"); got != "blocked" {
		t.Fatalf("X-Routing-Execution = %q, want blocked", got)
	}
	if got := rec.Header().Get("X-Routing-Unsupported-Features"); !strings.Contains(got, "dot") {
		t.Fatalf("X-Routing-Unsupported-Features = %q, want dot", got)
	}
	cfg := routingConfigResponseData(t, rec)
	if cfg.Mode != publicrouting.ModeObserve {
		t.Fatalf("Mode = %q, want observe", cfg.Mode)
	}
	if len(cfg.DNSResolvers) != 0 {
		t.Fatalf("DNSResolvers = %#v, want unsupported DoT resolver pruned", cfg.DNSResolvers)
	}
}

func TestRoutingConfigCacheKeyIncludesSupportedFeatures(t *testing.T) {
	routing := routingServiceForTest()
	cache := &publicRoutingCache{}
	blockedReq := httptest.NewRequest(http.MethodGet, "/v1/public/routing/config?user_id=10001", nil)
	blockedReq = blockedReq.WithContext(pkgmiddleware.WithUserID(blockedReq.Context(), 10001))
	blockedReq.Header.Set("X-Routing-Features", strings.Join(routingHandlerFeatures(), ","))
	blockedRec := httptest.NewRecorder()

	handleRoutingConfig(routing, cache).ServeHTTP(blockedRec, blockedReq)
	if blockedRec.Header().Get("X-Routing-Execution") != "blocked" {
		t.Fatalf("first request execution = %q, want blocked", blockedRec.Header().Get("X-Routing-Execution"))
	}

	eligibleReq := httptest.NewRequest(http.MethodGet, "/v1/public/routing/config?user_id=10001", nil)
	eligibleReq = eligibleReq.WithContext(pkgmiddleware.WithUserID(eligibleReq.Context(), 10001))
	eligibleReq.Header.Set("X-Routing-Features", strings.Join(routingHandlerFeatures("dot"), ","))
	eligibleRec := httptest.NewRecorder()
	handleRoutingConfig(routing, cache).ServeHTTP(eligibleRec, eligibleReq)

	if eligibleRec.Header().Get("X-Routing-Execution") != "eligible" {
		t.Fatalf("second request execution = %q, want eligible; body=%s", eligibleRec.Header().Get("X-Routing-Execution"), eligibleRec.Body.String())
	}
	cfg := routingConfigResponseData(t, eligibleRec)
	if cfg.Mode != "enforce" {
		t.Fatalf("Mode = %q, want enforce", cfg.Mode)
	}
	if len(cfg.DNSResolvers) != 1 || cfg.DNSResolvers[0].Proto != "dot" {
		t.Fatalf("DNSResolvers = %#v, want cached response separated by feature key", cfg.DNSResolvers)
	}
}

func TestRoutingConfigIfNoneMatchUsesCapabilityGatedHash(t *testing.T) {
	routing := routingServiceForTest()
	cache := &publicRoutingCache{}
	req := httptest.NewRequest(http.MethodGet, "/v1/public/routing/config?user_id=10001", nil)
	req = req.WithContext(pkgmiddleware.WithUserID(req.Context(), 10001))
	req.Header.Set("X-Routing-Features", strings.Join(routingHandlerFeatures("dot"), ","))
	rec := httptest.NewRecorder()
	handleRoutingConfig(routing, cache).ServeHTTP(rec, req)
	cfg := routingConfigResponseData(t, rec)

	notModifiedReq := httptest.NewRequest(http.MethodGet, "/v1/public/routing/config?user_id=10001", nil)
	notModifiedReq = notModifiedReq.WithContext(pkgmiddleware.WithUserID(notModifiedReq.Context(), 10001))
	notModifiedReq.Header.Set("X-Routing-Features", strings.Join(routingHandlerFeatures("dot"), ","))
	notModifiedReq.Header.Set("If-None-Match", cfg.RoutingHash)
	notModifiedRec := httptest.NewRecorder()
	handleRoutingConfig(routing, cache).ServeHTTP(notModifiedRec, notModifiedReq)

	if notModifiedRec.Code != http.StatusNotModified {
		t.Fatalf("status = %d body=%s, want 304", notModifiedRec.Code, notModifiedRec.Body.String())
	}
	if got := notModifiedRec.Header().Get("X-Routing-Execution"); got != "eligible" {
		t.Fatalf("X-Routing-Execution = %q, want eligible", got)
	}
	if notModifiedRec.Body.Len() != 0 {
		t.Fatalf("304 body = %q, want empty", notModifiedRec.Body.String())
	}
}

func TestRoutingPreviewHeadersExposeUnsupportedFeatures(t *testing.T) {
	body := `{"domain":"openai.com","user_id":"10001","supported_features":["routing_profile_v1","route_dns_resolver","route_outbound","route_fail_policy","route_fallback","doh"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/public/routing/preview", strings.NewReader(body))
	req = req.WithContext(pkgmiddleware.WithUserID(req.Context(), 10001))
	rec := httptest.NewRecorder()

	handleRoutingPreview(routingServiceForTest(), &publicRoutingCache{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Routing-Execution"); got != "blocked" {
		t.Fatalf("X-Routing-Execution = %q, want blocked", got)
	}
	if got := rec.Header().Get("X-Routing-Unsupported-Features"); !strings.Contains(got, "dot") {
		t.Fatalf("X-Routing-Unsupported-Features = %q, want dot", got)
	}
	var resp struct {
		Code int                         `json:"code"`
		Data publicrouting.PreviewResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode preview response: %v", err)
	}
	if !containsFeature(resp.Data.Unsupported, "dot") {
		t.Fatalf("Unsupported = %#v, want dot", resp.Data.Unsupported)
	}
	if resp.Data.ExecutionEnabled {
		t.Fatal("ExecutionEnabled = true, want false for unsupported features")
	}
}

func TestRoutingAuthenticationRejectsSelfReportedScope(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/public/routing/config?user_id=10001&node_id=7", nil)
	rec := httptest.NewRecorder()
	called := false

	requireRoutingAuthentication(nil, func(http.ResponseWriter, *http.Request) {
		called = true
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s, want 401", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("unauthenticated self-reported scope reached routing handler")
	}
}

func TestRoutingAuthenticationAcceptsSubscriptionTokenHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/public/routing/config", nil)
	req.Header.Set("X-Subscribe-Token", "legacy-subscription-token")
	rec := httptest.NewRecorder()
	called := false

	requireRoutingAuthentication(nil, func(http.ResponseWriter, *http.Request) {
		called = true
	}).ServeHTTP(rec, req)

	if !called {
		t.Fatalf("subscription-token request was rejected: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRoutingConfigOptionsUseOnlyTrustedScope(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/public/routing/config?user_id=999&subscribe_id=888&node_id=777&token=query-token", nil)
	req.Header.Set("token", "header-subscription-token")
	req = req.WithContext(pkgmiddleware.WithUserID(req.Context(), 10001))

	opts := routingConfigOptionsFromRequest(req, publicrouting.PreviewRequest{
		UserID:          "999",
		SubscribeID:     "888",
		UserSubscribeID: "666",
		SubscribeToken:  "body-token",
		NodeID:          "777",
	})

	if opts.UserID != 10001 || opts.SubscribeToken != "header-subscription-token" {
		t.Fatalf("trusted scope = %#v, want authenticated user and header token", opts)
	}
	if opts.SubscribeID != 0 || opts.UserSubscribeID != 0 || opts.NodeID != 0 {
		t.Fatalf("self-reported scope leaked into options: %#v", opts)
	}
}

func routingServiceForTest() *adminroutingservice.RoutingService {
	uc := adminroutingbiz.NewRoutingUsecase(routingHandlerFakeRepo{
		profiles: []*adminroutingbiz.RouteProfile{
			{
				ID:          1,
				Code:        "dot_profile",
				Name:        "DoT Profile",
				ScopeType:   "user",
				ScopeID:     "10001",
				Mode:        "enforce",
				Enabled:     true,
				ProfileJSON: `{"default_action":{"type":"proxy"},"default_dns_resolver_tag":"dns:cloudflare-dot","default_fallback_policy":"fallback_default"}`,
			},
		},
		dnsResolvers: []*adminroutingbiz.DNSResolver{
			{
				ID:           1,
				Tag:          "dns:cloudflare-dot",
				Name:         "Cloudflare DoT",
				Proto:        "dot",
				Address:      "1.1.1.1",
				Port:         853,
				Enabled:      true,
				ResolverJSON: `{"server_name":"cloudflare-dns.com","detour":{"type":"proxy"},"health_check":{"enabled":true,"domain":"www.cloudflare.com","interval_seconds":60}}`,
			},
		},
	}, log.DefaultLogger)
	return adminroutingservice.NewRoutingService(uc, log.DefaultLogger)
}

func routingConfigHandlerForTest() (func(http.HandlerFunc) http.Handler, *publicRoutingCache) {
	return func(handler http.HandlerFunc) http.Handler { return handler }, &publicRoutingCache{}
}

func routingConfigResponseData(t *testing.T, rec *httptest.ResponseRecorder) publicrouting.Envelope {
	t.Helper()
	var resp struct {
		Code int                    `json:"code"`
		Data publicrouting.Envelope `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode routing config response: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("response code = %d, want 200; body=%s", resp.Code, rec.Body.String())
	}
	return resp.Data
}

func routingHandlerFeatures(extra ...string) []string {
	features := []string{
		"routing_profile_v1",
		"route_dns_resolver",
		"route_outbound",
		"route_fail_policy",
		"route_fallback",
		"doh",
	}
	features = append(features, extra...)
	return features
}

func containsFeature(features []string, expected string) bool {
	for _, feature := range features {
		if feature == expected {
			return true
		}
	}
	return false
}
