package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	publicrouting "github.com/npanel-dev/NPanel-backend/internal/biz/public/routing"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"

	"github.com/go-kratos/kratos/v2/log"
)

type RouteProfile struct {
	ID          int64
	Code        string
	Name        string
	Description string
	ScopeType   string
	ScopeID     string
	Priority    int
	Mode        string
	Enabled     bool
	ProfileJSON string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RouteRule struct {
	ID          int64
	ProfileID   int64
	Name        string
	Priority    int
	Enabled     bool
	ServiceCode string
	MatcherJSON string
	ActionJSON  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DNSResolver struct {
	ID           int64
	Tag          string
	Name         string
	Proto        string
	Address      string
	Port         int
	Enabled      bool
	ResolverJSON string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RouteOutbound struct {
	ID           int64
	Tag          string
	Name         string
	Type         string
	Region       string
	Enabled      bool
	OutboundJSON string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UnlockService struct {
	ID          int64
	Code        string
	Name        string
	Category    string
	Enabled     bool
	ServiceJSON string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PreviewRequest = publicrouting.PreviewRequest
type PreviewResult = publicrouting.PreviewResult

type RoutingRepo interface {
	SaveProfile(context.Context, *RouteProfile) (*RouteProfile, error)
	UpdateProfile(context.Context, *RouteProfile) (*RouteProfile, error)
	FindProfileByID(context.Context, int64) (*RouteProfile, error)
	ListProfiles(context.Context, int, int, string, *bool) ([]*RouteProfile, int32, error)
	DeleteProfile(context.Context, int64) error

	SaveRule(context.Context, *RouteRule) (*RouteRule, error)
	UpdateRule(context.Context, *RouteRule) (*RouteRule, error)
	FindRuleByID(context.Context, int64) (*RouteRule, error)
	ListRules(context.Context, int, int, int64, string, *bool) ([]*RouteRule, int32, error)
	DeleteRule(context.Context, int64) error

	SaveDNSResolver(context.Context, *DNSResolver) (*DNSResolver, error)
	UpdateDNSResolver(context.Context, *DNSResolver) (*DNSResolver, error)
	FindDNSResolverByID(context.Context, int64) (*DNSResolver, error)
	ListDNSResolvers(context.Context, int, int, string, *bool) ([]*DNSResolver, int32, error)
	DeleteDNSResolver(context.Context, int64) error

	SaveOutbound(context.Context, *RouteOutbound) (*RouteOutbound, error)
	UpdateOutbound(context.Context, *RouteOutbound) (*RouteOutbound, error)
	FindOutboundByID(context.Context, int64) (*RouteOutbound, error)
	ListOutbounds(context.Context, int, int, string, *bool) ([]*RouteOutbound, int32, error)
	DeleteOutbound(context.Context, int64) error

	SaveUnlockService(context.Context, *UnlockService) (*UnlockService, error)
	UpdateUnlockService(context.Context, *UnlockService) (*UnlockService, error)
	FindUnlockServiceByID(context.Context, int64) (*UnlockService, error)
	ListUnlockServices(context.Context, int, int, string, *bool) ([]*UnlockService, int32, error)
	DeleteUnlockService(context.Context, int64) error
}

type RoutingUsecase struct {
	repo   RoutingRepo
	logger *log.Helper
}

func NewRoutingUsecase(repo RoutingRepo, logger log.Logger) *RoutingUsecase {
	return &RoutingUsecase{repo: repo, logger: log.NewHelper(logger)}
}

func (uc *RoutingUsecase) CreateProfile(ctx context.Context, item *RouteProfile) (*RouteProfile, error) {
	normalizeProfile(item)
	if err := validateProfile(item); err != nil {
		return nil, err
	}
	return uc.repo.SaveProfile(ctx, item)
}

func (uc *RoutingUsecase) UpdateProfile(ctx context.Context, item *RouteProfile) (*RouteProfile, error) {
	if item.ID <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	normalizeProfile(item)
	if err := validateProfile(item); err != nil {
		return nil, err
	}
	if found, err := uc.repo.FindProfileByID(ctx, item.ID); err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	} else if found == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrSystemNotFound)
	}
	result, err := uc.repo.UpdateProfile(ctx, item)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("update routing profile failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}
	return result, nil
}

func (uc *RoutingUsecase) ListProfiles(ctx context.Context, page, size int, search string, enabled *bool) ([]*RouteProfile, int32, error) {
	page, size = normalizePage(page, size)
	return uc.repo.ListProfiles(ctx, page, size, search, enabled)
}

func (uc *RoutingUsecase) DeleteProfile(ctx context.Context, id int64) error {
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return uc.repo.DeleteProfile(ctx, id)
}

func (uc *RoutingUsecase) CreateRule(ctx context.Context, item *RouteRule) (*RouteRule, error) {
	normalizeRule(item)
	if err := validateRule(item); err != nil {
		return nil, err
	}
	return uc.repo.SaveRule(ctx, item)
}

func (uc *RoutingUsecase) UpdateRule(ctx context.Context, item *RouteRule) (*RouteRule, error) {
	if item.ID <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	normalizeRule(item)
	if err := validateRule(item); err != nil {
		return nil, err
	}
	if found, err := uc.repo.FindRuleByID(ctx, item.ID); err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	} else if found == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrSystemNotFound)
	}
	return uc.repo.UpdateRule(ctx, item)
}

