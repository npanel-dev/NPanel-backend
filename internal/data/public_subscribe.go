package data

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyserver"
	"github.com/npanel-dev/NPanel-backend/ent/proxyservergroup"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribecategory"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribepriceoption"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	subscribeBiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/subscribe"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
	productlanguage "github.com/npanel-dev/NPanel-backend/internal/pkg/language"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

type publicSubscribeRepo struct {
	data *Data
	log  *log.Helper
}

type publicSubscribeTrafficLimit struct {
	StatType     string `json:"stat_type"`
	StatValue    int64  `json:"stat_value"`
	TrafficUsage int64  `json:"traffic_usage"`
	SpeedLimit   int64  `json:"speed_limit"`
}

// NewPublicSubscribeRepo 创建Public Subscribe仓库
func NewPublicSubscribeRepo(data *Data, logger log.Logger) subscribeBiz.SubscribeRepo {
	return &publicSubscribeRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// QuerySubscribeList 查询订阅列表
func (r *publicSubscribeRepo) QuerySubscribeList(ctx context.Context, language string, categoryID int64) ([]*subscribeBiz.Subscribe, int32, error) {
	language = productlanguage.NormalizeProductLanguage(language)
	// 查询条件: sell=true
	query := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.Sell(true),
		)

	// 语言过滤：与老项目 DefaultLanguage=true 行为保持一致
	if language != "" {
		query = query.Where(
			proxysubscribe.Or(
				proxysubscribe.Language(language),
				proxysubscribe.Language(""),
			),
		)
	} else {
		query = query.Where(proxysubscribe.Language(""))
	}

	if categoryID > 0 {
		query = query.Where(proxysubscribe.CategoryIDEQ(categoryID))
	}

	// 查询
	subscribes, err := query.Order(ent.Asc(proxysubscribe.FieldSort)).All(ctx)
	if err != nil {
		r.log.Errorf("QuerySubscribeList query error: %v", err)
		return nil, 0, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	total := int32(len(subscribes))
	result := make([]*subscribeBiz.Subscribe, 0, len(subscribes))
	categoryNames := r.publicSubscribeCategoryNames(ctx, subscribes)
	priceOptions := r.publicSubscribePriceOptions(ctx, subscribes)

	for _, s := range subscribes {
		// 处理Description（指针类型）
		desc := ""
		if s.Description != nil {
			desc = *s.Description
		}
		shortDescription := ""
		if s.ShortDescription != nil {
			shortDescription = *s.ShortDescription
		}
		features := ""
		if s.Features != nil {
			features = *s.Features
		}
		detailContent := ""
		if s.DetailContent != nil {
			detailContent = *s.DetailContent
		}

		// 处理ResetCycle和DeductionRatio（指针类型）
		resetCycle := int64(0)
		deductionRatio := int64(0)
		if s.ResetCycle != nil {
			resetCycle = int64(*s.ResetCycle)
		}
		if s.DeductionRatio != nil {
			deductionRatio = int64(*s.DeductionRatio)
		}

		// 处理Nodes（与老项目一致，按字符串列表解析）
		nodes64 := tool.StringToInt64Slice(s.Nodes)
		nodes := make([]int, 0, len(nodes64))
		for _, nodeID := range nodes64 {
			nodes = append(nodes, int(nodeID))
		}

		// 处理NodeTags（与老项目一致，按逗号分隔）
		var nodeTags []string
		if s.NodeTags != "" {
			for _, item := range strings.Split(s.NodeTags, ",") {
				if trimmed := strings.TrimSpace(item); trimmed != "" {
					nodeTags = append(nodeTags, trimmed)
				}
			}
		}

		nodeGroupID := int64(0)
		if s.NodeGroupID != nil {
			nodeGroupID = *s.NodeGroupID
		}

		var trafficLimit []*subscribeBiz.TrafficLimit
		if s.TrafficLimit != nil && strings.TrimSpace(*s.TrafficLimit) != "" {
			var limits []publicSubscribeTrafficLimit
			if err := json.Unmarshal([]byte(*s.TrafficLimit), &limits); err == nil {
				trafficLimit = make([]*subscribeBiz.TrafficLimit, 0, len(limits))
				for _, limit := range limits {
					trafficLimit = append(trafficLimit, &subscribeBiz.TrafficLimit{
						StatType:     limit.StatType,
						StatValue:    limit.StatValue,
						TrafficUsage: limit.TrafficUsage,
						SpeedLimit:   limit.SpeedLimit,
					})
				}
			}
		}

		item := &subscribeBiz.Subscribe{
			ID:                int64(s.ID),
			Name:              s.Name,
			Language:          s.Language,
			Description:       desc,
			ShortDescription:  shortDescription,
			Features:          features,
			DetailFormat:      s.DetailFormat,
			DetailContent:     detailContent,
			UnitPrice:         s.UnitPrice,
			UnitTime:          s.UnitTime,
			Replacement:       int64(s.Replacement),
			Inventory:         int64(s.Inventory),
			Traffic:           s.Traffic,
			SpeedLimit:        int64(s.SpeedLimit),
			DeviceLimit:       int64(s.DeviceLimit),
			Quota:             int64(s.Quota),
			CategoryID:        s.CategoryID,
			CategoryName:      categoryNames[s.CategoryID],
			Nodes:             nodes,
			NodeTags:          nodeTags,
			NodeGroupIds:      append([]int64{}, s.NodeGroupIds...),
			NodeGroupId:       nodeGroupID,
			TrafficLimit:      trafficLimit,
			Show:              s.Show,
			Sell:              s.Sell,
			Sort:              int64(s.Sort),
			DeductionRatio:    deductionRatio,
			AllowDeduction:    s.AllowDeduction,
			ResetCycle:        resetCycle,
			RenewalReset:      s.RenewalReset,
			ShowOriginalPrice: s.ShowOriginalPrice,
			PriceOptions:      priceOptions[int64(s.ID)],
			CreatedAt:         s.CreatedAt.UnixMilli(),
			UpdatedAt:         s.UpdatedAt.UnixMilli(),
		}

		// 解析Discount字段（指针类型）
		if s.Discount != nil && *s.Discount != "" {
			var discounts []*subscribeBiz.SubscribeDiscount
			if err := json.Unmarshal([]byte(*s.Discount), &discounts); err == nil {
				item.Discount = discounts
			}
		}

		result = append(result, item)
	}

	return result, total, nil
}

func (r *publicSubscribeRepo) QuerySubscribeCatalog(ctx context.Context, language string) (*subscribeBiz.SubscribeCatalog, error) {
	language = productlanguage.NormalizeProductLanguage(language)
	subscribes, total, err := r.QuerySubscribeList(ctx, language, 0)
	if err != nil {
		return nil, err
	}

	categoryQuery := r.data.db.ProxySubscribeCategory.Query().
		Where(proxysubscribecategory.ShowEQ(true))
	if language != "" {
		categoryQuery = categoryQuery.Where(
			proxysubscribecategory.Or(
				proxysubscribecategory.LanguageEQ(language),
				proxysubscribecategory.LanguageEQ(""),
			),
		)
	} else {
		categoryQuery = categoryQuery.Where(proxysubscribecategory.LanguageEQ(""))
	}
	categories, err := categoryQuery.
		Order(ent.Asc(proxysubscribecategory.FieldSort), ent.Asc(proxysubscribecategory.FieldID)).
		All(ctx)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	categoryMap := make(map[int64]*subscribeBiz.SubscribeCategory, len(categories))
	roots := make([]*subscribeBiz.SubscribeCategory, 0)
	for _, category := range categories {
		item := &subscribeBiz.SubscribeCategory{
			ID:          category.ID,
			ParentID:    category.ParentID,
			Name:        category.Name,
			Description: stringValue(category.Description),
			Language:    category.Language,
			Show:        category.Show,
			Sort:        int64(category.Sort),
		}
		categoryMap[category.ID] = item
	}
	for _, category := range categories {
		item := categoryMap[category.ID]
		if item.ParentID > 0 {
			if parent := categoryMap[item.ParentID]; parent != nil {
				parent.Children = append(parent.Children, item)
				continue
			}
		}
		roots = append(roots, item)
	}

	uncategorized := make([]*subscribeBiz.Subscribe, 0)
	for _, sub := range subscribes {
		if sub.CategoryID > 0 {
			if category := categoryMap[sub.CategoryID]; category != nil {
				category.List = append(category.List, sub)
				continue
			}
		}
		uncategorized = append(uncategorized, sub)
	}

	return &subscribeBiz.SubscribeCatalog{
		Categories:    roots,
		Uncategorized: uncategorized,
		Total:         total,
	}, nil
}

func (r *publicSubscribeRepo) publicSubscribeCategoryNames(ctx context.Context, subscribes []*ent.ProxySubscribe) map[int64]string {
	ids := make(map[int64]struct{})
	for _, sub := range subscribes {
		if sub != nil && sub.CategoryID > 0 {
			ids[sub.CategoryID] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return map[int64]string{}
	}
	idList := make([]int64, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	categories, err := r.data.db.ProxySubscribeCategory.Query().
		Where(proxysubscribecategory.IDIn(idList...)).
		All(ctx)
	if err != nil {
		return map[int64]string{}
	}
	result := make(map[int64]string, len(categories))
	for _, category := range categories {
		result[category.ID] = category.Name
	}
	return result
}

func (r *publicSubscribeRepo) publicSubscribePriceOptions(ctx context.Context, subscribes []*ent.ProxySubscribe) map[int64][]subscribeBiz.SubscribePriceOption {
	ids := make([]int64, 0, len(subscribes))
	for _, sub := range subscribes {
		if sub != nil {
			ids = append(ids, int64(sub.ID))
		}
	}
	result := make(map[int64][]subscribeBiz.SubscribePriceOption)
	if len(ids) == 0 {
		return result
	}
	items, err := r.data.db.ProxySubscribePriceOption.Query().
		Where(
			proxysubscribepriceoption.SubscribeIDIn(ids...),
			proxysubscribepriceoption.OptionTypeEQ("duration"),
			proxysubscribepriceoption.ShowEQ(true),
			proxysubscribepriceoption.SellEQ(true),
		).
		Order(ent.Desc(proxysubscribepriceoption.FieldSort), ent.Asc(proxysubscribepriceoption.FieldID)).
		All(ctx)
	if err != nil {
		return result
	}
	for _, item := range items {
		result[item.SubscribeID] = append(result[item.SubscribeID], subscribeBiz.SubscribePriceOption{
			ID:            item.ID,
			SubscribeID:   item.SubscribeID,
			Code:          item.Code,
			Type:          item.OptionType,
			Name:          item.Name,
			DurationUnit:  item.DurationUnit,
			DurationValue: item.DurationValue,
			Price:         item.Price,
			OriginalPrice: item.OriginalPrice,
			Inventory:     int64(item.Inventory),
			Show:          item.Show,
			Sell:          item.Sell,
			IsDefault:     item.IsDefault,
			Sort:          int64(item.Sort),
			CreatedAt:     item.CreatedAt.UnixMilli(),
			UpdatedAt:     item.UpdatedAt.UnixMilli(),
		})
	}
	return result
}

func (r *publicSubscribeRepo) QueryUserSubscribeNodeList(ctx context.Context, userID int64) ([]*subscribeBiz.UserSubscribeInfo, error) {
	userSubs, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.UserIDEQ(userID),
			proxyusersubscribe.StatusIn(0, 1, 2, 3),
		).
		Order(ent.Asc(proxyusersubscribe.FieldID)).
		All(ctx)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	userSubs = filterLegacyUserSubscribes(userSubs, time.Now())
	groupEnabled := legacyGroupEnabled(ctx, r.data)
	list := make([]*subscribeBiz.UserSubscribeInfo, 0, len(userSubs))
	for _, userSub := range userSubs {
		subscribePlan, err := r.data.db.ProxySubscribe.Query().
			Where(proxysubscribe.IDEQ(userSub.SubscribeID)).
			Only(ctx)
		if err != nil {
			return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
		}
		nodes, err := legacyNodeList(ctx, r.data, userSub, subscribePlan, groupEnabled)
		if err != nil {
			return nil, err
		}
		item := &subscribeBiz.UserSubscribeInfo{
			ID:          userSub.ID,
			UserID:      userSub.UserID,
			OrderID:     userSub.OrderID,
			SubscribeID: userSub.SubscribeID,
			StartTime:   userSub.StartTime.Unix(),
			ExpireTime:  legacyUnix(userSub.ExpireTime),
			FinishedAt:  unixTime(userSub.FinishedAt),
			ResetTime:   0,
			Traffic:     publicSubscribeInt64Value(userSub.Traffic),
			Download:    publicSubscribeInt64Value(userSub.Download),
			Upload:      publicSubscribeInt64Value(userSub.Upload),
			Token:       stringValue(userSub.Token),
			Status:      int32(int8Value(userSub.Status)),
			CreatedAt:   userSub.CreatedAt.Unix(),
			UpdatedAt:   userSub.UpdatedAt.Unix(),
			IsTryOut:    r.data.AppConf() != nil && r.data.AppConf().Register != nil && r.data.AppConf().Register.EnableTrial && r.data.AppConf().Register.TrialSubscribe == userSub.SubscribeID,
			Nodes:       nodes,
		}
		item.ResetTime = legacyUserResetTime(item)
		list = append(list, item)
	}
	return list, nil
}

func filterLegacyUserSubscribes(items []*ent.ProxyUserSubscribe, now time.Time) []*ent.ProxyUserSubscribe {
	result := make([]*ent.ProxyUserSubscribe, 0, len(items))
	for _, item := range items {
		if !shouldKeepLegacyUserSubscribe(item, now) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func publicSubscribeInt64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func int8Value(value *int8) int8 {
	if value == nil {
		return 0
	}
	return *value
}

func unixTime(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}

func legacyUserResetTime(item *subscribeBiz.UserSubscribeInfo) int64 {
	if item == nil || item.ExpireTime == 0 {
		return 0
	}
	return item.ExpireTime
}

func legacyUnix(value *time.Time) int64 {
	if value == nil || value.Unix() == 0 {
		return 0
	}
	return value.Unix()
}

func legacyGroupEnabled(ctx context.Context, d *Data) bool {
	if d == nil || d.db == nil {
		return false
	}
	item, err := d.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("group"),
			proxysystem.KeyIn("enabled", "Enabled"),
		).
		First(ctx)
	if err != nil {
		return false
	}
	return item.Value == "true" || item.Value == "1"
}

func legacyNodeList(ctx context.Context, d *Data, userSub *ent.ProxyUserSubscribe, subscribePlan *ent.ProxySubscribe, groupEnabled bool) ([]*subscribeBiz.UserSubscribeNodeInfo, error) {
	now := time.Now()
	if userSub.ExpireTime != nil && userSub.ExpireTime.Unix() != 0 && userSub.ExpireTime.Before(now) {
		return legacyExpiredNodes(ctx, d, userSub)
	}
	enabledNodes, err := d.db.ProxyNode.Query().
		Where(
			proxynode.EnabledEQ(true),
			proxynode.IsHiddenEQ(false),
		).
		Order(ent.Asc(proxynode.FieldSort)).
		All(ctx)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	selected := make([]*ent.ProxyNode, 0)
	seen := make(map[int64]struct{})
	if groupEnabled {
		nodeGroupID := publicSubscribeResolveNodeGroupID(userSub, subscribePlan)
		if nodeGroupID > 0 {
			groupInfo, err := legacyAccessibleNodeGroup(ctx, d, nodeGroupID)
			if err != nil {
				return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
			}
			if groupInfo == nil {
				nodeGroupID = 0
			}
		}
		directNodeIDs := tool.StringToInt64Slice(subscribePlan.Nodes)
		for _, node := range enabledNodes {
			if len(node.NodeGroupIds) == 0 {
				if _, ok := seen[node.ID]; !ok {
					seen[node.ID] = struct{}{}
					selected = append(selected, node)
				}
				continue
			}
			if nodeGroupID != 0 && tool.Contains(node.NodeGroupIds, nodeGroupID) {
				if _, ok := seen[node.ID]; !ok {
					seen[node.ID] = struct{}{}
					selected = append(selected, node)
				}
			}
		}
		for _, node := range enabledNodes {
			if tool.Contains(directNodeIDs, node.ID) {
				if _, ok := seen[node.ID]; !ok {
					seen[node.ID] = struct{}{}
					selected = append(selected, node)
				}
			}
		}
	} else {
		nodeIDs := tool.StringToInt64Slice(subscribePlan.Nodes)
		tags := make([]string, 0)
		for _, item := range strings.Split(subscribePlan.NodeTags, ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
		selectedIDs := make(map[int64]struct{})
		for _, node := range enabledNodes {
			if len(nodeIDs) == 0 && len(tags) == 0 {
				continue
			}
			matched := false
			if len(nodeIDs) > 0 && tool.Contains(nodeIDs, node.ID) {
				matched = true
			}
			if len(tags) > 0 && nodeMatchesTags(node.Tags, tags) {
				matched = true
			}
			if !matched {
				continue
			}
			if _, ok := selectedIDs[node.ID]; ok {
				continue
			}
			selectedIDs[node.ID] = struct{}{}
			selected = append(selected, node)
		}
	}
	return buildLegacyNodeInfos(ctx, d, userSub, selected)
}

func legacyExpiredNodes(ctx context.Context, d *Data, userSub *ent.ProxyUserSubscribe) ([]*subscribeBiz.UserSubscribeNodeInfo, error) {
	expiredGroup, err := d.db.ProxyServerGroup.Query().
		Where(proxyservergroup.IsExpiredGroupEQ(true)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if !isNodeGroupTypeAccessible(expiredGroup.GroupType, nodeGroupAccessApp) {
		return nil, nil
	}
	if userSub.ExpireTime == nil {
		return nil, nil
	}
	expiredDays := int(time.Since(*userSub.ExpireTime).Hours() / 24)
	if expiredDays > expiredGroup.ExpiredDaysLimit {
		return nil, nil
	}
	if expiredGroup.MaxTrafficGBExpired != nil && *expiredGroup.MaxTrafficGBExpired > 0 {
		usedTrafficGB := float64(publicSubscribeInt64Value(userSub.ExpiredDownload)+publicSubscribeInt64Value(userSub.ExpiredUpload)) / (1024 * 1024 * 1024)
		if usedTrafficGB >= float64(*expiredGroup.MaxTrafficGBExpired) {
			return nil, nil
		}
	}
	nodes, err := d.db.ProxyNode.Query().
		Where(
			proxynode.EnabledEQ(true),
			proxynode.IsHiddenEQ(false),
		).
		Order(ent.Asc(proxynode.FieldSort)).
		All(ctx)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	selected := make([]*ent.ProxyNode, 0)
	for _, node := range nodes {
		if tool.Contains(node.NodeGroupIds, expiredGroup.ID) {
			selected = append(selected, node)
		}
	}
	return buildLegacyNodeInfos(ctx, d, userSub, selected)
}

func publicSubscribeResolveNodeGroupID(userSub *ent.ProxyUserSubscribe, subscribePlan *ent.ProxySubscribe) int64 {
	if userSub != nil && userSub.NodeGroupID != 0 {
		return userSub.NodeGroupID
	}
	if subscribePlan != nil && subscribePlan.NodeGroupID != nil && *subscribePlan.NodeGroupID != 0 {
		return *subscribePlan.NodeGroupID
	}
	if subscribePlan != nil && len(subscribePlan.NodeGroupIds) > 0 {
		return subscribePlan.NodeGroupIds[0]
	}
	return 0
}

const (
	nodeGroupTypeCommon = "common"
	nodeGroupTypeApp    = "app"
	nodeGroupAccessApp  = "app"
)

func normalizeNodeGroupType(groupType string) string {
	switch strings.ToLower(strings.TrimSpace(groupType)) {
	case "", nodeGroupTypeCommon:
		return nodeGroupTypeCommon
	case nodeGroupTypeApp:
		return nodeGroupTypeApp
	default:
		return nodeGroupTypeCommon
	}
}

func isNodeGroupTypeAccessible(groupType, accessType string) bool {
	switch accessType {
	case nodeGroupAccessApp:
		resolved := normalizeNodeGroupType(groupType)
		return resolved == nodeGroupTypeCommon || resolved == nodeGroupTypeApp
	default:
		return false
	}
}

func legacyAccessibleNodeGroup(ctx context.Context, d *Data, nodeGroupID int64) (*ent.ProxyServerGroup, error) {
	if d == nil || d.db == nil || nodeGroupID == 0 {
		return nil, nil
	}
	item, err := d.db.ProxyServerGroup.Query().
		Where(proxyservergroup.IDEQ(nodeGroupID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if !isNodeGroupTypeAccessible(item.GroupType, nodeGroupAccessApp) {
		return nil, nil
	}
	return item, nil
}

func nodeMatchesTags(nodeTags string, tags []string) bool {
	for _, tag := range tags {
		for _, item := range strings.Split(nodeTags, ",") {
			if strings.TrimSpace(item) == tag {
				return true
			}
		}
	}
	return false
}

func buildLegacyNodeInfos(ctx context.Context, d *Data, userSub *ent.ProxyUserSubscribe, nodes []*ent.ProxyNode) ([]*subscribeBiz.UserSubscribeNodeInfo, error) {
	if len(nodes) == 0 {
		return []*subscribeBiz.UserSubscribeNodeInfo{}, nil
	}
	serverIDs := make([]int64, 0, len(nodes))
	serverSeen := make(map[int64]struct{}, len(nodes))
	for _, node := range nodes {
		if _, ok := serverSeen[node.ServerID]; ok {
			continue
		}
		serverSeen[node.ServerID] = struct{}{}
		serverIDs = append(serverIDs, node.ServerID)
	}
	servers, err := d.db.ProxyServer.Query().
		Where(proxyserver.IDIn(serverIDs...)).
		All(ctx)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	serverMap := make(map[int64]*ent.ProxyServer, len(servers))
	for _, server := range servers {
		serverMap[server.ID] = server
	}
	result := make([]*subscribeBiz.UserSubscribeNodeInfo, 0, len(nodes))
	for _, node := range nodes {
		server := serverMap[node.ServerID]
		if server == nil {
			continue
		}
		protocols := injectLegacySimnetUserCredentialsForClient(
			cleanLegacyNodeProtocolInstance(server.Protocol, node.Protocol, node.Port),
			userSub,
		)
		nodeInfo := &subscribeBiz.UserSubscribeNodeInfo{
			ID:              node.ID,
			Name:            node.Name,
			Uuid:            stringValue(userSub.UUID),
			Protocol:        node.Protocol,
			Protocols:       protocols,
			Port:            uint32(node.Port),
			Address:         node.Address,
			Tags:            strings.Split(node.Tags, ","),
			Country:         server.Country,
			City:            server.City,
			Longitude:       server.Longitude,
			Latitude:        server.Latitude,
			LatitudeCenter:  server.LatitudeCenter,
			LongitudeCenter: server.LongitudeCenter,
			CreatedAt:       node.CreatedAt.Unix(),
		}
		if matched := legacyMatchedServerProtocol(server.Protocol, node.Protocol, node.Port); matched != nil {
			applyLegacyOmniflowProtocol(nodeInfo, matched)
		}
		result = append(result, nodeInfo)
	}
	return result, nil
}

func legacyMatchedServerProtocol(protocolsJSON string, nodeProtocol string, nodePort uint16) *servermodel.Protocol {
	protocols, err := servermodel.UnmarshalProtocols(protocolsJSON)
	if err != nil {
		return nil
	}
	matched, _, _ := matchNodeProtocolConfig(protocols, nodeProtocol, nodePort)
	if matched != nil {
		matched.NormalizeOmniflow()
		return matched
	}
	return nil
}

func applyLegacyOmniflowProtocol(nodeInfo *subscribeBiz.UserSubscribeNodeInfo, protocol *servermodel.Protocol) {
	if nodeInfo == nil || protocol == nil {
		return
	}
	nodeInfo.SNI = protocol.SNI
	nodeInfo.OmniflowCarrier = protocol.OmniflowCarrier
	nodeInfo.OmniflowPath = protocol.OmniflowPath
	nodeInfo.OmniflowContentType = protocol.OmniflowContentType
	nodeInfo.OmniflowProfileJson = protocol.OmniflowProfileJson
	nodeInfo.OmniflowCaCertPath = protocol.OmniflowCaCertPath
	nodeInfo.OmniflowTargetMeta = protocol.OmniflowTargetMeta
	nodeInfo.OmniflowSpkiPin = protocol.OmniflowSpkiPin
	nodeInfo.OmniflowAdaptiveTlsEnabled = protocol.OmniflowAdaptiveTlsEnabled
	nodeInfo.OmniflowTlsFingerprint = protocol.OmniflowTlsFingerprint
	nodeInfo.OmniflowSniMode = protocol.OmniflowSniMode
	nodeInfo.OmniflowPaddingMode = protocol.OmniflowPaddingMode
	nodeInfo.OmniflowAfEnabled = protocol.OmniflowAfEnabled
	nodeInfo.OmniflowAfPathMode = protocol.OmniflowAfPathMode
	nodeInfo.OmniflowAfPathPrefix = protocol.OmniflowAfPathPrefix
	nodeInfo.OmniflowAfPathSuffix = protocol.OmniflowAfPathSuffix
	nodeInfo.OmniflowAfPathRotationSecs = int(protocol.OmniflowAfPathRotationSecs)
	nodeInfo.OmniflowAfPathSkewSlots = int(protocol.OmniflowAfPathSkewSlots)
}

func cleanLegacyNodeProtocols(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}
	var protocols []*servermodel.Protocol
	if err := json.Unmarshal([]byte(raw), &protocols); err != nil {
		return raw
	}
	for _, protocol := range protocols {
		cleanSimnetProtocolForClient(protocol)
	}
	cleaned, err := json.Marshal(protocols)
	if err != nil {
		return raw
	}
	return string(cleaned)
}

func cleanLegacyNodeProtocolInstance(raw string, nodeProtocol string, nodePort uint16) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}
	protocols, err := servermodel.UnmarshalProtocols(raw)
	if err != nil {
		return raw
	}
	matched, _, _ := matchNodeProtocolConfig(protocols, nodeProtocol, nodePort)
	if matched == nil {
		return "[]"
	}
	cleanSimnetProtocolForClient(matched)
	cleaned, err := json.Marshal([]*servermodel.Protocol{matched})
	if err != nil {
		return raw
	}
	return string(cleaned)
}

func injectLegacySimnetUserCredentialsForClient(raw string, userSub *ent.ProxyUserSubscribe) string {
	if strings.TrimSpace(raw) == "" || userSub == nil {
		return raw
	}
	userPSK := deriveLegacySimnetUserPSK(stringValue(userSub.UUID))
	if userPSK == "" {
		return raw
	}
	userKeyID := legacySimnetUserKeyID(userSub.ID)
	var protocols []map[string]any
	if err := json.Unmarshal([]byte(raw), &protocols); err != nil {
		return raw
	}
	changed := false
	for _, protocol := range protocols {
		if !isLegacySimnetProtocolMap(protocol) {
			continue
		}
		if serverPSK := firstStringFromMap(protocol, "simnet_server_psk", "simnetServerPsk", "simnet_psk", "simnetPsk"); serverPSK != "" {
			protocol["simnet_server_psk"] = serverPSK
		}
		protocol["simnet_server_key_id"] = firstIntFromMap(protocol, 0, "simnet_server_key_id", "simnetServerKeyId", "simnet_key_id", "simnetKeyId")
		protocol["simnet_user_psk"] = userPSK
		protocol["simnet_user_key_id"] = userKeyID
		changed = true
	}
	if !changed {
		return raw
	}
	encoded, err := json.Marshal(protocols)
	if err != nil {
		return raw
	}
	return string(encoded)
}

func isLegacySimnetProtocolMap(protocol map[string]any) bool {
	return strings.EqualFold(firstStringFromMap(protocol, "type", "protocol"), "simnet")
}

func legacySimnetUserKeyID(id int64) int64 {
	keyID := id % (1<<31 - 1)
	if keyID == 0 {
		return 1
	}
	return keyID
}

func deriveLegacySimnetUserPSK(value string) string {
	trimmed := strings.TrimSpace(value)
	if isCanonicalLegacyUUID(trimmed) {
		return strings.ToLower(strings.ReplaceAll(trimmed, "-", ""))
	}
	if len(trimmed) == 32 && isLegacyASCIIHex(trimmed) {
		return strings.ToLower(trimmed)
	}
	return hex.EncodeToString([]byte(trimmed))
}

func isCanonicalLegacyUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for idx, ch := range value {
		switch idx {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if !isLegacyASCIIHexRune(ch) {
				return false
			}
		}
	}
	return true
}

func isLegacyASCIIHex(value string) bool {
	for _, ch := range value {
		if !isLegacyASCIIHexRune(ch) {
			return false
		}
	}
	return true
}

func isLegacyASCIIHexRune(ch rune) bool {
	return (ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'f') ||
		(ch >= 'A' && ch <= 'F')
}

func firstStringFromMap(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if trimmed := strings.TrimSpace(stringFromAny(value)); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func firstIntFromMap(values map[string]any, fallback int64, keys ...string) int64 {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int:
			return int64(v)
		case int64:
			return v
		case int32:
			return int64(v)
		case float64:
			return int64(v)
		case json.Number:
			if parsed, err := v.Int64(); err == nil {
				return parsed
			}
		case string:
			if parsed, ok := parseLegacyIntString(v); ok {
				return parsed
			}
		}
	}
	return fallback
}

func parseLegacyIntString(value string) (int64, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}
	var result int64
	for _, ch := range trimmed {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		result = result*10 + int64(ch-'0')
	}
	return result, true
}

func cleanSimnetProtocolForClient(protocol *servermodel.Protocol) {
	if protocol == nil || protocol.Type != "simnet" {
		return
	}
	protocol.SimnetPsk = strings.TrimSpace(protocol.SimnetPsk)
	protocol.SimnetTicketID = strings.TrimSpace(protocol.SimnetTicketID)
	protocol.SimnetCarrier = defaultLegacySimnetString(protocol.SimnetCarrier, "h2")
	if strings.TrimSpace(protocol.SimnetPath) == "" {
		protocol.SimnetPath = "/simnet/session"
	} else {
		protocol.SimnetPath = strings.TrimSpace(protocol.SimnetPath)
	}
	protocol.NormalizeSimnet()
	if !protocol.SimnetAfEnabled {
		protocol.SimnetAfPathMode = ""
		protocol.SimnetAfMagicMode = ""
		protocol.SimnetAfPathPrefix = ""
		protocol.SimnetAfPathSuffix = ""
		protocol.SimnetAfResponseJitterMs = 0
		protocol.SimnetAfHandshakePolymorphism = false
		protocol.SimnetAfSettingsJitter = false
		protocol.SimnetAfFakeHeaderInjection = false
		return
	}
	protocol.SimnetAfPathMode = defaultLegacySimnetString(protocol.SimnetAfPathMode, "api")
	protocol.SimnetAfMagicMode = defaultLegacySimnetString(protocol.SimnetAfMagicMode, "derived")
	protocol.SimnetAfPathPrefix = strings.TrimSpace(protocol.SimnetAfPathPrefix)
	protocol.SimnetAfPathSuffix = strings.TrimSpace(protocol.SimnetAfPathSuffix)
	if protocol.SimnetAfResponseJitterMs == 0 {
		protocol.SimnetAfResponseJitterMs = 50
	}
	if !protocol.SimnetAfHandshakePolymorphism {
		protocol.SimnetAfHandshakePolymorphism = true
	}
	if !protocol.SimnetAfSettingsJitter {
		protocol.SimnetAfSettingsJitter = true
	}
	if !protocol.SimnetAfFakeHeaderInjection {
		protocol.SimnetAfFakeHeaderInjection = true
	}
}

func defaultLegacySimnetString(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}
