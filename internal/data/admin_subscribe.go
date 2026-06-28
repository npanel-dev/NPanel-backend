package data

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribecategory"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribegroup"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribepriceoption"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	"github.com/npanel-dev/NPanel-backend/internal/biz/admin/subscribe"
	"github.com/npanel-dev/NPanel-backend/internal/model"
	"github.com/npanel-dev/NPanel-backend/pkg/uuidx"
)

const subscribeModule = "data/admin_subscribe"

type subscribeRepo struct {
	data *Data
	log  *log.Helper
}

// NewSubscribeRepo create subscribe repository
func NewSubscribeRepo(data *Data, logger log.Logger) subscribe.SubscribeRepo {
	return &subscribeRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", subscribeModule)),
	}
}

// ==================== Subscribe Operations ====================

// CreateSubscribe create subscribe
func (r *subscribeRepo) CreateSubscribe(ctx context.Context, sub *model.Subscribe) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	created, err := tx.ProxySubscribe.Create().
		SetName(sub.Name).
		SetLanguage(sub.Language).
		SetDescription(sub.Description).
		SetShortDescription(sub.ShortDescription).
		SetFeatures(sub.Features).
		SetDetailFormat(sub.DetailFormat).
		SetDetailContent(sub.DetailContent).
		SetUnitPrice(sub.UnitPrice).
		SetUnitTime(sub.UnitTime).
		SetDiscount(sub.Discount).
		SetReplacement(sub.Replacement).
		SetInventory(int32(sub.Inventory)).
		SetTraffic(sub.Traffic).
		SetSpeedLimit(int32(sub.SpeedLimit)).
		SetDeviceLimit(int32(sub.DeviceLimit)).
		SetQuota(int32(sub.Quota)).
		SetCategoryID(sub.CategoryID).
		SetNodes(sub.Nodes).
		SetNodeTags(sub.NodeTags).
		SetNodeGroupIds(sub.NodeGroupIDs).
		SetNodeGroupID(sub.NodeGroupID).
		SetTrafficLimit(sub.TrafficLimit).
		SetShow(sub.Show).
		SetSell(sub.Sell).
		SetSort(int32(sub.Sort)).
		SetDeductionRatio(int32(sub.DeductionRatio)).
		SetAllowDeduction(sub.AllowDeduction).
		SetResetCycle(int32(sub.ResetCycle)).
		SetRenewalReset(sub.RenewalReset).
		SetShowOriginalPrice(sub.ShowOriginalPrice).
		Save(ctx)
	if err != nil {
		return rollback(tx, err)
	}

	if err := r.syncSubscribePriceOptions(ctx, tx, created.ID, sub.PriceOptions); err != nil {
		return rollback(tx, err)
	}

	return tx.Commit()
}

// GetSubscribeByID get subscribe by ID
func (r *subscribeRepo) GetSubscribeByID(ctx context.Context, id int) (*ent.ProxySubscribe, error) {
	return r.data.db.ProxySubscribe.Query().
		Where(proxysubscribe.ID(int64(id))).
		Only(ctx)
}

// UpdateSubscribe update subscribe
func (r *subscribeRepo) UpdateSubscribe(ctx context.Context, sub *model.Subscribe) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.ProxySubscribe.Update().
		Where(proxysubscribe.ID(sub.ID)).
		SetName(sub.Name).
		SetLanguage(sub.Language).
		SetDescription(sub.Description).
		SetShortDescription(sub.ShortDescription).
		SetFeatures(sub.Features).
		SetDetailFormat(sub.DetailFormat).
		SetDetailContent(sub.DetailContent).
		SetUnitPrice(sub.UnitPrice).
		SetUnitTime(sub.UnitTime).
		SetDiscount(sub.Discount).
		SetReplacement(sub.Replacement).
		SetInventory(int32(sub.Inventory)).
		SetTraffic(sub.Traffic).
		SetSpeedLimit(int32(sub.SpeedLimit)).
		SetDeviceLimit(int32(sub.DeviceLimit)).
		SetQuota(int32(sub.Quota)).
		SetCategoryID(sub.CategoryID).
		SetNodes(sub.Nodes).
		SetNodeTags(sub.NodeTags).
		SetNodeGroupIds(sub.NodeGroupIDs).
		SetNodeGroupID(sub.NodeGroupID).
		SetTrafficLimit(sub.TrafficLimit).
		SetShow(sub.Show).
		SetSell(sub.Sell).
		SetSort(int32(sub.Sort)).
		SetDeductionRatio(int32(sub.DeductionRatio)).
		SetAllowDeduction(sub.AllowDeduction).
		SetResetCycle(int32(sub.ResetCycle)).
		SetRenewalReset(sub.RenewalReset).
		SetShowOriginalPrice(sub.ShowOriginalPrice).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}
	if err := r.syncSubscribePriceOptions(ctx, tx, sub.ID, sub.PriceOptions); err != nil {
		return rollback(tx, err)
	}
	return tx.Commit()
}

