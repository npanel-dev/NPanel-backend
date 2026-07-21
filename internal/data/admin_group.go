package data

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/npanel-dev/NPanel-backend/api/admin/group/v1"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxygrouphistory"
	"github.com/npanel-dev/NPanel-backend/ent/proxygrouphistorydetail"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyservergroup"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	groupbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/group"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type adminGroupRepo struct {
	data   *Data
	logger *log.Helper
}

func int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func int32WrapperValue(v *wrapperspb.Int32Value, def int32) int32 {
	if v == nil {
		return def
	}
	return v.Value
}

func int64WrapperValue(v *wrapperspb.Int64Value, def int64) int64 {
	if v == nil {
		return def
	}
	return v.Value
}

func boolWrapperValue(v *wrapperspb.BoolValue, def bool) bool {
	if v == nil {
		return def
	}
	return v.Value
}

// NewAdminGroupRepo creates a new admin group repository
func NewAdminGroupRepo(d *Data, logger log.Logger) groupbiz.GroupRepo {
	return &adminGroupRepo{
		data:   d,
		logger: log.NewHelper(logger),
	}
}

// ===== Node Group CRUD =====

// CreateNodeGroup creates node group
func (r *adminGroupRepo) CreateNodeGroup(ctx context.Context, req *v1.CreateNodeGroupRequest) (int64, error) {
	groupType := req.Type
	if groupType == "" {
		groupType = "common"
	}
	isExpiredGroup := boolWrapperValue(req.IsExpiredGroup, false)
	forCalculation := boolWrapperValue(req.ForCalculation, true)
	expiredDaysLimit := int32WrapperValue(req.ExpiredDaysLimit, 7)
	speedLimit := int32WrapperValue(req.SpeedLimit, 0)
	maxTrafficGBExpired := int64WrapperValue(req.MaxTrafficGbExpired, 0)
	minTrafficGB := int64WrapperValue(req.MinTrafficGb, 0)
	maxTrafficGB := int64WrapperValue(req.MaxTrafficGb, 0)

	if isExpiredGroup {
		count, err := r.data.db.ProxyServerGroup.Query().
			Where(proxyservergroup.IsExpiredGroupEQ(true)).
			Count(ctx)
		if err != nil {
			r.logger.Errorf("Failed to check expired node group count: %v", err)
			return 0, err
		}
		if count > 0 {
			return 0, fmt.Errorf("system already has an expired node group, cannot create multiple")
		}
		forCalculation = false
	}

	if minTrafficGB != 0 || maxTrafficGB != 0 {
		if err := r.validateTrafficRange(ctx, 0, minTrafficGB, maxTrafficGB); err != nil {
			return 0, err
		}
	}

	group, err := r.data.db.ProxyServerGroup.Create().
		SetName(req.Name).
		SetGroupType(groupType).
		SetDescription(req.Description).
		SetSort(int(req.Sort)).
		SetForCalculation(forCalculation).
		SetIsExpiredGroup(isExpiredGroup).
		SetExpiredDaysLimit(int(expiredDaysLimit)).
		SetSpeedLimit(int(speedLimit)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		SetMinTrafficGB(minTrafficGB).
		SetMaxTrafficGB(maxTrafficGB).
		SetMaxTrafficGBExpired(maxTrafficGBExpired).
		Save(ctx)

	if err != nil {
		r.logger.Errorf("Failed to create node group: %v", err)
		return 0, err
	}

	return group.ID, nil
}

// UpdateNodeGroup updates node group
func (r *adminGroupRepo) UpdateNodeGroup(ctx context.Context, req *v1.UpdateNodeGroupRequest) error {
	groupID := req.Id
	if groupID <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	existing, err := r.data.db.ProxyServerGroup.Query().
		Where(proxyservergroup.IDEQ(groupID)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrServerGroupNotFound)
		}
		return err
	}

	if req.IsExpiredGroup != nil && req.IsExpiredGroup.Value {
		count, err := r.data.db.ProxyServerGroup.Query().
			Where(
				proxyservergroup.IsExpiredGroupEQ(true),
				proxyservergroup.IDNEQ(groupID),
			).
			Count(ctx)
		if err != nil {
			r.logger.Errorf("Failed to check expired node group count: %v", err)
			return err
		}
		if count > 0 {
			return fmt.Errorf("system already has an expired node group, cannot create multiple")
		}

		subscribeCount, err := r.data.db.ProxySubscribe.Query().
			Where(proxysubscribe.NodeGroupIDEQ(groupID)).
			Count(ctx)
		if err != nil {
			r.logger.Errorf("Failed to check default node group usage: %v", err)
			return err
		}
		if subscribeCount > 0 {
			return fmt.Errorf("this node group is used as default node group in subscription products, cannot set as expired group")
		}
	}

	newMinTrafficGB := int64Value(existing.MinTrafficGB)
	newMaxTrafficGB := int64Value(existing.MaxTrafficGB)
	if req.MinTrafficGb != nil {
		newMinTrafficGB = req.MinTrafficGb.Value
	}
	if req.MaxTrafficGb != nil {
		newMaxTrafficGB = req.MaxTrafficGb.Value
	}
	if req.MinTrafficGb != nil || req.MaxTrafficGb != nil {
		if err := r.validateTrafficRange(ctx, groupID, newMinTrafficGB, newMaxTrafficGB); err != nil {
			return err
		}
	}

	update := r.data.db.ProxyServerGroup.UpdateOneID(groupID)

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Type != "" {
		update.SetGroupType(req.Type)
	}
	if req.Sort != 0 {
		update.SetSort(int(req.Sort))
	}
	if req.ForCalculation != nil {
		update.SetForCalculation(req.ForCalculation.Value)
	}
	if req.IsExpiredGroup != nil {
		update.SetIsExpiredGroup(req.IsExpiredGroup.Value)
		if req.IsExpiredGroup.Value {
			update.SetForCalculation(false)
		}
	}
	if req.ExpiredDaysLimit != nil {
		update.SetExpiredDaysLimit(int(req.ExpiredDaysLimit.Value))
	}
	if req.MaxTrafficGbExpired != nil {
		update.SetMaxTrafficGBExpired(req.MaxTrafficGbExpired.Value)
	}
	if req.SpeedLimit != nil {
		update.SetSpeedLimit(int(req.SpeedLimit.Value))
	}
	if req.MinTrafficGb != nil {
		update.SetMinTrafficGB(req.MinTrafficGb.Value)
	}
	if req.MaxTrafficGb != nil {
		update.SetMaxTrafficGB(req.MaxTrafficGb.Value)
	}
	update.SetUpdatedAt(time.Now())

	return update.Exec(ctx)
}

// DeleteNodeGroup deletes node group
// 完全按照原项目逻辑：先检查是否有关联节点，有节点则不允许删除
func (r *adminGroupRepo) DeleteNodeGroup(ctx context.Context, id int64) error {
	// 1. 检查节点组是否存在
	_, err := r.data.db.ProxyServerGroup.Query().
		Where(proxyservergroup.IDEQ(id)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrServerGroupNotFound)
		}
		return err
	}

	// 2. 【关键】检查是否有关联节点
	nodeCount, err := r.data.db.ProxyNode.Query().
		Where(func(s *sql.Selector) {
			s.Where(jsonArrayContainsInt64(r.data.databaseDriver, proxynode.FieldNodeGroupIds, id))
		}).
		Count(ctx)

	if err != nil {
		r.logger.Errorf("Failed to count nodes in group: %v", err)
		return err
	}

	if nodeCount > 0 {
		r.logger.Infof("Cannot delete group %d: has %d associated nodes", id, nodeCount)
		return fmt.Errorf("cannot delete group with %d associated nodes, please migrate nodes first", nodeCount)
	}

	// 3. 删除节点组
	deletedCount, err := r.data.db.ProxyServerGroup.Delete().
		Where(proxyservergroup.IDEQ(id)).
		Exec(ctx)

	if err != nil {
		return err
	}

	if deletedCount == 0 {
		return responsecode.NewKratosError(responsecode.ErrServerGroupNotFound)
	}

	r.logger.Infof("Deleted node group: id=%d", id)
	return nil
}

