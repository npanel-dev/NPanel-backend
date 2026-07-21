package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystemlog"
	logbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/log"
)

type adminSystemLogRepo struct {
	data *Data
	log  *log.Helper
}

// NewAdminSystemLogRepo 创建系统日志仓库
func NewAdminSystemLogRepo(data *Data, logger log.Logger) logbiz.SystemLogRepo {
	return &adminSystemLogRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// FilterSystemLog 过滤系统日志（通用方法）
func (r *adminSystemLogRepo) FilterSystemLog(ctx context.Context, params *logbiz.FilterParams) ([]*ent.ProxySystemLog, int32, error) {
	// 设置默认值
	if params.Page == 0 {
		params.Page = 1
	}
	if params.Size == 0 {
		params.Size = 20
	}

	// 构建查询
	query := r.data.db.ProxySystemLog.
		Query().
		Where(func(s *sql.Selector) {
			// 日期过滤
			if params.Date != "" {
				s.Where(sql.EQ(s.C(proxysystemlog.FieldDate), params.Date))
			}

			// 类型过滤
			if params.Type != 0 {
				s.Where(sql.EQ(s.C(proxysystemlog.FieldType), params.Type))
			}

			// 日期过滤
			if params.Date != "" {
				s.Where(sql.EQ(s.C(proxysystemlog.FieldDate), params.Date))
			}

			// 对象ID过滤
			if params.ObjectID != nil && *params.ObjectID > 0 {
				s.Where(sql.EQ(s.C(proxysystemlog.FieldObjectID), *params.ObjectID))
			}

			// 搜索过滤（在content中搜索）
			if params.Search != "" {
				s.Where(sql.ContainsFold(s.C(proxysystemlog.FieldContent), params.Search))
			}
		})

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表（按ID降序排序）
	list, err := query.
		Order(ent.Desc(proxysystemlog.FieldID)).
		Limit(int(params.Size)).
		Offset(int((params.Page - 1) * params.Size)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return list, int32(total), nil
}
