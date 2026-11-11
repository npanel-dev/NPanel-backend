package application

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/npanel-dev/NPanel-backend/api/admin/application/v1"
	applicationbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/application"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

// SubscribeApplicationService 订阅应用配置服务
type SubscribeApplicationService struct {
	v1.UnimplementedSubscribeApplicationServiceServer

	uc     *applicationbiz.SubscribeApplicationUsecase
	logger *log.Helper
}

// NewSubscribeApplicationService 创建订阅应用配置服务
func NewSubscribeApplicationService(uc *applicationbiz.SubscribeApplicationUsecase, logger log.Logger) *SubscribeApplicationService {
	return &SubscribeApplicationService{
		uc:     uc,
		logger: log.NewHelper(logger),
	}
}

// CreateSubscribeApplication 创建订阅应用配置
func (s *SubscribeApplicationService) CreateSubscribeApplication(ctx context.Context, req *v1.CreateSubscribeApplicationRequest) (*v1.SubscribeApplicationReply, error) {

	// 序列化下载链接
	downloadLinkJSON, err := convertDownloadLinkToJSON(req.DownloadLink)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("marshal download link error: %v", err)
		return nil, err
	}

	app := &applicationbiz.SubscribeApplication{
		Name:              req.Name,
		Icon:              &req.Icon,
		Description:       &req.Description,
		Scheme:            req.Scheme,
		UserAgent:         req.UserAgent,
		Enabled:           req.Enabled == nil || *req.Enabled,
		IsDefault:         req.IsDefault,
		SubscribeTemplate: &req.SubscribeTemplate,
		OutputFormat:      req.OutputFormat,
		DownloadLink:      downloadLinkJSON,
	}

	result, err := s.uc.CreateSubscribeApplication(ctx, app)
	if err != nil {
		return nil, err
	}

	return &v1.SubscribeApplicationReply{
		Code:    int32(responsecode.AdminCreateSubscribeApplicationSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminCreateSubscribeApplicationSuccess],
		Data: &v1.SubscribeApplicationData{
			Application: s.convertToProto(result, req.DownloadLink),
		},
	}, nil
}

// UpdateSubscribeApplication 更新订阅应用配置
func (s *SubscribeApplicationService) UpdateSubscribeApplication(ctx context.Context, req *v1.UpdateSubscribeApplicationRequest) (*v1.SubscribeApplicationReply, error) {
	// 序列化下载链接
	downloadLinkJSON, err := convertDownloadLinkToJSON(req.DownloadLink)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("marshal download link error: %v", err)
		return nil, err
	}

	// 构建更新参数，注意：即使字段为空字符串也需要更新
	icon := req.Icon
	description := req.Description
	template := req.SubscribeTemplate
	if req.Id <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	app := &applicationbiz.SubscribeApplication{
		ID:                req.Id,
		Name:              req.Name,
		Icon:              &icon,
		Description:       &description,
		Scheme:            req.Scheme,
		UserAgent:         req.UserAgent,
		Enabled:           req.Enabled == nil || *req.Enabled,
		IsDefault:         req.IsDefault,
		SubscribeTemplate: &template,
		OutputFormat:      req.OutputFormat,
		DownloadLink:      downloadLinkJSON,
	}

	result, err := s.uc.UpdateSubscribeApplication(ctx, app)
	if err != nil {
		return nil, err
	}

	return &v1.SubscribeApplicationReply{
		Code:    int32(responsecode.AdminUpdateSubscribeApplicationSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminUpdateSubscribeApplicationSuccess],
		Data: &v1.SubscribeApplicationData{
			Application: s.convertToProto(result, req.DownloadLink),
		},
	}, nil
}