// validateTrafficRange 校验流量区间：不能重叠、不能留空档、最小值不能大于最大值
// 完全按照原项目逻辑实现
func (r *adminGroupRepo) validateTrafficRange(ctx context.Context, currentNodeGroupID int64, newMin, newMax int64) error {
	// 1. 检查最小值是否大于最大值
	if newMin > 0 && newMax > 0 && newMin > newMax {
		r.logger.Errorf("Invalid traffic range: min(%d) > max(%d)", newMin, newMax)
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 2. 如果两个值都为0，表示不参与流量分组，不需要校验
	if newMin == 0 && newMax == 0 {
		return nil
	}

	// 3. 查询所有其他设置了流量区间的节点组
	otherGroups, err := r.data.db.ProxyServerGroup.Query().
		Where(
			proxyservergroup.IDNEQ(currentNodeGroupID),
			proxyservergroup.Or(
				proxyservergroup.MinTrafficGBGT(0),
				proxyservergroup.MaxTrafficGBGT(0),
			),
		).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query other node groups: %v", err)
		return err
	}

	// 4. 检查是否有重叠
	for _, other := range otherGroups {
		otherMin := int64Value(other.MinTrafficGB)
		otherMax := int64Value(other.MaxTrafficGB)

		// 如果对方也没设置区间，跳过
		if otherMin == 0 && otherMax == 0 {
			continue
		}

		// 检查是否有重叠: 如果两个区间相交，就是重叠
		// 不重叠的条件是: newMax <= otherMin OR newMin >= otherMax
		overlaps := true
		if (newMax > 0 && otherMin > 0 && newMax <= otherMin) ||
			(newMin > 0 && otherMax > 0 && newMin >= otherMax) {
			overlaps = false
		}

		if overlaps {
			r.logger.Errorf("Traffic range overlap: current[%d-%d] overlaps with existing[%d-%d]",
				newMin, newMax, otherMin, otherMax)
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
	}

	return nil
}

// GetNodeGroupList gets node group list
func (r *adminGroupRepo) GetNodeGroupList(ctx context.Context, req *v1.GetNodeGroupListRequest) ([]*groupbiz.NodeGroupListItem, int32, error) {
	query := r.data.db.ProxyServerGroup.Query()
	if req.GroupId > 0 {
		query = query.Where(proxyservergroup.IDEQ(req.GroupId))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	groups, err := query.
		Order(ent.Asc(proxyservergroup.FieldSort), ent.Asc(proxyservergroup.FieldID)).
		Offset(int((req.Page - 1) * req.Size)).
		Limit(int(req.Size)).
		All(ctx)

	if err != nil {
		return nil, 0, err
	}

	items := make([]*groupbiz.NodeGroupListItem, 0, len(groups))
	for _, item := range groups {
		nodeCount, countErr := r.data.db.ProxyNode.Query().
			Where(func(s *sql.Selector) {
				s.Where(jsonArrayContainsInt64(r.data.databaseDriver, proxynode.FieldNodeGroupIds, item.ID))
			}).
			Count(ctx)
		if countErr != nil {
			return nil, 0, countErr
		}
		items = append(items, &groupbiz.NodeGroupListItem{
			Group:     item,
			NodeCount: int64(nodeCount),
		})
	}

	return items, int32(total), nil
}

// ===== Group Config =====

// GetGroupConfig gets group config
func (r *adminGroupRepo) GetGroupConfig(ctx context.Context) (*v1.GroupConfig, error) {
	enabled := false
	mode := "average"
	configMap := make(map[string]interface{})

	enabledSys, err := r.data.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("group"),
			proxysystem.KeyEQ("enabled"),
		).
		First(ctx)
	if err == nil {
		enabled = enabledSys.Value == "true" || enabledSys.Value == "1"
	} else if !ent.IsNotFound(err) {
		r.logger.Errorf("Failed to query group enabled config: %v", err)
		return nil, err
	}

	modeSys, err := r.data.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("group"),
			proxysystem.KeyEQ("mode"),
		).
		First(ctx)
	if err == nil && modeSys.Value != "" {
		mode = modeSys.Value
	} else if err != nil && !ent.IsNotFound(err) {
		r.logger.Errorf("Failed to query group mode config: %v", err)
		return nil, err
	}

	if averageSys, err := r.data.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("group"),
			proxysystem.KeyEQ("average_config"),
		).
		First(ctx); err == nil {
		var averageCfg map[string]interface{}
		if json.Unmarshal([]byte(averageSys.Value), &averageCfg) == nil {
			configMap["average_config"] = averageCfg
		}
	}

	if subscribeSys, err := r.data.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("group"),
			proxysystem.KeyEQ("subscribe_config"),
		).
		First(ctx); err == nil {
		var subscribeCfg map[string]interface{}
		if json.Unmarshal([]byte(subscribeSys.Value), &subscribeCfg) == nil {
			configMap["subscribe_config"] = subscribeCfg
		}
	}

	if trafficSys, err := r.data.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("group"),
			proxysystem.KeyEQ("traffic_config"),
		).
		First(ctx); err == nil {
		var trafficCfg map[string]interface{}
		if json.Unmarshal([]byte(trafficSys.Value), &trafficCfg) == nil {
			configMap["traffic_config"] = trafficCfg
		}
	}

	configBytes, err := json.Marshal(configMap)
	if err != nil {
		r.logger.Errorf("Failed to marshal group config: %v", err)
		return nil, err
	}

	return &v1.GroupConfig{
		Enabled: enabled,
		Mode:    mode,
		Config:  string(configBytes),
	}, nil
}

// UpdateGroupConfig updates group config
func (r *adminGroupRepo) UpdateGroupConfig(ctx context.Context, req *v1.UpdateGroupConfigRequest) error {
	if req.Mode != "" && req.Mode != "average" && req.Mode != "subscribe" && req.Mode != "traffic" {
		return fmt.Errorf("invalid mode: %s", req.Mode)
	}

	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		r.logger.Errorf("Failed to begin group config transaction: %v", err)
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	enabledValue := "false"
	if req.Enabled {
		enabledValue = "true"
	}
	if err = upsertGroupSystemConfigTx(ctx, tx, "enabled", enabledValue, "Group Feature Enabled"); err != nil {
		return err
	}

	if req.Mode != "" {
		if err = upsertGroupSystemConfigTx(ctx, tx, "mode", req.Mode, "Group Mode"); err != nil {
			return err
		}
	}

	if req.Config != "" {
		configMap := make(map[string]interface{})
		if err = json.Unmarshal([]byte(req.Config), &configMap); err != nil {
			r.logger.Errorf("Failed to unmarshal group config json: %v", err)
			return err
		}

		if averageConfig, ok := configMap["average_config"]; ok {
			jsonBytes, marshalErr := json.Marshal(averageConfig)
			if marshalErr != nil {
				return marshalErr
			}
			if err = upsertGroupSystemConfigTx(ctx, tx, "average_config", string(jsonBytes), "Average Group Config"); err != nil {
				return err
			}
		}

		if subscribeConfig, ok := configMap["subscribe_config"]; ok {
			jsonBytes, marshalErr := json.Marshal(subscribeConfig)
			if marshalErr != nil {
				return marshalErr
			}
			if err = upsertGroupSystemConfigTx(ctx, tx, "subscribe_config", string(jsonBytes), "Subscribe Group Config"); err != nil {
				return err
			}
		}

		if trafficConfig, ok := configMap["traffic_config"]; ok {
			jsonBytes, marshalErr := json.Marshal(trafficConfig)
			if marshalErr != nil {
				return marshalErr
			}
			if err = upsertGroupSystemConfigTx(ctx, tx, "traffic_config", string(jsonBytes), "Traffic Group Config"); err != nil {
				return err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		r.logger.Errorf("Failed to commit group config transaction: %v", err)
		return err
	}

	r.logger.Infof("Group config updated: enabled=%v, mode=%s", req.Enabled, req.Mode)
	return nil
}

func upsertGroupSystemConfigTx(ctx context.Context, tx *ent.Tx, key string, value string, desc string) error {
	existing, err := tx.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("group"),
			proxysystem.KeyEQ(key),
		).
		First(ctx)
	if err == nil {
		return tx.ProxySystem.UpdateOneID(existing.ID).
			SetValue(value).
			SetDesc(desc).
			SetUpdatedAt(time.Now()).
			Exec(ctx)
	}
	if !ent.IsNotFound(err) {
		return err
	}

	_, err = tx.ProxySystem.Create().
		SetCategory("group").
		SetKey(key).
		SetValue(value).
		SetDesc(desc).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

// ===== Group Recalculation =====

// RecalculateGroup recalculates groups with specified mode
func (r *adminGroupRepo) RecalculateGroup(ctx context.Context, mode string, triggerType string) (int64, error) {
	// 验证mode参数
	if mode != "average" && mode != "subscribe" && mode != "traffic" {
		return 0, fmt.Errorf("invalid mode: %s, must be one of: average, subscribe, traffic", mode)
	}

	// 设置默认trigger_type
	if triggerType == "" {
		triggerType = "manual"
	}

	// 开始事务
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		r.logger.Errorf("Failed to begin transaction: %v", err)
		return 0, err
	}

	// 确保事务在出错时回滚
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 创建历史记录
	history, err := tx.ProxyGroupHistory.Create().
		SetGroupMode(mode).
		SetTriggerType(triggerType).
		SetState("pending").
		SetTotalUsers(0).
		SetSuccessCount(0).
		SetFailedCount(0).
		SetCreatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		r.logger.Errorf("Failed to create group history: %v", err)
		return 0, err
	}

	r.logger.Infof("Group recalculation started: history_id=%d, mode=%s", history.ID, mode)

	// 更新状态为running
	err = r.updateHistoryStateTx(tx, ctx, history.ID, "running")
	if err != nil {
		r.logger.Errorf("Failed to update history state to running: %v", err)
		return 0, err
	}

	// 执行分组算法
	var affectedCount int
	var execErr error

	switch mode {
	case "average":
		affectedCount, execErr = r.executeAverageGroupingTx(tx, ctx, history.ID)
	case "subscribe":
		affectedCount, execErr = r.executeSubscribeGroupingTx(tx, ctx, history.ID)
	case "traffic":
		affectedCount, execErr = r.executeTrafficGroupingTx(tx, ctx, history.ID)
	default:
		execErr = fmt.Errorf("unsupported mode: %s", mode)
	}

	// 更新历史记录
	now := time.Now()
	if execErr != nil {
		// 失败 - 更新状态为failed
		updateErr := r.updateHistoryStateWithErrorTx(tx, ctx, history.ID, "failed", execErr.Error(), &now)
		if updateErr != nil {
			r.logger.Errorf("Failed to update history state to failed: %v", updateErr)
		}
		return 0, execErr
	}

	// 成功 - 更新状态为completed
	err = r.updateHistoryStateWithStatsTx(tx, ctx, history.ID, "completed", affectedCount, affectedCount, 0, &now)
	if err != nil {
		r.logger.Errorf("Failed to update history state to completed: %v", err)
		return 0, err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		r.logger.Errorf("Failed to commit transaction: %v", err)
		return 0, err
	}

	r.logger.Infof("Group recalculation completed: mode=%s, affected_users=%d", mode, affectedCount)
	return history.ID, nil
}