// DeleteSubscribe delete subscribe
func (r *subscribeRepo) DeleteSubscribe(ctx context.Context, id int) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.ProxySubscribePriceOption.Delete().
		Where(proxysubscribepriceoption.SubscribeIDEQ(int64(id))).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}
	if _, err := tx.ProxySubscribe.Delete().
		Where(proxysubscribe.ID(int64(id))).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}
	return tx.Commit()
}

// GetSubscribeList get subscribe list with pagination and filters
func (r *subscribeRepo) GetSubscribeList(ctx context.Context, req *model.SubscribeListParams) ([]*ent.ProxySubscribe, int32, error) {
	query := r.data.db.ProxySubscribe.Query()

	// Apply filters
	if req.Language != "" {
		query = query.Where(proxysubscribe.Language(req.Language))
	}

	if req.Search != "" {
		query = query.Where(proxysubscribe.NameContains(req.Search))
	}

	if req.NodeGroupID > 0 {
		query = query.Where(proxysubscribe.NodeGroupIDEQ(req.NodeGroupID))
	}

	if req.CategoryID > 0 {
		query = query.Where(proxysubscribe.CategoryIDEQ(req.CategoryID))
	}

	if len(req.IDs) > 0 {
		query = query.Where(proxysubscribe.IDIn(req.IDs...))
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (req.Page - 1) * req.Size
	list, err := query.
		Order(ent.Desc(proxysubscribe.FieldSort)).
		Offset(offset).
		Limit(req.Size).
		All(ctx)

	if err != nil {
		return nil, 0, err
	}

	return list, int32(total), nil
}

// CheckSubscribeInUse check if subscribe is being used by active user subscriptions
func (r *subscribeRepo) CheckSubscribeInUse(ctx context.Context, subscribeID int) (bool, error) {
	// Query user_subscribe table to check if there are active subscriptions
	// Note: This assumes user_subscribe table exists and has subscribe_id and status fields
	// Status 1 means active subscription
	// This is a simplified implementation - actual implementation should query the user service

	// For now, return false as we don't have the user_subscribe table schema
	// In production, you would query: SELECT COUNT(*) FROM user_subscribe WHERE subscribe_id=? AND status=1
	return false, nil
}

// BatchDeleteSubscribe batch delete subscribes
func (r *subscribeRepo) BatchDeleteSubscribe(ctx context.Context, ids []int) error {
	// Convert []int to []int64 for the query
	int64IDs := make([]int64, len(ids))
	for i, id := range ids {
		int64IDs[i] = int64(id)
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.ProxySubscribePriceOption.Delete().
		Where(proxysubscribepriceoption.SubscribeIDIn(int64IDs...)).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}
	if _, err := tx.ProxySubscribe.Delete().
		Where(proxysubscribe.IDIn(int64IDs...)).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}
	return tx.Commit()
}

func (r *subscribeRepo) GetSubscribePriceOptionsBySubscribeIDs(ctx context.Context, ids []int64) (map[int64][]*ent.ProxySubscribePriceOption, error) {
	result := make(map[int64][]*ent.ProxySubscribePriceOption)
	if len(ids) == 0 {
		return result, nil
	}
	items, err := r.data.db.ProxySubscribePriceOption.Query().
		Where(proxysubscribepriceoption.SubscribeIDIn(ids...)).
		Order(ent.Desc(proxysubscribepriceoption.FieldSort), ent.Asc(proxysubscribepriceoption.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.SubscribeID] = append(result[item.SubscribeID], item)
	}
	return result, nil
}

