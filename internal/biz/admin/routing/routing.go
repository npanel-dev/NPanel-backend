package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

type RoutingHealthItem struct {
	Kind                string
	Key                 string
	Name                string
	Status              string
	Source              string
	CheckedAt           time.Time
	RTTMS               int
	ConsecutiveFailures int
	LastError           string
	OutboundTag         string
	DNSResolverTag      string
}

type RoutingEnforceGuard struct {
	Key    string
	Label  string
	Passed bool
	Status string
	Reason string
}

type RoutingAuditEvent struct {
	ID           string
	ResourceType string
	ResourceID   string
	ResourceName string
	Action       string
	Summary      string
	CreatedAt    time.Time
}

type RoutingOverview struct {
	RoutingHash      string
	GeneratedAt      string
	ProfileCode      string
	ProfileName      string
	Mode             string
	ProfileEnabled   bool
	EnforceReady     bool
	ExecutionEnabled bool
	RollbackAction   string
	CompileError     string
	Health           []RoutingHealthItem
	Guards           []RoutingEnforceGuard
	AuditEvents      []RoutingAuditEvent
}

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

func (uc *RoutingUsecase) Overview(ctx context.Context) (*RoutingOverview, error) {
	now := time.Now()
	envelope, err := uc.BuildConfig(ctx, now)
	overview := routingOverviewFromEnvelope(envelope)
	if err != nil {
		overview.CompileError = err.Error()
	}
	profiles, _, profileErr := uc.repo.ListProfiles(ctx, 1, 1000, "", nil)
	if profileErr == nil {
		overview.AuditEvents = append(overview.AuditEvents, auditEventsForProfiles(profiles)...)
	}
	if rules, _, err := uc.repo.ListRules(ctx, 1, 1000, 0, "", nil); err == nil {
		overview.AuditEvents = append(overview.AuditEvents, auditEventsForRules(rules)...)
	}
	if resolvers, _, err := uc.repo.ListDNSResolvers(ctx, 1, 1000, "", nil); err == nil {
		overview.AuditEvents = append(overview.AuditEvents, auditEventsForDNSResolvers(resolvers)...)
	}
	if outbounds, _, err := uc.repo.ListOutbounds(ctx, 1, 1000, "", nil); err == nil {
		overview.AuditEvents = append(overview.AuditEvents, auditEventsForOutbounds(outbounds)...)
	}
	if services, _, err := uc.repo.ListUnlockServices(ctx, 1, 1000, "", nil); err == nil {
		overview.AuditEvents = append(overview.AuditEvents, auditEventsForUnlockServices(services)...)
	}
	sort.SliceStable(overview.AuditEvents, func(i, j int) bool {
		return overview.AuditEvents[i].CreatedAt.After(overview.AuditEvents[j].CreatedAt)
	})
	if len(overview.AuditEvents) > 20 {
		overview.AuditEvents = overview.AuditEvents[:20]
	}
	overview.Guards = routingEnforceGuards(envelope, overview.CompileError)
	overview.EnforceReady = guardsPassed(overview.Guards)
	overview.ExecutionEnabled = false
	overview.RollbackAction = "switch profile mode back to observe"
	return overview, nil
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
		if len(resolvers) > 0 {
			envelope.DNSResolvers = dnsResolversToRouting(resolvers)
		}
	} else {
		return fixture, err
	}
	if outbounds, _, err := uc.repo.ListOutbounds(ctx, 1, 1000, "", boolPtr(true)); err == nil {
		if len(outbounds) > 0 {
			envelope.Outbounds = outboundsToRouting(outbounds)
		}
	} else {
		return fixture, err
	}
	if services, _, err := uc.repo.ListUnlockServices(ctx, 1, 1000, "", boolPtr(true)); err == nil {
		if len(services) > 0 {
			envelope.UnlockServices = unlockServicesToRouting(services)
		}
	} else {
		return fixture, err
	}
	if rules, _, err := uc.repo.ListRules(ctx, 1, 1000, profile.ID, "", boolPtr(true)); err == nil {
		envelope.Rules = rulesToRouting(rules)
	} else {
		return fixture, err
	}

	envelope.HealthSnapshot = buildAdminHealthSnapshot(envelope)
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

func routingOverviewFromEnvelope(envelope publicrouting.Envelope) *RoutingOverview {
	return &RoutingOverview{
		RoutingHash:    envelope.RoutingHash,
		GeneratedAt:    envelope.GeneratedAt,
		ProfileCode:    envelope.Profile.Code,
		ProfileName:    envelope.Profile.Name,
		Mode:           envelope.Mode,
		ProfileEnabled: envelope.Profile.Enabled,
		Health:         routingHealthItems(envelope),
	}
}