// executeAverageGrouping implements average grouping algorithm
func (r *adminGroupRepo) executeAverageGrouping(ctx context.Context, historyID int64) (int, error) {
	r.logger.Infof("Executing average grouping for history_id=%d", historyID)

	// 1. 查询所有有效且未锁定的用户订阅
	type UserSubscribeInfo struct {
		ID                 int64
		UserID             int64
		SubscribeID        int64
		NodeGroupIDs       string // JSON string
		CurrentNodeGroupID int64
	}

	var userSubscribes []UserSubscribeInfo
	err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.Or(
				proxyusersubscribe.StatusIsNil(),
				proxyusersubscribe.StatusIn(0, 1),
			),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		Select(
			proxyusersubscribe.FieldID,
			proxyusersubscribe.FieldUserID,
			proxyusersubscribe.FieldSubscribeID,
			proxyusersubscribe.FieldNodeGroupID,
		).
		Scan(ctx, &userSubscribes)

	if err != nil {
		r.logger.Errorf("Failed to query user subscribes: %v", err)
		return 0, err
	}

	r.logger.Infof("Found %d user subscribes for average grouping", len(userSubscribes))

	// 2. 查询订阅套餐的node_group_ids映射
	subscribeMap := make(map[int64][]int64)
	var subscribeIDs []int64
	for _, us := range userSubscribes {
		subscribeIDs = append(subscribeIDs, us.SubscribeID)
	}

	if len(subscribeIDs) == 0 {
		return 0, nil
	}

	var subscribes []*ent.ProxySubscribe
	subscribes, err = r.data.db.ProxySubscribe.Query().
		Where(proxysubscribe.IDIn(subscribeIDs...)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query subscribes: %v", err)
		return 0, err
	}

	// 构建映射
	for _, sub := range subscribes {
		var nodeGroupIDs []int64
		if sub.NodeGroupIds != nil {
			nodeGroupIDs = sub.NodeGroupIds
		}
		subscribeMap[sub.ID] = nodeGroupIDs
	}

	// 3. 随机分配节点组ID
	affectedCount := 0

	type UserInfo struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	groupUsersMap := make(map[int64][]UserInfo) // node_group_id -> users

	for _, us := range userSubscribes {
		nodeGroupIDs := subscribeMap[us.SubscribeID]

		// 如果没有节点组ID，设置为0
		selectedNodeGroupID := int64(0)
		if len(nodeGroupIDs) > 0 {
			// 随机选择一个节点组ID
			randomIndex := rand.Intn(len(nodeGroupIDs))
			selectedNodeGroupID = nodeGroupIDs[randomIndex]
		}

		// 更新user_subscribe
		err = r.data.db.ProxyUserSubscribe.Update().
			Where(proxyusersubscribe.IDEQ(us.ID)).
			SetNodeGroupID(selectedNodeGroupID).
			Exec(ctx)

		if err != nil {
			r.logger.Errorf("Failed to update node_group_id for user_subscribe_id=%d: %v", us.ID, err)
			continue
		}

		// 记录到分组用户映射（只记录有节点组的用户）
		if selectedNodeGroupID > 0 {
			// TODO: 查询用户邮箱
			groupUsersMap[selectedNodeGroupID] = append(groupUsersMap[selectedNodeGroupID], UserInfo{
				ID:    us.UserID,
				Email: "",
			})
		}

		affectedCount++
	}

	// 4. 创建历史详情记录
	for nodeGroupID, users := range groupUsersMap {
		userCount := len(users)
		if userCount == 0 {
			continue
		}

		// 统计该节点组的节点数
		var nodeCount int
		if nodeGroupID > 0 {
			nodeCount, err = r.data.db.ProxyNode.Query().
				Where(func(s *sql.Selector) {
					s.Where(jsonArrayContainsInt64(r.data.databaseDriver, proxynode.FieldNodeGroupIds, nodeGroupID))
				}).
				Count(ctx)

			if err != nil {
				r.logger.Errorf("Failed to count nodes for node_group_id=%d: %v", nodeGroupID, err)
			}
		}

		// 序列化用户数据
		userDataJSON := "[]"
		if jsonData, err := json.Marshal(users); err == nil {
			userDataJSON = string(jsonData)
		}

		// 创建历史详情
		_, err = r.data.db.ProxyGroupHistoryDetail.Create().
			SetHistoryID(historyID).
			SetNodeGroupID(nodeGroupID).
			SetUserCount(userCount).
			SetNodeCount(int(nodeCount)).
			SetUserData(userDataJSON).
			SetCreatedAt(time.Now()).
			Save(ctx)

		if err != nil {
			r.logger.Errorf("Failed to create history detail for node_group_id=%d: %v", nodeGroupID, err)
		}

		r.logger.Infof("Average grouping: node_group_id=%d, users=%d, nodes=%d", nodeGroupID, userCount, nodeCount)
	}

	return affectedCount, nil
}

// executeSubscribeGrouping implements subscribe grouping algorithm
func (r *adminGroupRepo) executeSubscribeGrouping(ctx context.Context, historyID int64) (int, error) {
	r.logger.Infof("Executing subscribe grouping for history_id=%d", historyID)

	// 1. 查询所有有效且未锁定的用户订阅
	type UserSubscribeInfo struct {
		ID                    int64
		UserID                int64
		SubscribeID           int64
		CurrentNodeGroupID    int64
		SubscribeNodeGroupID  *int64
		SubscribeNodeGroupIds []int64
	}

	var userSubscribes []UserSubscribeInfo
	err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.Or(
				proxyusersubscribe.StatusIsNil(),
				proxyusersubscribe.StatusIn(0, 1),
			),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		Select(
			proxyusersubscribe.FieldID,
			proxyusersubscribe.FieldUserID,
			proxyusersubscribe.FieldSubscribeID,
			proxyusersubscribe.FieldNodeGroupID,
		).
		Scan(ctx, &userSubscribes)

	if err != nil {
		r.logger.Errorf("Failed to query user subscribes: %v", err)
		return 0, err
	}

	// 2. 查询订阅套餐信息
	var subscribeIDs []int64
	for _, us := range userSubscribes {
		subscribeIDs = append(subscribeIDs, us.SubscribeID)
	}

	if len(subscribeIDs) == 0 {
		return 0, nil
	}

	subscribes, err := r.data.db.ProxySubscribe.Query().
		Where(proxysubscribe.IDIn(subscribeIDs...)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query subscribes: %v", err)
		return 0, err
	}

	// 构建订阅映射
	subscribeMap := make(map[int64]*ent.ProxySubscribe)
	for _, sub := range subscribes {
		subscribeMap[sub.ID] = sub
	}

	// 3. 根据优先级选择节点组ID
	affectedCount := 0

	type UserInfo struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	groupUsersMap := make(map[int64][]UserInfo)

	for _, us := range userSubscribes {
		subscribe, ok := subscribeMap[us.SubscribeID]
		if !ok {
			r.logger.Warnf("Subscribe not found for subscribe_id=%d", us.SubscribeID)
			continue
		}

		// 注意：这里不检查过期，因为subscribe是套餐模板，不是用户订阅
		// 用户订阅的过期检查应该在查询时通过status字段过滤

		var selectedNodeGroupID int64 = 0

		// 按优先级选择节点组ID:
		// 优先级1: user_subscribe.node_group_id (如果已设置且不为0)
		// 优先级2: subscribe.node_group_id (如果设置了)
		// 优先级3: subscribe.node_group_ids[0] (第一个元素)
		if us.CurrentNodeGroupID > 0 {
			selectedNodeGroupID = us.CurrentNodeGroupID
		} else if subscribe.NodeGroupID != nil && *subscribe.NodeGroupID > 0 {
			selectedNodeGroupID = *subscribe.NodeGroupID
		} else if len(subscribe.NodeGroupIds) > 0 && subscribe.NodeGroupIds[0] > 0 {
			selectedNodeGroupID = subscribe.NodeGroupIds[0]
		}

		// 更新user_subscribe
		err = r.data.db.ProxyUserSubscribe.Update().
			Where(proxyusersubscribe.IDEQ(us.ID)).
			SetNodeGroupID(selectedNodeGroupID).
			Exec(ctx)

		if err != nil {
			r.logger.Errorf("Failed to update node_group_id for user_subscribe_id=%d: %v", us.ID, err)
			continue
		}

		// 记录到分组用户映射
		if selectedNodeGroupID > 0 {
			// TODO: 查询用户邮箱
			groupUsersMap[selectedNodeGroupID] = append(groupUsersMap[selectedNodeGroupID], UserInfo{
				ID:    us.UserID,
				Email: "",
			})
		}

		affectedCount++
	}

	// 4. 创建历史详情记录（与average模式相同）
	for nodeGroupID, users := range groupUsersMap {
		userCount := len(users)
		if userCount == 0 {
			continue
		}

		var nodeCount int
		if nodeGroupID > 0 {
			nodeCount, err = r.data.db.ProxyNode.Query().
				Where(func(s *sql.Selector) {
					s.Where(jsonArrayContainsInt64(r.data.databaseDriver, proxynode.FieldNodeGroupIds, nodeGroupID))
				}).
				Count(ctx)

			if err != nil {
				r.logger.Errorf("Failed to count nodes for node_group_id=%d: %v", nodeGroupID, err)
			}
		}

		userDataJSON := "[]"
		if jsonData, err := json.Marshal(users); err == nil {
			userDataJSON = string(jsonData)
		}

		_, err = r.data.db.ProxyGroupHistoryDetail.Create().
			SetHistoryID(historyID).
			SetNodeGroupID(nodeGroupID).
			SetUserCount(userCount).
			SetNodeCount(int(nodeCount)).
			SetUserData(userDataJSON).
			SetCreatedAt(time.Now()).
			Save(ctx)

		if err != nil {
			r.logger.Errorf("Failed to create history detail for node_group_id=%d: %v", nodeGroupID, err)
		}

		r.logger.Infof("Subscribe grouping: node_group_id=%d, users=%d, nodes=%d", nodeGroupID, userCount, nodeCount)
	}

	return affectedCount, nil
}

