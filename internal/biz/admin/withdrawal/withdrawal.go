package withdrawal

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/npanel-dev/NPanel-backend/api/admin/withdrawal/v1"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

const (
	StatusPending  int8 = 0
	StatusApproved int8 = 1
	StatusRejected int8 = 2
)

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

type WithdrawalRepo interface {
	ListWithdrawals(ctx context.Context, req *v1.GetWithdrawalListRequest) ([]*Withdrawal, int32, error)
	ApproveWithdrawal(ctx context.Context, id int64) error
	RejectWithdrawal(ctx context.Context, id int64, reason string) error
}

type WithdrawalUsecase struct {
	repo WithdrawalRepo
	log  *log.Helper
}

func NewWithdrawalUsecase(repo WithdrawalRepo, logger log.Logger) *WithdrawalUsecase {
	return &WithdrawalUsecase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "biz/admin/withdrawal")),
	}
}

func (uc *WithdrawalUsecase) ListWithdrawals(ctx context.Context, req *v1.GetWithdrawalListRequest) ([]*Withdrawal, int32, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.Size > 100 {
		req.Size = 100
	}
	return uc.repo.ListWithdrawals(ctx, req)
}

func (uc *WithdrawalUsecase) ApproveWithdrawal(ctx context.Context, id int64) error {
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if err := uc.repo.ApproveWithdrawal(ctx, id); err != nil {
		uc.log.Errorf("approve withdrawal failed: id=%d err=%v", id, err)
		return err
	}
	return nil
}

func (uc *WithdrawalUsecase) RejectWithdrawal(ctx context.Context, id int64, reason string) error {
	if id <= 0 || strings.TrimSpace(reason) == "" || len(reason) > 500 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if err := uc.repo.RejectWithdrawal(ctx, id, strings.TrimSpace(reason)); err != nil {
		uc.log.Errorf("reject withdrawal failed: id=%d err=%v", id, err)
		return err
	}
	return nil
}
