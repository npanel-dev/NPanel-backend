package withdrawal

import (
	"context"

	v1 "github.com/npanel-dev/NPanel-backend/api/admin/withdrawal/v1"
	withdrawalbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/withdrawal"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

type WithdrawalService struct {
	v1.UnimplementedWithdrawalServiceServer

	uc *withdrawalbiz.WithdrawalUsecase
}

func NewWithdrawalService(uc *withdrawalbiz.WithdrawalUsecase) *WithdrawalService {
	return &WithdrawalService{uc: uc}
}

func (s *WithdrawalService) GetWithdrawalList(ctx context.Context, req *v1.GetWithdrawalListRequest) (*v1.GetWithdrawalListReply, error) {
	list, total, err := s.uc.ListWithdrawals(ctx, req)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.Withdrawal, 0, len(list))
	for _, item := range list {
		items = append(items, convertWithdrawal(item))
	}
	return &v1.GetWithdrawalListReply{
		Code:    200,
		Message: responsecode.CodeMessages[200],
		Data: &v1.WithdrawalListData{
			List:  items,
			Total: total,
		},
	}, nil
}

func (s *WithdrawalService) ApproveWithdrawal(ctx context.Context, req *v1.ApproveWithdrawalRequest) (*v1.ApproveWithdrawalReply, error) {
	if err := s.uc.ApproveWithdrawal(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.ApproveWithdrawalReply{
		Code:    200,
		Message: responsecode.CodeMessages[200],
		Data:    &v1.OperationData{Success: true},
	}, nil
}

func (s *WithdrawalService) RejectWithdrawal(ctx context.Context, req *v1.RejectWithdrawalRequest) (*v1.RejectWithdrawalReply, error) {
	if err := s.uc.RejectWithdrawal(ctx, req.Id, req.Reason); err != nil {
		return nil, err
	}
	return &v1.RejectWithdrawalReply{
		Code:    200,
		Message: responsecode.CodeMessages[200],
		Data:    &v1.OperationData{Success: true},
	}, nil
}

func convertWithdrawal(item *withdrawalbiz.Withdrawal) *v1.Withdrawal {
	result := &v1.Withdrawal{
		Id:        item.ID,
		UserId:    item.UserID,
		Amount:    item.Amount,
		Method:    item.Method,
		Content:   item.Content,
		Status:    int32(item.Status),
		Reason:    item.Reason,
		CreatedAt: item.CreatedAt.UnixMilli(),
		UpdatedAt: item.UpdatedAt.UnixMilli(),
	}
	if item.ProcessedAt != nil {
		result.ProcessedAt = item.ProcessedAt.UnixMilli()
	}
	return result
}
