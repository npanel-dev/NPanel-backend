package withdrawal

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

// User 用户信息（简化版）
type User struct {
	ID         int64
	Commission int64
}

type InviteConfig struct {
	WithdrawalMinAmount int64
	WithdrawalMethods   string
}

// Withdrawal 提现记录
type Withdrawal struct {
	ID          int64
	UserID      int64
	Amount      int64
	Method      string
	Content     string
	Status      int8
	Reason      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ProcessedAt *time.Time
}

// CommissionWithdrawRequest 佣金提现请求
type CommissionWithdrawRequest struct {
	Amount  int64
	Content string
	Method  string
}

// WithdrawalRepo 提现仓储接口
type WithdrawalRepo interface {
	ProcessCommissionWithdraw(ctx context.Context, userID int64, amount int64, method string, content string) (*Withdrawal, error)
	TransferCommissionToBalance(ctx context.Context, userID int64, amount int64) error
	GetUserWithdrawals(ctx context.Context, userID int64, page, pageSize int32) ([]*Withdrawal, int32, error)
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	GetInviteConfig(ctx context.Context) (*InviteConfig, error)
}

// WithdrawalUsecase 提现用例
type WithdrawalUsecase struct {
	repo   WithdrawalRepo
	logger *log.Helper
}

// NewWithdrawalUsecase 创建提现用例
func NewWithdrawalUsecase(repo WithdrawalRepo, logger log.Logger) *WithdrawalUsecase {
	return &WithdrawalUsecase{
		repo:   repo,
		logger: log.NewHelper(logger),
	}
}

// CommissionWithdraw 佣金提现
// 对应原项目 commissionWithdrawLogic.go
func (uc *WithdrawalUsecase) CommissionWithdraw(ctx context.Context, userID int64, req *CommissionWithdrawRequest) (*Withdrawal, error) {
	if req == nil || req.Amount <= 0 || strings.TrimSpace(req.Method) == "" || strings.TrimSpace(req.Content) == "" {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	config, err := uc.repo.GetInviteConfig(ctx)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("get invite config failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if config.WithdrawalMinAmount > 0 && req.Amount < config.WithdrawalMinAmount {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if !isWithdrawalMethodEnabled(config.WithdrawalMethods, req.Method) {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 获取用户信息
	user, err := uc.repo.GetUserByID(ctx, userID)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("get user failed: userID=%d, error=%v", userID, err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	// 检查佣金余额
	if user.Commission < req.Amount {
		uc.logger.WithContext(ctx).Errorf("user %d has insufficient commission balance: %.2f, requested: %.2f",
			userID, float64(user.Commission)/100, float64(req.Amount)/100)
		return nil, responsecode.NewKratosError(responsecode.ErrUserCommissionNotEnough)
	}

	withdrawal, err := uc.repo.ProcessCommissionWithdraw(ctx, userID, req.Amount, req.Method, req.Content)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("process commission withdraw failed: userID=%d, error=%v", userID, err)
		return nil, err
	}

	uc.logger.WithContext(ctx).Infof("user %d commission withdraw successful: amount=%d", userID, req.Amount)

	return withdrawal, nil
}

func (uc *WithdrawalUsecase) TransferCommissionToBalance(ctx context.Context, userID int64, amount int64) error {
	if amount <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	user, err := uc.repo.GetUserByID(ctx, userID)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("get user failed: userID=%d, error=%v", userID, err)
		return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if user.Commission < amount {
		return responsecode.NewKratosError(responsecode.ErrUserCommissionNotEnough)
	}

	if err := uc.repo.TransferCommissionToBalance(ctx, userID, amount); err != nil {
		uc.logger.WithContext(ctx).Errorf("transfer commission to balance failed: userID=%d, error=%v", userID, err)
		return err
	}
	return nil
}

// QueryWithdrawalLog 查询提现日志
func (uc *WithdrawalUsecase) QueryWithdrawalLog(ctx context.Context, userID int64, page, pageSize int32) ([]*Withdrawal, int32, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	withdrawals, total, err := uc.repo.GetUserWithdrawals(ctx, userID, page, pageSize)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("query withdrawal log failed: userID=%d, error=%v", userID, err)
		return nil, 0, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	return withdrawals, total, nil
}

type withdrawalMethodConfig struct {
	Method  string `json:"method"`
	Name    string `json:"name"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

func isWithdrawalMethodEnabled(raw string, method string) bool {
	method = strings.ToLower(strings.TrimSpace(method))
	if method == "ustd" {
		method = "usdt"
	}
	if method == "" {
		return false
	}
	if strings.TrimSpace(raw) == "" {
		return method == "alipay" || method == "wechat" || method == "usdt"
	}

	var configs []withdrawalMethodConfig
	if err := json.Unmarshal([]byte(raw), &configs); err == nil && len(configs) > 0 {
		for _, item := range configs {
			key := strings.ToLower(strings.TrimSpace(item.Method))
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(item.Name))
			}
			if key == "ustd" {
				key = "usdt"
			}
			if key == method && item.Enabled {
				return true
			}
		}
		return false
	}

	for _, part := range strings.Split(raw, ",") {
		key := strings.ToLower(strings.TrimSpace(part))
		if key == "ustd" {
			key = "usdt"
		}
		if key == method {
			return true
		}
	}
	return false
}