func (uc *RoutingUsecase) ListRules(ctx context.Context, page, size int, profileID int64, search string, enabled *bool) ([]*RouteRule, int32, error) {
	page, size = normalizePage(page, size)
	return uc.repo.ListRules(ctx, page, size, profileID, search, enabled)
}

func (uc *RoutingUsecase) DeleteRule(ctx context.Context, id int64) error {
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return uc.repo.DeleteRule(ctx, id)
}

func (uc *RoutingUsecase) CreateDNSResolver(ctx context.Context, item *DNSResolver) (*DNSResolver, error) {
	normalizeDNSResolver(item)
	if err := validateDNSResolver(item); err != nil {
		return nil, err
	}
	return uc.repo.SaveDNSResolver(ctx, item)
}

func (uc *RoutingUsecase) UpdateDNSResolver(ctx context.Context, item *DNSResolver) (*DNSResolver, error) {
	if item.ID <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	normalizeDNSResolver(item)
	if err := validateDNSResolver(item); err != nil {
		return nil, err
	}
	if found, err := uc.repo.FindDNSResolverByID(ctx, item.ID); err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	} else if found == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrSystemNotFound)
	}
	return uc.repo.UpdateDNSResolver(ctx, item)
}

func (uc *RoutingUsecase) ListDNSResolvers(ctx context.Context, page, size int, search string, enabled *bool) ([]*DNSResolver, int32, error) {
	page, size = normalizePage(page, size)
	return uc.repo.ListDNSResolvers(ctx, page, size, search, enabled)
}

func (uc *RoutingUsecase) DeleteDNSResolver(ctx context.Context, id int64) error {
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return uc.repo.DeleteDNSResolver(ctx, id)
}

func (uc *RoutingUsecase) CreateOutbound(ctx context.Context, item *RouteOutbound) (*RouteOutbound, error) {
	normalizeOutbound(item)
	if err := validateOutbound(item); err != nil {
		return nil, err
	}
	return uc.repo.SaveOutbound(ctx, item)
}

func (uc *RoutingUsecase) UpdateOutbound(ctx context.Context, item *RouteOutbound) (*RouteOutbound, error) {
	if item.ID <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	normalizeOutbound(item)
	if err := validateOutbound(item); err != nil {
		return nil, err
	}
	if found, err := uc.repo.FindOutboundByID(ctx, item.ID); err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	} else if found == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrSystemNotFound)
	}
	return uc.repo.UpdateOutbound(ctx, item)
}

func (uc *RoutingUsecase) ListOutbounds(ctx context.Context, page, size int, search string, enabled *bool) ([]*RouteOutbound, int32, error) {
	page, size = normalizePage(page, size)
	return uc.repo.ListOutbounds(ctx, page, size, search, enabled)
}

func (uc *RoutingUsecase) DeleteOutbound(ctx context.Context, id int64) error {
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return uc.repo.DeleteOutbound(ctx, id)
}

func (uc *RoutingUsecase) CreateUnlockService(ctx context.Context, item *UnlockService) (*UnlockService, error) {
	normalizeUnlockService(item)
	if err := validateUnlockService(item); err != nil {
		return nil, err
	}
	return uc.repo.SaveUnlockService(ctx, item)
}

