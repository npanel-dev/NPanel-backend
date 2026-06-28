package subscribe

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/npanel-dev/NPanel-backend/api/admin/subscribe/v1"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/internal/model"
	productlanguage "github.com/npanel-dev/NPanel-backend/internal/pkg/language"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

const module = "biz/admin/subscribe"

func parseStringID(s string) (int, error) {
	val, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return int(val), nil
}

func parseStringID64(s string) (int64, error) {
	val, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return val, nil
}

func normalizeAdminProductLanguage(value string) (string, error) {
	normalized := productlanguage.NormalizeProductLanguage(value)
	if !productlanguage.IsSupportedProductLanguage(normalized) {
		return "", responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return normalized, nil
}

func normalizeSubscribeDetailFormat(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "markdown", "md":
		return "markdown", nil
	case "html":
		return "html", nil
	case "text", "plain":
		return "text", nil
	default:
		return "", responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
}

// SubscribeUseCase subscribe use case
type SubscribeUseCase struct {
	repo SubscribeRepo
	log  *log.Helper
}

// NewSubscribeUseCase create subscribe use case
func NewSubscribeUseCase(repo SubscribeRepo, logger log.Logger) *SubscribeUseCase {
	return &SubscribeUseCase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", module)),
	}
}

// SubscribeRepo subscribe repository interface
type SubscribeRepo interface {
	// Subscribe operations
	CreateSubscribe(ctx context.Context, sub *model.Subscribe) error
	GetSubscribeByID(ctx context.Context, id int) (*ent.ProxySubscribe, error)
	UpdateSubscribe(ctx context.Context, sub *model.Subscribe) error
	DeleteSubscribe(ctx context.Context, id int) error
	GetSubscribeList(ctx context.Context, req *model.SubscribeListParams) ([]*ent.ProxySubscribe, int32, error)
	GetSubscribePriceOptionsBySubscribeIDs(ctx context.Context, ids []int64) (map[int64][]*ent.ProxySubscribePriceOption, error)
	CheckSubscribeInUse(ctx context.Context, subscribeID int) (bool, error)
	BatchDeleteSubscribe(ctx context.Context, ids []int) error
	GetSubscribeMinSort(ctx context.Context, ids []int) (int64, error)
	BatchUpdateSubscribeSort(ctx context.Context, subscribes []*ent.ProxySubscribe) error

	// Subscribe category operations
	CreateSubscribeCategory(ctx context.Context, category *model.SubscribeCategory) error
	GetSubscribeCategoryByID(ctx context.Context, id int64) (*ent.ProxySubscribeCategory, error)
	UpdateSubscribeCategory(ctx context.Context, category *model.SubscribeCategory) error
	DeleteSubscribeCategory(ctx context.Context, id int64) error
	BatchDeleteSubscribeCategory(ctx context.Context, ids []int64) error
	GetSubscribeCategoryList(ctx context.Context, req *model.SubscribeCategoryListParams) ([]*ent.ProxySubscribeCategory, int32, error)
	CountSubscribeByCategoryID(ctx context.Context, categoryID int64) (int, error)
	CountSubscribeCategoryChildren(ctx context.Context, categoryID int64) (int, error)

	// Subscribe group operations
	CreateSubscribeGroup(ctx context.Context, group *model.SubscribeGroup) error
	GetSubscribeGroupByID(ctx context.Context, id int) (*ent.ProxySubscribeGroup, error)
	UpdateSubscribeGroup(ctx context.Context, group *model.SubscribeGroup) error
	DeleteSubscribeGroup(ctx context.Context, id int) error
	GetSubscribeGroupList(ctx context.Context) ([]*ent.ProxySubscribeGroup, int32, error)
	BatchDeleteSubscribeGroup(ctx context.Context, ids []int) error

	// User subscription query (for checking if subscribe is in use)
	GetActiveUserSubscriptionCount(ctx context.Context, subscribeID int) (int64, error)
	GetActiveUserSubscriptionCountByIDs(ctx context.Context, subscribeIDs []int64) (map[int64]int64, error)
	ResetAllSubscribeToken(ctx context.Context) error
}

// ==================== Subscribe Operations ====================