func (r *subscribeRepo) syncSubscribePriceOptions(ctx context.Context, tx *ent.Tx, subscribeID int64, options []model.SubscribePriceOption) error {
	existing, err := tx.ProxySubscribePriceOption.Query().
		Where(proxysubscribepriceoption.SubscribeIDEQ(subscribeID)).
		All(ctx)
	if err != nil {
		return err
	}
	existingByID := make(map[int64]*ent.ProxySubscribePriceOption, len(existing))
	for _, item := range existing {
		existingByID[item.ID] = item
	}
	submittedIDs := make(map[int64]struct{}, len(options))
	for _, option := range options {
		if option.ID > 0 {
			existingOption, ok := existingByID[option.ID]
			if !ok {
				return fmt.Errorf("price option %d does not belong to subscribe %d", option.ID, subscribeID)
			}
			if existingOption.SubscribeID != subscribeID {
				return fmt.Errorf("price option %d does not belong to subscribe %d", option.ID, subscribeID)
			}
			submittedIDs[option.ID] = struct{}{}
			if option.UpdatedAt > 0 && existingOption.UpdatedAt.Unix() > option.UpdatedAt {
				return model.ErrSubscribePriceOptionModified
			}
			if !priceOptionChanged(existingOption, option) {
				continue
			}
			affected, err := tx.ProxySubscribePriceOption.Update().
				Where(
					proxysubscribepriceoption.IDEQ(option.ID),
					proxysubscribepriceoption.SubscribeIDEQ(subscribeID),
					proxysubscribepriceoption.VersionEQ(option.Version),
				).
				SetCode(option.Code).
				SetOptionType(option.Type).
				SetName(option.Name).
				SetDurationUnit(option.DurationUnit).
				SetDurationValue(option.DurationValue).
				SetPrice(option.Price).
				SetOriginalPrice(option.OriginalPrice).
				SetInventory(int32(option.Inventory)).
				SetShow(option.Show).
				SetSell(option.Sell).
				SetIsDefault(option.IsDefault).
				SetSort(int32(option.Sort)).
				AddVersion(1).
				Save(ctx)
			if err != nil {
				return err
			}
			if affected == 0 {
				return model.ErrSubscribePriceOptionModified
			}
			continue
		}
		_, err := tx.ProxySubscribePriceOption.Create().
			SetSubscribeID(subscribeID).
			SetCode(option.Code).
			SetOptionType(option.Type).
			SetName(option.Name).
			SetDurationUnit(option.DurationUnit).
			SetDurationValue(option.DurationValue).
			SetPrice(option.Price).
			SetOriginalPrice(option.OriginalPrice).
			SetInventory(int32(option.Inventory)).
			SetShow(option.Show).
			SetSell(option.Sell).
			SetIsDefault(option.IsDefault).
			SetSort(int32(option.Sort)).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			return err
		}
	}
	archiveIDs := make([]int64, 0)
	for _, item := range existing {
		if _, ok := submittedIDs[item.ID]; !ok {
			archiveIDs = append(archiveIDs, item.ID)
		}
	}
	if len(archiveIDs) > 0 {
		if err := tx.ProxySubscribePriceOption.Update().
			Where(
				proxysubscribepriceoption.SubscribeIDEQ(subscribeID),
				proxysubscribepriceoption.IDIn(archiveIDs...),
			).
			SetShow(false).
			SetSell(false).
			SetIsDefault(false).
			AddVersion(1).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func priceOptionChanged(existing *ent.ProxySubscribePriceOption, option model.SubscribePriceOption) bool {
	return existing.Code != option.Code ||
		existing.OptionType != option.Type ||
		existing.Name != option.Name ||
		existing.DurationUnit != option.DurationUnit ||
		existing.DurationValue != option.DurationValue ||
		existing.Price != option.Price ||
		existing.OriginalPrice != option.OriginalPrice ||
		existing.Inventory != int32(option.Inventory) ||
		existing.Show != option.Show ||
		existing.Sell != option.Sell ||
		existing.IsDefault != option.IsDefault ||
		existing.Sort != int32(option.Sort)
}

// GetSubscribeMinSort get minimum sort value for given IDs
func (r *subscribeRepo) GetSubscribeMinSort(ctx context.Context, ids []int) (int64, error) {
	// Convert []int to []int64 for the query
	int64IDs := make([]int64, len(ids))
	for i, id := range ids {
		int64IDs[i] = int64(id)
	}
	subscribes, err := r.data.db.ProxySubscribe.Query().
		Where(proxysubscribe.IDIn(int64IDs...)).
		Order(ent.Asc(proxysubscribe.FieldSort)).
		Limit(1).
		All(ctx)

	if err != nil {
		return 0, err
	}

	if len(subscribes) == 0 {
		return 0, nil
	}

	return int64(subscribes[0].Sort), nil
}

// BatchUpdateSubscribeSort batch update subscribe sort values
func (r *subscribeRepo) BatchUpdateSubscribeSort(ctx context.Context, subscribes []*ent.ProxySubscribe) error {
	// Use transaction to update all subscribes
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}

	for _, sub := range subscribes {
		err = tx.ProxySubscribe.UpdateOneID(sub.ID).
			SetSort(int32(sub.Sort)).
			Exec(ctx)
		if err != nil {
			return rollback(tx, err)
		}
	}

	return tx.Commit()
}