func (uc *RoutingUsecase) UpdateUnlockService(ctx context.Context, item *UnlockService) (*UnlockService, error) {
	if item.ID <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	normalizeUnlockService(item)
	if err := validateUnlockService(item); err != nil {
		return nil, err
	}
	if found, err := uc.repo.FindUnlockServiceByID(ctx, item.ID); err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	} else if found == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrSystemNotFound)
	}
	return uc.repo.UpdateUnlockService(ctx, item)
}

func (uc *RoutingUsecase) ListUnlockServices(ctx context.Context, page, size int, search string, enabled *bool) ([]*UnlockService, int32, error) {
	page, size = normalizePage(page, size)
	return uc.repo.ListUnlockServices(ctx, page, size, search, enabled)
}

func (uc *RoutingUsecase) DeleteUnlockService(ctx context.Context, id int64) error {
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return uc.repo.DeleteUnlockService(ctx, id)
}

func (uc *RoutingUsecase) Preview(ctx context.Context, req PreviewRequest) (PreviewResult, error) {
	envelope, err := uc.BuildConfig(ctx, time.Now())
	if err != nil {
		return PreviewResult{}, err
	}
	return publicrouting.PreviewRouteConfig(envelope, req), nil
}

func (uc *RoutingUsecase) BuildConfig(ctx context.Context, now time.Time, opts ...publicrouting.ConfigOptions) (publicrouting.Envelope, error) {
	fixture := publicrouting.BuildPreviewConfig(now, firstConfigOptions(opts))
	profiles, _, err := uc.repo.ListProfiles(ctx, 1, 1000, "", boolPtr(true))
	if err != nil {
		return fixture, err
	}
	if len(profiles) == 0 {
		return fixture, nil
	}

	profile := profiles[0]
	envelope := fixture
	envelope.GeneratedAt = now.UTC().Format(time.RFC3339)
	envelope.ExpiresAt = now.UTC().Add(10 * time.Minute).Format(time.RFC3339)
	envelope.Mode = firstNonEmpty(profile.Mode, publicrouting.ModeObserve)
	envelope.Profile = profileToRoutingProfile(profile, fixture.Profile)

	if resolvers, _, err := uc.repo.ListDNSResolvers(ctx, 1, 1000, "", boolPtr(true)); err == nil {
		envelope.DNSResolvers = dnsResolversToRouting(resolvers)
	} else {
		return fixture, err
	}
	if outbounds, _, err := uc.repo.ListOutbounds(ctx, 1, 1000, "", boolPtr(true)); err == nil {
		envelope.Outbounds = outboundsToRouting(outbounds)
	} else {
		return fixture, err
	}
	if services, _, err := uc.repo.ListUnlockServices(ctx, 1, 1000, "", boolPtr(true)); err == nil {
		envelope.UnlockServices = unlockServicesToRouting(services)
	} else {
		return fixture, err
	}
	if rules, _, err := uc.repo.ListRules(ctx, 1, 1000, profile.ID, "", boolPtr(true)); err == nil {
		envelope.Rules = rulesToRouting(rules)
	} else {
		return fixture, err
	}

	envelope.HealthSnapshot = publicrouting.HealthSnapshot{
		GeneratedAt:  envelope.GeneratedAt,
		Outbounds:    unknownOutboundHealth(envelope.Outbounds, envelope.GeneratedAt),
		DNSResolvers: unknownDNSHealth(envelope.DNSResolvers, envelope.GeneratedAt),
		Services:     unknownServiceHealth(envelope.UnlockServices, envelope.GeneratedAt),
	}
	if err := validateCompiledEnvelope(envelope); err != nil {
		return fixture, err
	}
	envelope.RoutingHash = publicrouting.StableHash(envelope)
	return envelope, nil
}

func normalizePage(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	if size > 1000 {
		size = 1000
	}
	return page, size
}

func firstConfigOptions(opts []publicrouting.ConfigOptions) publicrouting.ConfigOptions {
	if len(opts) == 0 {
		return publicrouting.ConfigOptions{}
	}
	return opts[0]
}

func normalizeProfile(item *RouteProfile) {
	item.Code = strings.TrimSpace(item.Code)
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	item.ScopeType = firstNonEmpty(strings.TrimSpace(item.ScopeType), "global")
	item.ScopeID = firstNonEmpty(strings.TrimSpace(item.ScopeID), "default")
	item.Mode = firstNonEmpty(strings.TrimSpace(item.Mode), publicrouting.ModeObserve)
	item.ProfileJSON = normalizeJSON(item.ProfileJSON)
}

