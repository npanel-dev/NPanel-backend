package subscribe

import (
	"context"

	v1 "github.com/npanel-dev/NPanel-backend/api/public/subscribe/v1"
	subscribeBiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/subscribe"
	appmiddleware "github.com/npanel-dev/NPanel-backend/internal/pkg/middleware"
)

type SubscribeService struct {
	v1.UnimplementedPublicSubscribeServer
	uc *subscribeBiz.SubscribeUseCase
}

func NewSubscribeService(uc *subscribeBiz.SubscribeUseCase) *SubscribeService {
	return &SubscribeService{uc: uc}
}

func (s *SubscribeService) QuerySubscribeList(ctx context.Context, req *v1.QuerySubscribeListRequest) (*v1.QuerySubscribeListReply, error) {
	subscribes, total, err := s.uc.QuerySubscribeList(ctx, req.Language, req.CategoryId)
	if err != nil {
		return nil, err
	}

	list := make([]*v1.Subscribe, 0, len(subscribes))
	for _, sub := range subscribes {
		list = append(list, convertPublicSubscribe(sub))
	}

	return &v1.QuerySubscribeListReply{List: list, Total: total}, nil
}

func (s *SubscribeService) QuerySubscribeCatalog(ctx context.Context, req *v1.QuerySubscribeCatalogRequest) (*v1.QuerySubscribeCatalogReply, error) {
	catalog, err := s.uc.QuerySubscribeCatalog(ctx, req.Language)
	if err != nil {
		return nil, err
	}
	categories := make([]*v1.SubscribeCategory, 0, len(catalog.Categories))
	for _, category := range catalog.Categories {
		categories = append(categories, convertPublicSubscribeCategory(category))
	}
	uncategorized := make([]*v1.Subscribe, 0, len(catalog.Uncategorized))
	for _, sub := range catalog.Uncategorized {
		uncategorized = append(uncategorized, convertPublicSubscribe(sub))
	}
	return &v1.QuerySubscribeCatalogReply{
		Categories:    categories,
		Uncategorized: uncategorized,
		Total:         catalog.Total,
	}, nil
}

func convertPublicSubscribe(sub *subscribeBiz.Subscribe) *v1.Subscribe {
	if sub == nil {
		return nil
	}
	discounts := make([]*v1.SubscribeDiscount, 0, len(sub.Discount))
	for _, discount := range sub.Discount {
		if discount == nil {
			continue
		}
		discounts = append(discounts, &v1.SubscribeDiscount{
			Quantity: discount.Quantity,
			Discount: discount.Discount,
		})
	}

	trafficLimits := make([]*v1.TrafficLimit, 0, len(sub.TrafficLimit))
	for _, limit := range sub.TrafficLimit {
		if limit == nil {
			continue
		}
		trafficLimits = append(trafficLimits, &v1.TrafficLimit{
			StatType:     limit.StatType,
			StatValue:    limit.StatValue,
			TrafficUsage: limit.TrafficUsage,
			SpeedLimit:   int32(limit.SpeedLimit),
		})
	}

	return &v1.Subscribe{
		Id:                sub.ID,
		Name:              sub.Name,
		Language:          sub.Language,
		Description:       sub.Description,
		UnitPrice:         sub.UnitPrice,
		UnitTime:          sub.UnitTime,
		Discount:          discounts,
		Replacement:       sub.Replacement,
		Inventory:         int32(sub.Inventory),
		Traffic:           sub.Traffic,
		SpeedLimit:        int32(sub.SpeedLimit),
		DeviceLimit:       int32(sub.DeviceLimit),
		Quota:             int32(sub.Quota),
		CategoryId:        sub.CategoryID,
		CategoryName:      sub.CategoryName,
		Nodes:             convertIntSliceToInt64Slice(sub.Nodes),
		NodeTags:          sub.NodeTags,
		NodeGroupIds:      sub.NodeGroupIds,
		NodeGroupId:       sub.NodeGroupId,
		TrafficLimit:      trafficLimits,
		Show:              sub.Show,
		Sell:              sub.Sell,
		Sort:              int32(sub.Sort),
		DeductionRatio:    int32(sub.DeductionRatio),
		AllowDeduction:    sub.AllowDeduction,
		ResetCycle:        int32(sub.ResetCycle),
		RenewalReset:      sub.RenewalReset,
		ShowOriginalPrice: sub.ShowOriginalPrice,
		PriceOptions:      convertPublicSubscribePriceOptions(sub.PriceOptions),
		CreatedAt:         sub.CreatedAt,
		UpdatedAt:         sub.UpdatedAt,
	}
}