// ==================== Subscribe Category Operations ====================

// CreateSubscribeCategory create subscribe category.
func (r *subscribeRepo) CreateSubscribeCategory(ctx context.Context, category *model.SubscribeCategory) error {
	_, err := r.data.db.ProxySubscribeCategory.Create().
		SetParentID(category.ParentID).
		SetName(category.Name).
		SetDescription(category.Description).
		SetLanguage(category.Language).
		SetShow(category.Show).
		SetSort(int32(category.Sort)).
		Save(ctx)
	return err
}

// GetSubscribeCategoryByID get subscribe category by ID.
func (r *subscribeRepo) GetSubscribeCategoryByID(ctx context.Context, id int64) (*ent.ProxySubscribeCategory, error) {
	return r.data.db.ProxySubscribeCategory.Query().
		Where(proxysubscribecategory.ID(id)).
		Only(ctx)
}

// UpdateSubscribeCategory update subscribe category.
func (r *subscribeRepo) UpdateSubscribeCategory(ctx context.Context, category *model.SubscribeCategory) error {
	return r.data.db.ProxySubscribeCategory.Update().
		Where(proxysubscribecategory.ID(category.ID)).
		SetParentID(category.ParentID).
		SetName(category.Name).
		SetDescription(category.Description).
		SetLanguage(category.Language).
		SetShow(category.Show).
		SetSort(int32(category.Sort)).
		Exec(ctx)
}

// DeleteSubscribeCategory delete subscribe category.
func (r *subscribeRepo) DeleteSubscribeCategory(ctx context.Context, id int64) error {
	_, err := r.data.db.ProxySubscribeCategory.Delete().
		Where(proxysubscribecategory.ID(id)).
		Exec(ctx)
	return err
}

// BatchDeleteSubscribeCategory batch delete subscribe categories.
func (r *subscribeRepo) BatchDeleteSubscribeCategory(ctx context.Context, ids []int64) error {
	_, err := r.data.db.ProxySubscribeCategory.Delete().
		Where(proxysubscribecategory.IDIn(ids...)).
		Exec(ctx)
	return err
}