func validateProfile(item *RouteProfile) error {
	if item.Code == "" || item.Name == "" || !json.Valid([]byte(item.ProfileJSON)) {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return nil
}

func normalizeRule(item *RouteRule) {
	item.Name = strings.TrimSpace(item.Name)
	item.ServiceCode = strings.TrimSpace(item.ServiceCode)
	item.MatcherJSON = normalizeJSONWithDefault(item.MatcherJSON, `{"type":"domain_suffix","value":"example.com"}`)
	item.ActionJSON = normalizeJSONWithDefault(item.ActionJSON, `{"type":"outbound","outbound_tag":"proxy:default","fail_policy":"fallback_default"}`)
}

func validateRule(item *RouteRule) error {
	if item.Name == "" || !json.Valid([]byte(item.MatcherJSON)) || !json.Valid([]byte(item.ActionJSON)) {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return nil
}

func normalizeDNSResolver(item *DNSResolver) {
	item.Tag = strings.TrimSpace(item.Tag)
	item.Name = strings.TrimSpace(item.Name)
	item.Proto = firstNonEmpty(strings.TrimSpace(item.Proto), "system")
	item.Address = strings.TrimSpace(item.Address)
	item.ResolverJSON = normalizeJSON(item.ResolverJSON)
}

func validateDNSResolver(item *DNSResolver) error {
	if item.Tag == "" || item.Name == "" || !json.Valid([]byte(item.ResolverJSON)) {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return nil
}

func normalizeOutbound(item *RouteOutbound) {
	item.Tag = strings.TrimSpace(item.Tag)
	item.Name = strings.TrimSpace(item.Name)
	item.Type = firstNonEmpty(strings.TrimSpace(item.Type), "node_group")
	item.Region = strings.TrimSpace(item.Region)
	item.OutboundJSON = normalizeJSON(item.OutboundJSON)
}

func validateOutbound(item *RouteOutbound) error {
	if item.Tag == "" || item.Name == "" || !json.Valid([]byte(item.OutboundJSON)) {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return nil
}

func normalizeUnlockService(item *UnlockService) {
	item.Code = strings.TrimSpace(item.Code)
	item.Name = strings.TrimSpace(item.Name)
	item.Category = strings.TrimSpace(item.Category)
	item.ServiceJSON = normalizeJSON(item.ServiceJSON)
}

func validateUnlockService(item *UnlockService) error {
	if item.Code == "" || item.Name == "" || !json.Valid([]byte(item.ServiceJSON)) {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return nil
}

func normalizeJSON(raw string) string {
	return normalizeJSONWithDefault(raw, "{}")
}

func normalizeJSONWithDefault(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	return raw
}

func profileToRoutingProfile(item *RouteProfile, fallback publicrouting.Profile) publicrouting.Profile {
	profile := fallback
	_ = json.Unmarshal([]byte(item.ProfileJSON), &profile)
	profile.ID = fmt.Sprintf("profile_%d", item.ID)
	profile.Code = item.Code
	profile.Name = item.Name
	profile.Description = item.Description
	profile.Scope = publicrouting.ProfileScope{Type: item.ScopeType, ID: item.ScopeID}
	profile.Priority = item.Priority
	profile.Enabled = item.Enabled
	return profile
}

func dnsResolversToRouting(items []*DNSResolver) []publicrouting.DNSResolver {
	result := make([]publicrouting.DNSResolver, 0, len(items))
	for _, item := range items {
		resolver := publicrouting.DNSResolver{}
		_ = json.Unmarshal([]byte(item.ResolverJSON), &resolver)
		resolver.Tag = item.Tag
		resolver.Name = item.Name
		resolver.Proto = item.Proto
		resolver.Address = item.Address
		resolver.Port = item.Port
		resolver.Enabled = item.Enabled
		result = append(result, resolver)
	}
	return result
}

func outboundsToRouting(items []*RouteOutbound) []publicrouting.RouteOutbound {
	result := make([]publicrouting.RouteOutbound, 0, len(items))
	for _, item := range items {
		outbound := publicrouting.RouteOutbound{}
		_ = json.Unmarshal([]byte(item.OutboundJSON), &outbound)
		outbound.Tag = item.Tag
		outbound.Name = item.Name
		outbound.Type = item.Type
		outbound.Region = item.Region
		outbound.Enabled = item.Enabled
		result = append(result, outbound)
	}
	return result
}

func unlockServicesToRouting(items []*UnlockService) []publicrouting.UnlockService {
	result := make([]publicrouting.UnlockService, 0, len(items))
	for _, item := range items {
		service := publicrouting.UnlockService{}
		_ = json.Unmarshal([]byte(item.ServiceJSON), &service)
		service.Code = item.Code
		service.Name = item.Name
		service.Category = item.Category
		service.Enabled = item.Enabled
		result = append(result, service)
	}
	return result
}

func rulesToRouting(items []*RouteRule) []publicrouting.Rule {
	result := make([]publicrouting.Rule, 0, len(items))
	for _, item := range items {
		var matcher publicrouting.Matcher
		var action publicrouting.RouteAction
		_ = json.Unmarshal([]byte(item.MatcherJSON), &matcher)
		_ = json.Unmarshal([]byte(item.ActionJSON), &action)
		result = append(result, publicrouting.Rule{
			ID:          fmt.Sprintf("rule_%d", item.ID),
			Name:        item.Name,
			Priority:    item.Priority,
			Enabled:     item.Enabled,
			ServiceCode: item.ServiceCode,
			Matcher:     matcher,
			Action:      action,
		})
	}
	return result
}

func validateCompiledEnvelope(envelope publicrouting.Envelope) error {
	resolvers := map[string]struct{}{
		"":           {},
		"dns:system": {},
	}
	for _, resolver := range envelope.DNSResolvers {
		resolvers[resolver.Tag] = struct{}{}
	}

	outbounds := map[string]struct{}{
		"":              {},
		"proxy:default": {},
	}
	for _, outbound := range envelope.Outbounds {
		outbounds[outbound.Tag] = struct{}{}
	}

	if _, ok := resolvers[envelope.Profile.DefaultDNSResolverTag]; !ok {
		return fmt.Errorf("routing profile references missing dns resolver %q", envelope.Profile.DefaultDNSResolverTag)
	}
	if err := validateAction(envelope.Profile.DefaultAction, resolvers, outbounds); err != nil {
		return err
	}
	for _, rule := range envelope.Rules {
		if !rule.Enabled {
			continue
		}
		if err := validateAction(rule.Action, resolvers, outbounds); err != nil {
			return fmt.Errorf("routing rule %q: %w", rule.ID, err)
		}
	}
	return nil
}

func validateAction(action publicrouting.RouteAction, resolvers, outbounds map[string]struct{}) error {
	if _, ok := resolvers[action.DNSResolverTag]; !ok {
		return fmt.Errorf("missing dns resolver %q", action.DNSResolverTag)
	}
	if action.Type == "outbound" {
		if _, ok := outbounds[action.OutboundTag]; !ok {
			return fmt.Errorf("missing outbound %q", action.OutboundTag)
		}
	}
	return nil
}

func unknownOutboundHealth(items []publicrouting.RouteOutbound, now string) []publicrouting.HealthStatus {
	result := make([]publicrouting.HealthStatus, 0, len(items))
	for _, item := range items {
		result = append(result, publicrouting.HealthStatus{Tag: item.Tag, Status: "unknown", Source: "backend_admin", CheckedAt: now})
	}
	return result
}

func unknownDNSHealth(items []publicrouting.DNSResolver, now string) []publicrouting.HealthStatus {
	result := make([]publicrouting.HealthStatus, 0, len(items))
	for _, item := range items {
		result = append(result, publicrouting.HealthStatus{Tag: item.Tag, Status: "unknown", Source: "backend_admin", CheckedAt: now})
	}
	return result
}

func unknownServiceHealth(items []publicrouting.UnlockService, now string) []publicrouting.HealthStatus {
	result := make([]publicrouting.HealthStatus, 0, len(items))
	for _, item := range items {
		result = append(result, publicrouting.HealthStatus{Code: item.Code, Status: "unknown", Source: "backend_admin", CheckedAt: now, OutboundTag: item.DefaultOutboundTag, DNSResolverTag: item.DefaultDNSResolverTag})
	}
	return result
}

func boolPtr(value bool) *bool {
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