// executeTrafficGrouping implements traffic grouping algorithm
func (r *adminGroupRepo) executeTrafficGrouping(ctx context.Context, historyID int64) (int, error) {
	r.logger.Infof("Executing traffic grouping for history_id=%d", historyID)

	// 1. 获取所有设置了流量区间的节点组
	nodeGroups, err := r.data.db.ProxyServerGroup.Query().
		Where(
			proxyservergroup.ForCalculationEQ(true),
			proxyservergroup.Or(
				proxyservergroup.MinTrafficGBGT(0),
				proxyservergroup.MaxTrafficGBGT(0),
			),
		).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query node groups: %v", err)
		return 0, err
	}

	if len(nodeGroups) == 0 {
		r.logger.Infof("No node groups with traffic ranges configured")
		return 0, nil
	}

	r.logger.Infof("Found %d node groups with traffic ranges", len(nodeGroups))

	// 2. 查询所有有效且未锁定的用户订阅及其已用流量
	type UserSubscribeInfo struct {
		ID          int64
		UserID      int64
		Upload      int64
		Download    int64
		UsedTraffic int64 // 已用流量（字节）
	}

	var userSubscribes []UserSubscribeInfo
	err = r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.Or(
				proxyusersubscribe.StatusIsNil(),
				proxyusersubscribe.StatusIn(0, 1),
			),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		Select(
			proxyusersubscribe.FieldID,
			proxyusersubscribe.FieldUserID,
			proxyusersubscribe.FieldUpload,
			proxyusersubscribe.FieldDownload,
		).
		Scan(ctx, &userSubscribes)

	if err != nil {
		r.logger.Errorf("Failed to query user subscribes: %v", err)
		return 0, err
	}

	if len(userSubscribes) == 0 {
		r.logger.Infof("No valid and unlocked user subscribes found")
		return 0, nil
	}

	r.logger.Infof("Found %d user subscribes for traffic-based grouping", len(userSubscribes))

	// 3. 根据流量范围分配节点组ID
	affectedCount := 0

	type UserInfo struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	groupUsersMap := make(map[int64][]UserInfo)

	for _, us := range userSubscribes {
		// 将字节转换为GB
		usedTrafficGB := float64(us.UsedTraffic) / (1024 * 1024 * 1024)

		// 查找匹配的流量范围（使用左闭右开区间 [Min, Max)）
		var targetNodeGroupID int64 = 0
		for _, ng := range nodeGroups {
			minTraffic := float64(int64Value(ng.MinTrafficGB))
			maxTraffic := float64(int64Value(ng.MaxTrafficGB))

			// 检查是否在区间内 [min, max)
			if usedTrafficGB >= minTraffic && (maxTraffic == 0 || usedTrafficGB < maxTraffic) {
				targetNodeGroupID = ng.ID
				break
			}
		}

		// 更新user_subscribe的node_group_id字段
		err = r.data.db.ProxyUserSubscribe.Update().
			Where(proxyusersubscribe.IDEQ(us.ID)).
			SetNodeGroupID(targetNodeGroupID).
			Exec(ctx)

		if err != nil {
			r.logger.Errorf("Failed to update node_group_id for user_subscribe_id=%d: %v", us.ID, err)
			continue
		}

		// 记录到分组用户映射
		if targetNodeGroupID > 0 {
			groupUsersMap[targetNodeGroupID] = append(groupUsersMap[targetNodeGroupID], UserInfo{
				ID:    us.UserID,
				Email: "",
			})
		}

		affectedCount++
	}

	// 4. 创建历史详情记录
	for nodeGroupID, users := range groupUsersMap {
		userCount := len(users)
		if userCount == 0 {
			continue
		}

		// 统计该节点组的节点数
		var nodeCount int
		if nodeGroupID > 0 {
			nodeCount, err = r.data.db.ProxyNode.Query().
				Where(func(s *sql.Selector) {
					s.Where(jsonArrayContainsInt64(r.data.databaseDriver, proxynode.FieldNodeGroupIds, nodeGroupID))
				}).
				Count(ctx)

			if err != nil {
				r.logger.Errorf("Failed to count nodes for node_group_id=%d: %v", nodeGroupID, err)
			}
		}

		userDataJSON := "[]"
		if jsonData, err := json.Marshal(users); err == nil {
			userDataJSON = string(jsonData)
		}

		_, err = r.data.db.ProxyGroupHistoryDetail.Create().
			SetHistoryID(historyID).
			SetNodeGroupID(nodeGroupID).
			SetUserCount(userCount).
			SetNodeCount(int(nodeCount)).
			SetUserData(userDataJSON).
			SetCreatedAt(time.Now()).
			Save(ctx)

		if err != nil {
			r.logger.Errorf("Failed to create history detail for node_group_id=%d: %v", nodeGroupID, err)
		}

		r.logger.Infof("Traffic grouping: node_group_id=%d, users=%d, nodes=%d", nodeGroupID, userCount, nodeCount)
	}

	r.logger.Infof("Traffic grouping completed: affected=%d", affectedCount)
	return affectedCount, nil
}

// GetRecalculationStatus gets recalculation status
func (r *adminGroupRepo) GetRecalculationStatus(ctx context.Context) (*v1.RecalculationState, error) {
	// 查询最新的历史记录状态
	history, err := r.data.db.ProxyGroupHistory.Query().
		Order(ent.Desc(proxygrouphistory.FieldCreatedAt)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			// 没有历史记录，返回idle状态
			return &v1.RecalculationState{
				State:    "idle",
				Progress: 0,
				Total:    0,
			}, nil
		}
		return nil, err
	}

	var progress int64
	var total int64

	switch history.State {
	case "pending":
		progress = 0
		total = 0
	case "running":
		progress = 50 // 运行中设为50%
		total = int64(history.TotalUsers)
	case "completed":
		progress = 100
		total = int64(history.TotalUsers)
	case "failed":
		progress = 0
		total = int64(history.TotalUsers)
	}

	return &v1.RecalculationState{
		State:    history.State,
		Progress: progress,
		Total:    int32(total),
	}, nil
}

