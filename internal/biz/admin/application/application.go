package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	publicsubscriptionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/subscription"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

// DownloadLink 下载链接
type DownloadLink struct {
	Windows string `json:"windows"`
	Mac     string `json:"mac,omitempty"`
	MacOS   string `json:"macos"`
	Linux   string `json:"linux"`
	Android string `json:"android"`
	IOS     string `json:"ios"`
}

// Marshal 序列化为JSON
func (d *DownloadLink) Marshal() (string, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Unmarshal 从JSON反序列化
func (d *DownloadLink) Unmarshal(data string) error {
	if err := json.Unmarshal([]byte(data), d); err != nil {
		return err
	}
	if d.MacOS == "" {
		d.MacOS = d.Mac
	}
	return nil
}

// SubscribeApplication 订阅应用配置
type SubscribeApplication struct {
	ID                int64
	Name              string
	Icon              *string
	Description       *string
	Scheme            string
	UserAgent         string
	Enabled           bool
	IsDefault         bool
	SubscribeTemplate *string
	OutputFormat      string
	DownloadLink      string // JSON格式存储
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// SubscribeApplicationRepo 订阅应用配置仓储接口
type SubscribeApplicationRepo interface {
	Create(ctx context.Context, app *SubscribeApplication) (*SubscribeApplication, error)
	Update(ctx context.Context, app *SubscribeApplication) (*SubscribeApplication, error)
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*SubscribeApplication, error)
	List(ctx context.Context, page, size int) ([]*SubscribeApplication, int32, error)
	GetPreviewNodes(ctx context.Context) ([]*publicsubscriptionbiz.NodeInfo, error)
}

// SubscribeApplicationUsecase 订阅应用配置用例
type SubscribeApplicationUsecase struct {
	repo   SubscribeApplicationRepo
	logger *log.Helper
}

// NewSubscribeApplicationUsecase 创建订阅应用配置用例
func NewSubscribeApplicationUsecase(repo SubscribeApplicationRepo, logger log.Logger) *SubscribeApplicationUsecase {
	return &SubscribeApplicationUsecase{
		repo:   repo,
		logger: log.NewHelper(logger),
	}
}

// CreateSubscribeApplication 创建订阅应用配置
func (uc *SubscribeApplicationUsecase) CreateSubscribeApplication(ctx context.Context, app *SubscribeApplication) (*SubscribeApplication, error) {
	result, err := uc.repo.Create(ctx, app)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("create subscribe application error: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
	}
	return result, nil
}

// UpdateSubscribeApplication 更新订阅应用配置
func (uc *SubscribeApplicationUsecase) UpdateSubscribeApplication(ctx context.Context, app *SubscribeApplication) (*SubscribeApplication, error) {
	// 先查询是否存在
	existing, err := uc.repo.FindByID(ctx, app.ID)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("find subscribe application error: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if existing == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrSubscribeApplicationNotFound)
	}

	result, err := uc.repo.Update(ctx, app)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("update subscribe application error: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}
	return result, nil
}

// DeleteSubscribeApplication 删除订阅应用配置
func (uc *SubscribeApplicationUsecase) DeleteSubscribeApplication(ctx context.Context, id int64) error {
	existing, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("find subscribe application error: %v", err)
		return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if existing == nil {
		return responsecode.NewKratosError(responsecode.ErrSubscribeApplicationNotFound)
	}
	err = uc.repo.Delete(ctx, id)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("delete subscribe application error: %v", err)
		return responsecode.NewKratosError(responsecode.ErrDatabaseDelete)
	}
	return nil
}

// GetSubscribeApplicationList 获取订阅应用配置列表
func (uc *SubscribeApplicationUsecase) GetSubscribeApplicationList(ctx context.Context, page, size int) ([]*SubscribeApplication, int32, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	if size > 100 {
		size = 100
	}

	list, total, err := uc.repo.List(ctx, page, size)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("get subscribe application list error: %v", err)
		return nil, 0, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	return list, total, nil
}

// PreviewSubscribeTemplate 预览订阅模板
func (uc *SubscribeApplicationUsecase) PreviewSubscribeTemplate(ctx context.Context, id int64) (string, error) {
	app, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("find subscribe application error: %v", err)
		return "", responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if app == nil {
		return "", responsecode.NewKratosError(responsecode.ErrSubscribeApplicationNotFound)
	}

	nodes, err := uc.repo.GetPreviewNodes(ctx)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("get preview nodes error: %v", err)
		return "", responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	templateStr := ""
	if app.SubscribeTemplate != nil {
		templateStr = *app.SubscribeTemplate
	}

	userSubscribe := &publicsubscriptionbiz.UserSubscribe{
		UUID:          "test-password",
		SubscribeName: "Test Subscribe",
	}
	userInfo := publicsubscriptionbiz.UserInfo{
		Password:     "test-password",
		ExpiredAt:    time.Now().AddDate(1, 0, 0),
		Download:     0,
		Upload:       0,
		Traffic:      1000,
		SubscribeURL: "https://example.com/subscribe",
	}

	rendered, err := publicsubscriptionbiz.RenderTemplate(
		templateStr,
		app.OutputFormat,
		"NPanel",
		"Test Subscribe",
		nodes,
		userSubscribe,
		userInfo,
		map[string]string{},
	)
	if err != nil {
		uc.logger.WithContext(ctx).Errorf("render subscribe template error: %v", err)
		return "", responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return string(rendered), nil
}
