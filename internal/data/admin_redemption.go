package data

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/npanel-dev/NPanel-backend/api/admin/redemption/v1"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyredemptioncode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyredemptionrecord"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	redemptionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/redemption"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

type adminRedemptionRepo struct {
	data   *Data
	logger *log.Helper
}

func NewAdminRedemptionRepo(d *Data, logger log.Logger) redemptionbiz.RedemptionRepo {
	return &adminRedemptionRepo{data: d, logger: log.NewHelper(logger)}
}

func (r *adminRedemptionRepo) generateUniqueCode(ctx context.Context) (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const codeLength = 16

	for i := 0; i < 10; i++ {
		code := make([]byte, codeLength)
		for j := 0; j < codeLength; j++ {
			num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
			if err != nil {
				return "", fmt.Errorf("failed to generate random number: %w", err)
			}
			code[j] = charset[num.Int64()]
		}
		codeStr := string(code)
		_, err := r.data.db.ProxyRedemptionCode.Query().Where(proxyredemptioncode.Code(codeStr)).Only(ctx)
		if ent.IsNotFound(err) {
			return codeStr, nil
		}
		if err != nil {
			return "", fmt.Errorf("failed to check code existence: %w", err)
		}
	}
	return "", fmt.Errorf("failed to generate unique code after %d retries", 10)
}

func (r *adminRedemptionRepo) CreateRedemptionCode(ctx context.Context, req *v1.CreateRedemptionCodeRequest) (int64, error) {
	_, err := r.data.db.ProxySubscribe.Query().Where(proxysubscribe.IDEQ(req.SubscribePlan)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, responsecode.NewKratosError(responsecode.ErrInvalidParams)
		}
		return 0, err
	}

	var createdCount int64
	now := time.Now()
	for i := int64(0); i < req.BatchCount; i++ {
		code, err := r.generateUniqueCode(ctx)
		if err != nil {
			return 0, err
		}
		_, err = r.data.db.ProxyRedemptionCode.Create().
			SetCode(code).
			SetTotalCount(int32(req.TotalCount)).
			SetUsedCount(0).
			SetSubscribePlan(req.SubscribePlan).
			SetUnitTime(req.UnitTime).
			SetQuantity(int32(req.Quantity)).
			SetStatus(1).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if err != nil {
			return 0, err
		}
		createdCount++
	}
	return createdCount, nil
}

func (r *adminRedemptionRepo) UpdateRedemptionCode(ctx context.Context, req *v1.UpdateRedemptionCodeRequest) error {
	code, err := r.data.db.ProxyRedemptionCode.Query().
		Where(
			proxyredemptioncode.IDEQ(req.Id),
			proxyredemptioncode.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrRedemptionCodeNotFound)
		}
		return err
	}

	update := code.Update()
	if req.TotalCount != 0 {
		if req.TotalCount < int64(code.UsedCount) {
			return responsecode.NewKratosError(responsecode.ErrInvalidParams)
		}
		update.SetTotalCount(int32(req.TotalCount))
	}
	if req.SubscribePlan != 0 {
		update.SetSubscribePlan(req.SubscribePlan)
	}
	if req.UnitTime != "" {
		update.SetUnitTime(req.UnitTime)
	}
	if req.Quantity != 0 {
		update.SetQuantity(int32(req.Quantity))
	}
	update.SetUpdatedAt(time.Now())
	return update.Exec(ctx)
}

func (r *adminRedemptionRepo) ToggleRedemptionCodeStatus(ctx context.Context, req *v1.ToggleRedemptionCodeStatusRequest) error {
	code, err := r.data.db.ProxyRedemptionCode.Query().
		Where(
			proxyredemptioncode.IDEQ(req.Id),
			proxyredemptioncode.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrInvalidParams)
		}
		return err
	}
	return code.Update().SetStatus(int8(req.Status)).SetUpdatedAt(time.Now()).Exec(ctx)
}

func (r *adminRedemptionRepo) DeleteRedemptionCode(ctx context.Context, id int64) error {
	now := time.Now()
	affected, err := r.data.db.ProxyRedemptionCode.Update().
		Where(
			proxyredemptioncode.IDEQ(id),
			proxyredemptioncode.DeletedAtIsNil(),
		).
		SetStatus(0).
		SetDeletedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return responsecode.NewKratosError(responsecode.ErrRedemptionCodeNotFound)
	}
	return nil
}

func (r *adminRedemptionRepo) BatchDeleteRedemptionCode(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if err := r.DeleteRedemptionCode(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *adminRedemptionRepo) GetRedemptionCodeList(ctx context.Context, req *v1.GetRedemptionCodeListRequest) ([]*ent.ProxyRedemptionCode, int32, error) {
	query := r.data.db.ProxyRedemptionCode.Query().Where(proxyredemptioncode.DeletedAtIsNil())
	if req.SubscribePlan != 0 {
		query = query.Where(proxyredemptioncode.SubscribePlanEQ(req.SubscribePlan))
	}
	if req.UnitTime != "" {
		query = query.Where(proxyredemptioncode.UnitTimeEQ(req.UnitTime))
	}
	if req.Code != "" {
		query = query.Where(proxyredemptioncode.CodeContainsFold(req.Code))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	list, err := query.Order(ent.Desc(proxyredemptioncode.FieldCreatedAt)).Offset(int((req.Page - 1) * req.Size)).Limit(int(req.Size)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return list, int32(total), nil
}

func (r *adminRedemptionRepo) GetRedemptionRecordList(ctx context.Context, req *v1.GetRedemptionRecordListRequest) ([]*ent.ProxyRedemptionRecord, int32, error) {
	query := r.data.db.ProxyRedemptionRecord.Query()
	if req.UserId != 0 {
		query = query.Where(proxyredemptionrecord.UserIDEQ(req.UserId))
	}
	if req.CodeId != 0 {
		query = query.Where(proxyredemptionrecord.RedemptionCodeIDEQ(req.CodeId))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	list, err := query.Order(ent.Desc(proxyredemptionrecord.FieldCreatedAt)).Offset(int((req.Page - 1) * req.Size)).Limit(int(req.Size)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return list, int32(total), nil
}