func convertPublicSubscribePriceOptions(items []subscribeBiz.SubscribePriceOption) []*v1.SubscribePriceOption {
	if len(items) == 0 {
		return []*v1.SubscribePriceOption{}
	}
	result := make([]*v1.SubscribePriceOption, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.SubscribePriceOption{
			Id:            item.ID,
			SubscribeId:   item.SubscribeID,
			Name:          item.Name,
			DurationUnit:  item.DurationUnit,
			DurationValue: item.DurationValue,
			Price:         item.Price,
			OriginalPrice: item.OriginalPrice,
			Inventory:     int32(item.Inventory),
			Show:          item.Show,
			Sell:          item.Sell,
			IsDefault:     item.IsDefault,
			Sort:          int32(item.Sort),
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	return result
}

func convertPublicSubscribeCategory(category *subscribeBiz.SubscribeCategory) *v1.SubscribeCategory {
	if category == nil {
		return nil
	}
	list := make([]*v1.Subscribe, 0, len(category.List))
	for _, sub := range category.List {
		list = append(list, convertPublicSubscribe(sub))
	}
	children := make([]*v1.SubscribeCategory, 0, len(category.Children))
	for _, child := range category.Children {
		children = append(children, convertPublicSubscribeCategory(child))
	}
	return &v1.SubscribeCategory{
		Id:          category.ID,
		ParentId:    category.ParentID,
		Name:        category.Name,
		Description: category.Description,
		Language:    category.Language,
		Show:        category.Show,
		Sort:        int32(category.Sort),
		List:        list,
		Children:    children,
	}
}

func (s *SubscribeService) QueryUserSubscribeNodeList(ctx context.Context, req *v1.QueryUserSubscribeNodeListRequest) (*v1.QueryUserSubscribeNodeListReply, error) {
	userID := appmiddleware.GetUserID(ctx)
	list, err := s.uc.QueryUserSubscribeNodeList(ctx, userID)
	if err != nil {
		return nil, err
	}

	// simnet/omniflow 等新协议仅对自有客户端/SDK（UA 命中 omnxt 或 slaglab）放行，
	// 其它客户端一律剔除，避免下发开源客户端无法使用的节点配置。
	subscribeBiz.FilterExperimentalNodesForClient(list, appmiddleware.GetUserAgent(ctx))

	items := make([]*v1.UserSubscribeInfo, 0, len(list))
	for _, item := range list {
		nodes := make([]*v1.UserSubscribeNodeInfo, 0, len(item.Nodes))
		for _, node := range item.Nodes {
			if node == nil {
				continue
			}
			nodes = append(nodes, &v1.UserSubscribeNodeInfo{
				Id:              node.ID,
				Name:            node.Name,
				Uuid:            node.Uuid,
				Protocol:        node.Protocol,
				Protocols:       node.Protocols,
				Port:            node.Port,
				Address:         node.Address,
				Tags:            node.Tags,
				Country:         node.Country,
				City:            node.City,
				Longitude:       node.Longitude,
				Latitude:        node.Latitude,
				LatitudeCenter:  node.LatitudeCenter,
				LongitudeCenter: node.LongitudeCenter,
				CreatedAt:       node.CreatedAt,
				Sni:             node.SNI,

				OmniflowCarrier:            node.OmniflowCarrier,
				OmniflowPath:               node.OmniflowPath,
				OmniflowContentType:        node.OmniflowContentType,
				OmniflowProfileJson:        node.OmniflowProfileJson,
				OmniflowCaCertPath:         node.OmniflowCaCertPath,
				OmniflowTargetMeta:         node.OmniflowTargetMeta,
				OmniflowSpkiPin:            node.OmniflowSpkiPin,
				OmniflowAdaptiveTlsEnabled: node.OmniflowAdaptiveTlsEnabled,
				OmniflowTlsFingerprint:     node.OmniflowTlsFingerprint,
				OmniflowSniMode:            node.OmniflowSniMode,
				OmniflowPaddingMode:        node.OmniflowPaddingMode,
				OmniflowAfEnabled:          node.OmniflowAfEnabled,
				OmniflowAfPathMode:         node.OmniflowAfPathMode,
				OmniflowAfPathPrefix:       node.OmniflowAfPathPrefix,
				OmniflowAfPathSuffix:       node.OmniflowAfPathSuffix,
				OmniflowAfPathRotationSecs: int32(node.OmniflowAfPathRotationSecs),
				OmniflowAfPathSkewSlots:    int32(node.OmniflowAfPathSkewSlots),
			})
		}

		items = append(items, &v1.UserSubscribeInfo{
			Id:          item.ID,
			UserId:      item.UserID,
			OrderId:     item.OrderID,
			SubscribeId: item.SubscribeID,
			StartTime:   item.StartTime,
			ExpireTime:  item.ExpireTime,
			FinishedAt:  item.FinishedAt,
			ResetTime:   item.ResetTime,
			Traffic:     item.Traffic,
			Download:    item.Download,
			Upload:      item.Upload,
			Token:       item.Token,
			Status:      uint32(item.Status),
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
			IsTryOut:    item.IsTryOut,
			Nodes:       nodes,
		})
	}

	return &v1.QueryUserSubscribeNodeListReply{List: items}, nil
}

func convertIntSliceToInt64Slice(input []int) []int64 {
	if len(input) == 0 {
		return []int64{}
	}
	result := make([]int64, 0, len(input))
	for _, item := range input {
		result = append(result, int64(item))
	}
	return result
}
