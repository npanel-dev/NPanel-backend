package data

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyticket"
	"github.com/npanel-dev/NPanel-backend/ent/proxyticketfollow"
	ticketbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/ticket"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

type ticketRepo struct {
	data *Data
	log  *log.Helper
}

// NewTicketRepo 创建工单仓库
func NewTicketRepo(data *Data, logger log.Logger) ticketbiz.TicketRepo {
	return &ticketRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/admin/ticket")),
	}
}

// GetTicketById 获取工单基本信息（不含跟进列表）
func (r *ticketRepo) GetTicketById(ctx context.Context, id int) (*ticketbiz.Ticket, error) {
	po, err := r.data.db.ProxyTicket.Query().
		Where(
			proxyticket.ID(int64(id)),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New(responsecode.ErrTicketNotFound, "ticket not found", "TICKET_NOT_FOUND")
		}
		return nil, err
	}

	return &ticketbiz.Ticket{
		Id:          int64(po.ID),
		Title:       po.Title,
		Description: po.Description,
		UserId:      int64(po.UserID),
		Status:      po.Status,
		CreatedAt:   int(po.CreatedAt.UnixMilli()), // Convert to Unix milliseconds
		UpdatedAt:   int(po.UpdatedAt.UnixMilli()), // Convert to Unix milliseconds
	}, nil
}

// GetTicket 获取工单详情（包含跟进列表）
func (r *ticketRepo) GetTicket(ctx context.Context, id int) (*ticketbiz.Ticket, error) {
	// 获取工单基本信息
	ticket, err := r.GetTicketById(ctx, id)
	if err != nil {
		return nil, err
	}

	// 获取跟进列表
	follows, err := r.data.db.ProxyTicketFollow.Query().
		Where(
			proxyticketfollow.TicketID(int64(id)),
		).
		Order(ent.Asc(proxyticketfollow.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// 转换跟进列表
	ticket.Follow = make([]*ticketbiz.Follow, 0, len(follows))
	for _, f := range follows {
		ticket.Follow = append(ticket.Follow, &ticketbiz.Follow{
			Id:        int64(f.ID),
			TicketId:  int64(f.TicketID),
			From:      f.From,
			Type:      f.Type,
			Content:   f.Content,
			CreatedAt: int(f.CreatedAt.UnixMilli()), // Convert to Unix milliseconds
		})
	}

	return ticket, nil
}

// GetTicketList 获取工单列表
func (r *ticketRepo) GetTicketList(ctx context.Context, page, size int, userId int64, status *int8, search string) (int32, []*ticketbiz.Ticket, error) {
	query := r.data.db.ProxyTicket.Query()

	// 按用户ID筛选
	if userId > 0 {
		query = query.Where(proxyticket.UserID(int64(userId)))
	}

	// 按状态筛选；未指定状态时显示全部工单，包括已关闭工单。
	if status != nil {
		query = query.Where(proxyticket.Status(*status))
	}

	// 搜索（标题或描述）
	if search != "" {
		query = query.Where(
			proxyticket.Or(
				proxyticket.TitleContainsFold(search),
				proxyticket.DescriptionContainsFold(search),
			),
		)
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return 0, nil, err
	}

	// 分页查询
	list, err := query.
		Order(ent.Desc(proxyticket.FieldID)).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return 0, nil, err
	}

	// 转换结果
	// ⚠️ 注意：GetTicketList 不查询Follows字段（与老项目保持一致）
	// 只有GetTicket获取详情时才查询Follows
	tickets := make([]*ticketbiz.Ticket, 0, len(list))
	for _, po := range list {
		ticket := &ticketbiz.Ticket{
			Id:          int64(po.ID),
			Title:       po.Title,
			Description: po.Description,
			UserId:      int64(po.UserID),
			Status:      po.Status,
			CreatedAt:   int(po.CreatedAt.UnixMilli()), // Convert to Unix milliseconds
			UpdatedAt:   int(po.UpdatedAt.UnixMilli()), // Convert to Unix milliseconds
			Follow:      []*ticketbiz.Follow{},         // 列表接口不返回Follows，保持空数组
		}

		tickets = append(tickets, ticket)
	}

	return int32(total), tickets, nil
}

// UpdateTicketStatus 更新工单状态
func (r *ticketRepo) UpdateTicketStatus(ctx context.Context, id int, status int8) error {
	_, err := r.data.db.ProxyTicket.Update().
		Where(proxyticket.ID(int64(id))).
		SetStatus(status).
		Save(ctx)
	if err != nil {
		return err
	}
	return nil
}

// CreateTicketFollow 创建工单跟进
func (r *ticketRepo) CreateTicketFollow(ctx context.Context, follow *ticketbiz.Follow) error {
	_, err := r.data.db.ProxyTicketFollow.Create().
		SetTicketID(follow.TicketId).
		SetFrom(follow.From).
		SetType(follow.Type).
		SetContent(follow.Content).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create ticket follow: %w", err)
	}
	return nil
}