// CreateSubscribe create subscribe
func (uc *SubscribeUseCase) CreateSubscribe(ctx context.Context, req *v1.CreateSubscribeRequest) error {
	language, err := normalizeAdminProductLanguage(req.Language)
	if err != nil {
		return err
	}
	discountJSON, err := marshalJSON(convertDiscountToModel(req.Discount))
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "Marshal discount failed", "error", err)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	trafficLimitJSON, err := marshalJSON(convertTrafficLimitToModel(req.TrafficLimit))
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "Marshal traffic limit failed", "error", err)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	if err := uc.ensureSubscribeCategoryExists(ctx, req.CategoryId); err != nil {
		return err
	}
	detailFormat, err := normalizeSubscribeDetailFormat(req.DetailFormat)
	if err != nil {
		return err
	}
	priceOptions, err := convertPriceOptionsToModel(req.GetPriceOptions())
	if err != nil {
		return err
	}

	sub := &model.Subscribe{
		Name:              req.Name,
		Language:          language,
		Description:       req.Description,
		ShortDescription:  req.ShortDescription,
		Features:          req.Features,
		DetailFormat:      detailFormat,
		DetailContent:     req.DetailContent,
		UnitPrice:         req.UnitPrice,
		UnitTime:          req.UnitTime,
		Discount:          discountJSON,
		Replacement:       req.Replacement,
		Inventory:         int64(req.Inventory),
		Traffic:           req.Traffic,
		SpeedLimit:        int64(req.SpeedLimit),
		DeviceLimit:       int64(req.DeviceLimit),
		Quota:             int64(req.Quota),
		CategoryID:        req.CategoryId,
		Nodes:             int64SliceToString(req.Nodes),
		NodeTags:          stringSliceToString(req.NodeTags),
		NodeGroupIDs:      cloneInt64Slice(req.NodeGroupIds),
		NodeGroupID:       req.NodeGroupId,
		TrafficLimit:      trafficLimitJSON,
		Show:              getBoolValue(req.Show, false),
		Sell:              getBoolValue(req.Sell, false),
		Sort:              0,
		DeductionRatio:    int64(req.DeductionRatio),
		AllowDeduction:    getBoolValue(req.AllowDeduction, true),
		ResetCycle:        int64(req.ResetCycle),
		RenewalReset:      getBoolValue(req.RenewalReset, false),
		ShowOriginalPrice: req.ShowOriginalPrice,
		PriceOptions:      priceOptions,
	}

	if err := uc.repo.CreateSubscribe(ctx, sub); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "CreateSubscribe failed", "error", err)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return nil
}

// UpdateSubscribe update subscribe
func (uc *SubscribeUseCase) UpdateSubscribe(ctx context.Context, req *v1.UpdateSubscribeRequest) error {
	language, err := normalizeAdminProductLanguage(req.Language)
	if err != nil {
		return err
	}
	// Check if subscribe exists
	id := int(req.Id)
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	_, err = uc.repo.GetSubscribeByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			uc.log.WithContext(ctx).Errorw("msg", "UpdateSubscribe subscribe not found", "error", err, "id", req.Id)
			return responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		uc.log.WithContext(ctx).Errorw("msg", "UpdateSubscribe GetSubscribeByID error", "error", err, "id", req.Id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	discountJSON, err := marshalJSON(convertDiscountToModel(req.Discount))
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "Marshal discount failed", "error", err)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	trafficLimitJSON, err := marshalJSON(convertTrafficLimitToModel(req.TrafficLimit))
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "Marshal traffic limit failed", "error", err)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	if err := uc.ensureSubscribeCategoryExists(ctx, req.CategoryId); err != nil {
		return err
	}
	detailFormat, err := normalizeSubscribeDetailFormat(req.DetailFormat)
	if err != nil {
		return err
	}
	priceOptions, err := convertPriceOptionsToModel(req.GetPriceOptions())
	if err != nil {
		return err
	}

	sub := &model.Subscribe{
		ID:                int64(id),
		Name:              req.Name,
		Language:          language,
		Description:       req.Description,
		ShortDescription:  req.ShortDescription,
		Features:          req.Features,
		DetailFormat:      detailFormat,
		DetailContent:     req.DetailContent,
		UnitPrice:         req.UnitPrice,
		UnitTime:          req.UnitTime,
		Discount:          discountJSON,
		Replacement:       req.Replacement,
		Inventory:         int64(req.Inventory),
		Traffic:           req.Traffic,
		SpeedLimit:        int64(req.SpeedLimit),
		DeviceLimit:       int64(req.DeviceLimit),
		Quota:             int64(req.Quota),
		CategoryID:        req.CategoryId,
		Nodes:             int64SliceToString(req.Nodes),
		NodeTags:          stringSliceToString(req.NodeTags),
		NodeGroupIDs:      cloneInt64Slice(req.NodeGroupIds),
		NodeGroupID:       req.NodeGroupId,
		TrafficLimit:      trafficLimitJSON,
		Show:              getBoolValue(req.Show, false),
		Sell:              getBoolValue(req.Sell, false),
		Sort:              int64(req.Sort),
		DeductionRatio:    int64(req.DeductionRatio),
		AllowDeduction:    getBoolValue(req.AllowDeduction, true),
		ResetCycle:        int64(req.ResetCycle),
		RenewalReset:      getBoolValue(req.RenewalReset, false),
		ShowOriginalPrice: req.ShowOriginalPrice,
		PriceOptions:      priceOptions,
	}

	if err := uc.repo.UpdateSubscribe(ctx, sub); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "UpdateSubscribe failed", "error", err, "id", req.Id)
		if stderrors.Is(err, model.ErrSubscribePriceOptionModified) {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return nil
}

