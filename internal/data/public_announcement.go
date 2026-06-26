package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent/proxyannouncement"
	announcementBiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/announcement"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

type publicAnnouncementRepo struct {
	data *Data
	log  *log.Helper
}

// NewPublicAnnouncementRepo 创建Public Announcement仓库
func NewPublicAnnouncementRepo(data *Data, logger log.Logger) announcementBiz.AnnouncementRepo {
	return &publicAnnouncementRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// QueryAnnouncement 查询公告列表
func (r *publicAnnouncementRepo) QueryAnnouncement(ctx context.Context, page, size int32, pinned, popup *bool) ([]*announcementBiz.Announcement, int32, error) {
	// 查询条件: show=true
	query := r.data.db.ProxyAnnouncement.Query().
		Where(
			proxyannouncement.Show(true),
		)

	// pinned过滤
	if pinned != nil {
		query = query.Where(proxyannouncement.Pinned(*pinned))
	}

	// popup过滤
	if popup != nil {
		query = query.Where(proxyannouncement.Popup(*popup))
	}

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		r.log.Errorf("QueryAnnouncement count error: %v", err)
		return nil, 0, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	if size == 0 {
		size = 15
	}

	// 分页查询
	announcements, err := query.
		Order(func(s *sql.Selector) {
			s.OrderBy(
				sql.Desc(proxyannouncement.FieldPinned),
				sql.Desc(proxyannouncement.FieldCreatedAt),
				sql.Desc(proxyannouncement.FieldID),
			)
		}).
		Offset(int((page - 1) * size)).
		Limit(int(size)).
		All(ctx)

	if err != nil {
		r.log.Errorf("QueryAnnouncement query error: %v", err)
		return nil, 0, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	result := make([]*announcementBiz.Announcement, 0, len(announcements))
	for _, a := range announcements {
		result = append(result, &announcementBiz.Announcement{
			ID:        a.ID,
			Title:     a.Title,
			Content:   a.Content,
			Show:      a.Show,
			Pinned:    a.Pinned,
			Popup:     a.Popup,
			CreatedAt: a.CreatedAt.UnixMilli(),
			UpdatedAt: a.UpdatedAt.UnixMilli(),
		})
	}

	return result, int32(total), nil
}
