package data

import (
	"context"
	"encoding/json"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyroutingdnsresolver"
	"github.com/npanel-dev/NPanel-backend/ent/proxyroutinggrayrelease"
	"github.com/npanel-dev/NPanel-backend/ent/proxyroutinghealthreport"
	"github.com/npanel-dev/NPanel-backend/ent/proxyroutingoutbound"
	"github.com/npanel-dev/NPanel-backend/ent/proxyroutingprofile"
	"github.com/npanel-dev/NPanel-backend/ent/proxyroutingrouteevent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyroutingrule"
	"github.com/npanel-dev/NPanel-backend/ent/proxyroutingunlockservice"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	routingbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/routing"
)

type adminRoutingRepo struct {
	data   *Data
	logger *log.Helper
}

func NewAdminRoutingRepo(data *Data, logger log.Logger) routingbiz.RoutingRepo {
	return &adminRoutingRepo{data: data, logger: log.NewHelper(logger)}
}

func (r *adminRoutingRepo) SaveProfile(ctx context.Context, item *routingbiz.RouteProfile) (*routingbiz.RouteProfile, error) {
	po, err := r.data.db.ProxyRoutingProfile.Create().
		SetCode(item.Code).
		SetName(item.Name).
		SetDescription(item.Description).
		SetScopeType(item.ScopeType).
		SetScopeID(item.ScopeID).
		SetPriority(item.Priority).
		SetMode(item.Mode).
		SetEnabled(item.Enabled).
		SetProfileJSON(item.ProfileJSON).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return profileToModel(po), nil
}

func (r *adminRoutingRepo) UpdateProfile(ctx context.Context, item *routingbiz.RouteProfile) (*routingbiz.RouteProfile, error) {
	po, err := r.data.db.ProxyRoutingProfile.UpdateOneID(item.ID).
		SetCode(item.Code).
		SetName(item.Name).
		SetDescription(item.Description).
		SetScopeType(item.ScopeType).
		SetScopeID(item.ScopeID).
		SetPriority(item.Priority).
		SetMode(item.Mode).
		SetEnabled(item.Enabled).
		SetProfileJSON(item.ProfileJSON).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return profileToModel(po), nil
}

func (r *adminRoutingRepo) FindProfileByID(ctx context.Context, id int64) (*routingbiz.RouteProfile, error) {
	po, err := r.data.db.ProxyRoutingProfile.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return profileToModel(po), nil
}