// GetGroupHistory gets group history
func (r *adminGroupRepo) GetGroupHistory(ctx context.Context, req *v1.GetGroupHistoryRequest) ([]*ent.ProxyGroupHistory, int32, error) {
	query := r.data.db.ProxyGroupHistory.Query()

	// 可选过滤条件
	if req.GroupMode != "" {
		query = query.Where(proxygrouphistory.GroupModeEQ(req.GroupMode))
	}
	if req.TriggerType != "" {
		query = query.Where(proxygrouphistory.TriggerTypeEQ(req.TriggerType))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	histories, err := query.
		Order(ent.Desc(proxygrouphistory.FieldCreatedAt)).
		Offset(int((req.Page - 1) * req.Size)).
		Limit(int(req.Size)).
		All(ctx)

	if err != nil {
		return nil, 0, err
	}

	return histories, int32(total), nil
}

// GetGroupHistoryDetail gets group history detail
func (r *adminGroupRepo) GetGroupHistoryDetail(ctx context.Context, historyID int64) (*groupbiz.GroupHistoryDetailResult, error) {
	history, err := r.data.db.ProxyGroupHistory.Query().
		Where(proxygrouphistory.IDEQ(historyID)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewKratosError(responsecode.ErrServerGroupNotFound)
		}
		return nil, err
	}

	details, err := r.data.db.ProxyGroupHistoryDetail.Query().
		Where(proxygrouphistorydetail.HistoryIDEQ(historyID)).
		Order(ent.Asc(proxygrouphistorydetail.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	configSnapshot := ""
	var configKey string
	switch history.GroupMode {
	case "average":
		configKey = "average_config"
	case "subscribe":
		configKey = "subscribe_config"
	case "traffic":
		configKey = "traffic_config"
	}
	if configKey != "" {
		cfg, cfgErr := r.data.db.ProxySystem.Query().
			Where(
				proxysystem.CategoryEQ("group"),
				proxysystem.KeyEQ(configKey),
			).
			First(ctx)
		if cfgErr == nil {
			configSnapshot = cfg.Value
		} else if !ent.IsNotFound(cfgErr) {
			return nil, cfgErr
		}
	}

	return &groupbiz.GroupHistoryDetailResult{
		History:        history,
		Details:        details,
		ConfigSnapshot: configSnapshot,
	}, nil
}

// PreviewUserNodes previews user nodes with group permissions
func (r *adminGroupRepo) PreviewUserNodes(ctx context.Context, userID int64) ([]*groupbiz.PreviewNodeGroup, error) {
	// 1. 检查分组功能是否启用
	var enabledStr string
	enabledSys, err := r.data.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("group"),
			proxysystem.KeyEQ("enabled"),
		).
		Only(ctx)

	if err != nil {
		r.logger.Warnf("Failed to check group enabled status: %v", err)
		enabledStr = "false"
	} else {
		enabledStr = enabledSys.Value
	}
	enabled := enabledStr == "true" || enabledStr == "1"

	if !enabled {
		nodes, err := r.data.db.ProxyNode.Query().
			Where(
				proxynode.EnabledEQ(true),
				proxynode.IsHiddenEQ(false),
			).
			Order(ent.Asc(proxynode.FieldSort)).
			All(ctx)

		if err != nil {
			return nil, err
		}

		return []*groupbiz.PreviewNodeGroup{{
			ID:    0,
			Name:  "",
			Nodes: nodes,
		}}, nil
	}

	userSubscribes, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.UserIDEQ(userID),
			proxyusersubscribe.StatusIn(0, 1),
		).
		Order(ent.Desc(proxyusersubscribe.FieldExpireTime)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	if len(userSubscribes) == 0 {
		return []*groupbiz.PreviewNodeGroup{}, nil
	}

	subscribeIDs := make([]int64, 0, len(userSubscribes))
	for _, us := range userSubscribes {
		subscribeIDs = append(subscribeIDs, us.SubscribeID)
	}
	subscribes, err := r.data.db.ProxySubscribe.Query().
		Where(proxysubscribe.IDIn(subscribeIDs...)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	subscribeMap := make(map[int64]*ent.ProxySubscribe, len(subscribes))
	for _, sub := range subscribes {
		subscribeMap[sub.ID] = sub
	}

	nodeGroupIDSet := make(map[int64]struct{})
	for _, us := range userSubscribes {
		if us.NodeGroupID > 0 {
			nodeGroupIDSet[us.NodeGroupID] = struct{}{}
			continue
		}
		sub := subscribeMap[us.SubscribeID]
		if sub == nil {
			continue
		}
		if sub.NodeGroupID != nil && *sub.NodeGroupID > 0 {
			nodeGroupIDSet[*sub.NodeGroupID] = struct{}{}
			continue
		}
		if len(sub.NodeGroupIds) > 0 && sub.NodeGroupIds[0] > 0 {
			nodeGroupIDSet[sub.NodeGroupIds[0]] = struct{}{}
		}
	}

	nodes, err := r.data.db.ProxyNode.Query().
		Where(
			proxynode.EnabledEQ(true),
			proxynode.IsHiddenEQ(false),
		).
		Order(ent.Asc(proxynode.FieldSort), ent.Asc(proxynode.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	groupIDs := make([]int64, 0, len(nodeGroupIDSet))
	for id := range nodeGroupIDSet {
		groupIDs = append(groupIDs, id)
	}
	nodeGroupNameMap := make(map[int64]string)
	if len(groupIDs) > 0 {
		nodeGroups, err := r.data.db.ProxyServerGroup.Query().
			Where(proxyservergroup.IDIn(groupIDs...)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range nodeGroups {
			nodeGroupNameMap[item.ID] = item.Name
		}
	}

	grouped := make(map[int64][]*ent.ProxyNode)
	for _, node := range nodes {
		if len(node.NodeGroupIds) == 0 {
			grouped[0] = append(grouped[0], node)
			continue
		}
		firstGroupID := node.NodeGroupIds[0]
		if _, ok := nodeGroupIDSet[firstGroupID]; ok {
			grouped[firstGroupID] = append(grouped[firstGroupID], node)
			continue
		}
		grouped[0] = append(grouped[0], node)
	}

	resp := make([]*groupbiz.PreviewNodeGroup, 0, len(grouped))
	for _, groupID := range groupIDs {
		if len(grouped[groupID]) == 0 {
			continue
		}
		resp = append(resp, &groupbiz.PreviewNodeGroup{
			ID:    groupID,
			Name:  nodeGroupNameMap[groupID],
			Nodes: grouped[groupID],
		})
	}
	if len(grouped[0]) > 0 {
		resp = append(resp, &groupbiz.PreviewNodeGroup{
			ID:    0,
			Name:  "",
			Nodes: grouped[0],
		})
	}

	return resp, nil
}

// ResetGroups resets all user groups
func (r *adminGroupRepo) ResetGroups(ctx context.Context) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		r.logger.Errorf("Failed to begin reset groups transaction: %v", err)
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ProxyServerGroup.Delete().Exec(ctx); err != nil {
		r.logger.Errorf("Failed to delete all node groups: %v", err)
		return err
	}
	if err = tx.ProxySubscribe.Update().
		SetNodeGroupIds([]int64{}).
		SetNodeGroupID(0).
		Exec(ctx); err != nil {
		r.logger.Errorf("Failed to clear subscribe node groups: %v", err)
		return err
	}
	if err = tx.ProxyNode.Update().
		SetNodeGroupIds([]int64{}).
		Exec(ctx); err != nil {
		r.logger.Errorf("Failed to clear node node_group_ids: %v", err)
		return err
	}
	if err = tx.ProxyUserSubscribe.Update().
		SetNodeGroupID(0).
		Exec(ctx); err != nil {
		r.logger.Errorf("Failed to clear user subscribe node_group_id: %v", err)
		return err
	}
	if _, err = tx.ProxyGroupHistoryDetail.Delete().Exec(ctx); err != nil {
		r.logger.Errorf("Failed to delete group history details: %v", err)
		return err
	}
	if _, err = tx.ProxyGroupHistory.Delete().Exec(ctx); err != nil {
		r.logger.Errorf("Failed to delete group history: %v", err)
		return err
	}
	if _, err = tx.ProxySystem.Delete().
		Where(proxysystem.CategoryEQ("group")).
		Exec(ctx); err != nil {
		r.logger.Errorf("Failed to delete group config: %v", err)
		return err
	}

	if err = tx.Commit(); err != nil {
		r.logger.Errorf("Failed to commit reset groups transaction: %v", err)
		return err
	}

	r.logger.Info("All groups have been reset")
	return nil
}

// GetSubscribeGroupMapping gets subscribe group mapping
func (r *adminGroupRepo) GetSubscribeGroupMapping(ctx context.Context) ([]*v1.SubscribeGroupMappingItem, error) {
	// 1. 查询所有订阅套餐
	subscribes, err := r.data.db.ProxySubscribe.Query().All(ctx)
	if err != nil {
		r.logger.Errorf("Failed to query subscribes: %v", err)
		return nil, err
	}

	// 2. 查询所有节点组
	nodeGroups, err := r.data.db.ProxyServerGroup.Query().All(ctx)
	if err != nil {
		r.logger.Errorf("Failed to query node groups: %v", err)
		return nil, err
	}

	// 创建 node_group_id -> node_group_name 的映射
	nodeGroupMap := make(map[int64]string)
	for _, ng := range nodeGroups {
		nodeGroupMap[ng.ID] = ng.Name
	}

	// 3. 构建映射结果：套餐 -> 默认节点组（一对一）
	var mappingList []*v1.SubscribeGroupMappingItem

	for _, sub := range subscribes {
		// 获取套餐的默认节点组（node_group_ids 数组的第一个）
		nodeGroupName := ""
		if sub.NodeGroupIds != nil && len(sub.NodeGroupIds) > 0 {
			defaultNodeGroupId := sub.NodeGroupIds[0]
			nodeGroupName = nodeGroupMap[defaultNodeGroupId]
		}

		mappingList = append(mappingList, &v1.SubscribeGroupMappingItem{
			SubscribeName: sub.Name,
			NodeGroupName: nodeGroupName,
		})
	}

	return mappingList, nil
}

// ===== Helper Methods =====

// updateHistoryState updates history state
func (r *adminGroupRepo) updateHistoryState(ctx context.Context, historyID int64, state string) error {
	err := r.data.db.ProxyGroupHistory.Update().
		Where(proxygrouphistory.IDEQ(historyID)).
		SetState(state).
		Exec(ctx)

	return err
}

// updateHistoryStateWithError updates history state with error message
func (r *adminGroupRepo) updateHistoryStateWithError(ctx context.Context, historyID int64, state, errorMsg string, endTime *time.Time) error {
	update := r.data.db.ProxyGroupHistory.Update().
		Where(proxygrouphistory.IDEQ(historyID)).
		SetState(state).
		SetErrorMessage(errorMsg)

	if endTime != nil {
		update.SetEndTime(*endTime)
	}

	return update.Exec(ctx)
}

// updateHistoryStateWithStats updates history state with statistics
func (r *adminGroupRepo) updateHistoryStateWithStats(ctx context.Context, historyID int64, state string, totalUsers, successCount, failedCount int, endTime *time.Time) error {
	update := r.data.db.ProxyGroupHistory.Update().
		Where(proxygrouphistory.IDEQ(historyID)).
		SetState(state).
		SetTotalUsers(totalUsers).
		SetSuccessCount(successCount).
		SetFailedCount(failedCount).
		SetEndTime(*endTime)

	return update.Exec(ctx)
}

// getUserEmail queries user email by userID
func (r *adminGroupRepo) getUserEmail(ctx context.Context, userID int64) string {
	// 从proxy_user_auth_method表查询邮箱
	type EmailResult struct {
		AuthIdentifier string `json:"auth_identifier"`
	}

	var result EmailResult
	err := r.data.db.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.UserIDEQ(userID),
			proxyuserauthmethod.Or(
				proxyuserauthmethod.AuthTypeEQ("email"),
				proxyuserauthmethod.AuthTypeEQ("6"),
			),
		).
		Select(proxyuserauthmethod.FieldAuthIdentifier).
		Scan(ctx, &result)

	if err != nil {
		r.logger.Debugf("Failed to get email for user_id=%d: %v", userID, err)
		return ""
	}

	return result.AuthIdentifier
}

// ===== Transaction Helper Methods =====

// updateHistoryStateTx updates history state within a transaction
func (r *adminGroupRepo) updateHistoryStateTx(tx *ent.Tx, ctx context.Context, historyID int64, state string) error {
	err := tx.ProxyGroupHistory.Update().
		Where(proxygrouphistory.IDEQ(historyID)).
		SetState(state).
		Exec(ctx)

	return err
}

// updateHistoryStateWithErrorTx updates history state with error message within a transaction
func (r *adminGroupRepo) updateHistoryStateWithErrorTx(tx *ent.Tx, ctx context.Context, historyID int64, state, errorMsg string, endTime *time.Time) error {
	update := tx.ProxyGroupHistory.Update().
		Where(proxygrouphistory.IDEQ(historyID)).
		SetState(state).
		SetErrorMessage(errorMsg)

	if endTime != nil {
		update.SetEndTime(*endTime)
	}

	return update.Exec(ctx)
}

// updateHistoryStateWithStatsTx updates history state with statistics within a transaction
func (r *adminGroupRepo) updateHistoryStateWithStatsTx(tx *ent.Tx, ctx context.Context, historyID int64, state string, totalUsers, successCount, failedCount int, endTime *time.Time) error {
	update := tx.ProxyGroupHistory.Update().
		Where(proxygrouphistory.IDEQ(historyID)).
		SetState(state).
		SetTotalUsers(totalUsers).
		SetSuccessCount(successCount).
		SetFailedCount(failedCount).
		SetEndTime(*endTime)

	return update.Exec(ctx)
}

// getUserEmailTx queries user email by userID within a transaction
func (r *adminGroupRepo) getUserEmailTx(tx *ent.Tx, ctx context.Context, userID int64) string {
	// 从proxy_user_auth_method表查询邮箱
	type EmailResult struct {
		AuthIdentifier string `json:"auth_identifier"`
	}

	var result EmailResult
	err := tx.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.UserIDEQ(userID),
			proxyuserauthmethod.Or(
				proxyuserauthmethod.AuthTypeEQ("email"),
				proxyuserauthmethod.AuthTypeEQ("6"),
			),
		).
		Select(proxyuserauthmethod.FieldAuthIdentifier).
		Scan(ctx, &result)

	if err != nil {
		r.logger.Debugf("Failed to get email for user_id=%d: %v", userID, err)
		return ""
	}

	return result.AuthIdentifier
}

// ===== Transaction-based Grouping Algorithms =====
// These methods execute grouping algorithms within Ent transactions

// executeAverageGroupingTx implements average grouping algorithm within a transaction
// 完全按照原项目逻辑实现
func (r *adminGroupRepo) executeAverageGroupingTx(tx *ent.Tx, ctx context.Context, historyID int64) (int, error) {
	r.logger.Infof("Executing average grouping for history_id=%d", historyID)

	// 1. 查询所有有效且未锁定的用户订阅 (status IN (0, 1) AND group_locked = 0)
	var userSubscribes []*ent.ProxyUserSubscribe
	userSubscribes, err := tx.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.StatusIn(0, 1),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query user subscribes: %v", err)
		return 0, err
	}

	if len(userSubscribes) == 0 {
		r.logger.Infof("average grouping: no valid and unlocked user subscribes found")
		return 0, nil
	}

	r.logger.Infof("average grouping: found %d valid and unlocked user subscribes", len(userSubscribes))

	// 1.5 查询所有参与计算的节点组ID (for_calculation = true)
	var calculationNodeGroups []*ent.ProxyServerGroup
	calculationNodeGroups, err = tx.ProxyServerGroup.Query().
		Where(proxyservergroup.ForCalculation(true)).
		All(ctx)
	if err != nil {
		r.logger.Errorf("Failed to query calculation node groups: %v", err)
		return 0, err
	}

	// 创建参与计算的节点组ID集合（用于快速查找）
	calculationNodeGroupIds := make(map[int64]bool)
	for _, ng := range calculationNodeGroups {
		calculationNodeGroupIds[ng.ID] = true
	}

	r.logger.Infof("average grouping: found %d node groups with for_calculation=true", len(calculationNodeGroupIds))

	// 2. 批量查询订阅的节点组ID信息
	subscribeIds := make([]int64, len(userSubscribes))
	for i, us := range userSubscribes {
		subscribeIds[i] = us.SubscribeID
	}

	var subscribes []*ent.ProxySubscribe
	subscribes, err = tx.ProxySubscribe.Query().
		Where(proxysubscribe.IDIn(subscribeIds...)).
		All(ctx)
	if err != nil {
		r.logger.Errorf("Failed to query subscribe infos: %v", err)
		return 0, err
	}

	// 创建 subscribe_id -> SubscribeInfo 的映射
	subInfoMap := make(map[int64]*ent.ProxySubscribe)
	for _, si := range subscribes {
		subInfoMap[si.ID] = si
	}

	// 用于存储统计信息（按节点组ID统计用户数）
	groupUsersMap := make(map[int64][]struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	})

	// 3. 遍历所有用户订阅，按序轮询分配节点组（与原项目逻辑一致）
	affectedCount := 0
	failedCount := 0

	// 为每个订阅维护一个分配索引，用于按序循环分配（轮询）
	subscribeAllocationIndex := make(map[int64]int) // subscribe_id -> current_index

	for _, us := range userSubscribes {
		subInfo, ok := subInfoMap[us.SubscribeID]
		if !ok {
			r.logger.Infof("subscribe not found: user_subscribe_id=%d, subscribe_id=%d", us.ID, us.SubscribeID)
			failedCount++
			continue
		}

		// 解析订阅的节点组ID列表，并过滤出参与计算的节点组
		var nodeGroupIds []int64
		if subInfo.NodeGroupIds != nil && len(subInfo.NodeGroupIds) > 0 {
			allNodeGroupIds := subInfo.NodeGroupIds

			// 只保留参与计算的节点组
			for _, ngId := range allNodeGroupIds {
				if calculationNodeGroupIds[ngId] {
					nodeGroupIds = append(nodeGroupIds, ngId)
				}
			}

			if len(nodeGroupIds) == 0 && len(allNodeGroupIds) > 0 {
				r.logger.Debugf("all node_group_ids are not for calculation, setting to 0: subscribe_id=%d, total_node_groups=%d",
					subInfo.ID, len(allNodeGroupIds))
			}
		}

		// 按序选择节点组ID（循环轮询分配）
		selectedNodeGroupId := int64(0)
		if len(nodeGroupIds) > 0 {
			// 获取当前订阅的分配索引
			currentIndex := subscribeAllocationIndex[us.SubscribeID]
			// 选择当前索引对应的节点组
			selectedNodeGroupId = nodeGroupIds[currentIndex]
			// 更新索引，循环使用（轮询）
			subscribeAllocationIndex[us.SubscribeID] = (currentIndex + 1) % len(nodeGroupIds)

			r.logger.Debugf("assigning user_subscribe_id=%d (subscribe_id=%d) to node_group_id=%d (index=%d, total_options=%d, mode=sequential)",
				us.ID, us.SubscribeID, selectedNodeGroupId, currentIndex, len(nodeGroupIds))
		} else {
			r.logger.Debugf("no valid node_group_ids for subscribe_id=%d, setting to 0", subInfo.ID)
		}

		// 更新 user_subscribe 的 node_group_id 字段（单个ID）
		if err := tx.ProxyUserSubscribe.Update().
			Where(proxyusersubscribe.IDEQ(us.ID)).
			SetNodeGroupID(selectedNodeGroupId).
			Exec(ctx); err != nil {
			r.logger.Errorf("Failed to update user_subscribe node_group_id: user_subscribe_id=%d, error=%v", us.ID, err)
			failedCount++
			continue
		}

		// 只统计有节点组的用户
		if selectedNodeGroupId > 0 {
			// 查询用户邮箱，用于保存到历史记录
			email := r.getUserEmailTx(tx, ctx, us.UserID)
			groupUsersMap[selectedNodeGroupId] = append(groupUsersMap[selectedNodeGroupId], struct {
				ID    int64  `json:"id"`
				Email string `json:"email"`
			}{
				ID:    us.UserID,
				Email: email,
			})
		}

		affectedCount++
	}

	r.logger.Infof("average grouping completed: affected=%d, failed=%d", affectedCount, failedCount)

	// 4. 创建分组历史详情记录（按节点组ID统计）
	for nodeGroupId, users := range groupUsersMap {
		userCount := len(users)
		if userCount == 0 {
			continue
		}

		// 统计该节点组的节点数
		var nodeCount int = 0
		if nodeGroupId > 0 {
			if count, err := tx.ProxyNode.Query().
				Where(func(s *sql.Selector) {
					s.Where(jsonArrayContainsInt64(r.data.databaseDriver, proxynode.FieldNodeGroupIds, nodeGroupId))
				}).
				Count(ctx); err == nil {
				nodeCount = count
			} else {
				r.logger.Errorf("Failed to count nodes: node_group_id=%d, error=%v", nodeGroupId, err)
			}
		}

		// 序列化用户信息为 JSON
		userDataJSON := "[]"
		if jsonData, err := json.Marshal(users); err == nil {
			userDataJSON = string(jsonData)
		} else {
			r.logger.Errorf("Failed to marshal user data: node_group_id=%d, error=%v", nodeGroupId, err)
		}

		// 创建历史详情（使用 node_group_id 作为分组标识）
		_, err = tx.ProxyGroupHistoryDetail.Create().
			SetHistoryID(historyID).
			SetNodeGroupID(nodeGroupId).
			SetUserCount(userCount).
			SetNodeCount(int(nodeCount)).
			SetUserData(userDataJSON).
			SetCreatedAt(time.Now()).
			Save(ctx)

		if err != nil {
			r.logger.Errorf("Failed to create group history detail: node_group_id=%d, error=%v", nodeGroupId, err)
		}

		r.logger.Infof("Average Group (node_group_id=%d): users=%d, nodes=%d", nodeGroupId, userCount, nodeCount)
	}

	return affectedCount, nil
}

