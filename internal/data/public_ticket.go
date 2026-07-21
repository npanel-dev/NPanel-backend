package data

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyticket"
	"github.com/npanel-dev/NPanel-backend/ent/proxyticketfollow"
	ticketBiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/ticket"
)

type publicTicketRepo struct {
	data   *Data
	logger *log.Helper
}

// NewPublicTicketRepo creates a new ticket repository
func NewPublicTicketRepo(d *Data, logger log.Logger) ticketBiz.TicketRepo {
	return &publicTicketRepo{
		data:   d,
		logger: log.NewHelper(logger),
	}
}

// CreateTicket creates a new ticket
// 完整复刻原项目 createUserTicketLogic.go
func (r *publicTicketRepo) CreateTicket(ctx context.Context, userID int, title, description string) error {
	r.logger.Infof("[CreateTicket] userID: %d, title: %s", userID, title)

	// 创建工单
	err := r.data.db.ProxyTicket.Create().
		SetUserID(int64(userID)).
		SetTitle(title).
		SetDescription(description).
		SetStatus(int8(ticketBiz.StatusPending)).
		Exec(ctx)

	if err != nil {
		r.logger.Errorf("[CreateTicket] 创建工单失败: %v", err)
		return errors.InternalServer("DATABASE_ERROR", fmt.Sprintf("创建工单失败: %v", err))
	}

	r.logger.Infof("[CreateTicket] 工单创建成功")
	return nil
}

// GetTicketList gets user's ticket list
// 完整复刻原项目 getUserTicketListLogic.go
// 注意：当前实现使用单库模型
// 支持分页、状态过滤、搜索功能
func (r *publicTicketRepo) GetTicketList(ctx context.Context, userID int, page, size int, status *int32, search *string) (int32, []*ticketBiz.TicketInfo, error) {
	r.logger.Infof("[GetTicketList] userID: %d, page: %d, size: %d", userID, page, size)

	// 构建基础查询 - 包含user_id过滤
	query := r.data.db.ProxyTicket.Query().
		Where(proxyticket.UserIDEQ(int64(userID)))

	// 状态过滤；未指定状态时显示全部工单，包括已关闭工单。
	if status != nil {
		r.logger.Infof("[GetTicketList] 状态过滤: %d", *status)
		query = query.Where(proxyticket.StatusEQ(int8(*status)))
	}

	// 搜索功能（标题或描述包含关键字）
	if search != nil && *search != "" {
		r.logger.Infof("[GetTicketList] 搜索关键字: %s", *search)
		query = query.Where(proxyticket.Or(
			proxyticket.TitleContainsFold(*search),
			proxyticket.DescriptionContainsFold(*search),
		))
	}

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		r.logger.Errorf("[GetTicketList] 查询总数失败: %v", err)
		return 0, nil, errors.InternalServer("DATABASE_ERROR", fmt.Sprintf("查询总数失败: %v", err))
	}

	// 分页查询
	offset := (page - 1) * size
	tickets, err := query.
		Order(ent.Desc(proxyticket.FieldID)). // ⚠️ 重要：原项目按ID降序，不是created_at
		Offset(int(offset)).
		Limit(int(size)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("[GetTicketList] 查询工单列表失败: %v", err)
		return 0, nil, errors.InternalServer("DATABASE_ERROR", fmt.Sprintf("查询工单列表失败: %v", err))
	}

	// 转换为业务对象
	list := make([]*ticketBiz.TicketInfo, 0, len(tickets))
	for _, t := range tickets {
		list = append(list, &ticketBiz.TicketInfo{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			UserID:      t.UserID,
			Status:      int32(t.Status),
			CreatedAt:   int(t.CreatedAt.UnixMilli()), // Convert to Unix milliseconds
			UpdatedAt:   int(t.UpdatedAt.UnixMilli()), // Convert to Unix milliseconds
		})
	}

	r.logger.Infof("[GetTicketList] 查询成功, total: %d, count: %d", total, len(list))
	return int32(total), list, nil
}