// GetSubscribeCategoryList get subscribe category list.
func (r *subscribeRepo) GetSubscribeCategoryList(ctx context.Context, req *model.SubscribeCategoryListParams) ([]*ent.ProxySubscribeCategory, int32, error) {
	query := r.data.db.ProxySubscribeCategory.Query()
	if req.Language != "" {
		query = query.Where(proxysubscribecategory.Language(req.Language))
	}
	if req.ParentID != nil {
		query = query.Where(proxysubscribecategory.ParentIDEQ(*req.ParentID))
	}
	if req.Show != nil {
		query = query.Where(proxysubscribecategory.ShowEQ(*req.Show))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	list, err := query.
		Order(ent.Asc(proxysubscribecategory.FieldSort), ent.Asc(proxysubscribecategory.FieldID)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return list, int32(total), nil
}

// CountSubscribeByCategoryID counts subscribe plans bound to a category.
func (r *subscribeRepo) CountSubscribeByCategoryID(ctx context.Context, categoryID int64) (int, error) {
	return r.data.db.ProxySubscribe.Query().
		Where(proxysubscribe.CategoryIDEQ(categoryID)).
		Count(ctx)
}

// CountSubscribeCategoryChildren counts child categories.
func (r *subscribeRepo) CountSubscribeCategoryChildren(ctx context.Context, categoryID int64) (int, error) {
	return r.data.db.ProxySubscribeCategory.Query().
		Where(proxysubscribecategory.ParentIDEQ(categoryID)).
		Count(ctx)
}

// ==================== Subscribe Group Operations ====================

// CreateSubscribeGroup create subscribe group
func (r *subscribeRepo) CreateSubscribeGroup(ctx context.Context, group *model.SubscribeGroup) error {
	_, err := r.data.db.ProxySubscribeGroup.Create().
		SetName(group.Name).
		SetDescription(group.Description).
		SetIsExpiredGroup(group.IsExpiredGroup).
		SetExpiredDaysLimit(int32(group.ExpiredDaysLimit)).
		SetMaxTrafficGBExpired(int32(group.MaxTrafficGBExpired)).
		SetSpeedLimit(group.SpeedLimit).
		Save(ctx)

	return err
}

// GetSubscribeGroupByID get subscribe group by ID
func (r *subscribeRepo) GetSubscribeGroupByID(ctx context.Context, id int) (*ent.ProxySubscribeGroup, error) {
	return r.data.db.ProxySubscribeGroup.Query().
		Where(proxysubscribegroup.ID(int64(id))).
		Only(ctx)
}

// UpdateSubscribeGroup update subscribe group
func (r *subscribeRepo) UpdateSubscribeGroup(ctx context.Context, group *model.SubscribeGroup) error {
	return r.data.db.ProxySubscribeGroup.Update().
		Where(proxysubscribegroup.ID(group.ID)).
		SetName(group.Name).
		SetDescription(group.Description).
		SetIsExpiredGroup(group.IsExpiredGroup).
		SetExpiredDaysLimit(int32(group.ExpiredDaysLimit)).
		SetMaxTrafficGBExpired(int32(group.MaxTrafficGBExpired)).
		SetSpeedLimit(group.SpeedLimit).
		Exec(ctx)
}

// DeleteSubscribeGroup delete subscribe group
func (r *subscribeRepo) DeleteSubscribeGroup(ctx context.Context, id int) error {
	_, err := r.data.db.ProxySubscribeGroup.Delete().
		Where(proxysubscribegroup.ID(int64(id))).
		Exec(ctx)
	return err
}

// GetSubscribeGroupList get all subscribe groups (no pagination)
func (r *subscribeRepo) GetSubscribeGroupList(ctx context.Context) ([]*ent.ProxySubscribeGroup, int32, error) {
	list, err := r.data.db.ProxySubscribeGroup.Query().
		All(ctx)

	if err != nil {
		return nil, 0, err
	}

	return list, int32(len(list)), nil
}

// BatchDeleteSubscribeGroup batch delete subscribe groups
func (r *subscribeRepo) BatchDeleteSubscribeGroup(ctx context.Context, ids []int) error {
	// Convert []int to []int64 for the query
	int64IDs := make([]int64, len(ids))
	for i, id := range ids {
		int64IDs[i] = int64(id)
	}
	_, err := r.data.db.ProxySubscribeGroup.Delete().
		Where(proxysubscribegroup.IDIn(int64IDs...)).
		Exec(ctx)
	return err
}

// ==================== User Subscription Operations ====================

// GetActiveUserSubscriptionCount get active user subscription count for a subscribe
func (r *subscribeRepo) GetActiveUserSubscriptionCount(ctx context.Context, subscribeID int) (int64, error) {
	// 查询ProxyUserSubscribe表，统计该订阅套餐的活跃用户数
	// 活跃订阅：status=1（激活状态）
	status := int8(1) // 激活状态
	count, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.SubscribeIDEQ(int64(subscribeID)),
			proxyusersubscribe.StatusEQ(status),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

// GetActiveUserSubscriptionCountByIDs get active user subscription counts for multiple subscribes
func (r *subscribeRepo) GetActiveUserSubscriptionCountByIDs(ctx context.Context, subscribeIDs []int64) (map[int64]int64, error) {
	// 查询ProxyUserSubscribe表，统计多个订阅套餐的活跃用户数
	// 活跃订阅：status=1（激活状态）
	result := make(map[int64]int64)
	status := int8(1) // 激活状态

	// 为每个订阅套餐ID统计用户数
	for _, id := range subscribeIDs {
		count, err := r.data.db.ProxyUserSubscribe.Query().
			Where(
				proxyusersubscribe.SubscribeIDEQ(id),
				proxyusersubscribe.StatusEQ(status),
			).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		result[id] = int64(count)
	}
	return result, nil
}

func (r *subscribeRepo) ResetAllSubscribeToken(ctx context.Context) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}

	userSubs, err := tx.ProxyUserSubscribe.Query().
		Where(proxyusersubscribe.StatusIn(1, 2)).
		All(ctx)
	if err != nil {
		return rollback(tx, err)
	}

	nowMillis := time.Now().UnixMilli()
	oldTokens := make(map[int64]string, len(userSubs))
	for _, userSub := range userSubs {
		if userSub.Token != nil {
			oldTokens[userSub.ID] = *userSub.Token
		}
		token := uuidx.SubscribeToken(fmt.Sprintf("%d%d", nowMillis, userSub.ID))
		subscribeUUID := uuid.NewString()
		if _, err := tx.ProxyUserSubscribe.UpdateOneID(userSub.ID).
			SetToken(token).
			SetUUID(subscribeUUID).
			Save(ctx); err != nil {
			return rollback(tx, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// rollback helper function to rollback transaction
func rollback(tx *ent.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		return rerr
	}
	return err
}
