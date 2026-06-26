package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyserver"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribeapplication"
	applicationbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/application"
	publicsubscriptionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/subscription"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

type subscribeApplicationRepo struct {
	data *Data
	log  *log.Helper
}

// NewSubscribeApplicationRepo 创建订阅应用配置仓库
func NewSubscribeApplicationRepo(data *Data, logger log.Logger) applicationbiz.SubscribeApplicationRepo {
	return &subscribeApplicationRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// Create 创建订阅应用配置
func (r *subscribeApplicationRepo) Create(ctx context.Context, app *applicationbiz.SubscribeApplication) (*applicationbiz.SubscribeApplication, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	if !app.IsDefault {
		hasDefault, err := tx.ProxySubscribeApplication.Query().
			Where(proxysubscribeapplication.IsDefault(true)).
			Exist(ctx)
		if err != nil {
			return nil, rollback(tx, err)
		}
		if !hasDefault {
			app.IsDefault = true
		}
	}
	if app.IsDefault {
		if _, err := tx.ProxySubscribeApplication.Update().
			SetIsDefault(false).
			Save(ctx); err != nil {
			return nil, rollback(tx, err)
		}
	}

	po, err := tx.ProxySubscribeApplication.
		Create().
		SetName(app.Name).
		SetNillableIcon(app.Icon).
		SetNillableDescription(app.Description).
		SetScheme(app.Scheme).
		SetUserAgent(app.UserAgent).
		SetIsDefault(app.IsDefault).
		SetNillableSubscribeTemplate(app.SubscribeTemplate).
		SetOutputFormat(app.OutputFormat).
		SetDownloadLink(app.DownloadLink).
		Save(ctx)

	if err != nil {
		return nil, rollback(tx, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.convertToModel(po), nil
}

// Update 更新订阅应用配置
func (r *subscribeApplicationRepo) Update(ctx context.Context, app *applicationbiz.SubscribeApplication) (*applicationbiz.SubscribeApplication, error) {
	// 先查询确保应用配置存在
	existing, err := r.data.db.ProxySubscribeApplication.
		Query().
		Where(
			proxysubscribeapplication.ID(app.ID),
		).
		Only(ctx)

	if err != nil {
		return nil, err
	}
	if existing.IsDefault && !app.IsDefault {
		defaultCount, err := r.data.db.ProxySubscribeApplication.Query().
			Where(proxysubscribeapplication.IsDefault(true)).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		if defaultCount <= 1 {
			app.IsDefault = true
		}
	}

	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	if app.IsDefault {
		if _, err := tx.ProxySubscribeApplication.Update().
			Where(proxysubscribeapplication.IDNEQ(app.ID)).
			SetIsDefault(false).
			Save(ctx); err != nil {
			return nil, rollback(tx, err)
		}
	}

	// 构建更新操作，所有字段都直接设置（包括可选字段）
	updateBuilder := tx.ProxySubscribeApplication.
		UpdateOneID(app.ID).
		SetName(app.Name).
		SetScheme(app.Scheme).
		SetUserAgent(app.UserAgent).
		SetIsDefault(app.IsDefault).
		SetOutputFormat(app.OutputFormat).
		SetDownloadLink(app.DownloadLink).
		SetNillableIcon(app.Icon).
		SetNillableDescription(app.Description).
		SetNillableSubscribeTemplate(app.SubscribeTemplate)

	po, err := updateBuilder.Save(ctx)
	if err != nil {
		return nil, rollback(tx, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.convertToModel(po), nil
}

// FindByID 根据ID查找订阅应用配置
func (r *subscribeApplicationRepo) FindByID(ctx context.Context, id int64) (*applicationbiz.SubscribeApplication, error) {
	po, err := r.data.db.ProxySubscribeApplication.
		Query().
		Where(
			proxysubscribeapplication.ID(id),
		).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return r.convertToModel(po), nil
}

// List 查询订阅应用配置列表
func (r *subscribeApplicationRepo) List(ctx context.Context, page, size int) ([]*applicationbiz.SubscribeApplication, int32, error) {
	query := r.data.db.ProxySubscribeApplication.Query()

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	pos, err := query.
		Order(ent.Desc(proxysubscribeapplication.FieldCreatedAt)).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	apps := make([]*applicationbiz.SubscribeApplication, 0, len(pos))
	for _, po := range pos {
		apps = append(apps, r.convertToModel(po))
	}

	return apps, int32(total), nil
}

// GetPreviewNodes returns all enabled nodes converted for template preview rendering.
func (r *subscribeApplicationRepo) GetPreviewNodes(ctx context.Context) ([]*publicsubscriptionbiz.NodeInfo, error) {
	nodes, err := r.data.db.ProxyNode.Query().
		Where(proxynode.Enabled(true)).
		Order(ent.Asc(proxynode.FieldSort), ent.Asc(proxynode.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*publicsubscriptionbiz.NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		server, err := r.data.db.ProxyServer.Query().
			Where(proxyserver.IDEQ(node.ServerID)).
			Only(ctx)
		if err != nil {
			r.log.Warnf("preview skip node %d: query server failed: %v", node.ID, err)
			continue
		}

		protocols, err := servermodel.UnmarshalProtocols(server.Protocol)
		if err != nil {
			r.log.Warnf("preview skip server %d: unmarshal protocols failed: %v", server.ID, err)
			continue
		}

		matched, _, _ := matchNodeProtocolConfig(protocols, node.Protocol, node.Port)
		if matched == nil {
			r.log.Warnf("preview skip node %d: no protocol match for %s", node.ID, node.Protocol)
			continue
		}

		nodeInfo := &publicsubscriptionbiz.NodeInfo{
			ID:          node.ID,
			Sort:        int(node.Sort),
			Name:        node.Name,
			Server:      node.Address,
			Port:        node.Port,
			Type:        node.Protocol,
			Tags:        tool.StringToStringSlice(node.Tags),
			NodeGroupID: 0,

			Security:                           matched.Security,
			SNI:                                matched.SNI,
			AllowInsecure:                      matched.AllowInsecure,
			Fingerprint:                        matched.Fingerprint,
			RealityServerAddr:                  matched.RealityServerAddr,
			RealityServerPort:                  int(matched.RealityServerPort),
			RealityPrivateKey:                  matched.RealityPrivateKey,
			RealityPublicKey:                   matched.RealityPublicKey,
			RealityShortId:                     matched.RealityShortId,
			Transport:                          matched.Transport,
			Host:                               matched.Host,
			Path:                               matched.Path,
			ServiceName:                        matched.ServiceName,
			Mc1Mode:                            matched.Mc1Mode,
			Mc1CidrSegments:                    matched.Mc1CidrSegments,
			MundoUsername:                      matched.MundoUsername,
			MundoCertificateFingerprint:        matched.MundoCertificateFingerprint,
			MundoFakeTitle:                     matched.MundoFakeTitle,
			MundoFakeMessage:                   matched.MundoFakeMessage,
			MundoAcceptProxyProtocol:           matched.MundoAcceptProxyProtocol,
			MundoUseTLSCertificate:             matched.MundoUseTLSCertificate,
			Method:                             matched.Cipher,
			ServerKey:                          matched.ServerKey,
			Flow:                               matched.Flow,
			HopPorts:                           matched.HopPorts,
			HopInterval:                        int(matched.HopInterval),
			ObfsPassword:                       matched.ObfsPassword,
			UpMbps:                             int(matched.UpMbps),
			DownMbps:                           int(matched.DownMbps),
			DisableSNI:                         matched.DisableSNI,
			ReduceRtt:                          matched.ReduceRtt,
			UDPRelayMode:                       matched.UDPRelayMode,
			CongestionController:               matched.CongestionController,
			PaddingScheme:                      matched.PaddingScheme,
			Multiplex:                          matched.Multiplex,
			XhttpMode:                          matched.XhttpMode,
			XhttpExtra:                         matched.XhttpExtra,
			Encryption:                         matched.Encryption,
			EncryptionMode:                     matched.EncryptionMode,
			EncryptionRtt:                      matched.EncryptionRtt,
			EncryptionTicket:                   matched.EncryptionTicket,
			EncryptionServerPadding:            matched.EncryptionServerPadding,
			EncryptionPrivateKey:               matched.EncryptionPrivateKey,
			EncryptionClientPadding:            matched.EncryptionClientPadding,
			EncryptionPassword:                 matched.EncryptionPassword,
			Ratio:                              matched.Ratio,
			CertMode:                           matched.CertMode,
			CertDNSProvider:                    matched.CertDNSProvider,
			CertDNSEnv:                         matched.CertDNSEnv,
			SimnetPsk:                          matched.SimnetPsk,
			SimnetKeyID:                        int(matched.SimnetKeyID),
			SimnetTicketID:                     matched.SimnetTicketID,
			SimnetPath:                         matched.SimnetPath,
			SimnetCarrier:                      matched.SimnetCarrier,
			SimnetAfEnabled:                    matched.SimnetAfEnabled,
			SimnetAfPathMode:                   matched.SimnetAfPathMode,
			SimnetAfPathPrefix:                 matched.SimnetAfPathPrefix,
			SimnetAfPathSuffix:                 matched.SimnetAfPathSuffix,
			SimnetAfMagicMode:                  matched.SimnetAfMagicMode,
			SimnetAfResponseJitterMs:           int(matched.SimnetAfResponseJitterMs),
			SimnetAfHandshakePolymorphism:      matched.SimnetAfHandshakePolymorphism,
			SimnetAfSettingsJitter:             matched.SimnetAfSettingsJitter,
			SimnetAfFakeHeaderInjection:        matched.SimnetAfFakeHeaderInjection,
			SimnetClientMaxConcurrentStreams:   int(matched.SimnetClientMaxConcurrentStreams),
			SimnetClientMaxStreamsPerSession:   int(matched.SimnetClientMaxStreamsPerSession),
			SimnetClientSessionIdleTimeoutSecs: int(matched.SimnetClientSessionIdleTimeoutSecs),
			SimnetClientMaxUDPSessions:         int(matched.SimnetClientMaxUDPSessions),
		}
		nodeInfo.NormalizeSimnet()
		result = append(result, nodeInfo)
	}

	return result, nil
}

// Delete 删除订阅应用配置
func (r *subscribeApplicationRepo) Delete(ctx context.Context, id int64) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}

	deletingDefault := false
	existing, err := tx.ProxySubscribeApplication.Query().
		Where(proxysubscribeapplication.ID(id)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return rollback(tx, err)
		}
	} else {
		deletingDefault = existing.IsDefault
	}

	_, err = tx.ProxySubscribeApplication.
		Delete().
		Where(
			proxysubscribeapplication.ID(id),
		).
		Exec(ctx)
	if err != nil {
		return rollback(tx, err)
	}

	if deletingDefault {
		hasDefault, err := tx.ProxySubscribeApplication.Query().
			Where(proxysubscribeapplication.IsDefault(true)).
			Exist(ctx)
		if err != nil {
			return rollback(tx, err)
		}
		if !hasDefault {
			nextDefault, err := tx.ProxySubscribeApplication.Query().
				Order(ent.Asc(proxysubscribeapplication.FieldID)).
				First(ctx)
			if err != nil && !ent.IsNotFound(err) {
				return rollback(tx, err)
			}
			if nextDefault != nil {
				if _, err := tx.ProxySubscribeApplication.Update().
					Where(proxysubscribeapplication.ID(nextDefault.ID)).
					SetIsDefault(true).
					Save(ctx); err != nil {
					return rollback(tx, err)
				}
			}
		}
	}

	return tx.Commit()
}

// convertToModel 转换为业务模型
func (r *subscribeApplicationRepo) convertToModel(po *ent.ProxySubscribeApplication) *applicationbiz.SubscribeApplication {
	if po == nil {
		return nil
	}

	return &applicationbiz.SubscribeApplication{
		ID:                po.ID,
		Name:              po.Name,
		Icon:              po.Icon,
		Description:       po.Description,
		Scheme:            po.Scheme,
		UserAgent:         po.UserAgent,
		IsDefault:         po.IsDefault,
		SubscribeTemplate: po.SubscribeTemplate,
		OutputFormat:      po.OutputFormat,
		DownloadLink:      po.DownloadLink,
		CreatedAt:         po.CreatedAt,
		UpdatedAt:         po.UpdatedAt,
	}
}