// GetTicketByID gets ticket by ID
func (r *publicTicketRepo) GetTicketByID(ctx context.Context, ticketID int) (*ticketBiz.TicketInfo, error) {
	r.logger.Infof("[GetTicketByID] ticketID: %d", ticketID)

	// 查询工单
	ticket, err := r.data.db.ProxyTicket.Query().
		Where(proxyticket.IDEQ(int64(ticketID))).
		Only(ctx)

	if err != nil {
		r.logger.Errorf("[GetTicketByID] 查询工单失败: %v", err)
		return nil, errors.InternalServer("DATABASE_ERROR", fmt.Sprintf("查询工单失败: %v", err))
	}

	r.logger.Infof("[GetTicketByID] 查询成功")
	return &ticketBiz.TicketInfo{
		ID:          ticket.ID,
		Title:       ticket.Title,
		Description: ticket.Description,
		UserID:      ticket.UserID,
		Status:      int32(ticket.Status),
		CreatedAt:   int(ticket.CreatedAt.UnixMilli()), // Convert to Unix milliseconds
		UpdatedAt:   int(ticket.UpdatedAt.UnixMilli()), // Convert to Unix milliseconds
	}, nil
}

// UpdateTicketStatus updates ticket status
// 完整复刻原项目 updateUserTicketStatusLogic.go
// 使用user_id过滤确保权限安全
func (r *publicTicketRepo) UpdateTicketStatus(ctx context.Context, userID, ticketID int64, status int32) error {
	r.logger.Infof("[UpdateTicketStatus] userID: %d, ticketID: %d, status: %d",
		userID, ticketID, status)

	// 更新状态 - 使用user_id过滤确保安全
	err := r.data.db.ProxyTicket.Update().
		Where(proxyticket.IDEQ(ticketID),
			proxyticket.UserIDEQ(userID)).
		SetStatus(int8(status)).
		Exec(ctx)

	if err != nil {
		r.logger.Errorf("[UpdateTicketStatus] 更新状态失败: %v", err)
		return errors.InternalServer("DATABASE_ERROR", fmt.Sprintf("更新状态失败: %v", err))
	}

	r.logger.Infof("[UpdateTicketStatus] 更新成功")
	return nil
}

// CreateTicketFollow creates a follow-up record
// 完整复刻原项目 createUserTicketFollowLogic.go
func (r *publicTicketRepo) CreateTicketFollow(ctx context.Context, ticketID int64, from string, followType int32, content string) error {
	r.logger.Infof("[CreateTicketFollow] ticketID: %d, from: %s, type: %d",
		ticketID, from, followType)

	// 创建跟进记录
	err := r.data.db.ProxyTicketFollow.Create().
		SetTicketID(ticketID).
		SetFrom(from).
		SetType(int8(followType)).
		SetContent(content).
		Exec(ctx)

	if err != nil {
		r.logger.Errorf("[CreateTicketFollow] 创建跟进记录失败: %v", err)
		return errors.InternalServer("DATABASE_ERROR", fmt.Sprintf("创建跟进记录失败: %v", err))
	}

	r.logger.Infof("[CreateTicketFollow] 创建成功")
	return nil
}

// GetTicketFollows gets all follow-ups for a ticket
func (r *publicTicketRepo) GetTicketFollows(ctx context.Context, ticketID int) ([]*ticketBiz.TicketFollow, error) {
	r.logger.Infof("[GetTicketFollows] ticketID: %d", ticketID)

	// 查询跟进记录
	follows, err := r.data.db.ProxyTicketFollow.Query().
		Where(proxyticketfollow.TicketIDEQ(int64(ticketID))).
		Order(ent.Asc(proxyticketfollow.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("[GetTicketFollows] 查询跟进记录失败: %v", err)
		return nil, errors.InternalServer("DATABASE_ERROR", fmt.Sprintf("查询跟进记录失败: %v", err))
	}

	// 转换为业务对象
	list := make([]*ticketBiz.TicketFollow, 0, len(follows))
	for _, f := range follows {
		list = append(list, &ticketBiz.TicketFollow{
			ID:        f.ID,
			TicketID:  f.TicketID,
			From:      f.From,
			Type:      int32(f.Type),
			Content:   f.Content,
			CreatedAt: int(f.CreatedAt.UnixMilli()), // Convert to Unix milliseconds
		})
	}

	r.logger.Infof("[GetTicketFollows] 查询成功, count: %d", len(list))
	return list, nil
}