func routingHealthItems(envelope publicrouting.Envelope) []RoutingHealthItem {
	names := map[string]string{}
	for _, outbound := range envelope.Outbounds {
		names["outbound:"+outbound.Tag] = outbound.Name
	}
	for _, resolver := range envelope.DNSResolvers {
		names["dns_resolver:"+resolver.Tag] = resolver.Name
	}
	for _, service := range envelope.UnlockServices {
		names["service:"+service.Code] = service.Name
	}

	var result []RoutingHealthItem
	for _, item := range envelope.HealthSnapshot.Outbounds {
		result = append(result, healthItemFromStatus("outbound", item.Tag, names["outbound:"+item.Tag], item))
	}
	for _, item := range envelope.HealthSnapshot.DNSResolvers {
		result = append(result, healthItemFromStatus("dns_resolver", item.Tag, names["dns_resolver:"+item.Tag], item))
	}
	for _, item := range envelope.HealthSnapshot.Services {
		result = append(result, healthItemFromStatus("service", item.Code, names["service:"+item.Code], item))
	}
	return result
}

func healthItemFromStatus(kind, key, name string, item publicrouting.HealthStatus) RoutingHealthItem {
	return RoutingHealthItem{
		Kind:                kind,
		Key:                 key,
		Name:                name,
		Status:              item.Status,
		Source:              item.Source,
		CheckedAt:           parseRFC3339(item.CheckedAt),
		RTTMS:               item.RTTMS,
		ConsecutiveFailures: item.ConsecutiveFailures,
		LastError:           item.LastError,
		OutboundTag:         item.OutboundTag,
		DNSResolverTag:      item.DNSResolverTag,
	}
}

func routingEnforceGuards(envelope publicrouting.Envelope, compileError string) []RoutingEnforceGuard {
	compileOK := compileError == ""
	profileEnabled := envelope.Profile.Enabled
	modeEnforce := envelope.Mode == "enforce"
	grayScope := envelope.Profile.Scope.Type != "" && envelope.Profile.Scope.Type != "global"
	healthOK := routingHealthOK(envelope.HealthSnapshot)
	capabilityOK := len(envelope.CapabilityRequirements.RequiredFeatures) > 0
	rollbackOK := true

	return []RoutingEnforceGuard{
		guard("compile_ok", "Config compiles", compileOK, "ok", compileError),
		guard("profile_enabled", "Profile enabled", profileEnabled, "ok", "enable the target profile before enforce"),
		guard("mode_enforce", "Profile mode enforce", modeEnforce, envelope.Mode, "keep observe until dry-run and health are verified"),
		guard("gray_scope", "Gray scope selected", grayScope, envelope.Profile.Scope.Type, "use user/subscribe scope before global enforce"),
		guard("health_ok", "Health snapshot healthy", healthOK, healthSummary(envelope.HealthSnapshot), "unknown or failing health blocks enforce"),
		guard("capability_contract", "Capability contract present", capabilityOK, "ok", "required_features must be declared"),
		guard("rollback_ready", "Rollback path ready", rollbackOK, "ok", "switch profile mode back to observe"),
		guard("execution_wired", "Client execution wired", false, "dry-run only", "P4 keeps real execution disabled"),
	}
}

func guard(key, label string, passed bool, status, failReason string) RoutingEnforceGuard {
	if passed {
		return RoutingEnforceGuard{Key: key, Label: label, Passed: true, Status: status, Reason: "passed"}
	}
	if failReason == "" {
		failReason = "blocked"
	}
	return RoutingEnforceGuard{Key: key, Label: label, Passed: false, Status: status, Reason: failReason}
}

func guardsPassed(items []RoutingEnforceGuard) bool {
	for _, item := range items {
		if !item.Passed {
			return false
		}
	}
	return true
}

func routingHealthOK(snapshot publicrouting.HealthSnapshot) bool {
	for _, item := range append(append([]publicrouting.HealthStatus{}, snapshot.Outbounds...), append(snapshot.DNSResolvers, snapshot.Services...)...) {
		if item.Status != "healthy" && item.Status != "ok" && item.Status != "disabled" {
			return false
		}
	}
	return len(snapshot.Outbounds)+len(snapshot.DNSResolvers)+len(snapshot.Services) > 0
}

