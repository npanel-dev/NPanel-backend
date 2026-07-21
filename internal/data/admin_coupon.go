package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxycoupon"
	"github.com/npanel-dev/NPanel-backend/internal/biz/admin/coupon"
)

const couponModule = "data/admin_coupon"

type couponRepo struct {
	data *Data
	log  *log.Helper
}

// NewCouponRepo create coupon repository
func NewCouponRepo(data *Data, logger log.Logger) coupon.CouponRepo {
	return &couponRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", couponModule)),
	}
}

// CreateCoupon 创建优惠券
func (r *couponRepo) CreateCoupon(ctx context.Context, name, code string, count int, typ int32, discount, startTime, expireTime, userLimit int64, subscribe string, usedCount int64, enable bool) error {
	_, err := r.data.db.ProxyCoupon.Create().
		SetName(name).
		SetCode(code).
		SetCount(int32(count)).
		SetType(int8(typ)).
		SetDiscount(discount).
		SetStartTime(startTime).
		SetExpireTime(expireTime).
		SetUserLimit(userLimit).
		SetSubscribe(subscribe).
		SetUsedCount(int8(usedCount)).
		SetEnable(enable).
		Save(ctx)

	return err
}

// UpdateCoupon 更新优惠券
func (r *couponRepo) UpdateCoupon(ctx context.Context, id int, name, code string, count int, typ int32, discount, startTime, expireTime, userLimit int64, subscribe string, usedCount int64, enable bool) error {
	return r.data.db.ProxyCoupon.Update().
		Where(
			proxycoupon.ID(int64(id)),
		).
		SetName(name).
		SetCode(code).
		SetCount(int32(count)).
		SetType(int8(typ)).
		SetDiscount(discount).
		SetStartTime(startTime).
		SetExpireTime(expireTime).
		SetUserLimit(userLimit).
		SetSubscribe(subscribe).
		SetUsedCount(int8(usedCount)).
		SetEnable(enable).
		Exec(ctx)
}

// DeleteCoupon 删除优惠券
func (r *couponRepo) DeleteCoupon(ctx context.Context, id int) error {
	_, err := r.data.db.ProxyCoupon.Delete().
		Where(
			proxycoupon.ID(int64(id)),
		).
		Exec(ctx)
	return err
}

// BatchDeleteCoupon 批量删除优惠券
func (r *couponRepo) BatchDeleteCoupon(ctx context.Context, ids []int) error {
	// Convert int IDs to int64
	int64IDs := make([]int64, len(ids))
	for i, id := range ids {
		int64IDs[i] = int64(id)
	}
	_, err := r.data.db.ProxyCoupon.Delete().
		Where(
			proxycoupon.IDIn(int64IDs...),
		).
		Exec(ctx)
	return err
}

// GetCouponList 获取优惠券列表
func (r *couponRepo) GetCouponList(ctx context.Context, page, size, subscribe int64, search string) ([]*ent.ProxyCoupon, int32, error) {
	query := r.data.db.ProxyCoupon.Query()

	if subscribe != 0 {
		query = query.Where(func(s *sql.Selector) {
			s.Where(csvContains(r.data.databaseDriver, proxycoupon.FieldSubscribe, subscribe))
		})
	}

	// 如果有搜索关键字
	if search != "" {
		query = query.Where(
			proxycoupon.Or(
				proxycoupon.NameContainsFold(search),
				proxycoupon.CodeContainsFold(search),
			),
		)
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	list, err := query.
		Limit(int(size)).
		Offset(int((page - 1) * size)).
		Order(ent.Desc(proxycoupon.FieldID)).
		All(ctx)

	return list, int32(total), err
}

// FindCouponByCode 根据代码查找优惠券
func (r *couponRepo) FindCouponByCode(ctx context.Context, code string) (*ent.ProxyCoupon, error) {
	return r.data.db.ProxyCoupon.Query().
		Where(proxycoupon.Code(code)).
		Only(ctx)
}