// DeleteSubscribeApplication 删除订阅应用配置
func (s *SubscribeApplicationService) DeleteSubscribeApplication(ctx context.Context, req *v1.DeleteSubscribeApplicationRequest) (*v1.DeleteSubscribeApplicationReply, error) {
	if req.Id <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	err := s.uc.DeleteSubscribeApplication(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &v1.DeleteSubscribeApplicationReply{
		Code:    int32(responsecode.AdminDeleteSubscribeApplicationSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminDeleteSubscribeApplicationSuccess],
		Data: &v1.DeleteSubscribeApplicationData{
			Success: true,
		},
	}, nil
}

// GetSubscribeApplicationList 获取订阅应用配置列表
func (s *SubscribeApplicationService) GetSubscribeApplicationList(ctx context.Context, req *v1.GetSubscribeApplicationListRequest) (*v1.GetSubscribeApplicationListReply, error) {
	list, total, err := s.uc.GetSubscribeApplicationList(ctx, int(req.Page), int(req.Size))
	if err != nil {
		return nil, err
	}

	// 转换为protobuf格式
	items := make([]*v1.SubscribeApplication, 0, len(list))
	for _, app := range list {
		// 反序列化下载链接
		downloadLink := &v1.DownloadLink{}
		if app.DownloadLink != "" {
			link := &applicationbiz.DownloadLink{}
			if err := link.Unmarshal(app.DownloadLink); err == nil {
				downloadLink.Windows = link.Windows
				downloadLink.Macos = link.MacOS
				downloadLink.Linux = link.Linux
				downloadLink.Android = link.Android
				downloadLink.Ios = link.IOS
			}
		}

		items = append(items, s.convertToProto(app, downloadLink))
	}

	return &v1.GetSubscribeApplicationListReply{
		Code:    int32(responsecode.AdminGetSubscribeApplicationListSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminGetSubscribeApplicationListSuccess],
		Data: &v1.GetSubscribeApplicationListData{
			List:  items,
			Total: total,
		},
	}, nil
}

// PreviewSubscribeTemplate 预览订阅模板
func (s *SubscribeApplicationService) PreviewSubscribeTemplate(ctx context.Context, req *v1.PreviewSubscribeTemplateRequest) (*v1.PreviewSubscribeTemplateReply, error) {
	if req.Id <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	template, err := s.uc.PreviewSubscribeTemplate(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &v1.PreviewSubscribeTemplateReply{
		Code:    int32(responsecode.AdminPreviewSubscribeTemplateSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminPreviewSubscribeTemplateSuccess],
		Data: &v1.PreviewSubscribeTemplateData{
			Template: template,
		},
	}, nil
}

// convertToProto 将业务模型转换为protobuf消息
func (s *SubscribeApplicationService) convertToProto(app *applicationbiz.SubscribeApplication, downloadLink *v1.DownloadLink) *v1.SubscribeApplication {
	result := &v1.SubscribeApplication{
		Id:           app.ID,
		Name:         app.Name,
		Scheme:       app.Scheme,
		UserAgent:    app.UserAgent,
		Enabled:      &app.Enabled,
		IsDefault:    app.IsDefault,
		OutputFormat: app.OutputFormat,
		DownloadLink: downloadLink,
		CreatedAt:    timestamppb.New(app.CreatedAt),
		UpdatedAt:    timestamppb.New(app.UpdatedAt),
	}

	// 处理可选字段
	if app.Icon != nil {
		result.Icon = *app.Icon
	}
	if app.Description != nil {
		result.Description = *app.Description
	}
	if app.SubscribeTemplate != nil {
		result.SubscribeTemplate = *app.SubscribeTemplate
	}

	return result
}

// convertDownloadLinkToJSON 将下载链接转换为JSON
func convertDownloadLinkToJSON(link *v1.DownloadLink) (string, error) {
	if link == nil {
		return "", nil
	}

	bizLink := &applicationbiz.DownloadLink{
		Windows: link.Windows,
		MacOS:   link.Macos,
		Linux:   link.Linux,
		Android: link.Android,
		IOS:     link.Ios,
	}

	return bizLink.Marshal()
}