func healthSummary(snapshot publicrouting.HealthSnapshot) string {
	counts := map[string]int{}
	for _, item := range append(append([]publicrouting.HealthStatus{}, snapshot.Outbounds...), append(snapshot.DNSResolvers, snapshot.Services...)...) {
		counts[item.Status]++
	}
	if len(counts) == 0 {
		return "empty"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}

func buildAdminHealthSnapshot(envelope publicrouting.Envelope) publicrouting.HealthSnapshot {
	return publicrouting.HealthSnapshot{
		GeneratedAt:  envelope.GeneratedAt,
		Outbounds:    adminOutboundHealth(envelope.Outbounds, envelope.GeneratedAt),
		DNSResolvers: adminDNSHealth(envelope.DNSResolvers, envelope.GeneratedAt),
		Services:     adminServiceHealth(envelope.UnlockServices, envelope.GeneratedAt),
	}
}

func adminOutboundHealth(items []publicrouting.RouteOutbound, now string) []publicrouting.HealthStatus {
	result := make([]publicrouting.HealthStatus, 0, len(items))
	for _, item := range items {
		status, lastError := configuredHealthStatus(item.Enabled, item.HealthCheck.Enabled)
		result = append(result, publicrouting.HealthStatus{
			Tag:       item.Tag,
			Region:    item.Region,
			Status:    status,
			Source:    "admin_config",
			CheckedAt: now,
			LastError: lastError,
		})
	}
	return result
}

func adminDNSHealth(items []publicrouting.DNSResolver, now string) []publicrouting.HealthStatus {
	result := make([]publicrouting.HealthStatus, 0, len(items))
	for _, item := range items {
		status, lastError := configuredHealthStatus(item.Enabled, item.HealthCheck.Enabled)
		result = append(result, publicrouting.HealthStatus{
			Tag:       item.Tag,
			Status:    status,
			Source:    "admin_config",
			CheckedAt: now,
			LastError: lastError,
		})
	}
	return result
}

func adminServiceHealth(items []publicrouting.UnlockService, now string) []publicrouting.HealthStatus {
	result := make([]publicrouting.HealthStatus, 0, len(items))
	for _, item := range items {
		status, lastError := configuredHealthStatus(item.Enabled, item.HealthCheckURL != "")
		result = append(result, publicrouting.HealthStatus{
			Code:           item.Code,
			Region:         item.DefaultRegion,
			Status:         status,
			Source:         "admin_config",
			CheckedAt:      now,
			LastError:      lastError,
			OutboundTag:    item.DefaultOutboundTag,
			DNSResolverTag: item.DefaultDNSResolverTag,
		})
	}
	return result
}

func configuredHealthStatus(enabled, hasCheck bool) (string, string) {
	if !enabled {
		return "disabled", ""
	}
	if !hasCheck {
		return "unknown", "health check is not configured"
	}
	return "unknown", "waiting for node/client health reports"
}

func unknownOutboundHealth(items []publicrouting.RouteOutbound, now string) []publicrouting.HealthStatus {
	result := make([]publicrouting.HealthStatus, 0, len(items))
	for _, item := range items {
		result = append(result, publicrouting.HealthStatus{Tag: item.Tag, Status: "unknown", Source: "backend_admin", CheckedAt: now})
	}
	return result
}

func auditEventsForProfiles(items []*RouteProfile) []RoutingAuditEvent {
	events := make([]RoutingAuditEvent, 0, len(items))
	for _, item := range items {
		events = append(events, auditEvent("profile", item.ID, item.Code, "upsert", fmt.Sprintf("mode=%s enabled=%t scope=%s:%s", item.Mode, item.Enabled, item.ScopeType, item.ScopeID), item.UpdatedAt))
	}
	return events
}

func auditEventsForRules(items []*RouteRule) []RoutingAuditEvent {
	events := make([]RoutingAuditEvent, 0, len(items))
	for _, item := range items {
		events = append(events, auditEvent("rule", item.ID, item.Name, "upsert", fmt.Sprintf("profile_id=%d enabled=%t", item.ProfileID, item.Enabled), item.UpdatedAt))
	}
	return events
}

func auditEventsForDNSResolvers(items []*DNSResolver) []RoutingAuditEvent {
	events := make([]RoutingAuditEvent, 0, len(items))
	for _, item := range items {
		events = append(events, auditEvent("dns_resolver", item.ID, item.Tag, "upsert", fmt.Sprintf("proto=%s enabled=%t", item.Proto, item.Enabled), item.UpdatedAt))
	}
	return events
}

func auditEventsForOutbounds(items []*RouteOutbound) []RoutingAuditEvent {
	events := make([]RoutingAuditEvent, 0, len(items))
	for _, item := range items {
		events = append(events, auditEvent("outbound", item.ID, item.Tag, "upsert", fmt.Sprintf("type=%s region=%s enabled=%t", item.Type, item.Region, item.Enabled), item.UpdatedAt))
	}
	return events
}

func auditEventsForUnlockServices(items []*UnlockService) []RoutingAuditEvent {
	events := make([]RoutingAuditEvent, 0, len(items))
	for _, item := range items {
		events = append(events, auditEvent("unlock_service", item.ID, item.Code, "upsert", fmt.Sprintf("category=%s enabled=%t", item.Category, item.Enabled), item.UpdatedAt))
	}
	return events
}

func auditEvent(resourceType string, resourceID int64, name, action, summary string, at time.Time) RoutingAuditEvent {
	if at.IsZero() {
		at = time.Now()
	}
	id := fmt.Sprintf("%s:%d:%d", resourceType, resourceID, at.Unix())
	return RoutingAuditEvent{
		ID:           id,
		ResourceType: resourceType,
		ResourceID:   fmt.Sprintf("%d", resourceID),
		ResourceName: name,
		Action:       action,
		Summary:      summary,
		CreatedAt:    at,
	}
}

func parseRFC3339(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
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
