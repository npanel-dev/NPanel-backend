package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/npanel-dev/NPanel-backend/api/admin/withdrawal/v1"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/predicate"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserwithdrawal"
	withdrawalbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/withdrawal"
	systemlog "github.com/npanel-dev/NPanel-backend/internal/model/log"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

type adminWithdrawalRepo struct {
	data *Data
	log  *log.Helper
}

func NewAdminWithdrawalRepo(data *Data, logger log.Logger) withdrawalbiz.WithdrawalRepo {
	return &adminWithdrawalRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *adminWithdrawalRepo) ListWithdrawals(ctx context.Context, req *v1.GetWithdrawalListRequest) ([]*withdrawalbiz.Withdrawal, int32, error) {
	where := make([]predicate.ProxyUserWithdrawal, 0, 3)
	if req.UserId != nil && *req.UserId > 0 {
		where = append(where, proxyuserwithdrawal.UserIDEQ(*req.UserId))
	}
	if req.Status != nil {
		where = append(where, proxyuserwithdrawal.StatusEQ(int8(*req.Status)))
	}
	if req.Method != "" {
		where = append(where, proxyuserwithdrawal.MethodEQ(req.Method))
	}

	query := r.data.db.ProxyUserWithdrawal.Query().Where(where...)
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	entities, err := query.
		Order(ent.Desc(proxyuserwithdrawal.FieldCreatedAt)).
		Offset(int((req.Page - 1) * req.Size)).
		Limit(int(req.Size)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*withdrawalbiz.Withdrawal, 0, len(entities))
	for _, entity := range entities {
		items = append(items, convertAdminWithdrawal(entity))
	}
	return items, int32(total), nil
}

func (r *adminWithdrawalRepo) ApproveWithdrawal(ctx context.Context, id int64) error {
	now := time.Now()
	affected, err := r.data.db.ProxyUserWithdrawal.Update().
		Where(proxyuserwithdrawal.IDEQ(id), proxyuserwithdrawal.StatusEQ(withdrawalbiz.StatusPending)).
		SetStatus(withdrawalbiz.StatusApproved).
		SetProcessedAt(now).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return nil
}

func (r *adminWithdrawalRepo) RejectWithdrawal(ctx context.Context, id int64, reason string) error {
	now := time.Now()
	return r.data.db.TX(ctx, func(tx *ent.Tx) error {
		record, err := tx.ProxyUserWithdrawal.Query().
			Where(proxyuserwithdrawal.IDEQ(id), proxyuserwithdrawal.StatusEQ(withdrawalbiz.StatusPending)).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
			}
			return err
		}

		affected, err := tx.ProxyUserWithdrawal.Update().
			Where(proxyuserwithdrawal.IDEQ(record.ID), proxyuserwithdrawal.StatusEQ(withdrawalbiz.StatusPending)).
			SetStatus(withdrawalbiz.StatusRejected).
			SetReason(reason).
			SetProcessedAt(now).
			Save(ctx)
		if err != nil {
			return err
		}
		if affected == 0 {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}

		if err := tx.ProxyUser.UpdateOneID(record.UserID).
			AddCommission(record.Amount).
			Exec(ctx); err != nil {
			return err
		}

		payload, err := (&systemlog.Commission{
			Type:      systemlog.CommissionTypeRefund,
			Amount:    record.Amount,
			Timestamp: now.UnixMilli(),
		}).Marshal()
		if err != nil {
			return err
		}
		_, err = tx.ProxySystemLog.Create().
			SetType(int8(systemlog.TypeCommission)).
			SetDate(now.Format(time.DateOnly)).
			SetObjectID(record.UserID).
			SetContent(string(payload)).
			SetCreatedAt(now).
			Save(ctx)
		return err
	})
}

func convertAdminWithdrawal(entity *ent.ProxyUserWithdrawal) *withdrawalbiz.Withdrawal {
	item := &withdrawalbiz.Withdrawal{
		ID:          entity.ID,
		UserID:      entity.UserID,
		Amount:      entity.Amount,
		Status:      int8(entity.Status),
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
		ProcessedAt: entity.ProcessedAt,
	}
	if entity.Method != nil {
		item.Method = *entity.Method
	}
	if entity.Content != nil {
		item.Content = *entity.Content
	}
	if entity.Reason != nil {
		item.Reason = *entity.Reason
	}
	return item
}
