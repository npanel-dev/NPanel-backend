package data

import (
	"context"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserwithdrawal"
	"github.com/npanel-dev/NPanel-backend/internal/biz/public/withdrawal"
	systemlog "github.com/npanel-dev/NPanel-backend/internal/model/log"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

// withdrawalRepo 提现数据仓储
type withdrawalRepo struct {
	data *Data
	log  *log.Helper
}

// NewWithdrawalRepo 创建提现数据仓储
func NewWithdrawalRepo(data *Data, logger log.Logger) withdrawal.WithdrawalRepo {
	return &withdrawalRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *withdrawalRepo) ProcessCommissionWithdraw(ctx context.Context, userID int64, amount int64, method string, content string) (*withdrawal.Withdrawal, error) {
	var created *ent.ProxyUserWithdrawal
	err := r.data.db.TX(ctx, func(tx *ent.Tx) error {
		affected, err := tx.ProxyUser.Update().
			Where(proxyuser.IDEQ(userID), proxyuser.CommissionGTE(amount)).
			AddCommission(-amount).
			Save(ctx)
		if err != nil {
			return err
		}
		if affected == 0 {
			return responsecode.NewKratosError(responsecode.ErrUserCommissionNotEnough)
		}

		payload, err := (&systemlog.Commission{
			Type:      systemlog.CommissionTypeWithdraw,
			Amount:    amount,
			Timestamp: time.Now().UnixMilli(),
		}).Marshal()
		if err != nil {
			return err
		}

		if _, err := tx.ProxySystemLog.Create().
			SetType(int8(systemlog.TypeCommission)).
			SetDate(time.Now().Format("2006-01-02")).
			SetObjectID(userID).
			SetContent(string(payload)).
			SetCreatedAt(time.Now()).
			Save(ctx); err != nil {
			return err
		}

		created, err = tx.ProxyUserWithdrawal.Create().
			SetUserID(userID).
			SetAmount(amount).
			SetMethod(method).
			SetContent(content).
			SetStatus(0).
			SetReason("").
			Save(ctx)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.convertToModel(created), nil
}

func (r *withdrawalRepo) TransferCommissionToBalance(ctx context.Context, userID int64, amount int64) error {
	now := time.Now()
	return r.data.db.TX(ctx, func(tx *ent.Tx) error {
		affected, err := tx.ProxyUser.Update().
			Where(proxyuser.IDEQ(userID), proxyuser.CommissionGTE(amount)).
			AddCommission(-amount).
			AddBalance(amount).
			Save(ctx)
		if err != nil {
			return err
		}
		if affected == 0 {
			return responsecode.NewKratosError(responsecode.ErrUserCommissionNotEnough)
		}

		commissionPayload, err := (&systemlog.Commission{
			Type:      systemlog.CommissionTypeConvertBalance,
			Amount:    amount,
			Timestamp: now.UnixMilli(),
		}).Marshal()
		if err != nil {
			return err
		}
		if _, err := tx.ProxySystemLog.Create().
			SetType(int8(systemlog.TypeCommission)).
			SetDate(now.Format(time.DateOnly)).
			SetObjectID(userID).
			SetContent(string(commissionPayload)).
			SetCreatedAt(now).
			Save(ctx); err != nil {
			return err
		}

		user, err := tx.ProxyUser.Get(ctx, userID)
		if err != nil {
			return err
		}
		balance := int64(0)
		if user.Balance != nil {
			balance = *user.Balance
		}
		balancePayload, err := (&systemlog.Balance{
			Type:      systemlog.BalanceTypeCommissionTransfer,
			Amount:    amount,
			Balance:   balance,
			Timestamp: now.UnixMilli(),
		}).Marshal()
		if err != nil {
			return err
		}
		_, err = tx.ProxySystemLog.Create().
			SetType(int8(systemlog.TypeBalance)).
			SetDate(now.Format(time.DateOnly)).
			SetObjectID(userID).
			SetContent(string(balancePayload)).
			SetCreatedAt(now).
			Save(ctx)
		return err
	})
}

// GetWithdrawalByID 根据ID获取提现记录
func (r *withdrawalRepo) GetWithdrawalByID(ctx context.Context, id int64) (*withdrawal.Withdrawal, error) {
	entity, err := r.data.db.ProxyUserWithdrawal.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.convertToModel(entity), nil
}

// GetUserWithdrawals 获取用户提现记录列表
func (r *withdrawalRepo) GetUserWithdrawals(ctx context.Context, userID int64, page, pageSize int32) ([]*withdrawal.Withdrawal, int32, error) {
	query := r.data.db.ProxyUserWithdrawal.Query().
		Where(proxyuserwithdrawal.UserID(userID)).
		Order(ent.Desc(proxyuserwithdrawal.FieldCreatedAt))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	entities, err := query.
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	withdrawals := make([]*withdrawal.Withdrawal, len(entities))
	for i, entity := range entities {
		withdrawals[i] = r.convertToModel(entity)
	}

	return withdrawals, int32(total), nil
}

// UpdateUserCommission 更新用户佣金
func (r *withdrawalRepo) UpdateUserCommission(ctx context.Context, userID int64, commission int64) error {
	return r.data.db.ProxyUser.UpdateOneID(userID).
		SetCommission(commission).
		Exec(ctx)
}

// GetUserByID 根据ID获取用户
func (r *withdrawalRepo) GetUserByID(ctx context.Context, userID int64) (*withdrawal.User, error) {
	entity, err := r.data.db.ProxyUser.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	commission := int64(0)
	if entity.Commission != nil {
		commission = *entity.Commission
	}
	return &withdrawal.User{
		ID:         entity.ID,
		Commission: commission,
	}, nil
}

func (r *withdrawalRepo) GetInviteConfig(ctx context.Context) (*withdrawal.InviteConfig, error) {
	entries, err := r.data.db.ProxySystem.Query().
		Where(proxysystem.CategoryEQ("invite")).
		All(ctx)
	if err != nil {
		return nil, err
	}
	config := &withdrawal.InviteConfig{}
	for _, entry := range entries {
		switch entry.Key {
		case "WithdrawalMinAmount", "withdrawal_min_amount":
			value, err := strconv.ParseInt(entry.Value, 10, 64)
			if err == nil {
				config.WithdrawalMinAmount = value
			}
		case "WithdrawalMethods", "withdrawal_methods", "WithdrawalMethod":
			config.WithdrawalMethods = entry.Value
		}
	}
	return config, nil
}

// convertToModel 转换为业务模型
func (r *withdrawalRepo) convertToModel(entity *ent.ProxyUserWithdrawal) *withdrawal.Withdrawal {
	content := ""
	if entity.Content != nil {
		content = *entity.Content
	}
	reason := ""
	if entity.Reason != nil {
		reason = *entity.Reason
	}
	return &withdrawal.Withdrawal{
		ID:          entity.ID,
		UserID:      entity.UserID,
		Amount:      entity.Amount,
		Method:      valueOrEmpty(entity.Method),
		Content:     content,
		Status:      int8(entity.Status),
		Reason:      reason,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
		ProcessedAt: entity.ProcessedAt,
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
