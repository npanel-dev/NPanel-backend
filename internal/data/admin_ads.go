package data

import (
	"context"
	"time"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyads"
	adsbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/ads"

	"github.com/go-kratos/kratos/v2/log"
)

var _ adsbiz.AdsRepo = (*adsRepo)(nil)

type adsRepo struct {
	data *Data
	log  *log.Helper
}

// NewAdsRepo 创建广告数据仓库
func NewAdsRepo(data *Data, logger log.Logger) adsbiz.AdsRepo {
	return &adsRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// GetAdsListByPage 分页获取广告列表
func (r *adsRepo) GetAdsListByPage(ctx context.Context, page, size int, filter adsbiz.AdsFilter) (total int32, list []*adsbiz.Ads, err error) {
	query := r.data.db.ProxyAds.Query()

	// 应用搜索过滤
	if filter.Search != "" {
		query = query.Where(proxyads.Or(
			proxyads.TitleContainsFold(filter.Search),
			proxyads.ContentContainsFold(filter.Search),
		))
	}

	// 应用状态过滤
	if filter.Status != nil {
		query = query.Where(proxyads.Status(int(*filter.Status)))
	}

	// 获取总数
	count, err := query.Count(ctx)
	if err != nil {
		r.log.WithContext(ctx).Errorf("Failed to count ads: %v", err)
		return 0, nil, err
	}
	total = int32(count)

	// 分页查询
	offset := (page - 1) * size
	entAds, err := query.
		Offset(offset).
		Limit(size).
		All(ctx)
	if err != nil {
		r.log.WithContext(ctx).Errorf("Failed to query ads: %v", err)
		return 0, nil, err
	}

	// 转换为业务对象
	list = make([]*adsbiz.Ads, len(entAds))
	for i, entAd := range entAds {
		list[i] = r.entAdsToBiz(entAd)
	}

	return total, list, nil
}

// GetAdsByID 根据ID获取广告详情
func (r *adsRepo) GetAdsByID(ctx context.Context, id int64) (*adsbiz.Ads, error) {
	entAd, err := r.data.db.ProxyAds.Query().
		Where(
			proxyads.ID(id),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // 返回nil表示未找到
		}
		r.log.WithContext(ctx).Errorf("Failed to get ads by id %d: %v", id, err)
		return nil, err
	}

	return r.entAdsToBiz(entAd), nil
}

// CreateAds 创建广告
func (r *adsRepo) CreateAds(ctx context.Context, ads *adsbiz.Ads) (*adsbiz.Ads, error) {
	create := r.data.db.ProxyAds.Create().
		SetTitle(ads.Title).
		SetType(ads.Type).
		SetContent(ads.Content).
		SetDescription(ads.Description).
		SetTargetURL(ads.TargetURL).
		SetStatus(int(ads.Status))

	// 设置时间字段（如果不为零值）
	if !ads.StartTime.IsZero() {
		create = create.SetStartTime(ads.StartTime)
	}
	if !ads.EndTime.IsZero() {
		create = create.SetEndTime(ads.EndTime)
	}

	entAd, err := create.Save(ctx)
	if err != nil {
		r.log.WithContext(ctx).Errorf("Failed to create ads: %v", err)
		return nil, err
	}

	return r.entAdsToBiz(entAd), nil
}

// UpdateAds 更新广告
func (r *adsRepo) UpdateAds(ctx context.Context, ads *adsbiz.Ads) (*adsbiz.Ads, error) {
	update := r.data.db.ProxyAds.UpdateOneID(ads.ID).
		SetTitle(ads.Title).
		SetType(ads.Type).
		SetContent(ads.Content).
		SetDescription(ads.Description).
		SetTargetURL(ads.TargetURL).
		SetStatus(int(ads.Status)).
		SetUpdatedAt(time.Now())

	// 设置时间字段（如果不为零值）
	if !ads.StartTime.IsZero() {
		update = update.SetStartTime(ads.StartTime)
	}
	if !ads.EndTime.IsZero() {
		update = update.SetEndTime(ads.EndTime)
	}

	entAd, err := update.Save(ctx)
	if err != nil {
		r.log.WithContext(ctx).Errorf("Failed to update ads: %v", err)
		return nil, err
	}

	return r.entAdsToBiz(entAd), nil
}

// DeleteAds 删除广告
func (r *adsRepo) DeleteAds(ctx context.Context, id int64) error {
	err := r.data.db.ProxyAds.DeleteOneID(id).
		Exec(ctx)
	if err != nil {
		r.log.WithContext(ctx).Errorf("Failed to delete ads: %v", err)
		return err
	}

	return nil
}

// entAdsToBiz 将Ent对象转换为业务对象
func (r *adsRepo) entAdsToBiz(entAd *ent.ProxyAds) *adsbiz.Ads {
	if entAd == nil {
		return nil
	}

	return &adsbiz.Ads{
		ID:          entAd.ID,
		Title:       entAd.Title,
		Type:        entAd.Type,
		Content:     entAd.Content,
		Description: entAd.Description,
		TargetURL:   entAd.TargetURL,
		StartTime:   entAd.StartTime,
		EndTime:     entAd.EndTime,
		Status:      int8(entAd.Status),
		CreatedAt:   entAd.CreatedAt,
		UpdatedAt:   entAd.UpdatedAt,
	}
}
