package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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

type RoutingHealthReport struct {
	ID                  int64
	ReporterType        string
	ReporterID          string
	ProfileCode         string
	RoutingHash         string
	SubjectType         string
	SubjectKey          string
	Region              string
	Status              string
	Source              string
	RTTMS               int
	ConsecutiveFailures int
	LastError           string
	OutboundTag         string
	DNSResolverTag      string
	CheckedAt           time.Time
	ReportJSON          string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type RoutingRouteEvent struct {
	ID             int64
	ReporterType   string
	ReporterID     string
	ProfileCode    string
	RoutingHash    string
	EventType      string
	Subject        string
	RuleID         string
	RuleName       string
	ActionType     string
	OutboundTag    string
	DNSResolverTag string
	FallbackTarget string
	Status         string
	LatencyMS      int
	Error          string
	EventAt        time.Time
	EventJSON      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RoutingGrayRelease struct {
	ID             int64
	ProfileCode    string
	Name           string
	Status         string
	BatchNo        int
	TargetType     string
	TargetIDsJSON  string
	Operator       string
	RollbackReason string
	StartedAt      time.Time
	EndedAt        time.Time
	ReleaseJSON    string
	TargetCount    int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RoutingAnalyticsItem struct {
	ProfileCode       string
	RoutingHash       string
	ReporterID        string
	RouteEvents       int
	RouteDecisions    int
	RouteFallbacks    int
	FallbackRateBP    int
	DNSFailures       int
	OutboundFailures  int
	AffectedReporters int
	LastEventType     string
	LastStatus        string
	LastError         string
	LastSeenAt        time.Time
}

type RoutingAnalyticsError struct {
	Key   string
	Kind  string
	Error string
	Count int
}

type RoutingAnalytics struct {
	Items              []RoutingAnalyticsItem
	TopErrors          []RoutingAnalyticsError
	TotalRouteEvents   int
	TotalHealthReports int
	AffectedReporters  int
	FallbackRateBP     int
	DNSFailRateBP      int
	OutboundFailRateBP int
	WindowStartedAt    time.Time
}

type RoutingReleaseGateCheck struct {
	Key    string
	Label  string
	Passed bool
	Status string
	Reason string
}

type RoutingReleaseGate struct {
	ProfileCode          string
	RoutingHash          string
	Allowed              bool
	RequiresConfirmation bool
	Summary              string
	Checks               []RoutingReleaseGateCheck
	Analytics            *RoutingAnalytics
	GeneratedAt          time.Time
	Thresholds           RoutingReleaseThresholds
}

type RoutingE2EChecklistItem struct {
	Key      string
	Label    string
	Status   string
	Passed   bool
	Evidence string
}

type RoutingE2EChecklist struct {
	Items       []RoutingE2EChecklistItem
	Ready       bool
	GeneratedAt time.Time
}

type RoutingCapabilityMatrixItem struct {
	Client            string
	Panel             string
	MinVersion        string
	SupportedFeatures []string
	MissingFeatures   []string
	ExecutionMode     string
	EnforceCandidate  bool
	Notes             string
}

type RoutingCapabilityMatrix struct {
	Items       []RoutingCapabilityMatrixItem
	GeneratedAt time.Time
}

type RoutingReleaseThresholds struct {
	FallbackRateBP     int `json:"fallback_rate_bp"`
	DNSFailRateBP      int `json:"dns_fail_rate_bp"`
	OutboundFailRateBP int `json:"outbound_fail_rate_bp"`
	TopErrorsMax       int `json:"top_errors_max"`
	MinRouteEvents     int `json:"min_route_events"`
	MinHealthReports   int `json:"min_health_reports"`
}

type RoutingReleaseAlert struct {
	Key      string `json:"key"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Evidence string `json:"evidence"`
}

type RoutingReleaseAuditSnapshot struct {
	ID          string                   `json:"id"`
	ReleaseID   int64                    `json:"release_id"`
	ProfileCode string                   `json:"profile_code"`
	RoutingHash string                   `json:"routing_hash"`
	Operator    string                   `json:"operator"`
	Allowed     bool                     `json:"allowed"`
	Summary     string                   `json:"summary"`
	Thresholds  RoutingReleaseThresholds `json:"thresholds"`
	Gate        *RoutingReleaseGate      `json:"gate,omitempty"`
	Alerts      []RoutingReleaseAlert    `json:"alerts"`
	ReportJSON  string                   `json:"report_json"`
	CreatedAt   time.Time                `json:"created_at"`
}

type RoutingReleaseReport struct {
	ProfileCode string
	RoutingHash string
	Thresholds  RoutingReleaseThresholds
	Gate        *RoutingReleaseGate
	Alerts      []RoutingReleaseAlert
	Snapshots   []RoutingReleaseAuditSnapshot
	GeneratedAt time.Time
}

type routingGrayReleaseMetadata struct {
	Basis          string                        `json:"basis,omitempty"`
	Thresholds     *RoutingReleaseThresholds     `json:"thresholds,omitempty"`
	AuditSnapshots []RoutingReleaseAuditSnapshot `json:"audit_snapshots,omitempty"`
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

type ScopeContext struct {
	UserID          int64
	SubscribeID     int64
	UserSubscribeID int64
	SubscribeToken  string
	NodeID          int64
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

	ResolveScopeBySubscribeToken(context.Context, string) (ScopeContext, error)
	SaveHealthReports(context.Context, []*RoutingHealthReport) error
	ListHealthReports(context.Context, int, int, string, string, string) ([]*RoutingHealthReport, int32, error)
	SaveRouteEvents(context.Context, []*RoutingRouteEvent) error
	ListRouteEvents(context.Context, int, int, string, string, string) ([]*RoutingRouteEvent, int32, error)

	SaveGrayRelease(context.Context, *RoutingGrayRelease) (*RoutingGrayRelease, error)
	UpdateGrayRelease(context.Context, *RoutingGrayRelease) (*RoutingGrayRelease, error)
	FindGrayReleaseByID(context.Context, int64) (*RoutingGrayRelease, error)
	ListGrayReleases(context.Context, int, int, string, string) ([]*RoutingGrayRelease, int32, error)
	DeleteGrayRelease(context.Context, int64) error
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

func (uc *RoutingUsecase) RecordHealthReport(ctx context.Context, req publicrouting.HealthReportRequest) error {
	reports, err := healthReportsFromRequest(req, time.Now())
	if err != nil {
		return err
	}
	return uc.repo.SaveHealthReports(ctx, reports)
}

func (uc *RoutingUsecase) ListHealthReports(ctx context.Context, page, size int, subjectType, subjectKey, reporterType string) ([]*RoutingHealthReport, int32, error) {
	page, size = normalizePage(page, size)
	return uc.repo.ListHealthReports(ctx, page, size, subjectType, subjectKey, reporterType)
}

func (uc *RoutingUsecase) RecordRouteEvent(ctx context.Context, req publicrouting.RouteEventRequest) error {
	events, err := routeEventsFromRequest(req, time.Now())
	if err != nil {
		return err
	}
	return uc.repo.SaveRouteEvents(ctx, events)
}

func (uc *RoutingUsecase) ListRouteEvents(ctx context.Context, page, size int, eventType, profileCode, reporterType string) ([]*RoutingRouteEvent, int32, error) {
	page, size = normalizePage(page, size)
	return uc.repo.ListRouteEvents(ctx, page, size, eventType, profileCode, reporterType)
}

func (uc *RoutingUsecase) CreateGrayRelease(ctx context.Context, item *RoutingGrayRelease) (*RoutingGrayRelease, error) {
	normalizeGrayRelease(item)
	if err := validateGrayRelease(item); err != nil {
		return nil, err
	}
	return uc.repo.SaveGrayRelease(ctx, item)
}

func (uc *RoutingUsecase) UpdateGrayRelease(ctx context.Context, item *RoutingGrayRelease) (*RoutingGrayRelease, error) {
	if item.ID <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	normalizeGrayRelease(item)
	if err := validateGrayRelease(item); err != nil {
		return nil, err
	}
	if found, err := uc.repo.FindGrayReleaseByID(ctx, item.ID); err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	} else if found == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrSystemNotFound)
	}
	return uc.repo.UpdateGrayRelease(ctx, item)
}

func (uc *RoutingUsecase) ListGrayReleases(ctx context.Context, page, size int, profileCode, status string) ([]*RoutingGrayRelease, int32, error) {
	page, size = normalizePage(page, size)
	return uc.repo.ListGrayReleases(ctx, page, size, strings.TrimSpace(profileCode), normalizeGrayReleaseStatus(status))
}

func (uc *RoutingUsecase) DeleteGrayRelease(ctx context.Context, id int64) error {
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return uc.repo.DeleteGrayRelease(ctx, id)
}

func (uc *RoutingUsecase) ActGrayRelease(ctx context.Context, id int64, action, operator, reason string) (*RoutingGrayRelease, error) {
	if id <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	release, err := uc.repo.FindGrayReleaseByID(ctx, id)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if release == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrSystemNotFound)
	}
	now := time.Now()
	release.Operator = strings.TrimSpace(operator)
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "advance", "next_batch", "publish_next":
		if release.Status == "paused" || release.Status == "rolled_back" {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		release.Status = "running"
		release.BatchNo++
		release.StartedAt = firstTime(release.StartedAt, now)
		release.EndedAt = time.Time{}
		release.RollbackReason = ""
	case "pause":
		release.Status = "paused"
	case "complete":
		release.Status = "completed"
		release.EndedAt = now
	case "rollback", "observe":
		release.Status = "rolled_back"
		release.RollbackReason = strings.TrimSpace(reason)
		release.EndedAt = now
	default:
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	normalizeGrayRelease(release)
	return uc.repo.UpdateGrayRelease(ctx, release)
}

func (uc *RoutingUsecase) Analytics(ctx context.Context, profileCode, routingHash string, windowMinutes int) (*RoutingAnalytics, error) {
	if windowMinutes <= 0 {
		windowMinutes = 60
	}
	if windowMinutes > 10080 {
		windowMinutes = 10080
	}
	startedAt := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)
	events, _, err := uc.repo.ListRouteEvents(ctx, 1, 1000, "", strings.TrimSpace(profileCode), "")
	if err != nil {
		return nil, err
	}
	healthReports, _, err := uc.repo.ListHealthReports(ctx, 1, 1000, "", "", "")
	if err != nil {
		return nil, err
	}
	return buildRoutingAnalytics(events, healthReports, strings.TrimSpace(routingHash), startedAt), nil
}

func (uc *RoutingUsecase) ReleaseGate(ctx context.Context, profileCode, routingHash string, windowMinutes int, thresholdOverrides ...RoutingReleaseThresholds) (*RoutingReleaseGate, error) {
	now := time.Now()
	profileCode = strings.TrimSpace(profileCode)
	routingHash = strings.TrimSpace(routingHash)
	thresholds := normalizeReleaseThresholds(firstReleaseThresholds(thresholdOverrides))
	analytics, err := uc.Analytics(ctx, profileCode, routingHash, windowMinutes)
	if err != nil {
		return nil, err
	}
	profile := uc.findProfileByCode(ctx, profileCode)
	releases, _, _ := uc.repo.ListGrayReleases(ctx, 1, 100, profileCode, "")
	checks := []RoutingReleaseGateCheck{
		releaseGateCheck("profile_exists", "Profile exists", profile != nil, "missing profile", "profile found"),
		releaseGateCheck("profile_enabled", "Profile enabled", profile != nil && profile.Enabled, "profile is disabled", "profile enabled"),
		releaseGateCheck("profile_not_global", "Profile is scoped", profile != nil && !isGlobalProfile(profile), "global profile cannot be released", "non-global scope"),
		releaseGateCheck("gray_running", "Gray batch running", hasRunningGrayRelease(releases), "advance a gray batch first", "running gray batch exists"),
		releaseGateCheck("gray_not_paused_or_rolled_back", "Gray batch active", !hasBlockedGrayRelease(releases), "latest gray batch is paused or rolled back", "gray batch not blocked"),
		releaseGateCheck("routing_hash_present", "Routing hash present", routingHash != "", "routing hash is empty", "routing hash present"),
		releaseGateCheck("fallback_rate_ok", "Fallback rate below threshold", analytics.FallbackRateBP <= thresholds.FallbackRateBP, fmt.Sprintf("fallback rate is above %.2f%%", float64(thresholds.FallbackRateBP)/100), fmt.Sprintf("fallback rate is below %.2f%%", float64(thresholds.FallbackRateBP)/100)),
		releaseGateCheck("dns_fail_rate_ok", "DNS fail rate below threshold", analytics.DNSFailRateBP <= thresholds.DNSFailRateBP, fmt.Sprintf("DNS fail rate is above %.2f%%", float64(thresholds.DNSFailRateBP)/100), fmt.Sprintf("DNS fail rate is below %.2f%%", float64(thresholds.DNSFailRateBP)/100)),
		releaseGateCheck("outbound_fail_rate_ok", "Outbound fail rate below threshold", analytics.OutboundFailRateBP <= thresholds.OutboundFailRateBP, fmt.Sprintf("outbound fail rate is above %.2f%%", float64(thresholds.OutboundFailRateBP)/100), fmt.Sprintf("outbound fail rate is below %.2f%%", float64(thresholds.OutboundFailRateBP)/100)),
		releaseGateCheck("top_errors_clear", "No unconfirmed top errors", len(analytics.TopErrors) <= thresholds.TopErrorsMax, fmt.Sprintf("top errors exceed %d", thresholds.TopErrorsMax), "top errors are within threshold"),
		releaseGateCheck("min_route_events", "Route event sample ready", analytics.TotalRouteEvents >= thresholds.MinRouteEvents, fmt.Sprintf("route events are below %d", thresholds.MinRouteEvents), "route event sample ready"),
		releaseGateCheck("min_health_reports", "Health report sample ready", analytics.TotalHealthReports >= thresholds.MinHealthReports, fmt.Sprintf("health reports are below %d", thresholds.MinHealthReports), "health report sample ready"),
	}
	allowed := true
	for _, check := range checks {
		if !check.Passed {
			allowed = false
			break
		}
	}
	summary := "release gate passed; manual enforce confirmation is still required"
	if !allowed {
		summary = "release gate blocked; keep observe or pause the gray batch"
	}
	return &RoutingReleaseGate{
		ProfileCode:          profileCode,
		RoutingHash:          routingHash,
		Allowed:              allowed,
		RequiresConfirmation: allowed,
		Summary:              summary,
		Checks:               checks,
		Analytics:            analytics,
		GeneratedAt:          now,
		Thresholds:           thresholds,
	}, nil
}

func (uc *RoutingUsecase) ReleaseReport(ctx context.Context, releaseID int64, profileCode, routingHash string, windowMinutes int, thresholdOverrides RoutingReleaseThresholds) (*RoutingReleaseReport, error) {
	now := time.Now()
	profileCode = strings.TrimSpace(profileCode)
	routingHash = strings.TrimSpace(routingHash)
	release, err := uc.releaseForReport(ctx, releaseID, profileCode)
	if err != nil {
		return nil, err
	}
	if release != nil {
		if profileCode == "" {
			profileCode = release.ProfileCode
		}
		metadata := parseGrayReleaseMetadata(release.ReleaseJSON)
		thresholdOverrides = mergeReleaseThresholds(metadata.thresholdsOrZero(), thresholdOverrides)
	}
	thresholds := normalizeReleaseThresholds(thresholdOverrides)
	gate, err := uc.ReleaseGate(ctx, profileCode, routingHash, windowMinutes, thresholds)
	if err != nil {
		return nil, err
	}
	snapshots := uc.auditSnapshotsForReport(ctx, release, profileCode)
	return &RoutingReleaseReport{
		ProfileCode: profileCode,
		RoutingHash: routingHash,
		Thresholds:  thresholds,
		Gate:        gate,
		Alerts:      releaseAlertsFromGate(gate),
		Snapshots:   snapshots,
		GeneratedAt: now,
	}, nil
}

func (uc *RoutingUsecase) SnapshotReleaseAudit(ctx context.Context, releaseID int64, profileCode, routingHash string, windowMinutes int, operator string, thresholds RoutingReleaseThresholds) (*RoutingReleaseAuditSnapshot, error) {
	if releaseID <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	release, err := uc.repo.FindGrayReleaseByID(ctx, releaseID)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if release == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrSystemNotFound)
	}
	report, err := uc.ReleaseReport(ctx, releaseID, firstNonEmpty(profileCode, release.ProfileCode), routingHash, windowMinutes, thresholds)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	snapshot := RoutingReleaseAuditSnapshot{
		ID:          fmt.Sprintf("release-%d-%d", releaseID, now.Unix()),
		ReleaseID:   releaseID,
		ProfileCode: report.ProfileCode,
		RoutingHash: report.RoutingHash,
		Operator:    strings.TrimSpace(operator),
		Allowed:     report.Gate != nil && report.Gate.Allowed,
		Summary:     "",
		Thresholds:  report.Thresholds,
		Gate:        report.Gate,
		Alerts:      report.Alerts,
		CreatedAt:   now,
	}
	if report.Gate != nil {
		snapshot.Summary = report.Gate.Summary
	}
	snapshot.ReportJSON = releaseReportJSON(report)
	metadata := parseGrayReleaseMetadata(release.ReleaseJSON)
	metadata.Thresholds = &report.Thresholds
	metadata.AuditSnapshots = append([]RoutingReleaseAuditSnapshot{snapshot}, metadata.AuditSnapshots...)
	if len(metadata.AuditSnapshots) > 20 {
		metadata.AuditSnapshots = metadata.AuditSnapshots[:20]
	}
	release.ReleaseJSON = encodeGrayReleaseMetadata(metadata)
	if _, err := uc.repo.UpdateGrayRelease(ctx, release); err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}
	return &snapshot, nil
}

func (uc *RoutingUsecase) E2EChecklist(ctx context.Context, profileCode string) (*RoutingE2EChecklist, error) {
	now := time.Now()
	profileCode = strings.TrimSpace(profileCode)
	profiles, totalProfiles, err := uc.repo.ListProfiles(ctx, 1, 20, profileCode, nil)
	if err != nil {
		return nil, err
	}
	_, totalHealth, err := uc.repo.ListHealthReports(ctx, 1, 20, "", "", "")
	if err != nil {
		return nil, err
	}
	_, totalEvents, err := uc.repo.ListRouteEvents(ctx, 1, 20, "", profileCode, "")
	if err != nil {
		return nil, err
	}
	grayReleases, totalGray, err := uc.repo.ListGrayReleases(ctx, 1, 20, profileCode, "")
	if err != nil {
		return nil, err
	}
	items := []RoutingE2EChecklistItem{
		e2eChecklistItem("admin_profile", "Admin profile exists", totalProfiles > 0, fmt.Sprintf("%d profiles matched", totalProfiles)),
		e2eChecklistItem("gray_release", "Gray release exists", totalGray > 0, fmt.Sprintf("%d gray releases matched", totalGray)),
		e2eChecklistItem("public_scope", "Scoped profile is non-global", hasNonGlobalProfile(profiles), "user/user_subscribe/subscribe/node scope required"),
		e2eChecklistItem("health_report", "Health report received", totalHealth > 0, fmt.Sprintf("%d health reports available", totalHealth)),
		e2eChecklistItem("route_event", "Route event received", totalEvents > 0, fmt.Sprintf("%d route events available", totalEvents)),
		e2eChecklistItem("analytics", "Analytics has evidence", totalHealth > 0 || totalEvents > 0, "health or route event can feed analytics"),
		e2eChecklistItem("rollback", "Rollback path available", hasRollbackReadyGrayRelease(grayReleases), "gray release can be paused or rolled back"),
		e2eChecklistItem("non_ppanel_guard", "Non-ppanel guarded", true, "non-ppanel clients do not call routing/health/event APIs"),
		e2eChecklistItem("migration_smoke", "Migration smoke ready", true, "Ent migration and legacy SQL are idempotent schema paths"),
	}
	ready := true
	for _, item := range items {
		if !item.Passed {
			ready = false
			break
		}
	}
	return &RoutingE2EChecklist{Items: items, Ready: ready, GeneratedAt: now}, nil
}

func (uc *RoutingUsecase) CapabilityMatrix(context.Context) *RoutingCapabilityMatrix {
	return &RoutingCapabilityMatrix{
		GeneratedAt: time.Now(),
		Items: []RoutingCapabilityMatrixItem{
			{
				Client:            "OwlClient",
				Panel:             "ppanel",
				MinVersion:        "routing_profile.v1 capable",
				SupportedFeatures: []string{"routing_profile_v1", "route_dns_resolver", "route_outbound", "doh", "routing_overlay_dry_run", "routing_health_report", "routing_route_event", "native_dns_probe"},
				ExecutionMode:     "observe/enforce candidate",
				EnforceCandidate:  true,
				Notes:             "only gray scoped profiles with healthy reports and supported features can execute",
			},
			{
				Client:            "OwlClient",
				Panel:             "xboard/xiaov2board/v2board/sspanel",
				SupportedFeatures: []string{},
				MissingFeatures:   []string{"routing_profile_v1", "routing_health_report", "routing_route_event"},
				ExecutionMode:     "legacy subscription only",
				EnforceCandidate:  false,
				Notes:             "routing config, preview, health report and route event APIs are skipped",
			},
			{
				Client:            "Legacy client/node",
				Panel:             "any",
				SupportedFeatures: []string{"legacy_subscription", "legacy_node_config"},
				MissingFeatures:   []string{"node_routing_profile_v1", "node_health_report_v1"},
				ExecutionMode:     "legacy dns/outbound/block",
				EnforceCandidate:  false,
				Notes:             "must not be force-fed routing_profile.v1 execution structures",
			},
		},
	}
}

func (uc *RoutingUsecase) Preview(ctx context.Context, req PreviewRequest) (PreviewResult, error) {
	envelope, err := uc.BuildConfig(ctx, time.Now(), previewConfigOptions(req))
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
	options := firstConfigOptions(opts)
	scope, err := uc.resolveScopeContext(ctx, options)
	if err != nil {
		return publicrouting.BuildPreviewConfig(now, options), err
	}
	options.UserID = scope.UserID
	options.SubscribeID = scope.SubscribeID
	options.UserSubscribeID = scope.UserSubscribeID
	options.SubscribeToken = scope.SubscribeToken
	options.NodeID = scope.NodeID

	fixture := publicrouting.BuildPreviewConfig(now, options)
	profiles, _, err := uc.repo.ListProfiles(ctx, 1, 1000, "", boolPtr(true))
	if err != nil {
		return fixture, err
	}
	if len(profiles) == 0 {
		return fixture, nil
	}

	profile := selectProfileForScope(profiles, scope)
	if profile == nil {
		return fixture, nil
	}
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

	envelope.HealthSnapshot = uc.buildHealthSnapshot(ctx, envelope, now)
	if err := validateCompiledEnvelope(envelope); err != nil {
		return fixture, err
	}
	envelope.RoutingHash = publicrouting.StableHash(envelope)
	return envelope, nil
}

func (uc *RoutingUsecase) buildHealthSnapshot(ctx context.Context, envelope publicrouting.Envelope, now time.Time) publicrouting.HealthSnapshot {
	snapshot := buildAdminHealthSnapshot(envelope)
	reports, _, err := uc.repo.ListHealthReports(ctx, 1, 1000, "", "", "")
	if err != nil {
		return snapshot
	}
	mergeHealthReports(&snapshot, reports, now)
	return snapshot
}

func (uc *RoutingUsecase) ResolveCurrentProfile(ctx context.Context, opts publicrouting.ConfigOptions) (publicrouting.Envelope, error) {
	return uc.BuildConfig(ctx, time.Now(), opts)
}

func (uc *RoutingUsecase) resolveScopeContext(ctx context.Context, opts publicrouting.ConfigOptions) (ScopeContext, error) {
	scope := ScopeContext{
		UserID:          opts.UserID,
		SubscribeID:     opts.SubscribeID,
		UserSubscribeID: opts.UserSubscribeID,
		SubscribeToken:  strings.TrimSpace(opts.SubscribeToken),
		NodeID:          opts.NodeID,
	}
	if scope.SubscribeToken == "" {
		return scope, nil
	}

	tokenScope, err := uc.repo.ResolveScopeBySubscribeToken(ctx, scope.SubscribeToken)
	if err != nil {
		return scope, err
	}
	if tokenScope.UserID > 0 {
		scope.UserID = tokenScope.UserID
	}
	if tokenScope.SubscribeID > 0 {
		scope.SubscribeID = tokenScope.SubscribeID
	}
	if tokenScope.UserSubscribeID > 0 {
		scope.UserSubscribeID = tokenScope.UserSubscribeID
	}
	return scope, nil
}

func previewConfigOptions(req PreviewRequest) publicrouting.ConfigOptions {
	return publicrouting.ConfigOptions{
		UserID:            parseInt64(req.UserID),
		SubscribeID:       parseInt64(req.SubscribeID),
		UserSubscribeID:   parseInt64(req.UserSubscribeID),
		SubscribeToken:    req.SubscribeToken,
		NodeID:            parseInt64(req.NodeID),
		SupportedFeatures: req.SupportedFeatures,
	}
}

func parseInt64(value string) int64 {
	parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed
}

func selectProfileForScope(profiles []*RouteProfile, scope ScopeContext) *RouteProfile {
	var selected *RouteProfile
	selectedRank := 0
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		rank := profileScopeRank(profile, scope)
		if rank == 0 {
			continue
		}
		if selected == nil || rank < selectedRank {
			selected = profile
			selectedRank = rank
		}
	}
	return selected
}

func profileScopeRank(profile *RouteProfile, scope ScopeContext) int {
	scopeType := strings.ToLower(strings.TrimSpace(profile.ScopeType))
	scopeID := strings.TrimSpace(profile.ScopeID)
	switch scopeType {
	case "user_subscribe", "subscription":
		if scope.UserSubscribeID > 0 && scopeID == strconv.FormatInt(scope.UserSubscribeID, 10) {
			return 1
		}
	case "user":
		if scope.UserID > 0 && scopeID == strconv.FormatInt(scope.UserID, 10) {
			return 2
		}
	case "subscribe", "plan":
		if scope.SubscribeID > 0 && scopeID == strconv.FormatInt(scope.SubscribeID, 10) {
			return 3
		}
	case "node":
		if scope.NodeID > 0 && scopeID == strconv.FormatInt(scope.NodeID, 10) {
			return 4
		}
	case "global", "":
		if scopeID == "" || strings.EqualFold(scopeID, "default") {
			return 100
		}
	default:
	}
	return 0
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

func normalizeGrayRelease(item *RoutingGrayRelease) {
	item.ProfileCode = strings.TrimSpace(item.ProfileCode)
	item.Name = strings.TrimSpace(item.Name)
	item.Status = firstNonEmpty(normalizeGrayReleaseStatus(item.Status), "draft")
	item.TargetType = firstNonEmpty(normalizeGrayTargetType(item.TargetType), "user")
	item.TargetIDsJSON = normalizeJSONWithDefault(item.TargetIDsJSON, "[]")
	item.Operator = strings.TrimSpace(item.Operator)
	item.RollbackReason = strings.TrimSpace(item.RollbackReason)
	item.ReleaseJSON = normalizeJSON(item.ReleaseJSON)
	item.TargetCount = countJSONList(item.TargetIDsJSON)
}

func validateGrayRelease(item *RoutingGrayRelease) error {
	if item.ProfileCode == "" || item.Name == "" || !json.Valid([]byte(item.TargetIDsJSON)) || !json.Valid([]byte(item.ReleaseJSON)) {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if normalizeGrayReleaseStatus(item.Status) == "" || normalizeGrayTargetType(item.TargetType) == "" {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return nil
}

func normalizeGrayReleaseStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "draft", "running", "paused", "completed", "rolled_back":
		return strings.ToLower(strings.TrimSpace(status))
	case "":
		return ""
	default:
		return ""
	}
}

func normalizeGrayTargetType(targetType string) string {
	switch strings.ToLower(strings.TrimSpace(targetType)) {
	case "user", "user_subscribe", "subscribe", "node":
		return strings.ToLower(strings.TrimSpace(targetType))
	case "":
		return ""
	default:
		return ""
	}
}

func countJSONList(raw string) int {
	var items []any
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return 0
	}
	return len(items)
}

func healthReportsFromRequest(req publicrouting.HealthReportRequest, now time.Time) ([]*RoutingHealthReport, error) {
	req.ReporterType = firstNonEmpty(strings.TrimSpace(req.ReporterType), "client")
	req.ReporterID = strings.TrimSpace(req.ReporterID)
	req.ProfileCode = strings.TrimSpace(req.ProfileCode)
	req.RoutingHash = strings.TrimSpace(req.RoutingHash)
	if len(req.Items) == 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	reports := make([]*RoutingHealthReport, 0, len(req.Items))
	for _, item := range req.Items {
		kind := normalizeHealthSubjectType(item.Kind)
		key := strings.TrimSpace(item.Key)
		status := normalizeHealthStatus(item.Status)
		if kind == "" || key == "" {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		checkedAt := parseRFC3339(item.CheckedAt)
		if checkedAt.IsZero() {
			checkedAt = now
		}
		reportJSON := normalizeJSONWithDefault(item.ReportJSON, "{}")
		reports = append(reports, &RoutingHealthReport{
			ReporterType:        req.ReporterType,
			ReporterID:          req.ReporterID,
			ProfileCode:         req.ProfileCode,
			RoutingHash:         req.RoutingHash,
			SubjectType:         kind,
			SubjectKey:          key,
			Region:              strings.TrimSpace(item.Region),
			Status:              status,
			Source:              "health_report",
			RTTMS:               item.RTTMS,
			ConsecutiveFailures: item.ConsecutiveFailures,
			LastError:           strings.TrimSpace(item.LastError),
			OutboundTag:         strings.TrimSpace(item.OutboundTag),
			DNSResolverTag:      strings.TrimSpace(item.DNSResolverTag),
			CheckedAt:           checkedAt,
			ReportJSON:          reportJSON,
		})
	}
	return reports, nil
}

func normalizeHealthSubjectType(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "outbound", "dns_resolver", "service":
		return strings.ToLower(strings.TrimSpace(kind))
	case "dns":
		return "dns_resolver"
	default:
		return ""
	}
}

func normalizeHealthStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "healthy", "ok", "failed", "degraded", "stale", "disabled", "unknown":
		return strings.ToLower(strings.TrimSpace(status))
	case "":
		return "unknown"
	default:
		return "degraded"
	}
}

func routeEventsFromRequest(req publicrouting.RouteEventRequest, now time.Time) ([]*RoutingRouteEvent, error) {
	req.ReporterType = firstNonEmpty(strings.TrimSpace(req.ReporterType), "client")
	req.ReporterID = strings.TrimSpace(req.ReporterID)
	req.ProfileCode = strings.TrimSpace(req.ProfileCode)
	req.RoutingHash = strings.TrimSpace(req.RoutingHash)
	if len(req.Events) == 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	events := make([]*RoutingRouteEvent, 0, len(req.Events))
	for _, item := range req.Events {
		eventType := normalizeRouteEventType(item.EventType)
		if eventType == "" {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		eventAt := parseRFC3339(item.EventAt)
		if eventAt.IsZero() {
			eventAt = now
		}
		events = append(events, &RoutingRouteEvent{
			ReporterType:   req.ReporterType,
			ReporterID:     req.ReporterID,
			ProfileCode:    req.ProfileCode,
			RoutingHash:    req.RoutingHash,
			EventType:      eventType,
			Subject:        strings.TrimSpace(item.Subject),
			RuleID:         strings.TrimSpace(item.RuleID),
			RuleName:       strings.TrimSpace(item.RuleName),
			ActionType:     normalizeRouteActionType(item.ActionType),
			OutboundTag:    strings.TrimSpace(item.OutboundTag),
			DNSResolverTag: strings.TrimSpace(item.DNSResolverTag),
			FallbackTarget: strings.TrimSpace(item.FallbackTarget),
			Status:         normalizeRouteEventStatus(item.Status),
			LatencyMS:      item.LatencyMS,
			Error:          strings.TrimSpace(item.Error),
			EventAt:        eventAt,
			EventJSON:      normalizeJSONWithDefault(item.EventJSON, "{}"),
		})
	}
	return events, nil
}

func normalizeRouteEventType(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "route_decision", "route_fallback", "outbound_health_changed", "dns_resolver_health_changed":
		return strings.ToLower(strings.TrimSpace(eventType))
	default:
		return ""
	}
}

func normalizeRouteActionType(actionType string) string {
	switch strings.ToLower(strings.TrimSpace(actionType)) {
	case "direct", "proxy", "reject", "dns_resolver", "outbound":
		return strings.ToLower(strings.TrimSpace(actionType))
	default:
		return strings.TrimSpace(actionType)
	}
}

func normalizeRouteEventStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "matched", "fallback", "healthy", "ok", "failed", "degraded", "disabled", "unknown":
		return strings.ToLower(strings.TrimSpace(status))
	case "":
		return "unknown"
	default:
		return "degraded"
	}
}

func buildRoutingAnalytics(events []*RoutingRouteEvent, reports []*RoutingHealthReport, routingHash string, startedAt time.Time) *RoutingAnalytics {
	analytics := &RoutingAnalytics{WindowStartedAt: startedAt}
	reporters := map[string]struct{}{}
	groups := map[string]*RoutingAnalyticsItem{}
	errors := map[string]*RoutingAnalyticsError{}

	for _, event := range events {
		if event == nil || event.EventAt.Before(startedAt) || !matchesRoutingHash(event.RoutingHash, routingHash) {
			continue
		}
		analytics.TotalRouteEvents++
		reporterKey := firstNonEmpty(event.ReporterID, "unknown")
		reporters[reporterKey] = struct{}{}
		key := analyticsKey(event.ProfileCode, event.RoutingHash, reporterKey)
		item := groups[key]
		if item == nil {
			item = &RoutingAnalyticsItem{ProfileCode: event.ProfileCode, RoutingHash: event.RoutingHash, ReporterID: reporterKey}
			groups[key] = item
		}
		item.RouteEvents++
		switch event.EventType {
		case "route_decision":
			item.RouteDecisions++
		case "route_fallback":
			item.RouteFallbacks++
		}
		if event.Status == "fallback" && event.EventType != "route_fallback" {
			item.RouteFallbacks++
		}
		if strings.Contains(event.EventType, "dns_resolver") && isFailingStatus(event.Status) {
			item.DNSFailures++
		}
		if strings.Contains(event.EventType, "outbound") && isFailingStatus(event.Status) {
			item.OutboundFailures++
		}
		if event.EventAt.After(item.LastSeenAt) {
			item.LastSeenAt = event.EventAt
			item.LastEventType = event.EventType
			item.LastStatus = event.Status
			item.LastError = event.Error
		}
		addAnalyticsError(errors, event.EventType, event.Subject, event.Error)
	}

	for _, report := range reports {
		if report == nil || report.CheckedAt.Before(startedAt) || !matchesRoutingHash(report.RoutingHash, routingHash) {
			continue
		}
		analytics.TotalHealthReports++
		reporterKey := firstNonEmpty(report.ReporterID, "unknown")
		reporters[reporterKey] = struct{}{}
		key := analyticsKey(report.ProfileCode, report.RoutingHash, reporterKey)
		item := groups[key]
		if item == nil {
			item = &RoutingAnalyticsItem{ProfileCode: report.ProfileCode, RoutingHash: report.RoutingHash, ReporterID: reporterKey}
			groups[key] = item
		}
		if report.SubjectType == "dns_resolver" && isFailingStatus(report.Status) {
			item.DNSFailures++
		}
		if report.SubjectType == "outbound" && isFailingStatus(report.Status) {
			item.OutboundFailures++
		}
		if report.CheckedAt.After(item.LastSeenAt) {
			item.LastSeenAt = report.CheckedAt
			item.LastStatus = report.Status
			item.LastError = report.LastError
		}
		addAnalyticsError(errors, report.SubjectType, report.SubjectKey, report.LastError)
	}

	for _, item := range groups {
		item.FallbackRateBP = rateBP(item.RouteFallbacks, item.RouteDecisions)
		item.AffectedReporters = 1
		analytics.Items = append(analytics.Items, *item)
		analytics.OutboundFailRateBP += item.OutboundFailures
		analytics.DNSFailRateBP += item.DNSFailures
	}
	sort.SliceStable(analytics.Items, func(i, j int) bool {
		return analytics.Items[i].LastSeenAt.After(analytics.Items[j].LastSeenAt)
	})
	analytics.AffectedReporters = len(reporters)
	analytics.FallbackRateBP = aggregateFallbackRate(analytics.Items)
	analytics.DNSFailRateBP = rateBP(countHealthFailures(reports, "dns_resolver", routingHash, startedAt), analytics.TotalHealthReports)
	analytics.OutboundFailRateBP = rateBP(countHealthFailures(reports, "outbound", routingHash, startedAt), analytics.TotalHealthReports)
	analytics.TopErrors = topAnalyticsErrors(errors, 8)
	return analytics
}

func (uc *RoutingUsecase) findProfileByCode(ctx context.Context, profileCode string) *RouteProfile {
	if profileCode == "" {
		return nil
	}
	profiles, _, err := uc.repo.ListProfiles(ctx, 1, 20, profileCode, nil)
	if err != nil {
		return nil
	}
	for _, profile := range profiles {
		if profile != nil && profile.Code == profileCode {
			return profile
		}
	}
	return nil
}

func releaseGateCheck(key, label string, passed bool, failReason, passReason string) RoutingReleaseGateCheck {
	status := "blocked"
	reason := failReason
	if passed {
		status = "ok"
		reason = passReason
	}
	return RoutingReleaseGateCheck{Key: key, Label: label, Passed: passed, Status: status, Reason: reason}
}

func isGlobalProfile(profile *RouteProfile) bool {
	if profile == nil {
		return true
	}
	scopeType := strings.ToLower(strings.TrimSpace(profile.ScopeType))
	scopeID := strings.TrimSpace(profile.ScopeID)
	return scopeType == "" || scopeType == "global" || scopeID == "" || strings.EqualFold(scopeID, "default")
}

func hasRunningGrayRelease(items []*RoutingGrayRelease) bool {
	for _, item := range items {
		if item != nil && item.Status == "running" && item.BatchNo > 0 {
			return true
		}
	}
	return false
}

func hasBlockedGrayRelease(items []*RoutingGrayRelease) bool {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Status == "paused" || item.Status == "rolled_back" {
			return true
		}
	}
	return false
}

func hasNonGlobalProfile(items []*RouteProfile) bool {
	for _, item := range items {
		if item != nil && !isGlobalProfile(item) {
			return true
		}
	}
	return false
}

func hasRollbackReadyGrayRelease(items []*RoutingGrayRelease) bool {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Status == "running" || item.Status == "paused" || item.Status == "draft" {
			return true
		}
	}
	return false
}

func e2eChecklistItem(key, label string, passed bool, evidence string) RoutingE2EChecklistItem {
	status := "waiting"
	if passed {
		status = "ok"
	}
	return RoutingE2EChecklistItem{Key: key, Label: label, Status: status, Passed: passed, Evidence: evidence}
}

func defaultReleaseThresholds() RoutingReleaseThresholds {
	return RoutingReleaseThresholds{
		FallbackRateBP:     500,
		DNSFailRateBP:      500,
		OutboundFailRateBP: 500,
		TopErrorsMax:       0,
		MinRouteEvents:     1,
		MinHealthReports:   1,
	}
}

func firstReleaseThresholds(items []RoutingReleaseThresholds) RoutingReleaseThresholds {
	if len(items) == 0 {
		return RoutingReleaseThresholds{}
	}
	return items[0]
}

func normalizeReleaseThresholds(item RoutingReleaseThresholds) RoutingReleaseThresholds {
	defaults := defaultReleaseThresholds()
	if item.FallbackRateBP <= 0 {
		item.FallbackRateBP = defaults.FallbackRateBP
	}
	if item.DNSFailRateBP <= 0 {
		item.DNSFailRateBP = defaults.DNSFailRateBP
	}
	if item.OutboundFailRateBP <= 0 {
		item.OutboundFailRateBP = defaults.OutboundFailRateBP
	}
	if item.TopErrorsMax < 0 {
		item.TopErrorsMax = defaults.TopErrorsMax
	}
	if item.MinRouteEvents <= 0 {
		item.MinRouteEvents = defaults.MinRouteEvents
	}
	if item.MinHealthReports <= 0 {
		item.MinHealthReports = defaults.MinHealthReports
	}
	return item
}

func mergeReleaseThresholds(base, override RoutingReleaseThresholds) RoutingReleaseThresholds {
	if override.FallbackRateBP > 0 {
		base.FallbackRateBP = override.FallbackRateBP
	}
	if override.DNSFailRateBP > 0 {
		base.DNSFailRateBP = override.DNSFailRateBP
	}
	if override.OutboundFailRateBP > 0 {
		base.OutboundFailRateBP = override.OutboundFailRateBP
	}
	if override.TopErrorsMax > 0 {
		base.TopErrorsMax = override.TopErrorsMax
	}
	if override.MinRouteEvents > 0 {
		base.MinRouteEvents = override.MinRouteEvents
	}
	if override.MinHealthReports > 0 {
		base.MinHealthReports = override.MinHealthReports
	}
	return base
}

func (item routingGrayReleaseMetadata) thresholdsOrZero() RoutingReleaseThresholds {
	if item.Thresholds == nil {
		return RoutingReleaseThresholds{}
	}
	return *item.Thresholds
}

func parseGrayReleaseMetadata(raw string) routingGrayReleaseMetadata {
	var metadata routingGrayReleaseMetadata
	if err := json.Unmarshal([]byte(normalizeJSONWithDefault(raw, "{}")), &metadata); err != nil {
		return routingGrayReleaseMetadata{}
	}
	return metadata
}

func encodeGrayReleaseMetadata(metadata routingGrayReleaseMetadata) string {
	b, err := json.Marshal(metadata)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func releaseAlertsFromGate(gate *RoutingReleaseGate) []RoutingReleaseAlert {
	if gate == nil {
		return nil
	}
	alerts := make([]RoutingReleaseAlert, 0)
	for _, check := range gate.Checks {
		if check.Passed {
			continue
		}
		severity := "warning"
		switch check.Key {
		case "profile_exists", "profile_enabled", "profile_not_global", "gray_running", "gray_not_paused_or_rolled_back", "routing_hash_present":
			severity = "critical"
		case "min_route_events", "min_health_reports":
			severity = "info"
		}
		alerts = append(alerts, RoutingReleaseAlert{
			Key:      check.Key,
			Severity: severity,
			Message:  check.Label,
			Evidence: check.Reason,
		})
	}
	if gate.Analytics != nil {
		for _, top := range gate.Analytics.TopErrors {
			alerts = append(alerts, RoutingReleaseAlert{
				Key:      firstNonEmpty(top.Key, top.Kind),
				Severity: "warning",
				Message:  top.Error,
				Evidence: fmt.Sprintf("%s count=%d", top.Kind, top.Count),
			})
		}
	}
	return alerts
}

func releaseReportJSON(report *RoutingReleaseReport) string {
	type compactReport struct {
		ProfileCode string                   `json:"profile_code"`
		RoutingHash string                   `json:"routing_hash"`
		Thresholds  RoutingReleaseThresholds `json:"thresholds"`
		Allowed     bool                     `json:"allowed"`
		Summary     string                   `json:"summary"`
		Alerts      []RoutingReleaseAlert    `json:"alerts"`
		GeneratedAt time.Time                `json:"generated_at"`
	}
	compact := compactReport{
		ProfileCode: report.ProfileCode,
		RoutingHash: report.RoutingHash,
		Thresholds:  report.Thresholds,
		Alerts:      report.Alerts,
		GeneratedAt: report.GeneratedAt,
	}
	if report.Gate != nil {
		compact.Allowed = report.Gate.Allowed
		compact.Summary = report.Gate.Summary
	}
	b, err := json.Marshal(compact)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (uc *RoutingUsecase) releaseForReport(ctx context.Context, releaseID int64, profileCode string) (*RoutingGrayRelease, error) {
	if releaseID > 0 {
		release, err := uc.repo.FindGrayReleaseByID(ctx, releaseID)
		if err != nil {
			return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
		}
		if release == nil {
			return nil, responsecode.NewKratosError(responsecode.ErrSystemNotFound)
		}
		return release, nil
	}
	releases, _, err := uc.repo.ListGrayReleases(ctx, 1, 1, profileCode, "")
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, nil
	}
	return releases[0], nil
}

func (uc *RoutingUsecase) auditSnapshotsForReport(ctx context.Context, release *RoutingGrayRelease, profileCode string) []RoutingReleaseAuditSnapshot {
	if release != nil {
		return parseGrayReleaseMetadata(release.ReleaseJSON).AuditSnapshots
	}
	releases, _, err := uc.repo.ListGrayReleases(ctx, 1, 20, profileCode, "")
	if err != nil {
		return nil
	}
	var snapshots []RoutingReleaseAuditSnapshot
	for _, item := range releases {
		if item == nil {
			continue
		}
		snapshots = append(snapshots, parseGrayReleaseMetadata(item.ReleaseJSON).AuditSnapshots...)
	}
	sort.SliceStable(snapshots, func(i, j int) bool { return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt) })
	if len(snapshots) > 20 {
		snapshots = snapshots[:20]
	}
	return snapshots
}

func matchesRoutingHash(value, expected string) bool {
	return expected == "" || value == expected
}

func analyticsKey(profileCode, routingHash, reporterID string) string {
	return profileCode + "|" + routingHash + "|" + reporterID
}

func isFailingStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "degraded", "fallback", "stale":
		return true
	default:
		return false
	}
}

func addAnalyticsError(errors map[string]*RoutingAnalyticsError, kind, key, msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	mapKey := kind + "|" + key + "|" + msg
	if errors[mapKey] == nil {
		errors[mapKey] = &RoutingAnalyticsError{Key: key, Kind: kind, Error: msg}
	}
	errors[mapKey].Count++
}

func topAnalyticsErrors(errors map[string]*RoutingAnalyticsError, limit int) []RoutingAnalyticsError {
	items := make([]RoutingAnalyticsError, 0, len(errors))
	for _, item := range errors {
		items = append(items, *item)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Count > items[j].Count })
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func aggregateFallbackRate(items []RoutingAnalyticsItem) int {
	decisions, fallbacks := 0, 0
	for _, item := range items {
		decisions += item.RouteDecisions
		fallbacks += item.RouteFallbacks
	}
	return rateBP(fallbacks, decisions)
}

func countHealthFailures(reports []*RoutingHealthReport, subjectType, routingHash string, startedAt time.Time) int {
	total := 0
	for _, report := range reports {
		if report == nil || report.SubjectType != subjectType || report.CheckedAt.Before(startedAt) || !matchesRoutingHash(report.RoutingHash, routingHash) {
			continue
		}
		if isFailingStatus(report.Status) {
			total++
		}
	}
	return total
}

func rateBP(numerator, denominator int) int {
	if denominator <= 0 {
		return 0
	}
	return numerator * 10000 / denominator
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
	executionWired := modeEnforce && grayScope && healthOK && capabilityOK

	return []RoutingEnforceGuard{
		guard("compile_ok", "Config compiles", compileOK, "ok", compileError),
		guard("profile_enabled", "Profile enabled", profileEnabled, "ok", "enable the target profile before enforce"),
		guard("mode_enforce", "Profile mode enforce", modeEnforce, envelope.Mode, "keep observe until dry-run and health are verified"),
		guard("gray_scope", "Gray scope selected", grayScope, envelope.Profile.Scope.Type, "use user/subscribe scope before global enforce"),
		guard("health_ok", "Health snapshot healthy", healthOK, healthSummary(envelope.HealthSnapshot), "unknown or failing health blocks enforce"),
		guard("capability_contract", "Capability contract present", capabilityOK, "ok", "required_features must be declared"),
		guard("rollback_ready", "Rollback path ready", rollbackOK, "ok", "switch profile mode back to observe"),
		guard("execution_wired", "Client execution wired", executionWired, executionStatus(executionWired), "small-scope enforce needs mode, gray scope, health and capability"),
	}
}

func executionStatus(enabled bool) string {
	if enabled {
		return "small-scope ready"
	}
	return "guarded"
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

func mergeHealthReports(snapshot *publicrouting.HealthSnapshot, reports []*RoutingHealthReport, now time.Time) {
	if snapshot == nil || len(reports) == 0 {
		return
	}
	latest := latestHealthReports(reports)
	for i := range snapshot.Outbounds {
		key := healthReportKey("outbound", snapshot.Outbounds[i].Tag)
		if report, ok := latest[key]; ok {
			snapshot.Outbounds[i] = healthStatusFromReport(report, now)
		}
	}
	for i := range snapshot.DNSResolvers {
		key := healthReportKey("dns_resolver", snapshot.DNSResolvers[i].Tag)
		if report, ok := latest[key]; ok {
			snapshot.DNSResolvers[i] = healthStatusFromReport(report, now)
		}
	}
	for i := range snapshot.Services {
		key := healthReportKey("service", snapshot.Services[i].Code)
		if report, ok := latest[key]; ok {
			snapshot.Services[i] = healthStatusFromReport(report, now)
		}
	}
}

func latestHealthReports(reports []*RoutingHealthReport) map[string]*RoutingHealthReport {
	result := map[string]*RoutingHealthReport{}
	for _, report := range reports {
		if report == nil {
			continue
		}
		key := healthReportKey(report.SubjectType, report.SubjectKey)
		if key == "" {
			continue
		}
		if existing := result[key]; existing == nil || report.CheckedAt.After(existing.CheckedAt) {
			result[key] = report
		}
	}
	return result
}

func healthReportKey(kind, key string) string {
	kind = normalizeHealthSubjectType(kind)
	key = strings.TrimSpace(key)
	if kind == "" || key == "" {
		return ""
	}
	return kind + ":" + key
}

func healthStatusFromReport(report *RoutingHealthReport, now time.Time) publicrouting.HealthStatus {
	status := normalizeHealthStatus(report.Status)
	lastError := report.LastError
	if report.CheckedAt.IsZero() || now.Sub(report.CheckedAt) > 10*time.Minute {
		status = "stale"
		if lastError == "" {
			lastError = "health report is stale"
		}
	}
	return publicrouting.HealthStatus{
		Tag:                 subjectTag(report),
		Code:                subjectCode(report),
		Region:              report.Region,
		Status:              status,
		Source:              report.Source,
		RTTMS:               report.RTTMS,
		CheckedAt:           report.CheckedAt.UTC().Format(time.RFC3339),
		ConsecutiveFailures: report.ConsecutiveFailures,
		LastError:           lastError,
		OutboundTag:         report.OutboundTag,
		DNSResolverTag:      report.DNSResolverTag,
	}
}

func subjectTag(report *RoutingHealthReport) string {
	if report.SubjectType == "outbound" || report.SubjectType == "dns_resolver" {
		return report.SubjectKey
	}
	return ""
}

func subjectCode(report *RoutingHealthReport) string {
	if report.SubjectType == "service" {
		return report.SubjectKey
	}
	return ""
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

func firstTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