// executeSubscribeGroupingTx implements subscribe grouping algorithm within a transaction
// 完全按照原项目逻辑实现
func (r *adminGroupRepo) executeSubscribeGroupingTx(tx *ent.Tx, ctx context.Context, historyID int64) (int, error) {
	r.logger.Infof("Executing subscribe grouping for history_id=%d", historyID)

	// 1. 查询所有有效且未锁定的用户订阅 (status IN (0, 1) AND group_locked = 0)
	var userSubscribes []*ent.ProxyUserSubscribe
	userSubscribes, err := tx.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.StatusIn(0, 1),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query user subscribes: %v", err)
		return 0, err
	}

	if len(userSubscribes) == 0 {
		r.logger.Infof("subscribe grouping: no valid and unlocked user subscribes found")
		return 0, nil
	}

	r.logger.Infof("subscribe grouping: found %d valid and unlocked user subscribes", len(userSubscribes))

	// 1.5 查询所有参与计算的节点组ID (for_calculation = true)
	var calculationNodeGroups []*ent.ProxyServerGroup
	calculationNodeGroups, err = tx.ProxyServerGroup.Query().
		Where(proxyservergroup.ForCalculation(true)).
		All(ctx)
	if err != nil {
		r.logger.Errorf("Failed to query calculation node groups: %v", err)
		return 0, err
	}

	// 创建参与计算的节点组ID集合（用于快速查找）
	calculationNodeGroupIds := make(map[int64]bool)
	for _, ng := range calculationNodeGroups {
		calculationNodeGroupIds[ng.ID] = true
	}

	r.logger.Infof("subscribe grouping: found %d node groups with for_calculation=true", len(calculationNodeGroupIds))

	// 2. 批量查询订阅的节点组ID信息
	subscribeIds := make([]int64, len(userSubscribes))
	for i, us := range userSubscribes {
		subscribeIds[i] = us.SubscribeID
	}

	var subscribes []*ent.ProxySubscribe
	subscribes, err = tx.ProxySubscribe.Query().
		Where(proxysubscribe.IDIn(subscribeIds...)).
		All(ctx)
	if err != nil {
		r.logger.Errorf("Failed to query subscribe infos: %v", err)
		return 0, err
	}

	// 创建 subscribe_id -> SubscribeInfo 的映射
	subInfoMap := make(map[int64]*ent.ProxySubscribe)
	for _, si := range subscribes {
		subInfoMap[si.ID] = si
	}

	// 用于存储统计信息（按节点组ID统计用户数）
	type UserInfo struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	groupUsersMap := make(map[int64][]UserInfo)

	// 3. 遍历所有用户订阅，取第一个节点组ID
	affectedCount := 0
	failedCount := 0

	for _, us := range userSubscribes {
		subInfo, ok := subInfoMap[us.SubscribeID]
		if !ok {
			r.logger.Infof("subscribe not found: user_subscribe_id=%d, subscribe_id=%d", us.ID, us.SubscribeID)
			failedCount++
			continue
		}

		// 解析订阅的节点组ID列表，并过滤出参与计算的节点组
		var nodeGroupIds []int64
		if subInfo.NodeGroupIds != nil && len(subInfo.NodeGroupIds) > 0 {
			allNodeGroupIds := subInfo.NodeGroupIds

			// 只保留参与计算的节点组
			for _, ngId := range allNodeGroupIds {
				if calculationNodeGroupIds[ngId] {
					nodeGroupIds = append(nodeGroupIds, ngId)
				}
			}

			if len(nodeGroupIds) == 0 && len(allNodeGroupIds) > 0 {
				r.logger.Debugf("all node_group_ids are not for calculation, setting to 0: subscribe_id=%d, total_node_groups=%d",
					subInfo.ID, len(allNodeGroupIds))
			}
		}

		// 取第一个参与计算的节点组ID（如果有），否则设置为 0
		selectedNodeGroupId := int64(0)
		if len(nodeGroupIds) > 0 {
			selectedNodeGroupId = nodeGroupIds[0]
		}

		r.logger.Debugf("assigning user_subscribe_id=%d (subscribe_id=%d) to node_group_id=%d (total_options=%d, selected_first)",
			us.ID, us.SubscribeID, selectedNodeGroupId, len(nodeGroupIds))

		// 更新 user_subscribe 的 node_group_id 字段
		if err := tx.ProxyUserSubscribe.Update().
			Where(proxyusersubscribe.IDEQ(us.ID)).
			SetNodeGroupID(selectedNodeGroupId).
			Exec(ctx); err != nil {
			r.logger.Errorf("Failed to update user_subscribe node_group_id: user_subscribe_id=%d, error=%v", us.ID, err)
			failedCount++
			continue
		}

		// 只统计有节点组的用户
		if selectedNodeGroupId > 0 {
			// 查询用户邮箱，用于保存到历史记录
			email := r.getUserEmailTx(tx, ctx, us.UserID)
			groupUsersMap[selectedNodeGroupId] = append(groupUsersMap[selectedNodeGroupId], UserInfo{
				ID:    us.UserID,
				Email: email,
			})
		}

		affectedCount++
	}

	r.logger.Infof("subscribe grouping completed: affected=%d, failed=%d", affectedCount, failedCount)

	// 4. 处理订阅过期/失效的用户，设置 node_group_id 为 0
	// 查询所有没有有效订阅且未锁定的用户订阅记录
	var expiredUserSubscribes []*ent.ProxyUserSubscribe
	expiredUserSubscribes, err = tx.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.GroupLockedEQ(false),
			proxyusersubscribe.StatusNotIn(0, 1),
		).
		All(ctx)
	if err != nil {
		r.logger.Errorf("Failed to query expired user subscribes: %v", err)
		// 继续处理，不因为过期用户查询失败而影响
	} else {
		r.logger.Infof("found %d expired user subscribes for subscribe-based grouping, will set node_group_id to 0", len(expiredUserSubscribes))

		expiredAffectedCount := 0
		for _, eu := range expiredUserSubscribes {
			// 更新 user_subscribe 表的 node_group_id 字段到 0
			if err := tx.ProxyUserSubscribe.Update().
				Where(proxyusersubscribe.IDEQ(eu.ID)).
				SetNodeGroupID(0).
				Exec(ctx); err != nil {
				r.logger.Errorf("Failed to update expired user subscribe node_group_id: user_subscribe_id=%d, error=%v", eu.ID, err)
				continue
			}

			expiredAffectedCount++
		}

		r.logger.Infof("expired user subscribes grouping completed: affected=%d", expiredAffectedCount)
	}

	// 5. 创建分组历史详情记录（按节点组ID统计）
	for nodeGroupId, users := range groupUsersMap {
		userCount := len(users)
		if userCount == 0 {
			continue
		}

		// 统计该节点组的节点数
		var nodeCount int = 0
		if nodeGroupId > 0 {
			if count, err := tx.ProxyNode.Query().
				Where(func(s *sql.Selector) {
					s.Where(jsonArrayContainsInt64(r.data.databaseDriver, proxynode.FieldNodeGroupIds, nodeGroupId))
				}).
				Count(ctx); err == nil {
				nodeCount = count
			} else {
				r.logger.Errorf("Failed to count nodes: node_group_id=%d, error=%v", nodeGroupId, err)
			}
		}

		// 序列化用户信息为 JSON
		userDataJSON := "[]"
		if jsonData, err := json.Marshal(users); err == nil {
			userDataJSON = string(jsonData)
		} else {
			r.logger.Errorf("Failed to marshal user data: node_group_id=%d, error=%v", nodeGroupId, err)
		}

		// 创建历史详情
		_, err = tx.ProxyGroupHistoryDetail.Create().
			SetHistoryID(historyID).
			SetNodeGroupID(nodeGroupId).
			SetUserCount(userCount).
			SetNodeCount(int(nodeCount)).
			SetUserData(userDataJSON).
			SetCreatedAt(time.Now()).
			Save(ctx)

		if err != nil {
			r.logger.Errorf("Failed to create group history detail: node_group_id=%d, error=%v", nodeGroupId, err)
		}

		r.logger.Infof("Subscribe Group (node_group_id=%d): users=%d, nodes=%d", nodeGroupId, userCount, nodeCount)
	}

	return affectedCount, nil
}