// DeleteSubscribe delete subscribe
func (uc *SubscribeUseCase) DeleteSubscribe(ctx context.Context, id int) error {
	// Check if subscribe is in use by active user subscriptions
	inUse, err := uc.repo.CheckSubscribeInUse(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "DeleteSubscribe CheckSubscribeInUse error", "error", err, "id", id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	if inUse {
		uc.log.WithContext(ctx).Warnw("msg", "DeleteSubscribe subscribe is in use", "id", id)
		return responsecode.NewKratosError(responsecode.ErrSubscribeInUse)
	}

	if err := uc.repo.DeleteSubscribe(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "DeleteSubscribe failed", "error", err, "id", id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return nil
}

// BatchDeleteSubscribe batch delete subscribes
func (uc *SubscribeUseCase) BatchDeleteSubscribe(ctx context.Context, ids []int) error {
	// Check each subscribe if it's in use
	for _, id := range ids {
		inUse, err := uc.repo.CheckSubscribeInUse(ctx, id)
		if err != nil {
			uc.log.WithContext(ctx).Errorw("msg", "BatchDeleteSubscribe CheckSubscribeInUse error", "error", err, "id", id)
			return responsecode.NewKratosError(responsecode.ErrInternalError)
		}

		if inUse {
			uc.log.WithContext(ctx).Warnw("msg", "BatchDeleteSubscribe subscribe is in use", "id", id)
			return responsecode.NewKratosError(responsecode.ErrSubscribeInUse)
		}
	}

	// Delete all subscribes
	if err := uc.repo.BatchDeleteSubscribe(ctx, ids); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "BatchDeleteSubscribe failed", "error", err, "ids", ids)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return nil
}

// GetSubscribeDetails get subscribe details
func (uc *SubscribeUseCase) GetSubscribeDetails(ctx context.Context, id int) (*v1.SubscribeInfo, error) {
	sub, err := uc.repo.GetSubscribeByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeDetails subscribe not found", "error", err, "id", id)
			return nil, responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeDetails failed", "error", err, "id", id)
		return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	item := convertSubscribeToProto(sub)
	priceOptions, err := uc.repo.GetSubscribePriceOptionsBySubscribeIDs(ctx, []int64{int64(sub.ID)})
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeDetails price options error", "error", err, "id", id)
		return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	item.PriceOptions = convertPriceOptionsToProto(priceOptions[int64(sub.ID)])
	if sub.CategoryID > 0 {
		if category, err := uc.repo.GetSubscribeCategoryByID(ctx, sub.CategoryID); err == nil {
			item.CategoryName = category.Name
		}
	}
	return item, nil
}

// GetSubscribeList get subscribe list
func (uc *SubscribeUseCase) GetSubscribeList(ctx context.Context, req *v1.GetSubscribeListRequest) (*v1.GetSubscribeListData, error) {
	params := &model.SubscribeListParams{
		Page:        int(req.Page),
		Size:        int(req.Size),
		Language:    productlanguage.NormalizeProductLanguage(req.Language),
		Search:      req.Search,
		NodeGroupID: req.NodeGroupId,
		CategoryID:  req.CategoryId,
	}

	list, total, err := uc.repo.GetSubscribeList(ctx, params)
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeList failed", "error", err, "params", params)
		return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	// Get subscribe IDs for querying sold counts
	subscribeIDs := make([]int64, 0, len(list))
	for _, sub := range list {
		subscribeIDs = append(subscribeIDs, sub.ID)
	}

	// Get active user subscription counts (sold count)
	soldCounts := make(map[int64]int64)
	if len(subscribeIDs) > 0 {
		soldCounts, err = uc.repo.GetActiveUserSubscriptionCountByIDs(ctx, subscribeIDs)
		if err != nil {
			uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeList GetActiveUserSubscriptionCountByIDs error", "error", err)
			// Don't fail the request, just log the error
		}
	}

	// Convert to proto
	categoryNames := uc.subscribeCategoryNames(ctx, list)
	priceOptions, err := uc.repo.GetSubscribePriceOptionsBySubscribeIDs(ctx, subscribeIDs)
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeList price options error", "error", err)
		return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	items := make([]*v1.SubscribeItem, 0, len(list))
	for _, sub := range list {
		item := convertSubscribeToProtoItem(sub)
		item.CategoryName = categoryNames[sub.CategoryID]
		item.Sold = soldCounts[int64(sub.ID)]
		item.PriceOptions = convertPriceOptionsToProto(priceOptions[int64(sub.ID)])
		items = append(items, item)
	}

	return &v1.GetSubscribeListData{
		List:  items,
		Total: total,
	}, nil
}

// SubscribeSort subscribe sort
func (uc *SubscribeUseCase) SubscribeSort(ctx context.Context, req *v1.SubscribeSortRequest) error {
	if len(req.Sort) == 0 {
		return nil
	}

	// Extract IDs
	ids := make([]int, 0, len(req.Sort))
	sortMap := make(map[int64]int64)
	for i, item := range req.Sort {
		if item.Id <= 0 {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		id := int(item.Id)
		ids = append(ids, id)
		sortMap[int64(id)] = int64(i)
	}

	// Get minimum sort value
	minSort, err := uc.repo.GetSubscribeMinSort(ctx, ids)
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "SubscribeSort GetSubscribeMinSort error", "error", err, "ids", ids)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	// Get subscribes
	idsInt64 := make([]int64, len(ids))
	for i, v := range ids {
		idsInt64[i] = int64(v)
	}
	params := &model.SubscribeListParams{
		Page: 1,
		Size: 9999,
		IDs:  idsInt64,
	}
	subscribes, _, err := uc.repo.GetSubscribeList(ctx, params)
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "SubscribeSort GetSubscribeList error", "error", err, "ids", ids)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	// Update sort values
	for _, sub := range subscribes {
		if newSort, ok := sortMap[sub.ID]; ok {
			sub.Sort = int32(minSort + newSort)
		}
	}

	// Batch update
	if err := uc.repo.BatchUpdateSubscribeSort(ctx, subscribes); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "SubscribeSort BatchUpdateSubscribeSort error", "error", err)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return nil
}

