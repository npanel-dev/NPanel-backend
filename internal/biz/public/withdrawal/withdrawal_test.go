package withdrawal

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

type withdrawalRepoMock struct {
	page     int32
	pageSize int32
}

func (m *withdrawalRepoMock) ProcessCommissionWithdraw(context.Context, int64, int64, string, string) (*Withdrawal, error) {
	return nil, nil
}

func (m *withdrawalRepoMock) TransferCommissionToBalance(context.Context, int64, int64) error {
	return nil
}

func (m *withdrawalRepoMock) GetUserWithdrawals(_ context.Context, _ int64, page, pageSize int32) ([]*Withdrawal, int32, error) {
	m.page = page
	m.pageSize = pageSize
	return nil, 0, nil
}

func (m *withdrawalRepoMock) GetUserByID(context.Context, int64) (*User, error) {
	return nil, nil
}

func (m *withdrawalRepoMock) GetInviteConfig(context.Context) (*InviteConfig, error) {
	return nil, nil
}

func TestQueryWithdrawalLogNormalizesPagination(t *testing.T) {
	repo := &withdrawalRepoMock{}
	uc := NewWithdrawalUsecase(repo, log.NewStdLogger(io.Discard))

	if _, _, err := uc.QueryWithdrawalLog(context.Background(), 1, 0, 0); err != nil {
		t.Fatalf("QueryWithdrawalLog() error = %v", err)
	}

	if repo.page != 1 || repo.pageSize != 10 {
		t.Fatalf("pagination = (%d, %d), want (1, 10)", repo.page, repo.pageSize)
	}
}

func TestQueryWithdrawalLogCapsPageSize(t *testing.T) {
	repo := &withdrawalRepoMock{}
	uc := NewWithdrawalUsecase(repo, log.NewStdLogger(io.Discard))

	if _, _, err := uc.QueryWithdrawalLog(context.Background(), 1, 2, 500); err != nil {
		t.Fatalf("QueryWithdrawalLog() error = %v", err)
	}

	if repo.page != 2 || repo.pageSize != 100 {
		t.Fatalf("pagination = (%d, %d), want (2, 100)", repo.page, repo.pageSize)
	}
}