// executeTrafficGroupingTx implements traffic grouping algorithm within a transaction
// 完全按照原项目逻辑实现
func (r *adminGroupRepo) executeTrafficGroupingTx(tx *ent.Tx, ctx context.Context, historyID int64) (int, error) {
	r.logger.Infof("Executing traffic grouping for history_id=%d", historyID)

	// 用于存储每个节点组的用户信息（id 和 email）
	type UserInfo struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	groupUsersMap := make(map[int64][]UserInfo) // node_group_id -> []UserInfo

	// 1. 获取所有设置了流量区间的节点组 (for_calculation = true AND (min_traffic_gb > 0 OR max_traffic_gb > 0))
	nodeGroups, err := tx.ProxyServerGroup.Query().
		Where(proxyservergroup.ForCalculation(true)).
		Where(
			proxyservergroup.Or(
				proxyservergroup.MinTrafficGBGT(0),
				proxyservergroup.MaxTrafficGBGT(0),
			),
		).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query node groups: %v", err)
		return 0, err
	}

	if len(nodeGroups) == 0 {
		r.logger.Infof("no node groups with traffic ranges configured")
		return 0, nil
	}

	r.logger.Infof("executeTrafficGrouping loaded node groups: node_groups_count=%d", len(nodeGroups))

	// 2. 查询所有有效且未锁定的用户订阅及其已用流量 (status IN (0, 1) AND group_locked = 0)
	var userSubscribes []*ent.ProxyUserSubscribe
	userSubscribes, err = tx.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.StatusIn(0, 1),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		All(ctx)
	userSubscribes, err = tx.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.StatusIn(0, 1),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		All(ctx)
	userSubscribes, err = tx.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.StatusIn(0, 1),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		All(ctx)
	userSubscribes, err = tx.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.StatusIn(0, 1),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		All(ctx)
	userSubscribes, err = tx.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.StatusIn(0, 1),
			proxyusersubscribe.GroupLockedEQ(false),
		).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query user subscribes: %v", err)
		return 0, err
	}

	if len(userSubscribes) == 0 {
		r.logger.Infof("no valid and unlocked user subscribes found")
		return 0, nil
	}

	r.logger.Infof("found user subscribes for traffic-based grouping: count=%d", len(userSubscribes))

	// 3. 根据流量范围分配节点组ID到用户订阅
	affectedCount := 0
	groupUserCount := make(map[int64]int) // node_group_id -> user_count

	for _, us := range userSubscribes {
		// 计算已用流量（字节转GB）
		var upload, download int64
		if us.Upload != nil {
			upload = *us.Upload
		}
		if us.Download != nil {
			download = *us.Download
		}
		usedTrafficGB := float64(upload+download) / (1024 * 1024 * 1024)

		// 查找匹配的流量范围（使用左闭右开区间 [Min, Max)）
		var targetNodeGroupId int64 = 0
		for _, ng := range nodeGroups {
			minTraffic := float64(int64Value(ng.MinTrafficGB))
			maxTraffic := float64(int64Value(ng.MaxTrafficGB))

			// 检查是否在区间内 [min, max)
			if usedTrafficGB >= minTraffic && usedTrafficGB < maxTraffic {
				targetNodeGroupId = ng.ID
				break
			}
		}

		// 如果没有匹配到任何范围，targetNodeGroupId 保持为 0（不分配节点组）

		// 更新 user_subscribe 的 node_group_id 字段
		if err := tx.ProxyUserSubscribe.Update().
			Where(proxyusersubscribe.IDEQ(us.ID)).
			SetNodeGroupID(targetNodeGroupId).
			Exec(ctx); err != nil {
			r.logger.Errorf("Failed to update user subscribe node_group_id: user_subscribe_id=%d, target_node_group_id=%d, error=%v",
				us.ID, targetNodeGroupId, err)
			continue
		}

		// 只有分配了节点组的用户才记录到历史
		if targetNodeGroupId > 0 {
			// 查询用户邮箱，用于保存到历史记录
			email := r.getUserEmailTx(tx, ctx, us.UserID)
			userInfo := UserInfo{
				ID:    us.UserID,
				Email: email,
			}
			groupUsersMap[targetNodeGroupId] = append(groupUsersMap[targetNodeGroupId], userInfo)
			groupUserCount[targetNodeGroupId]++

			r.logger.Debugf("assigned user subscribe %d (traffic: %.2fGB) to node group %d",
				us.ID, usedTrafficGB, targetNodeGroupId)
		} else {
			r.logger.Debugf("user subscribe %d (traffic: %.2fGB) not assigned to any node group",
				us.ID, usedTrafficGB)
		}

		affectedCount++
	}

	r.logger.Infof("traffic-based grouping completed: affected_subscribes=%d", affectedCount)

	// 4. 创建分组历史详情记录（只统计有用户的节点组）
	nodeGroupCount := make(map[int64]int) // node_group_id -> node_count
	for _, ng := range nodeGroups {
		nodeGroupCount[ng.ID] = 1 // 每个节点组计为1
	}

	for nodeGroupId, userCount := range groupUserCount {
		userDataJSON, err := json.Marshal(groupUsersMap[nodeGroupId])
		if err != nil {
			r.logger.Errorf("Failed to marshal user data: node_group_id=%d, error=%v", nodeGroupId, err)
			continue
		}

		_, err = tx.ProxyGroupHistoryDetail.Create().
			SetHistoryID(historyID).
			SetNodeGroupID(nodeGroupId).
			SetUserCount(userCount).
			SetNodeCount(nodeGroupCount[nodeGroupId]).
			SetUserData(string(userDataJSON)).
			SetCreatedAt(time.Now()).
			Save(ctx)

		if err != nil {
			r.logger.Errorf("Failed to create group history detail: history_id=%d, node_group_id=%d, error=%v",
				historyID, nodeGroupId, err)
		}
	}

	return affectedCount, nil
}