// ==================== Subscribe Category Operations ====================

// CreateSubscribeCategory create subscribe category.
func (uc *SubscribeUseCase) CreateSubscribeCategory(ctx context.Context, req *v1.CreateSubscribeCategoryRequest) error {
	language, err := normalizeAdminProductLanguage(req.Language)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Name) == "" {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if err := uc.ensureSubscribeCategoryParent(ctx, 0, req.ParentId); err != nil {
		return err
	}
	category := &model.SubscribeCategory{
		ParentID:    req.ParentId,
		Name:        req.Name,
		Description: req.Description,
		Language:    language,
		Show:        getBoolValue(req.Show, true),
		Sort:        int64(req.Sort),
	}
	if err := uc.repo.CreateSubscribeCategory(ctx, category); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "CreateSubscribeCategory failed", "error", err)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	return nil
}

// UpdateSubscribeCategory update subscribe category.
func (uc *SubscribeUseCase) UpdateSubscribeCategory(ctx context.Context, req *v1.UpdateSubscribeCategoryRequest) error {
	language, err := normalizeAdminProductLanguage(req.Language)
	if err != nil {
		return err
	}
	if req.Id <= 0 || strings.TrimSpace(req.Name) == "" {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if _, err := uc.repo.GetSubscribeCategoryByID(ctx, req.Id); err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		uc.log.WithContext(ctx).Errorw("msg", "UpdateSubscribeCategory GetSubscribeCategoryByID error", "error", err, "id", req.Id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	if err := uc.ensureSubscribeCategoryParent(ctx, req.Id, req.ParentId); err != nil {
		return err
	}
	category := &model.SubscribeCategory{
		ID:          req.Id,
		ParentID:    req.ParentId,
		Name:        req.Name,
		Description: req.Description,
		Language:    language,
		Show:        getBoolValue(req.Show, true),
		Sort:        int64(req.Sort),
	}
	if err := uc.repo.UpdateSubscribeCategory(ctx, category); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "UpdateSubscribeCategory failed", "error", err, "id", req.Id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	return nil
}

// DeleteSubscribeCategory delete subscribe category.
func (uc *SubscribeUseCase) DeleteSubscribeCategory(ctx context.Context, id int64) error {
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if err := uc.ensureSubscribeCategoryExists(ctx, id); err != nil {
		return err
	}
	if childCount, err := uc.repo.CountSubscribeCategoryChildren(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "DeleteSubscribeCategory CountSubscribeCategoryChildren error", "error", err, "id", id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	} else if childCount > 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if planCount, err := uc.repo.CountSubscribeByCategoryID(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "DeleteSubscribeCategory CountSubscribeByCategoryID error", "error", err, "id", id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	} else if planCount > 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if err := uc.repo.DeleteSubscribeCategory(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "DeleteSubscribeCategory failed", "error", err, "id", id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	return nil
}

// BatchDeleteSubscribeCategory batch delete subscribe categories.
func (uc *SubscribeUseCase) BatchDeleteSubscribeCategory(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	for _, id := range ids {
		if id <= 0 {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		if err := uc.ensureSubscribeCategoryExists(ctx, id); err != nil {
			return err
		}
		if childCount, err := uc.repo.CountSubscribeCategoryChildren(ctx, id); err != nil {
			uc.log.WithContext(ctx).Errorw("msg", "BatchDeleteSubscribeCategory CountSubscribeCategoryChildren error", "error", err, "id", id)
			return responsecode.NewKratosError(responsecode.ErrInternalError)
		} else if childCount > 0 {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		if planCount, err := uc.repo.CountSubscribeByCategoryID(ctx, id); err != nil {
			uc.log.WithContext(ctx).Errorw("msg", "BatchDeleteSubscribeCategory CountSubscribeByCategoryID error", "error", err, "id", id)
			return responsecode.NewKratosError(responsecode.ErrInternalError)
		} else if planCount > 0 {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
	}
	if err := uc.repo.BatchDeleteSubscribeCategory(ctx, ids); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "BatchDeleteSubscribeCategory failed", "error", err, "ids", ids)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	return nil
}

// GetSubscribeCategoryList get subscribe category list.
func (uc *SubscribeUseCase) GetSubscribeCategoryList(ctx context.Context, req *v1.GetSubscribeCategoryListRequest) (*v1.GetSubscribeCategoryListData, error) {
	params := &model.SubscribeCategoryListParams{
		Language: productlanguage.NormalizeProductLanguage(req.Language),
	}
	if req.ParentId != nil {
		params.ParentID = req.ParentId
	}
	if req.Show != nil {
		params.Show = req.Show
	}
	list, total, err := uc.repo.GetSubscribeCategoryList(ctx, params)
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeCategoryList failed", "error", err, "params", params)
		return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	items := make([]*v1.SubscribeCategoryInfo, 0, len(list))
	for _, category := range list {
		items = append(items, convertSubscribeCategoryToProto(category))
	}
	return &v1.GetSubscribeCategoryListData{
		List:  items,
		Total: total,
	}, nil
}

// ==================== Subscribe Group Operations ====================

// CreateSubscribeGroup create subscribe group
func (uc *SubscribeUseCase) CreateSubscribeGroup(ctx context.Context, req *v1.CreateSubscribeGroupRequest) error {
	group := &model.SubscribeGroup{
		Name:                req.Name,
		Description:         req.Description,
		IsExpiredGroup:      req.IsExpiredGroup,
		ExpiredDaysLimit:    req.ExpiredDaysLimit,
		MaxTrafficGBExpired: req.MaxTrafficGbExpired,
		SpeedLimit:          int64(req.SpeedLimit),
	}

	if err := uc.repo.CreateSubscribeGroup(ctx, group); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "CreateSubscribeGroup failed", "error", err)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return nil
}

// UpdateSubscribeGroup update subscribe group
func (uc *SubscribeUseCase) UpdateSubscribeGroup(ctx context.Context, req *v1.UpdateSubscribeGroupRequest) error {
	id := req.Id
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	group := &model.SubscribeGroup{
		ID:                  id,
		Name:                req.Name,
		Description:         req.Description,
		IsExpiredGroup:      req.IsExpiredGroup,
		ExpiredDaysLimit:    req.ExpiredDaysLimit,
		MaxTrafficGBExpired: req.MaxTrafficGbExpired,
		SpeedLimit:          int64(req.SpeedLimit),
	}

	if err := uc.repo.UpdateSubscribeGroup(ctx, group); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "UpdateSubscribeGroup failed", "error", err, "id", req.Id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return nil
}

// DeleteSubscribeGroup delete subscribe group
func (uc *SubscribeUseCase) DeleteSubscribeGroup(ctx context.Context, id int) error {
	if err := uc.repo.DeleteSubscribeGroup(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "DeleteSubscribeGroup failed", "error", err, "id", id)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return nil
}

// BatchDeleteSubscribeGroup batch delete subscribe groups
func (uc *SubscribeUseCase) BatchDeleteSubscribeGroup(ctx context.Context, ids []int) error {
	if err := uc.repo.BatchDeleteSubscribeGroup(ctx, ids); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "BatchDeleteSubscribeGroup failed", "error", err, "ids", ids)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return nil
}

// GetSubscribeGroupList get subscribe group list
func (uc *SubscribeUseCase) GetSubscribeGroupList(ctx context.Context) (*v1.GetSubscribeGroupListData, error) {
	list, total, err := uc.repo.GetSubscribeGroupList(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeGroupList failed", "error", err)
		return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	// Convert to proto
	groups := make([]*v1.SubscribeGroupInfo, 0, len(list))
	for _, group := range list {
		desc := ""
		if group.Description != nil {
			desc = *group.Description
		}
		groups = append(groups, &v1.SubscribeGroupInfo{
			Id:                  int64(group.ID),
			Name:                group.Name,
			Description:         desc,
			IsExpiredGroup:      group.IsExpiredGroup,
			ExpiredDaysLimit:    derefInt32(group.ExpiredDaysLimit),
			MaxTrafficGbExpired: derefInt32(group.MaxTrafficGBExpired),
			SpeedLimit:          int32(derefInt64(group.SpeedLimit)),
			CreatedAt:           group.CreatedAt.Unix(),
			UpdatedAt:           group.UpdatedAt.Unix(),
		})
	}

	return &v1.GetSubscribeGroupListData{
		List:  groups,
		Total: total,
	}, nil
}

func (uc *SubscribeUseCase) ResetAllSubscribeToken(ctx context.Context) error {
	if err := uc.repo.ResetAllSubscribeToken(ctx); err != nil {
		uc.log.WithContext(ctx).Errorw("msg", "ResetAllSubscribeToken failed", "error", err)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	return nil
}

// ==================== Helper Functions ====================

// convertDiscountToModel convert proto discount to model discount
func convertDiscountToModel(discounts []*v1.SubscribeDiscount) []model.SubscribeDiscount {
	result := make([]model.SubscribeDiscount, 0, len(discounts))
	for _, d := range discounts {
		result = append(result, model.SubscribeDiscount{
			Quantity: d.Quantity,
			Discount: d.Discount,
		})
	}
	return result
}

// convertDiscountFromJSON convert JSON discount to proto discount
func convertDiscountFromJSON(discountJSON string) []*v1.SubscribeDiscount {
	if discountJSON == "" {
		return nil
	}

	var discounts []model.SubscribeDiscount
	if err := json.Unmarshal([]byte(discountJSON), &discounts); err != nil {
		return nil
	}

	result := make([]*v1.SubscribeDiscount, 0, len(discounts))
	for _, d := range discounts {
		result = append(result, &v1.SubscribeDiscount{
			Quantity: d.Quantity,
			Discount: d.Discount,
		})
	}
	return result
}

func convertTrafficLimitToModel(limits []*v1.TrafficLimit) []model.TrafficLimit {
	result := make([]model.TrafficLimit, 0, len(limits))
	for _, limit := range limits {
		result = append(result, model.TrafficLimit{
			StatType:     limit.StatType,
			StatValue:    limit.StatValue,
			TrafficUsage: limit.TrafficUsage,
			SpeedLimit:   int64(limit.SpeedLimit),
		})
	}
	return result
}

func convertTrafficLimitFromJSON(raw string) []*v1.TrafficLimit {
	if raw == "" {
		return nil
	}

	var limits []model.TrafficLimit
	if err := json.Unmarshal([]byte(raw), &limits); err != nil {
		return nil
	}

	result := make([]*v1.TrafficLimit, 0, len(limits))
	for _, limit := range limits {
		result = append(result, &v1.TrafficLimit{
			StatType:     limit.StatType,
			StatValue:    limit.StatValue,
			TrafficUsage: limit.TrafficUsage,
			SpeedLimit:   int32(int64(limit.SpeedLimit)),
		})
	}
	return result
}

func marshalJSON(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	switch vv := v.(type) {
	case []model.SubscribeDiscount:
		if len(vv) == 0 {
			return "", nil
		}
	case []model.TrafficLimit:
		if len(vv) == 0 {
			return "", nil
		}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// int64SliceToString convert int64 slice to comma-separated string
func int64SliceToString(slice []int64) string {
	if len(slice) == 0 {
		return ""
	}
	strs := make([]string, 0, len(slice))
	for _, v := range slice {
		strs = append(strs, fmt.Sprintf("%d", v))
	}
	return strings.Join(strs, ",")
}

// stringToInt64Slice convert comma-separated string to int64 slice
func stringToInt64Slice(s string) []int64 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var val int64
		fmt.Sscanf(p, "%d", &val)
		result = append(result, val)
	}
	return result
}

// stringSliceToString convert string slice to comma-separated string
func stringSliceToString(slice []string) string {
	return strings.Join(slice, ",")
}

// stringToStringSlice convert comma-separated string to string slice
func stringToStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// getBoolValue get bool value from optional bool pointer
func getBoolValue(ptr *bool, defaultValue bool) bool {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

func cloneInt64Slice(input []int64) []int64 {
	if input == nil {
		return nil
	}
	out := make([]int64, len(input))
	copy(out, input)
	return out
}

func derefInt32(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}
func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func (uc *SubscribeUseCase) ensureSubscribeCategoryExists(ctx context.Context, categoryID int64) error {
	if categoryID < 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if categoryID == 0 {
		return nil
	}
	if _, err := uc.repo.GetSubscribeCategoryByID(ctx, categoryID); err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeCategoryByID error", "error", err, "category_id", categoryID)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	return nil
}

func (uc *SubscribeUseCase) ensureSubscribeCategoryParent(ctx context.Context, categoryID, parentID int64) error {
	if parentID < 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if parentID == 0 {
		return nil
	}

	seen := make(map[int64]struct{})
	currentID := parentID
	for depth := 0; currentID > 0; depth++ {
		if depth > 64 {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		if categoryID > 0 && currentID == categoryID {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		if _, ok := seen[currentID]; ok {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		seen[currentID] = struct{}{}

		category, err := uc.repo.GetSubscribeCategoryByID(ctx, currentID)
		if err != nil {
			if ent.IsNotFound(err) {
				return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
			}
			uc.log.WithContext(ctx).Errorw("msg", "GetSubscribeCategoryByID error", "error", err, "category_id", currentID)
			return responsecode.NewKratosError(responsecode.ErrInternalError)
		}
		currentID = category.ParentID
	}
	return nil
}

func convertSubscribeCategoryToProto(category *ent.ProxySubscribeCategory) *v1.SubscribeCategoryInfo {
	if category == nil {
		return nil
	}
	return &v1.SubscribeCategoryInfo{
		Id:          category.ID,
		ParentId:    category.ParentID,
		Name:        category.Name,
		Description: derefString(category.Description),
		Language:    category.Language,
		Show:        category.Show,
		Sort:        category.Sort,
		CreatedAt:   category.CreatedAt.Unix(),
		UpdatedAt:   category.UpdatedAt.Unix(),
	}
}

func (uc *SubscribeUseCase) subscribeCategoryNames(ctx context.Context, subscribes []*ent.ProxySubscribe) map[int64]string {
	ids := make(map[int64]struct{})
	for _, sub := range subscribes {
		if sub != nil && sub.CategoryID > 0 {
			ids[sub.CategoryID] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return map[int64]string{}
	}
	result := make(map[int64]string, len(ids))
	for id := range ids {
		category, err := uc.repo.GetSubscribeCategoryByID(ctx, id)
		if err != nil {
			continue
		}
		result[id] = category.Name
	}
	return result
}

var validPriceOptionDurationUnits = map[string]struct{}{
	"Minute":  {},
	"Hour":    {},
	"Day":     {},
	"Week":    {},
	"Month":   {},
	"Year":    {},
	"NoLimit": {},
}

var validPriceOptionTypes = map[string]struct{}{
	"duration":     {},
	"traffic_pack": {},
	"reset_pack":   {},
}

func defaultPriceOptionCode(optionType, unit string, durationValue int64) string {
	if optionType == "" {
		optionType = "duration"
	}
	if optionType != "duration" {
		return optionType
	}
	unit = strings.ToLower(strings.TrimSpace(unit))
	if unit == "" {
		unit = "month"
	}
	if unit == "nolimit" {
		return "duration_no_limit"
	}
	if durationValue <= 0 {
		durationValue = 1
	}
	return fmt.Sprintf("duration_%d_%s", durationValue, unit)
}

func convertPriceOptionsToModel(items []*v1.SubscribePriceOption) ([]model.SubscribePriceOption, error) {
	if len(items) == 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	result := make([]model.SubscribePriceOption, 0, len(items))
	hasDefault := false
	firstSellableIndex := -1
	seenCodes := make(map[string]struct{}, len(items))
	seenVisibleDurations := make(map[string]struct{}, len(items))
	for i, item := range items {
		if item == nil {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		optionType := strings.TrimSpace(item.Type)
		if optionType == "" {
			optionType = "duration"
		}
		if _, ok := validPriceOptionTypes[optionType]; !ok {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		unit := strings.TrimSpace(item.DurationUnit)
		if _, ok := validPriceOptionDurationUnits[unit]; !ok {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		durationValue := item.DurationValue
		if unit == "NoLimit" {
			durationValue = 0
		} else if durationValue <= 0 {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		if item.Price < 0 || item.OriginalPrice < 0 || item.Inventory < -1 {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		if item.Id > 0 && item.Version <= 0 {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		code := strings.TrimSpace(item.Code)
		if code == "" {
			code = defaultPriceOptionCode(optionType, unit, durationValue)
		}
		if _, ok := seenCodes[code]; ok {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		seenCodes[code] = struct{}{}
		option := model.SubscribePriceOption{
			ID:            item.Id,
			Code:          code,
			Type:          optionType,
			Name:          strings.TrimSpace(item.Name),
			DurationUnit:  unit,
			DurationValue: durationValue,
			Price:         item.Price,
			OriginalPrice: item.OriginalPrice,
			Inventory:     int64(item.Inventory),
			Show:          item.Show,
			Sell:          item.Sell,
			IsDefault:     item.IsDefault,
			Sort:          int64(item.Sort),
			Version:       item.Version,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		}
		isSellableDuration := option.Type == "duration" && option.Sell
		if isSellableDuration && option.Show {
			durationKey := fmt.Sprintf("%s:%d", option.DurationUnit, option.DurationValue)
			if _, ok := seenVisibleDurations[durationKey]; ok {
				return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
			}
			seenVisibleDurations[durationKey] = struct{}{}
		}
		if option.Name == "" {
			if unit == "NoLimit" {
				option.Name = "NoLimit"
			} else {
				option.Name = fmt.Sprintf("%d %s", durationValue, unit)
			}
		}
		if isSellableDuration && firstSellableIndex < 0 {
			firstSellableIndex = len(result)
		}
		if option.IsDefault && isSellableDuration {
			if hasDefault {
				option.IsDefault = false
			} else {
				hasDefault = true
			}
		} else if option.IsDefault {
			option.IsDefault = false
		}
		if option.Sort == 0 {
			option.Sort = int64(len(items) - i)
		}
		result = append(result, option)
	}
	if firstSellableIndex < 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if !hasDefault {
		result[firstSellableIndex].IsDefault = true
	}
	return result, nil
}

func convertPriceOptionsToProto(items []*ent.ProxySubscribePriceOption) []*v1.SubscribePriceOption {
	if len(items) == 0 {
		return []*v1.SubscribePriceOption{}
	}
	result := make([]*v1.SubscribePriceOption, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.SubscribePriceOption{
			Id:            item.ID,
			SubscribeId:   item.SubscribeID,
			Code:          item.Code,
			Type:          item.OptionType,
			Name:          item.Name,
			DurationUnit:  item.DurationUnit,
			DurationValue: item.DurationValue,
			Price:         item.Price,
			OriginalPrice: item.OriginalPrice,
			Inventory:     item.Inventory,
			Show:          item.Show,
			Sell:          item.Sell,
			IsDefault:     item.IsDefault,
			Sort:          item.Sort,
			Version:       item.Version,
			CreatedAt:     item.CreatedAt.Unix(),
			UpdatedAt:     item.UpdatedAt.Unix(),
		})
	}
	return result
}

// convertSubscribeToProto convert ent subscribe to proto subscribe info
func convertSubscribeToProto(sub *ent.ProxySubscribe) *v1.SubscribeInfo {
	desc := ""
	if sub.Description != nil {
		desc = *sub.Description
	}
	shortDescription := ""
	if sub.ShortDescription != nil {
		shortDescription = *sub.ShortDescription
	}
	features := ""
	if sub.Features != nil {
		features = *sub.Features
	}
	detailContent := ""
	if sub.DetailContent != nil {
		detailContent = *sub.DetailContent
	}
	discount := ""
	if sub.Discount != nil {
		discount = *sub.Discount
	}
	trafficLimit := ""
	if sub.TrafficLimit != nil {
		trafficLimit = *sub.TrafficLimit
	}
	deductionRatio := int64(0)
	if sub.DeductionRatio != nil {
		deductionRatio = int64(*sub.DeductionRatio)
	}
	allowDeduction := sub.AllowDeduction
	resetCycle := int64(0)
	if sub.ResetCycle != nil {
		resetCycle = int64(*sub.ResetCycle)
	}
	renewalReset := sub.RenewalReset

	return &v1.SubscribeInfo{
		Id:                int64(sub.ID),
		Name:              sub.Name,
		Language:          sub.Language,
		Description:       desc,
		ShortDescription:  shortDescription,
		Features:          features,
		DetailFormat:      sub.DetailFormat,
		DetailContent:     detailContent,
		UnitPrice:         int64(sub.UnitPrice),
		UnitTime:          sub.UnitTime,
		Discount:          convertDiscountFromJSON(discount),
		Replacement:       int64(sub.Replacement),
		Inventory:         int32(sub.Inventory),
		Traffic:           int64(sub.Traffic),
		SpeedLimit:        int32(sub.SpeedLimit),
		DeviceLimit:       int32(sub.DeviceLimit),
		Quota:             int32(sub.Quota),
		CategoryId:        sub.CategoryID,
		Nodes:             stringToInt64Slice(sub.Nodes),
		NodeTags:          stringToStringSlice(sub.NodeTags),
		NodeGroupIds:      cloneInt64Slice(sub.NodeGroupIds),
		NodeGroupId:       derefInt64(sub.NodeGroupID),
		TrafficLimit:      convertTrafficLimitFromJSON(trafficLimit),
		Show:              sub.Show,
		Sell:              sub.Sell,
		Sort:              int32(sub.Sort),
		DeductionRatio:    int32(deductionRatio),
		AllowDeduction:    allowDeduction,
		ResetCycle:        int32(resetCycle),
		RenewalReset:      renewalReset,
		ShowOriginalPrice: sub.ShowOriginalPrice,
		CreatedAt:         sub.CreatedAt.Unix(),
		UpdatedAt:         sub.UpdatedAt.Unix(),
	}
}

// convertSubscribeToProtoItem convert ent subscribe to proto subscribe item
func convertSubscribeToProtoItem(sub *ent.ProxySubscribe) *v1.SubscribeItem {
	desc := ""
	if sub.Description != nil {
		desc = *sub.Description
	}
	shortDescription := ""
	if sub.ShortDescription != nil {
		shortDescription = *sub.ShortDescription
	}
	features := ""
	if sub.Features != nil {
		features = *sub.Features
	}
	detailContent := ""
	if sub.DetailContent != nil {
		detailContent = *sub.DetailContent
	}
	discount := ""
	if sub.Discount != nil {
		discount = *sub.Discount
	}
	trafficLimit := ""
	if sub.TrafficLimit != nil {
		trafficLimit = *sub.TrafficLimit
	}
	deductionRatio := int64(0)
	if sub.DeductionRatio != nil {
		deductionRatio = int64(*sub.DeductionRatio)
	}
	allowDeduction := sub.AllowDeduction
	resetCycle := int64(0)
	if sub.ResetCycle != nil {
		resetCycle = int64(*sub.ResetCycle)
	}
	renewalReset := sub.RenewalReset

	return &v1.SubscribeItem{
		Id:                int64(sub.ID),
		Name:              sub.Name,
		Language:          sub.Language,
		Description:       desc,
		ShortDescription:  shortDescription,
		Features:          features,
		DetailFormat:      sub.DetailFormat,
		DetailContent:     detailContent,
		UnitPrice:         int64(sub.UnitPrice),
		UnitTime:          sub.UnitTime,
		Discount:          convertDiscountFromJSON(discount),
		Replacement:       int64(sub.Replacement),
		Inventory:         int32(sub.Inventory),
		Traffic:           int64(sub.Traffic),
		SpeedLimit:        int32(sub.SpeedLimit),
		DeviceLimit:       int32(sub.DeviceLimit),
		Quota:             int32(sub.Quota),
		CategoryId:        sub.CategoryID,
		Nodes:             stringToInt64Slice(sub.Nodes),
		NodeTags:          stringToStringSlice(sub.NodeTags),
		NodeGroupIds:      cloneInt64Slice(sub.NodeGroupIds),
		NodeGroupId:       derefInt64(sub.NodeGroupID),
		TrafficLimit:      convertTrafficLimitFromJSON(trafficLimit),
		Show:              sub.Show,
		Sell:              sub.Sell,
		Sort:              int32(sub.Sort),
		DeductionRatio:    int32(deductionRatio),
		AllowDeduction:    allowDeduction,
		ResetCycle:        int32(resetCycle),
		RenewalReset:      renewalReset,
		ShowOriginalPrice: sub.ShowOriginalPrice,
		CreatedAt:         sub.CreatedAt.Unix(),
		UpdatedAt:         sub.UpdatedAt.Unix(),
		Sold:              0, // Will be set by caller
	}
}
