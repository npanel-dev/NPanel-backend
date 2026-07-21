package data

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxytask"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	marketingbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/marketing"
	"github.com/npanel-dev/NPanel-backend/internal/model"
	taskmodel "github.com/npanel-dev/NPanel-backend/internal/model/task"
	queuetypes "github.com/npanel-dev/NPanel-backend/internal/queue/types"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

type adminMarketingRepo struct {
	data *Data
	log  *log.Helper
}

// NewAdminMarketingRepo 创建营销仓库
func NewAdminMarketingRepo(data *Data, logger log.Logger) marketingbiz.MarketingRepo {
	return &adminMarketingRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// ========== Email Task Methods ==========

// CreateBatchSendEmailTask 创建批量发送邮件任务
func (r *adminMarketingRepo) CreateBatchSendEmailTask(ctx context.Context, subject, content string, scope uint32,
	registerStartTime, registerEndTime int64, additional string, scheduled int64, interval uint32, limit uint64) error {

	scopeType := taskmodel.ScopeType(scope)

	var emails []string

	// 基础查询：获取 auth_type = 'email' 的用户邮箱地址
	baseQuery := func() *ent.ProxyUserAuthMethodQuery {
		query := r.data.db.ProxyUserAuthMethod.Query().
			Where(func(s *sql.Selector) {
				// JOIN user ON user.id = user_auth_methods.user_id
				t := sql.Table(proxyuser.Table)
				s.Join(t).On(s.C(proxyuserauthmethod.FieldUserID), t.C(proxyuser.FieldID))

				// WHERE auth_type = 'email'
				s.Where(sql.EQ(s.C(proxyuserauthmethod.FieldAuthType), model.AuthTypeEmail))

				// 注册时间范围过滤
				if registerStartTime != 0 {
					s.Where(sql.GTE(t.C(proxyuser.FieldCreatedAt), time.UnixMilli(registerStartTime)))
				}
				if registerEndTime != 0 {
					s.Where(sql.LTE(t.C(proxyuser.FieldCreatedAt), time.UnixMilli(registerEndTime)))
				}
			})
		return query
	}

	var query *ent.ProxyUserAuthMethodQuery
	var err error

	switch scopeType {
	case taskmodel.ScopeAll:
		// 所有有email认证的用户
		query = baseQuery()

	case taskmodel.ScopeActive:
		// 有激活订阅的用户（status IN (1,2)）
		query = baseQuery().
			Where(func(s *sql.Selector) {
				t := sql.Table(proxyusersubscribe.Table)
				// Ent 给第一个 JOIN 自动设置了别名 t1，所以这里要引用 t1.id
				s.Join(t).OnP(sql.P(func(b *sql.Builder) {
					b.WriteString("t1").WriteByte('.').WriteString(proxyuser.FieldID)
					b.WriteString(" = t2.")
					b.WriteString(proxyusersubscribe.FieldUserID)
				}))
				s.Where(sql.In(t.C(proxyusersubscribe.FieldStatus), uint8(model.UserSubscribeStatusActive), uint8(model.UserSubscribeStatusFinish)))
			})

	case taskmodel.ScopeExpired:
		// 订阅过期的用户（status = 3）
		query = baseQuery().
			Where(func(s *sql.Selector) {
				t := sql.Table(proxyusersubscribe.Table)
				s.Join(t).OnP(sql.P(func(b *sql.Builder) {
					b.WriteString("t1").WriteByte('.').WriteString(proxyuser.FieldID)
					b.WriteString(" = t2.")
					b.WriteString(proxyusersubscribe.FieldUserID)
				}))
				s.Where(sql.EQ(t.C(proxyusersubscribe.FieldStatus), uint8(model.UserSubscribeStatusExpired)))
			})

	case taskmodel.ScopeNone:
		// 没有订阅的用户
		query = baseQuery().
			Where(func(s *sql.Selector) {
				t := sql.Table(proxyusersubscribe.Table)
				s.LeftJoin(t).OnP(sql.P(func(b *sql.Builder) {
					b.WriteString("t1").WriteByte('.').WriteString(proxyuser.FieldID)
					b.WriteString(" = t2.")
					b.WriteString(proxyusersubscribe.FieldUserID)
				}))
				s.Where(sql.IsNull(t.C(proxyusersubscribe.FieldUserID)))
			})

	case taskmodel.ScopeSkip:
		// 跳过scope，不查询用户
		query = nil

	default:
		return fmt.Errorf("invalid email scope: %d", scope)
	}

	// 执行查询获取邮箱列表
	if query != nil {
		// 使用原生SQL查询auth_identifier字段
		emailList, err := query.Select(proxyuserauthmethod.FieldAuthIdentifier).Strings(ctx)
		if err != nil {
			r.log.Errorf("CreateBatchSendEmailTask: failed to fetch emails: %v", err)
			return fmt.Errorf("failed to fetch emails: %w", err)
		}
		emails = emailList
	}

	// 邮箱列表为空且不是Skip模式，返回错误
	if len(emails) == 0 && scopeType != taskmodel.ScopeSkip {
		r.log.Errorf("CreateBatchSendEmailTask: no email addresses found for scope %d", scope)
		return fmt.Errorf("no email addresses found for the specified scope")
	}

	// 邮箱去重
	emails = tool.RemoveDuplicateElements(emails...)

	// 处理额外邮箱地址（不覆盖）
	var additionalEmails []string
	if additional != "" {
		additionalEmails = tool.RemoveDuplicateElements(strings.Split(additional, "\n")...)
	}
	if len(additionalEmails) == 0 && scopeType == taskmodel.ScopeSkip {
		r.log.Errorf("CreateBatchSendEmailTask: no additional emails for skip scope")
		return fmt.Errorf("no additional email addresses provided for skip scope")
	}

	// 设置定时执行时间（默认延迟10秒执行，防止任务创建和执行时间过于接近）
	scheduledAt := time.Now().Add(10 * time.Second)
	if scheduled != 0 {
		scheduledAt = time.Unix(int64(scheduled), 0)
		if scheduledAt.Before(time.Now()) {
			scheduledAt = time.Now()
		}
	}

	// 构建EmailScope
	scopeInfo := &taskmodel.EmailScope{
		Type:              int8(scopeType),
		RegisterStartTime: registerStartTime,
		RegisterEndTime:   registerEndTime,
		Recipients:        emails,
		Additional:        additionalEmails,
		Scheduled:         int64(scheduled),
		Interval:          uint8(interval),
		Limit:             uint64(limit),
	}
	scopeJSON, err := taskmodel.MarshalEmailScope(scopeInfo)
	if err != nil {
		r.log.Errorf("CreateBatchSendEmailTask: failed to marshal scope: %v", err)
		return fmt.Errorf("failed to marshal scope: %w", err)
	}

	// 构建EmailContent
	taskContent := &taskmodel.EmailContent{
		Subject: subject,
		Content: content,
	}
	contentJSON, err := taskmodel.MarshalEmailContent(taskContent)
	if err != nil {
		r.log.Errorf("CreateBatchSendEmailTask: failed to marshal content: %v", err)
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	// 计算总数
	var total uint64
	if len(additionalEmails) > 0 {
		allEmails := append(emails, additionalEmails...)
		total = uint64(len(tool.RemoveDuplicateElements(allEmails...)))
	} else {
		total = uint64(len(emails))
	}

	// 创建任务记录
	taskRecord, err := r.data.db.ProxyTask.Create().
		SetType(int8(taskmodel.TypeEmail)).
		SetScope(scopeJSON).
		SetContent(contentJSON).
		SetStatus(int8(taskmodel.StatusPending)).
		SetErrors("").
		SetTotal(uint32(total)).
		SetCurrent(0).
		Save(ctx)

	if err != nil {
		r.log.Errorf("CreateBatchSendEmailTask: failed to create task: %v", err)
		return fmt.Errorf("failed to create task: %w", err)
	}

	r.log.Infof("CreateBatchSendEmailTask: successfully created task with ID: %d", taskRecord.ID)

	// 创建Asynq任务并加入队列
	asynqTask := asynq.NewTask(queuetypes.ScheduledBatchSendEmail, []byte(strconv.FormatInt(int64(taskRecord.ID), 10)))
	info, err := r.data.queue.EnqueueContext(ctx, asynqTask, asynq.ProcessAt(scheduledAt))
	if err != nil {
		r.log.Errorf("CreateBatchSendEmailTask: failed to enqueue task: %v", err)
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	r.log.Infof("CreateBatchSendEmailTask: successfully enqueued task ID: %s, scheduled at: %s", info.ID, scheduledAt.Format(time.DateTime))

	return nil
}

// GetBatchSendEmailTaskList 获取批量发送邮件任务列表
func (r *adminMarketingRepo) GetBatchSendEmailTaskList(ctx context.Context, page, size int32, scope, status *uint32) ([]*ent.ProxyTask, int32, error) {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 10
	}

	query := r.data.db.ProxyTask.Query().
		Where(func(s *sql.Selector) {
			s.Where(sql.And(
				sql.EQ(s.C(proxytask.FieldType), int8(taskmodel.TypeEmail)),
			))
			if status != nil {
				s.Where(sql.EQ(s.C(proxytask.FieldStatus), int8(*status)))
			}
			if scope != nil {
				s.Where(sql.Contains(s.C(proxytask.FieldScope), fmt.Sprintf(`"type":%d`, *scope)))
			}
		})

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	list, err := query.
		Order(ent.Desc(proxytask.FieldCreatedAt)).
		Limit(int(size)).
		Offset(int((page - 1) * size)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return list, int32(total), nil
}

// StopBatchSendEmailTask 停止批量发送邮件任务
func (r *adminMarketingRepo) StopBatchSendEmailTask(ctx context.Context, id int) error {
	// 注意：原始项目使用 email.Manager.RemoveWorker(id) 停止工作线程
	// 当前实现使用 Asynq 队列系统，无法直接停止正在执行的任务
	// 仅更新数据库状态为 Completed (status=2)，与原始逻辑保持一致
	affected, err := r.data.db.ProxyTask.Update().
		Where(func(s *sql.Selector) {
			s.Where(sql.And(
				sql.EQ(s.C(proxytask.FieldID), id),
				sql.EQ(s.C(proxytask.FieldType), int8(taskmodel.TypeEmail)),
			))
		}).
		SetStatus(int8(taskmodel.StatusCompleted)).
		Save(ctx)

	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("task not found or access denied")
	}

	return nil
}

// GetPreSendEmailCount 获取预发送邮件数量
func (r *adminMarketingRepo) GetPreSendEmailCount(ctx context.Context, scope uint32, registerStartTime, registerEndTime int64) (int64, error) {
	scopeType := taskmodel.ScopeType(scope)

	// 基础查询：auth_type = 'email' 的用户认证方法
	baseQuery := func() *ent.ProxyUserAuthMethodQuery {
		query := r.data.db.ProxyUserAuthMethod.Query().
			Where(func(s *sql.Selector) {
				// JOIN user ON user.id = user_auth_methods.user_id
				t := sql.Table(proxyuser.Table)
				s.Join(t).On(s.C(proxyuserauthmethod.FieldUserID), t.C(proxyuser.FieldID))

				// WHERE auth_type = 'email'
				s.Where(sql.EQ(s.C(proxyuserauthmethod.FieldAuthType), model.AuthTypeEmail))

				// 注册时间范围过滤
				if registerStartTime != 0 {
					s.Where(sql.GTE(t.C(proxyuser.FieldCreatedAt), time.UnixMilli(registerStartTime)))
				}
				if registerEndTime != 0 {
					s.Where(sql.LTE(t.C(proxyuser.FieldCreatedAt), time.UnixMilli(registerEndTime)))
				}
			})
		return query
	}

	var count int
	var err error

	switch scopeType {
	case taskmodel.ScopeAll:
		// 所有有email认证的用户
		count, err = baseQuery().Count(ctx)

	case taskmodel.ScopeActive:
		// 有激活订阅的用户（status IN (1,2)）
		count, err = baseQuery().
			Where(func(s *sql.Selector) {
				// JOIN user_subscribe ON user.id = user_subscribe.user_id
				t := sql.Table(proxyusersubscribe.Table)
				s.Join(t).OnP(sql.P(func(b *sql.Builder) {
					b.WriteString("t1").WriteByte('.').WriteString(proxyuser.FieldID)
					b.WriteString(" = t2.")
					b.WriteString(proxyusersubscribe.FieldUserID)
				}))
				s.Where(sql.In(t.C(proxyusersubscribe.FieldStatus), uint8(model.UserSubscribeStatusActive), uint8(model.UserSubscribeStatusFinish)))
			}).
			Count(ctx)

	case taskmodel.ScopeExpired:
		// 订阅过期的用户（status = 3）
		count, err = baseQuery().
			Where(func(s *sql.Selector) {
				t := sql.Table(proxyusersubscribe.Table)
				s.Join(t).OnP(sql.P(func(b *sql.Builder) {
					b.WriteString("t1").WriteByte('.').WriteString(proxyuser.FieldID)
					b.WriteString(" = t2.")
					b.WriteString(proxyusersubscribe.FieldUserID)
				}))
				s.Where(sql.EQ(t.C(proxyusersubscribe.FieldStatus), uint8(model.UserSubscribeStatusExpired)))
			}).
			Count(ctx)

	case taskmodel.ScopeNone:
		// 没有订阅的用户
		count, err = baseQuery().
			Where(func(s *sql.Selector) {
				// LEFT JOIN user_subscribe ON user.id = user_subscribe.user_id
				// WHERE user_subscribe.user_id IS NULL
				t := sql.Table(proxyusersubscribe.Table)
				s.LeftJoin(t).OnP(sql.P(func(b *sql.Builder) {
					b.WriteString("t1").WriteByte('.').WriteString(proxyuser.FieldID)
					b.WriteString(" = t2.")
					b.WriteString(proxyusersubscribe.FieldUserID)
				}))
				s.Where(sql.IsNull(t.C(proxyusersubscribe.FieldUserID)))
			}).
			Count(ctx)

	case taskmodel.ScopeSkip:
		// 跳过scope，不需要统计
		return 0, nil

	default:
		return 0, fmt.Errorf("invalid email scope: %d", scope)
	}

	if err != nil {
		r.log.Errorf("GetPreSendEmailCount failed: %v", err)
		return 0, err
	}

	return int64(count), nil
}

// GetBatchSendEmailTaskStatus 获取批量发送邮件任务状态
func (r *adminMarketingRepo) GetBatchSendEmailTaskStatus(ctx context.Context, id int) (*ent.ProxyTask, error) {
	task, err := r.data.db.ProxyTask.Query().
		Where(func(s *sql.Selector) {
			s.Where(sql.And(
				sql.EQ(s.C(proxytask.FieldID), id),
				sql.EQ(s.C(proxytask.FieldType), int8(taskmodel.TypeEmail)),
			))
		}).
		Only(ctx)

	if err != nil {
		return nil, err
	}

	return task, nil
}

// ========== Quota Task Methods ==========

// CreateQuotaTask 创建配额任务
func (r *adminMarketingRepo) CreateQuotaTask(ctx context.Context, subscribers []int, isActive *bool,
	startTime, endTime int64, resetTraffic bool, days uint64, giftType uint32, giftValue uint64) error {

	// 查询符合条件的用户订阅记录
	// 将[]int转换为[]int64
	subscribersInt64 := make([]int64, len(subscribers))
	for i, id := range subscribers {
		subscribersInt64[i] = int64(id)
	}

	query := r.data.db.ProxyUserSubscribe.Query().
		Where(func(s *sql.Selector) {
			// 订阅ID列表过滤
			if len(subscribers) > 0 {
				s.Where(sql.In(s.C(proxyusersubscribe.FieldSubscribeID), toInterfaceSlice(subscribersInt64)...))
			}

			// 是否仅活跃订阅（status IN (0,1,2)）
			if isActive != nil && *isActive {
				s.Where(sql.In(s.C(proxyusersubscribe.FieldStatus),
					uint8(model.UserSubscribeStatusPending),
					uint8(model.UserSubscribeStatusActive),
					uint8(model.UserSubscribeStatusFinish),
				))
			}

			// 开始时间过滤：start_time <= ?
			if startTime != 0 {
				s.Where(sql.LTE(s.C(proxyusersubscribe.FieldStartTime), time.UnixMilli(startTime)))
			}

			// 结束时间过滤：expire_time >= ?
			if endTime != 0 {
				s.Where(sql.GTE(s.C(proxyusersubscribe.FieldExpireTime), time.UnixMilli(endTime)))
			}
		})

	subs, err := query.All(ctx)
	if err != nil {
		r.log.Errorf("CreateQuotaTask: failed to query subscribers: %v", err)
		return fmt.Errorf("failed to query subscribers: %w", err)
	}

	if len(subs) == 0 {
		r.log.Errorf("CreateQuotaTask: no subscribers found")
		return fmt.Errorf("no subscribers found")
	}

	// 提取订阅ID列表
	var subIds []int64
	for _, sub := range subs {
		subIds = append(subIds, int64(sub.ID))
	}

	// 构建QuotaScope
	scopeInfo := &taskmodel.QuotaScope{
		Subscribers: subscribersInt64,
		IsActive:    isActive,
		StartTime:   startTime,
		EndTime:     endTime,
		Objects:     subIds,
	}
	scopeJSON, err := taskmodel.MarshalQuotaScope(scopeInfo)
	if err != nil {
		r.log.Errorf("CreateQuotaTask: failed to marshal scope: %v", err)
		return fmt.Errorf("failed to marshal scope: %w", err)
	}

	// 构建QuotaContent
	contentInfo := &taskmodel.QuotaContent{
		ResetTraffic: resetTraffic,
		Days:         days,
		GiftType:     uint8(giftType),
		GiftValue:    giftValue,
	}
	contentJSON, err := taskmodel.MarshalQuotaContent(contentInfo)
	if err != nil {
		r.log.Errorf("CreateQuotaTask: failed to marshal content: %v", err)
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	// 创建任务记录
	taskRecord, err := r.data.db.ProxyTask.Create().
		SetType(int8(taskmodel.TypeQuota)).
		SetScope(scopeJSON).
		SetContent(contentJSON).
		SetStatus(int8(taskmodel.StatusPending)).
		SetErrors("").
		SetTotal(uint32(len(subIds))).
		SetCurrent(0).
		Save(ctx)

	if err != nil {
		r.log.Errorf("CreateQuotaTask: failed to create task: %v", err)
		return fmt.Errorf("failed to create task: %w", err)
	}

	r.log.Infof("CreateQuotaTask: successfully created task with ID: %d", taskRecord.ID)

	// 创建Asynq任务并立即加入队列
	asynqTask := asynq.NewTask(queuetypes.ForthwithQuotaTask, []byte(strconv.FormatInt(int64(taskRecord.ID), 10)))
	info, err := r.data.queue.EnqueueContext(ctx, asynqTask)
	if err != nil {
		r.log.Errorf("CreateQuotaTask: failed to enqueue task: %v", err)
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	r.log.Infof("CreateQuotaTask: successfully enqueued task with ID: %s", info.ID)

	return nil
}

// QueryQuotaTaskPreCount 查询配额任务预计数量
func (r *adminMarketingRepo) QueryQuotaTaskPreCount(ctx context.Context, subscribers []int, isActive *bool, startTime, endTime int64) (int64, error) {
	// 将[]int转换为[]int64
	subscribersInt64 := make([]int64, len(subscribers))
	for i, id := range subscribers {
		subscribersInt64[i] = int64(id)
	}
	query := r.data.db.ProxyUserSubscribe.Query().
		Where(func(s *sql.Selector) {
			// 订阅ID列表过滤
			if len(subscribers) > 0 {
				s.Where(sql.In(s.C(proxyusersubscribe.FieldSubscribeID), toInterfaceSlice(subscribersInt64)...))
			}

			// 是否仅活跃订阅（status IN (0,1,2)）
			if isActive != nil && *isActive {
				s.Where(sql.In(s.C(proxyusersubscribe.FieldStatus),
					uint8(model.UserSubscribeStatusPending),
					uint8(model.UserSubscribeStatusActive),
					uint8(model.UserSubscribeStatusFinish),
				))
			}

			// 开始时间过滤：start_time <= ?
			if startTime != 0 {
				s.Where(sql.LTE(s.C(proxyusersubscribe.FieldStartTime), time.UnixMilli(startTime)))
			}

			// 结束时间过滤：expire_time >= ?
			if endTime != 0 {
				s.Where(sql.GTE(s.C(proxyusersubscribe.FieldExpireTime), time.UnixMilli(endTime)))
			}
		})

	count, err := query.Count(ctx)
	if err != nil {
		r.log.Errorf("QueryQuotaTaskPreCount failed: %v", err)
		return 0, err
	}

	return int64(count), nil
}

// QueryQuotaTaskList 查询配额任务列表
func (r *adminMarketingRepo) QueryQuotaTaskList(ctx context.Context, page, size int32, status *uint32) ([]*ent.ProxyTask, int32, error) {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}

	query := r.data.db.ProxyTask.Query().
		Where(func(s *sql.Selector) {
			s.Where(sql.And(
				sql.EQ(s.C(proxytask.FieldType), int8(taskmodel.TypeQuota)),
			))
			if status != nil {
				s.Where(sql.EQ(s.C(proxytask.FieldStatus), int8(*status)))
			}
		})

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	list, err := query.
		Order(ent.Desc(proxytask.FieldCreatedAt)).
		Limit(int(size)).
		Offset(int((page - 1) * size)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return list, int32(total), nil
}

// ========== Helper Functions ==========

// toInterfaceSlice 将 []int64 转换为 []interface{} 用于 sql.In 查询
func toInterfaceSlice(slice []int64) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}