// ExportGroupResult exports group result as CSV
func (r *adminGroupRepo) ExportGroupResult(ctx context.Context, historyID *int64) ([]byte, string, error) {
	var records [][]string

	// CSV 表头
	records = append(records, []string{"用户ID", "节点组ID", "节点组名称"})

	if historyID != nil {
		// 导出指定历史的详细结果
		r.logger.Infof("Exporting group result for history_id=%d", *historyID)

		// 1. 查询分组历史详情
		details, err := r.data.db.ProxyGroupHistoryDetail.Query().
			Where(proxygrouphistorydetail.HistoryIDEQ(*historyID)).
			All(ctx)

		if err != nil {
			r.logger.Errorf("Failed to get group history details: %v", err)
			return nil, "", err
		}

		// 2. 为每个组生成记录
		for _, detail := range details {
			// 从 UserData JSON 解析用户信息
			type UserInfo struct {
				ID    int64  `json:"id"`
				Email string `json:"email"`
			}
			var users []UserInfo
			if detail.UserData != "" && detail.UserData != "[]" {
				if err := json.Unmarshal([]byte(detail.UserData), &users); err != nil {
					r.logger.Errorf("Failed to parse user data: %v", err)
					continue
				}
			}

			// 查询节点组名称
			nodeGroup, err := r.data.db.ProxyServerGroup.Query().
				Where(proxyservergroup.IDEQ(detail.NodeGroupID)).
				First(ctx)

			var nodeGroupName string
			if err == nil && nodeGroup != nil {
				nodeGroupName = nodeGroup.Name
			} else {
				nodeGroupName = fmt.Sprintf("Group-%d", detail.NodeGroupID)
			}

			// 为每个用户生成记录
			for _, user := range users {
				records = append(records, []string{
					fmt.Sprintf("%d", user.ID),
					fmt.Sprintf("%d", detail.NodeGroupID),
					nodeGroupName,
				})
			}
		}
	} else {
		// 导出当前所有用户的分组情况
		r.logger.Infof("Exporting current group result for all users")

		// 查询所有有节点组的用户订阅（去重）
		type UserNodeGroupInfo struct {
			UserID      int64 `json:"user_id"`
			NodeGroupID int64 `json:"node_group_id"`
		}

		var userGroups []UserNodeGroupInfo
		err := r.data.db.ProxyUserSubscribe.Query().
			Where(
				proxyusersubscribe.NodeGroupIDNEQ(0),
			).
			Select(proxyusersubscribe.FieldUserID, proxyusersubscribe.FieldNodeGroupID).
			Scan(ctx, &userGroups)

		if err != nil {
			r.logger.Errorf("Failed to get users: %v", err)
			return nil, "", err
		}

		// 为每个用户生成记录
		for _, ug := range userGroups {
			// 查询节点组信息
			nodeGroup, err := r.data.db.ProxyServerGroup.Query().
				Where(proxyservergroup.IDEQ(ug.NodeGroupID)).
				First(ctx)

			var nodeGroupName string
			if err == nil && nodeGroup != nil {
				nodeGroupName = nodeGroup.Name
			} else {
				nodeGroupName = fmt.Sprintf("Group-%d", ug.NodeGroupID)
			}

			records = append(records, []string{
				fmt.Sprintf("%d", ug.UserID),
				fmt.Sprintf("%d", ug.NodeGroupID),
				nodeGroupName,
			})
		}
	}

	// 生成 CSV 数据
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.WriteAll(records); err != nil {
		r.logger.Errorf("Failed to write csv: %v", err)
		return nil, "", err
	}
	writer.Flush()

	if err := writer.Error(); err != nil {
		r.logger.Errorf("Failed to flush csv writer: %v", err)
		return nil, "", err
	}

	// 添加 UTF-8 BOM
	bom := []byte{0xEF, 0xBB, 0xBF}
	csvData := buf.Bytes()
	result := make([]byte, 0, len(bom)+len(csvData))
	result = append(result, bom...)
	result = append(result, csvData...)

	// 生成文件名
	filename := fmt.Sprintf("group_result_%d.csv", time.Now().Unix())

	return result, filename, nil
}
