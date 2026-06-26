package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyannouncement"
	announcementbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/announcement"
)

type adminAnnouncementRepo struct {
	data   *Data
	logger *log.Helper
}

// NewAdminAnnouncementRepo 创建公告数据仓库
func NewAdminAnnouncementRepo(data *Data, logger log.Logger) announcementbiz.AnnouncementRepo {
	return &adminAnnouncementRepo{
		data:   data,
		logger: log.NewHelper(logger),
	}
}

// Save 保存公告
func (r *adminAnnouncementRepo) Save(ctx context.Context, announcement *announcementbiz.Announcement) (*announcementbiz.Announcement, error) {
	pinned := getBoolPtrValue(announcement.Pinned)
	if pinned {
		tx, err := r.data.db.Tx(ctx)
		if err != nil {
			return nil, err
		}

		builder := tx.ProxyAnnouncement.Create().
			SetTitle(announcement.Title).
			SetShow(getBoolPtrValue(announcement.Show)).
			SetPinned(false).
			SetPopup(getBoolPtrValue(announcement.Popup))

		if announcement.Content != nil {
			builder.SetContent(*announcement.Content)
		}

		po, err := builder.Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		if _, err = tx.ProxyAnnouncement.Update().SetPinned(false).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		if po, err = tx.ProxyAnnouncement.UpdateOneID(po.ID).SetPinned(true).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		if err = tx.Commit(); err != nil {
			return nil, err
		}

		return r.convertToModel(po), nil
	}

	builder := r.data.db.ProxyAnnouncement.Create().
		SetTitle(announcement.Title).
		SetShow(getBoolPtrValue(announcement.Show)).
		SetPinned(pinned).
		SetPopup(getBoolPtrValue(announcement.Popup))

	if announcement.Content != nil {
		builder.SetContent(*announcement.Content)
	}

	po, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToModel(po), nil
}

// Update 更新公告
func (r *adminAnnouncementRepo) Update(ctx context.Context, announcement *announcementbiz.Announcement) (*announcementbiz.Announcement, error) {
	if announcement.Pinned != nil && *announcement.Pinned {
		tx, err := r.data.db.Tx(ctx)
		if err != nil {
			return nil, err
		}

		if _, err = tx.ProxyAnnouncement.Update().SetPinned(false).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		builder := tx.ProxyAnnouncement.UpdateOneID(announcement.ID).
			SetTitle(announcement.Title).
			SetUpdatedAt(time.Now())

		if announcement.Show != nil {
			builder.SetShow(*announcement.Show)
		}
		builder.SetPinned(true)
		if announcement.Popup != nil {
			builder.SetPopup(*announcement.Popup)
		}

		if announcement.Content != nil {
			builder.SetContent(*announcement.Content)
		} else {
			builder.ClearContent()
		}

		po, err := builder.Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			if ent.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}

		if err = tx.Commit(); err != nil {
			return nil, err
		}

		return r.convertToModel(po), nil
	}

	builder := r.data.db.ProxyAnnouncement.UpdateOneID(announcement.ID).
		SetTitle(announcement.Title).
		SetUpdatedAt(time.Now())

	if announcement.Show != nil {
		builder.SetShow(*announcement.Show)
	}
	if announcement.Pinned != nil {
		builder.SetPinned(*announcement.Pinned)
	}
	if announcement.Popup != nil {
		builder.SetPopup(*announcement.Popup)
	}

	if announcement.Content != nil {
		builder.SetContent(*announcement.Content)
	} else {
		builder.ClearContent()
	}

	po, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return r.convertToModel(po), nil
}

// FindByID 根据ID查找公告
func (r *adminAnnouncementRepo) FindByID(ctx context.Context, id int64) (*announcementbiz.Announcement, error) {
	po, err := r.data.db.ProxyAnnouncement.Query().
		Where(
			proxyannouncement.ID(id),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return r.convertToModel(po), nil
}

// ListAll 获取公告列表
func (r *adminAnnouncementRepo) ListAll(ctx context.Context, page, size int, search string, show, pinned, popup *bool) ([]*announcementbiz.Announcement, int32, error) {
	query := r.data.db.ProxyAnnouncement.Query()

	// 根据条件过滤
	if search != "" {
		query = query.Where(proxyannouncement.TitleContains(search))
	}
	if show != nil {
		query = query.Where(proxyannouncement.Show(*show))
	}
	if pinned != nil {
		query = query.Where(proxyannouncement.Pinned(*pinned))
	}
	if popup != nil {
		query = query.Where(proxyannouncement.Popup(*popup))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询，按照置顶和创建时间排序
	pos, err := query.
		Order(func(s *sql.Selector) {
			s.OrderBy(
				sql.Desc(proxyannouncement.FieldPinned),    // 置顶的在前
				sql.Desc(proxyannouncement.FieldCreatedAt), // 然后按创建时间倒序
			)
		}).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 转换为业务模型
	announcements := make([]*announcementbiz.Announcement, 0, len(pos))
	for _, po := range pos {
		announcements = append(announcements, r.convertToModel(po))
	}

	return announcements, int32(total), nil
}

// Delete 删除公告
func (r *adminAnnouncementRepo) Delete(ctx context.Context, id int64) error {
	deleted, err := r.data.db.ProxyAnnouncement.Delete().
		Where(
			proxyannouncement.ID(id),
		).
		Exec(ctx)
	if err != nil {
		return err
	}

	if deleted == 0 {
		return &ent.NotFoundError{}
	}

	return nil
}

// convertToModel 将ent实体转换为业务模型
func (r *adminAnnouncementRepo) convertToModel(po *ent.ProxyAnnouncement) *announcementbiz.Announcement {
	model := &announcementbiz.Announcement{
		ID:        po.ID,
		Title:     po.Title,
		Show:      &po.Show,
		Pinned:    &po.Pinned,
		Popup:     &po.Popup,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}

	// 处理可选字段
	if po.Content != "" {
		model.Content = &po.Content
	}

	return model
}

func getBoolPtrValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}