func (r *adminRoutingRepo) ListProfiles(ctx context.Context, page, size int, search string, enabled *bool) ([]*routingbiz.RouteProfile, int32, error) {
	query := r.data.db.ProxyRoutingProfile.Query()
	if search != "" {
		query = query.Where(proxyroutingprofile.Or(
			proxyroutingprofile.CodeContainsFold(search),
			proxyroutingprofile.NameContainsFold(search),
			proxyroutingprofile.ScopeTypeContainsFold(search),
			proxyroutingprofile.ScopeIDContainsFold(search),
		))
	}
	if enabled != nil {
		query = query.Where(proxyroutingprofile.Enabled(*enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Order(func(s *sql.Selector) {
		s.OrderBy(sql.Asc(proxyroutingprofile.FieldPriority), sql.Desc(proxyroutingprofile.FieldUpdatedAt))
	}).Offset((page - 1) * size).Limit(size).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*routingbiz.RouteProfile, 0, len(pos))
	for _, po := range pos {
		items = append(items, profileToModel(po))
	}
	return items, int32(total), nil
}

func (r *adminRoutingRepo) DeleteProfile(ctx context.Context, id int64) error {
	return r.data.db.ProxyRoutingProfile.DeleteOneID(id).Exec(ctx)
}

func (r *adminRoutingRepo) SaveRule(ctx context.Context, item *routingbiz.RouteRule) (*routingbiz.RouteRule, error) {
	po, err := r.data.db.ProxyRoutingRule.Create().
		SetProfileID(item.ProfileID).
		SetName(item.Name).
		SetPriority(item.Priority).
		SetEnabled(item.Enabled).
		SetServiceCode(item.ServiceCode).
		SetMatcherJSON(item.MatcherJSON).
		SetActionJSON(item.ActionJSON).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return ruleToModel(po), nil
}

func (r *adminRoutingRepo) UpdateRule(ctx context.Context, item *routingbiz.RouteRule) (*routingbiz.RouteRule, error) {
	po, err := r.data.db.ProxyRoutingRule.UpdateOneID(item.ID).
		SetProfileID(item.ProfileID).
		SetName(item.Name).
		SetPriority(item.Priority).
		SetEnabled(item.Enabled).
		SetServiceCode(item.ServiceCode).
		SetMatcherJSON(item.MatcherJSON).
		SetActionJSON(item.ActionJSON).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return ruleToModel(po), nil
}

func (r *adminRoutingRepo) FindRuleByID(ctx context.Context, id int64) (*routingbiz.RouteRule, error) {
	po, err := r.data.db.ProxyRoutingRule.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return ruleToModel(po), nil
}

func (r *adminRoutingRepo) ListRules(ctx context.Context, page, size int, profileID int64, search string, enabled *bool) ([]*routingbiz.RouteRule, int32, error) {
	query := r.data.db.ProxyRoutingRule.Query()
	if profileID > 0 {
		query = query.Where(proxyroutingrule.ProfileID(profileID))
	}
	if search != "" {
		query = query.Where(proxyroutingrule.Or(proxyroutingrule.NameContainsFold(search), proxyroutingrule.ServiceCodeContainsFold(search)))
	}
	if enabled != nil {
		query = query.Where(proxyroutingrule.Enabled(*enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Order(func(s *sql.Selector) {
		s.OrderBy(sql.Asc(proxyroutingrule.FieldPriority), sql.Desc(proxyroutingrule.FieldUpdatedAt))
	}).Offset((page - 1) * size).Limit(size).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*routingbiz.RouteRule, 0, len(pos))
	for _, po := range pos {
		items = append(items, ruleToModel(po))
	}
	return items, int32(total), nil
}

func (r *adminRoutingRepo) DeleteRule(ctx context.Context, id int64) error {
	return r.data.db.ProxyRoutingRule.DeleteOneID(id).Exec(ctx)
}

func (r *adminRoutingRepo) SaveDNSResolver(ctx context.Context, item *routingbiz.DNSResolver) (*routingbiz.DNSResolver, error) {
	po, err := r.data.db.ProxyRoutingDNSResolver.Create().
		SetTag(item.Tag).
		SetName(item.Name).
		SetProto(item.Proto).
		SetAddress(item.Address).
		SetPort(item.Port).
		SetEnabled(item.Enabled).
		SetResolverJSON(item.ResolverJSON).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return dnsResolverToModel(po), nil
}

func (r *adminRoutingRepo) UpdateDNSResolver(ctx context.Context, item *routingbiz.DNSResolver) (*routingbiz.DNSResolver, error) {
	po, err := r.data.db.ProxyRoutingDNSResolver.UpdateOneID(item.ID).
		SetTag(item.Tag).
		SetName(item.Name).
		SetProto(item.Proto).
		SetAddress(item.Address).
		SetPort(item.Port).
		SetEnabled(item.Enabled).
		SetResolverJSON(item.ResolverJSON).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return dnsResolverToModel(po), nil
}

func (r *adminRoutingRepo) FindDNSResolverByID(ctx context.Context, id int64) (*routingbiz.DNSResolver, error) {
	po, err := r.data.db.ProxyRoutingDNSResolver.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return dnsResolverToModel(po), nil
}

func (r *adminRoutingRepo) ListDNSResolvers(ctx context.Context, page, size int, search string, enabled *bool) ([]*routingbiz.DNSResolver, int32, error) {
	query := r.data.db.ProxyRoutingDNSResolver.Query()
	if search != "" {
		query = query.Where(proxyroutingdnsresolver.Or(proxyroutingdnsresolver.TagContainsFold(search), proxyroutingdnsresolver.NameContainsFold(search)))
	}
	if enabled != nil {
		query = query.Where(proxyroutingdnsresolver.Enabled(*enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Order(proxyroutingdnsresolver.ByTag()).Offset((page - 1) * size).Limit(size).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*routingbiz.DNSResolver, 0, len(pos))
	for _, po := range pos {
		items = append(items, dnsResolverToModel(po))
	}
	return items, int32(total), nil
}

func (r *adminRoutingRepo) DeleteDNSResolver(ctx context.Context, id int64) error {
	return r.data.db.ProxyRoutingDNSResolver.DeleteOneID(id).Exec(ctx)
}

func (r *adminRoutingRepo) SaveOutbound(ctx context.Context, item *routingbiz.RouteOutbound) (*routingbiz.RouteOutbound, error) {
	po, err := r.data.db.ProxyRoutingOutbound.Create().
		SetTag(item.Tag).
		SetName(item.Name).
		SetType(item.Type).
		SetRegion(item.Region).
		SetEnabled(item.Enabled).
		SetOutboundJSON(item.OutboundJSON).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return outboundToModel(po), nil
}

func (r *adminRoutingRepo) UpdateOutbound(ctx context.Context, item *routingbiz.RouteOutbound) (*routingbiz.RouteOutbound, error) {
	po, err := r.data.db.ProxyRoutingOutbound.UpdateOneID(item.ID).
		SetTag(item.Tag).
		SetName(item.Name).
		SetType(item.Type).
		SetRegion(item.Region).
		SetEnabled(item.Enabled).
		SetOutboundJSON(item.OutboundJSON).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return outboundToModel(po), nil
}

func (r *adminRoutingRepo) FindOutboundByID(ctx context.Context, id int64) (*routingbiz.RouteOutbound, error) {
	po, err := r.data.db.ProxyRoutingOutbound.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return outboundToModel(po), nil
}

func (r *adminRoutingRepo) ListOutbounds(ctx context.Context, page, size int, search string, enabled *bool) ([]*routingbiz.RouteOutbound, int32, error) {
	query := r.data.db.ProxyRoutingOutbound.Query()
	if search != "" {
		query = query.Where(proxyroutingoutbound.Or(proxyroutingoutbound.TagContainsFold(search), proxyroutingoutbound.NameContainsFold(search)))
	}
	if enabled != nil {
		query = query.Where(proxyroutingoutbound.Enabled(*enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Order(proxyroutingoutbound.ByTag()).Offset((page - 1) * size).Limit(size).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*routingbiz.RouteOutbound, 0, len(pos))
	for _, po := range pos {
		items = append(items, outboundToModel(po))
	}
	return items, int32(total), nil
}

func (r *adminRoutingRepo) DeleteOutbound(ctx context.Context, id int64) error {
	return r.data.db.ProxyRoutingOutbound.DeleteOneID(id).Exec(ctx)
}

func (r *adminRoutingRepo) SaveUnlockService(ctx context.Context, item *routingbiz.UnlockService) (*routingbiz.UnlockService, error) {
	po, err := r.data.db.ProxyRoutingUnlockService.Create().
		SetCode(item.Code).
		SetName(item.Name).
		SetCategory(item.Category).
		SetEnabled(item.Enabled).
		SetServiceJSON(item.ServiceJSON).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return unlockServiceToModel(po), nil
}

func (r *adminRoutingRepo) UpdateUnlockService(ctx context.Context, item *routingbiz.UnlockService) (*routingbiz.UnlockService, error) {
	po, err := r.data.db.ProxyRoutingUnlockService.UpdateOneID(item.ID).
		SetCode(item.Code).
		SetName(item.Name).
		SetCategory(item.Category).
		SetEnabled(item.Enabled).
		SetServiceJSON(item.ServiceJSON).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return unlockServiceToModel(po), nil
}

func (r *adminRoutingRepo) FindUnlockServiceByID(ctx context.Context, id int64) (*routingbiz.UnlockService, error) {
	po, err := r.data.db.ProxyRoutingUnlockService.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return unlockServiceToModel(po), nil
}

func (r *adminRoutingRepo) ListUnlockServices(ctx context.Context, page, size int, search string, enabled *bool) ([]*routingbiz.UnlockService, int32, error) {
	query := r.data.db.ProxyRoutingUnlockService.Query()
	if search != "" {
		query = query.Where(proxyroutingunlockservice.Or(proxyroutingunlockservice.CodeContainsFold(search), proxyroutingunlockservice.NameContainsFold(search)))
	}
	if enabled != nil {
		query = query.Where(proxyroutingunlockservice.Enabled(*enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Order(proxyroutingunlockservice.ByCode()).Offset((page - 1) * size).Limit(size).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*routingbiz.UnlockService, 0, len(pos))
	for _, po := range pos {
		items = append(items, unlockServiceToModel(po))
	}
	return items, int32(total), nil
}

func (r *adminRoutingRepo) DeleteUnlockService(ctx context.Context, id int64) error {
	return r.data.db.ProxyRoutingUnlockService.DeleteOneID(id).Exec(ctx)
}

func (r *adminRoutingRepo) ResolveScopeBySubscribeToken(ctx context.Context, token string) (routingbiz.ScopeContext, error) {
	if token == "" {
		return routingbiz.ScopeContext{}, nil
	}
	userSub, err := r.data.db.ProxyUserSubscribe.Query().
		Where(proxyusersubscribe.TokenEQ(token)).
		Only(ctx)
	if err != nil {
		return routingbiz.ScopeContext{}, err
	}
	return routingbiz.ScopeContext{
		UserID:          userSub.UserID,
		SubscribeID:     userSub.SubscribeID,
		UserSubscribeID: userSub.ID,
		SubscribeToken:  token,
	}, nil
}

func (r *adminRoutingRepo) SaveHealthReports(ctx context.Context, reports []*routingbiz.RoutingHealthReport) error {
	for _, report := range reports {
		if report == nil {
			continue
		}
		if _, err := r.data.db.ProxyRoutingHealthReport.Create().
			SetReporterType(report.ReporterType).
			SetReporterID(report.ReporterID).
			SetProfileCode(report.ProfileCode).
			SetRoutingHash(report.RoutingHash).
			SetSubjectType(report.SubjectType).
			SetSubjectKey(report.SubjectKey).
			SetRegion(report.Region).
			SetStatus(report.Status).
			SetSource(report.Source).
			SetRttMs(report.RTTMS).
			SetConsecutiveFailures(report.ConsecutiveFailures).
			SetLastError(report.LastError).
			SetOutboundTag(report.OutboundTag).
			SetDNSResolverTag(report.DNSResolverTag).
			SetCheckedAt(report.CheckedAt).
			SetReportJSON(report.ReportJSON).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *adminRoutingRepo) ListHealthReports(ctx context.Context, page, size int, subjectType, subjectKey, reporterType string) ([]*routingbiz.RoutingHealthReport, int32, error) {
	query := r.data.db.ProxyRoutingHealthReport.Query()
	if subjectType != "" {
		query = query.Where(proxyroutinghealthreport.SubjectTypeEQ(subjectType))
	}
	if subjectKey != "" {
		query = query.Where(proxyroutinghealthreport.SubjectKeyContainsFold(subjectKey))
	}
	if reporterType != "" {
		query = query.Where(proxyroutinghealthreport.ReporterTypeEQ(reporterType))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Order(proxyroutinghealthreport.ByCheckedAt(sql.OrderDesc()), proxyroutinghealthreport.ByID(sql.OrderDesc())).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*routingbiz.RoutingHealthReport, 0, len(pos))
	for _, po := range pos {
		items = append(items, healthReportToModel(po))
	}
	return items, int32(total), nil
}

func (r *adminRoutingRepo) SaveRouteEvents(ctx context.Context, events []*routingbiz.RoutingRouteEvent) error {
	for _, event := range events {
		if event == nil {
			continue
		}
		if _, err := r.data.db.ProxyRoutingRouteEvent.Create().
			SetReporterType(event.ReporterType).
			SetReporterID(event.ReporterID).
			SetProfileCode(event.ProfileCode).
			SetRoutingHash(event.RoutingHash).
			SetEventType(event.EventType).
			SetSubject(event.Subject).
			SetRuleID(event.RuleID).
			SetRuleName(event.RuleName).
			SetActionType(event.ActionType).
			SetOutboundTag(event.OutboundTag).
			SetDNSResolverTag(event.DNSResolverTag).
			SetFallbackTarget(event.FallbackTarget).
			SetStatus(event.Status).
			SetLatencyMs(event.LatencyMS).
			SetError(event.Error).
			SetEventAt(event.EventAt).
			SetEventJSON(event.EventJSON).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *adminRoutingRepo) ListRouteEvents(ctx context.Context, page, size int, eventType, profileCode, reporterType string) ([]*routingbiz.RoutingRouteEvent, int32, error) {
	query := r.data.db.ProxyRoutingRouteEvent.Query()
	if eventType != "" {
		query = query.Where(proxyroutingrouteevent.EventTypeEQ(eventType))
	}
	if profileCode != "" {
		query = query.Where(proxyroutingrouteevent.ProfileCodeContainsFold(profileCode))
	}
	if reporterType != "" {
		query = query.Where(proxyroutingrouteevent.ReporterTypeEQ(reporterType))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Order(proxyroutingrouteevent.ByEventAt(sql.OrderDesc()), proxyroutingrouteevent.ByID(sql.OrderDesc())).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*routingbiz.RoutingRouteEvent, 0, len(pos))
	for _, po := range pos {
		items = append(items, routeEventToModel(po))
	}
	return items, int32(total), nil
}

func (r *adminRoutingRepo) SaveGrayRelease(ctx context.Context, item *routingbiz.RoutingGrayRelease) (*routingbiz.RoutingGrayRelease, error) {
	create := r.data.db.ProxyRoutingGrayRelease.Create().
		SetProfileCode(item.ProfileCode).
		SetName(item.Name).
		SetStatus(item.Status).
		SetBatchNo(item.BatchNo).
		SetTargetType(item.TargetType).
		SetTargetIdsJSON(item.TargetIDsJSON).
		SetOperator(item.Operator).
		SetRollbackReason(item.RollbackReason).
		SetReleaseJSON(item.ReleaseJSON)
	if !item.StartedAt.IsZero() {
		create.SetStartedAt(item.StartedAt)
	}
	if !item.EndedAt.IsZero() {
		create.SetEndedAt(item.EndedAt)
	}
	po, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return grayReleaseToModel(po), nil
}

func (r *adminRoutingRepo) UpdateGrayRelease(ctx context.Context, item *routingbiz.RoutingGrayRelease) (*routingbiz.RoutingGrayRelease, error) {
	update := r.data.db.ProxyRoutingGrayRelease.UpdateOneID(item.ID).
		SetProfileCode(item.ProfileCode).
		SetName(item.Name).
		SetStatus(item.Status).
		SetBatchNo(item.BatchNo).
		SetTargetType(item.TargetType).
		SetTargetIdsJSON(item.TargetIDsJSON).
		SetOperator(item.Operator).
		SetRollbackReason(item.RollbackReason).
		SetReleaseJSON(item.ReleaseJSON)
	if item.StartedAt.IsZero() {
		update.ClearStartedAt()
	} else {
		update.SetStartedAt(item.StartedAt)
	}
	if item.EndedAt.IsZero() {
		update.ClearEndedAt()
	} else {
		update.SetEndedAt(item.EndedAt)
	}
	po, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return grayReleaseToModel(po), nil
}

func (r *adminRoutingRepo) FindGrayReleaseByID(ctx context.Context, id int64) (*routingbiz.RoutingGrayRelease, error) {
	po, err := r.data.db.ProxyRoutingGrayRelease.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return grayReleaseToModel(po), nil
}

func (r *adminRoutingRepo) ListGrayReleases(ctx context.Context, page, size int, profileCode, status string) ([]*routingbiz.RoutingGrayRelease, int32, error) {
	query := r.data.db.ProxyRoutingGrayRelease.Query()
	if profileCode != "" {
		query = query.Where(proxyroutinggrayrelease.ProfileCodeContainsFold(profileCode))
	}
	if status != "" {
		query = query.Where(proxyroutinggrayrelease.StatusEQ(status))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := query.Order(proxyroutinggrayrelease.ByUpdatedAt(sql.OrderDesc()), proxyroutinggrayrelease.ByID(sql.OrderDesc())).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*routingbiz.RoutingGrayRelease, 0, len(pos))
	for _, po := range pos {
		items = append(items, grayReleaseToModel(po))
	}
	return items, int32(total), nil
}

func (r *adminRoutingRepo) DeleteGrayRelease(ctx context.Context, id int64) error {
	return r.data.db.ProxyRoutingGrayRelease.DeleteOneID(id).Exec(ctx)
}

func profileToModel(po *ent.ProxyRoutingProfile) *routingbiz.RouteProfile {
	return &routingbiz.RouteProfile{
		ID:          po.ID,
		Code:        po.Code,
		Name:        po.Name,
		Description: po.Description,
		ScopeType:   po.ScopeType,
		ScopeID:     po.ScopeID,
		Priority:    po.Priority,
		Mode:        po.Mode,
		Enabled:     po.Enabled,
		ProfileJSON: po.ProfileJSON,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func healthReportToModel(po *ent.ProxyRoutingHealthReport) *routingbiz.RoutingHealthReport {
	return &routingbiz.RoutingHealthReport{
		ID:                  po.ID,
		ReporterType:        po.ReporterType,
		ReporterID:          po.ReporterID,
		ProfileCode:         po.ProfileCode,
		RoutingHash:         po.RoutingHash,
		SubjectType:         po.SubjectType,
		SubjectKey:          po.SubjectKey,
		Region:              po.Region,
		Status:              po.Status,
		Source:              po.Source,
		RTTMS:               po.RttMs,
		ConsecutiveFailures: po.ConsecutiveFailures,
		LastError:           po.LastError,
		OutboundTag:         po.OutboundTag,
		DNSResolverTag:      po.DNSResolverTag,
		CheckedAt:           po.CheckedAt,
		ReportJSON:          po.ReportJSON,
		CreatedAt:           po.CreatedAt,
		UpdatedAt:           po.UpdatedAt,
	}
}

func routeEventToModel(po *ent.ProxyRoutingRouteEvent) *routingbiz.RoutingRouteEvent {
	return &routingbiz.RoutingRouteEvent{
		ID:             po.ID,
		ReporterType:   po.ReporterType,
		ReporterID:     po.ReporterID,
		ProfileCode:    po.ProfileCode,
		RoutingHash:    po.RoutingHash,
		EventType:      po.EventType,
		Subject:        po.Subject,
		RuleID:         po.RuleID,
		RuleName:       po.RuleName,
		ActionType:     po.ActionType,
		OutboundTag:    po.OutboundTag,
		DNSResolverTag: po.DNSResolverTag,
		FallbackTarget: po.FallbackTarget,
		Status:         po.Status,
		LatencyMS:      po.LatencyMs,
		Error:          po.Error,
		EventAt:        po.EventAt,
		EventJSON:      po.EventJSON,
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

func grayReleaseToModel(po *ent.ProxyRoutingGrayRelease) *routingbiz.RoutingGrayRelease {
	return &routingbiz.RoutingGrayRelease{
		ID:             po.ID,
		ProfileCode:    po.ProfileCode,
		Name:           po.Name,
		Status:         po.Status,
		BatchNo:        po.BatchNo,
		TargetType:     po.TargetType,
		TargetIDsJSON:  po.TargetIdsJSON,
		Operator:       po.Operator,
		RollbackReason: po.RollbackReason,
		StartedAt:      po.StartedAt,
		EndedAt:        po.EndedAt,
		ReleaseJSON:    po.ReleaseJSON,
		TargetCount:    countJSONList(po.TargetIdsJSON),
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

func ruleToModel(po *ent.ProxyRoutingRule) *routingbiz.RouteRule {
	return &routingbiz.RouteRule{
		ID:          po.ID,
		ProfileID:   po.ProfileID,
		Name:        po.Name,
		Priority:    po.Priority,
		Enabled:     po.Enabled,
		ServiceCode: po.ServiceCode,
		MatcherJSON: po.MatcherJSON,
		ActionJSON:  po.ActionJSON,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func dnsResolverToModel(po *ent.ProxyRoutingDNSResolver) *routingbiz.DNSResolver {
	return &routingbiz.DNSResolver{
		ID:           po.ID,
		Tag:          po.Tag,
		Name:         po.Name,
		Proto:        po.Proto,
		Address:      po.Address,
		Port:         po.Port,
		Enabled:      po.Enabled,
		ResolverJSON: po.ResolverJSON,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}

func outboundToModel(po *ent.ProxyRoutingOutbound) *routingbiz.RouteOutbound {
	return &routingbiz.RouteOutbound{
		ID:           po.ID,
		Tag:          po.Tag,
		Name:         po.Name,
		Type:         po.Type,
		Region:       po.Region,
		Enabled:      po.Enabled,
		OutboundJSON: po.OutboundJSON,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}

func unlockServiceToModel(po *ent.ProxyRoutingUnlockService) *routingbiz.UnlockService {
	return &routingbiz.UnlockService{
		ID:          po.ID,
		Code:        po.Code,
		Name:        po.Name,
		Category:    po.Category,
		Enabled:     po.Enabled,
		ServiceJSON: po.ServiceJSON,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func countJSONList(raw string) int {
	var items []any
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return 0
	}
	return len(items)
}
