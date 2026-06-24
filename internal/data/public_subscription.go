package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyserver"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribeapplication"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	subscriptionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/subscription"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

type publicSubscriptionRepo struct {
	data *Data
	log  *log.Helper
}

// NewPublicSubscriptionRepo 创建Public Subscription数据仓储实例
func NewPublicSubscriptionRepo(data *Data, logger log.Logger) subscriptionbiz.SubscriptionRepo {
	return &publicSubscriptionRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// ValidateTokenAndGetSubscribe 验证token并获取用户订阅信息
func (r *publicSubscriptionRepo) ValidateTokenAndGetSubscribe(ctx context.Context, token string) (*subscriptionbiz.UserSubscribe, error) {
	// 查询用户订阅
	userSub, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.TokenEQ(token),
		).
		Only(ctx)

	if err != nil {
		r.log.Errorf("Failed to query user subscribe by token: %v", err)
		return nil, fmt.Errorf("invalid subscribe token")
	}

	// 注意：按照原项目逻辑，这里不检查订阅状态和过期时间
	// 原项目注释："Ignore expiration check"
	// 过期检查会在 getServers (GetAvailableNodes) 中进行
	// 这样可以返回过期提示节点而不是直接拒绝访问

	// 获取订阅套餐信息
	subscribePlan, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(userSub.SubscribeID),
		).
		Only(ctx)

	if err != nil {
		r.log.Errorf("Failed to query subscribe plan: %v", err)
		return nil, fmt.Errorf("subscribe plan not found")
	}

	// 转换为业务层对象
	result := &subscriptionbiz.UserSubscribe{
		ID:          int64(userSub.ID),
		UserID:      userSub.UserID,
		SubscribeID: userSub.SubscribeID,
		Token:       getStringValue(userSub.Token),
		UUID:        getStringValue(userSub.UUID),
		StartTime:   userSub.StartTime.UnixMilli(),
		Status:      int(getInt8Value(userSub.Status)),
	}

	if userSub.ExpireTime != nil {
		result.ExpireTime = userSub.ExpireTime.UnixMilli()
	}
	if userSub.Traffic != nil {
		result.Traffic = *userSub.Traffic
	}
	if userSub.Download != nil {
		result.Download = *userSub.Download
	}
	if userSub.Upload != nil {
		result.Upload = *userSub.Upload
	}
	if userSub.ExpiredDownload != nil {
		result.ExpiredDownload = *userSub.ExpiredDownload
	}
	if userSub.ExpiredUpload != nil {
		result.ExpiredUpload = *userSub.ExpiredUpload
	}

	// 套餐信息
	result.SubscribeName = subscribePlan.Name
	result.NodeGroups = subscribePlan.NodeGroupIds
	result.NodeTags = tool.StringToStringSlice(subscribePlan.NodeTags)
	result.NodeGroupID = userSub.NodeGroupID

	return result, nil
}

// GetAvailableNodes 获取可用节点列表 - 按照原项目逻辑实现
func (r *publicSubscriptionRepo) GetAvailableNodes(ctx context.Context, userSubscribe *subscriptionbiz.UserSubscribe) ([]*subscriptionbiz.NodeInfo, error) {
	isGroupMode := r.isGroupEnabled(ctx)

	// 1. 过期订阅只允许在开启分组时走“过期节点组”；否则返回默认过期提示节点。
	if r.isSubscriptionExpired(userSubscribe) {
		if isGroupMode {
			return r.createExpiredNodesFromDB(ctx, userSubscribe), nil
		}
		return r.createExpiredNodesDefault(), nil
	}

	// 2. 获取订阅套餐详情
	subscribePlan, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(userSubscribe.SubscribeID),
		).
		Only(ctx)

	if err != nil {
		r.log.Errorf("Failed to query subscribe plan: %v", err)
		return nil, fmt.Errorf("subscribe plan not found")
	}

	var nodes []*ent.ProxyNode

	if isGroupMode {
		// === 分组模式：使用 node_group_id 获取节点 ===
		r.log.Info("Using group mode to get nodes")
		nodes, err = r.getNodesByGroup(ctx, userSubscribe, subscribePlan)
		if err != nil {
			r.log.Errorf("Failed to get nodes by group: %v", err)
			return nil, err
		}
	} else {
		// === 标签模式：使用 node_ids 和 tags 获取节点 ===
		r.log.Info("Using tag mode to get nodes")
		nodes, err = r.getNodesByTag(ctx, subscribePlan)
		if err != nil {
			r.log.Errorf("Failed to get nodes by tag: %v", err)
			return nil, err
		}
	}

	r.log.Infof("Found %d nodes", len(nodes))

	// 4. 转换为NodeInfo并获取服务器信息
	nodeInfos := make([]*subscriptionbiz.NodeInfo, 0, len(nodes))
	effectiveNodeGroupID, _ := resolveSubscriptionNodeGroupID(userSubscribe, subscribePlan)

	for _, node := range nodes {
		// 获取服务器信息
		server, err := r.data.db.ProxyServer.Query().
			Where(proxyserver.IDEQ(node.ServerID)).
			Only(ctx)

		if err != nil {
			r.log.Warnf("Failed to query server for node %d: %v", node.ID, err)
			continue
		}

		protocols, err := servermodel.UnmarshalProtocols(server.Protocol)
		if err != nil {
			r.log.Warnf("Failed to unmarshal server protocols for server %d: %v", server.ID, err)
			continue
		}

		var matched *servermodel.Protocol
		var firstEnabled *servermodel.Protocol
		var firstAvailable *servermodel.Protocol
		for _, protocol := range protocols {
			if protocol == nil {
				continue
			}
			if firstAvailable == nil {
				firstAvailable = protocol
			}
			if protocol.Enable && firstEnabled == nil {
				firstEnabled = protocol
			}
			if strings.EqualFold(strings.TrimSpace(protocol.Type), strings.TrimSpace(node.Protocol)) {
				matched = protocol
				break
			}
		}
		if matched == nil {
			matched = firstEnabled
		}
		if matched == nil {
			matched = firstAvailable
		}
		if matched == nil {
			r.log.Warnf("No protocol config found for node %d with protocol %s", node.ID, node.Protocol)
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(matched.Type), strings.TrimSpace(node.Protocol)) {
			r.log.Warnf("Node %d protocol %s not found in server %d, fallback to protocol %s", node.ID, node.Protocol, server.ID, matched.Type)
		}

		nodeInfo := &subscriptionbiz.NodeInfo{
			ID:          int64(node.ID),
			Sort:        int(node.Sort),
			Name:        node.Name,
			Server:      node.Address,
			Port:        node.Port,
			Type:        matched.Type,
			Tags:        tool.StringToStringSlice(node.Tags),
			NodeGroupID: resolveNodeGroupID(node.NodeGroupIds, effectiveNodeGroupID),

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
			OmniflowCarrier:                    matched.OmniflowCarrier,
			OmniflowPath:                       matched.OmniflowPath,
			OmniflowContentType:                matched.OmniflowContentType,
			OmniflowProfileJson:                matched.OmniflowProfileJson,
			OmniflowCaCertPath:                 matched.OmniflowCaCertPath,
			OmniflowTargetMeta:                 matched.OmniflowTargetMeta,
			OmniflowSpkiPin:                    matched.OmniflowSpkiPin,
			OmniflowAdaptiveTlsEnabled:         matched.OmniflowAdaptiveTlsEnabled,
			OmniflowTlsFingerprint:             matched.OmniflowTlsFingerprint,
			OmniflowSniMode:                    matched.OmniflowSniMode,
			OmniflowPaddingMode:                matched.OmniflowPaddingMode,
			OmniflowAfEnabled:                  matched.OmniflowAfEnabled,
			OmniflowAfPathMode:                 matched.OmniflowAfPathMode,
			OmniflowAfPathPrefix:               matched.OmniflowAfPathPrefix,
			OmniflowAfPathSuffix:               matched.OmniflowAfPathSuffix,
			OmniflowAfPathRotationSecs:         int(matched.OmniflowAfPathRotationSecs),
			OmniflowAfPathSkewSlots:            int(matched.OmniflowAfPathSkewSlots),
		}

		nodeInfo.NormalizeSimnet()
		nodeInfos = append(nodeInfos, nodeInfo)
	}

	return nodeInfos, nil
}

// isSubscriptionExpired 检查订阅是否过期
func (r *publicSubscriptionRepo) isSubscriptionExpired(userSubscribe *subscriptionbiz.UserSubscribe) bool {
	if userSubscribe.ExpireTime == 0 {
		return false
	}

	expireTime := time.UnixMilli(userSubscribe.ExpireTime)
	return expireTime.Before(time.Now())
}

// createExpiredNodes 创建过期提示节点 - 按照原项目逻辑返回两个节点
func (r *publicSubscriptionRepo) createExpiredNodes() []*subscriptionbiz.NodeInfo {
	host := r.getFirstHostLine()

	return []*subscriptionbiz.NodeInfo{
		{
			Name:   "Subscribe Expired",
			Server: "127.0.0.1",
			Port:   18080,
			Type:   "shadowsocks",
			Tags:   []string{},
			Sort:   1,
			Method: "aes-256-gcm",
		},
		{
			Name:   host,
			Server: "127.0.0.1",
			Port:   18080,
			Type:   "shadowsocks",
			Tags:   []string{},
			Sort:   2,
			Method: "aes-256-gcm",
		},
	}
}

// getFirstHostLine 获取配置中 Host 的第一行（用于过期提示节点）
func (r *publicSubscriptionRepo) getFirstHostLine() string {
	host := ""
	if siteValues, err := loadSystemConfigMap(context.Background(), r.data.db, "site"); err == nil {
		if value, ok := systemConfigLookup(siteValues, "Host", "host"); ok {
			host = value
		}
	}
	if host == "" && r.data.conf != nil && r.data.conf.Site != nil {
		host = r.data.conf.Site.Host
	}
	if host != "" {
		lines := strings.Split(host, "\n")
		if len(lines) > 0 {
			return lines[0]
		}
		return host
	}
	return "example.com" // 默认值
}

// isGroupEnabled 判断分组功能是否启用
func (r *publicSubscriptionRepo) isGroupEnabled(ctx context.Context) bool {
	var value string
	err := r.data.db.ProxySystem.Query().
		Where(func(s *sql.Selector) {
			s.Where(sql.EQ("category", "group")).
				Where(sql.EQ("key", "enabled"))
		}).
		Select("value").
		Scan(ctx, &value)

	if err != nil {
		r.log.Debugf("Check group enabled failed: %v", err)
		return false
	}
	return value == "true" || value == "1"
}

// filterNodesByGroup 按分组过滤节点
// node_group_id 为 0（NULL）= 公共节点，所有人可见
// node_group_id 与 user_group_id 匹配 = 专属节点，只有该组用户可见
func (r *publicSubscriptionRepo) filterNodesByGroup(nodes []*subscriptionbiz.NodeInfo, userGroupId int64) []*subscriptionbiz.NodeInfo {
	var result []*subscriptionbiz.NodeInfo

	for _, n := range nodes {
		// node_group_id 为 0（NULL）= 公共节点，所有人可见
		if n.NodeGroupID == 0 {
			result = append(result, n)
			continue
		}

		// node_group_id 与 user_group_id 匹配 = 专属节点
		if n.NodeGroupID == userGroupId {
			result = append(result, n)
		}
	}

	return result
}

// GetSubscribeDomain 获取订阅域名配置（用于生成订阅URL）
func (r *publicSubscriptionRepo) GetSubscribeDomain(ctx context.Context) string {
	if subscribeValues, err := loadSystemConfigMap(ctx, r.data.db, "subscribe"); err == nil {
		if value, ok := systemConfigLookup(subscribeValues, "SubscribeDomain", "subscribe_domain"); ok {
			return value
		}
	}
	if r.data.conf != nil && r.data.conf.Subscribe != nil {
		return r.data.conf.Subscribe.SubscribeDomain
	}
	return ""
}

func (r *publicSubscriptionRepo) GetSubscribePath(ctx context.Context) string {
	if subscribeValues, err := loadSystemConfigMap(ctx, r.data.db, "subscribe"); err == nil {
		if value, ok := systemConfigLookup(subscribeValues, "SubscribePath", "subscribe_path"); ok {
			return value
		}
	}
	if r.data.conf != nil && r.data.conf.Subscribe != nil {
		return r.data.conf.Subscribe.SubscribePath
	}
	return "/api/subscribe"
}

// GetSiteName 获取站点名称
func (r *publicSubscriptionRepo) GetSiteName(ctx context.Context) string {
	if siteValues, err := loadSystemConfigMap(ctx, r.data.db, "site"); err == nil {
		if value, ok := systemConfigLookup(siteValues, "SiteName", "site_name"); ok {
			return value
		}
	}
	if r.data.conf != nil && r.data.conf.Site != nil {
		return r.data.conf.Site.SiteName
	}
	return "npanel"
}

func (r *publicSubscriptionRepo) GetSubscribeRuntimeConfig(ctx context.Context) (*subscriptionbiz.SubscribeRuntimeConfig, error) {
	result := &subscriptionbiz.SubscribeRuntimeConfig{}

	subscribeValues, err := loadSystemConfigMap(ctx, r.data.db, "subscribe")
	if err == nil {
		if value, ok := systemConfigLookup(subscribeValues, "PanDomain", "pan_domain"); ok {
			result.PanDomain = parseSystemBool(value)
		} else if r.data.conf != nil && r.data.conf.Subscribe != nil {
			result.PanDomain = r.data.conf.Subscribe.PanDomain
		}
		if value, ok := systemConfigLookup(subscribeValues, "UserAgentLimit", "user_agent_limit"); ok {
			result.UserAgentLimit = parseSystemBool(value)
		} else if r.data.conf != nil && r.data.conf.Subscribe != nil {
			result.UserAgentLimit = r.data.conf.Subscribe.UserAgentLimit
		}
		if value, ok := systemConfigLookup(subscribeValues, "UserAgentList", "user_agent_list"); ok {
			result.UserAgentList = value
		} else if r.data.conf != nil && r.data.conf.Subscribe != nil {
			result.UserAgentList = r.data.conf.Subscribe.UserAgentList
		}
		return result, nil
	}

	if r.data.conf != nil && r.data.conf.Subscribe != nil {
		result.PanDomain = r.data.conf.Subscribe.PanDomain
		result.UserAgentLimit = r.data.conf.Subscribe.UserAgentLimit
		result.UserAgentList = r.data.conf.Subscribe.UserAgentList
	}

	return result, nil
}

// GetUserInfo 获取用户信息
func (r *publicSubscriptionRepo) GetUserInfo(ctx context.Context, userID int64) (*subscriptionbiz.UserInfo, error) {
	user, err := r.data.db.ProxyUser.Query().
		Where(proxyuser.IDEQ(userID)).
		Only(ctx)

	if err != nil {
		r.log.Errorf("Failed to query user: %v", err)
		return nil, err
	}

	userInfo := &subscriptionbiz.UserInfo{
		ID:         int64(user.ID),
		Email:      "", // 需要从ProxyUserAuthMethod查询
		InviteCode: getStringValue(user.ReferCode),
	}

	// 获取邮箱
	authMethod, err := r.data.db.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.UserIDEQ(userID),
			proxyuserauthmethod.AuthTypeEQ("email"),
		).
		First(ctx)

	if err == nil && authMethod != nil {
		userInfo.Email = authMethod.AuthIdentifier
	}

	return userInfo, nil
}

// GetSubscribeInfo 获取订阅信息头（upload/download/total/expire）
func (r *publicSubscriptionRepo) GetSubscribeInfo(ctx context.Context, userSubscribe *subscriptionbiz.UserSubscribe) string {
	upload := userSubscribe.Upload
	download := userSubscribe.Download
	total := userSubscribe.Traffic
	expire := int64(0)
	if userSubscribe.ExpireTime > 0 {
		expire = time.UnixMilli(userSubscribe.ExpireTime).Unix()
	}

	return fmt.Sprintf("upload=%d;download=%d;total=%d;expire=%d",
		upload, download, total, expire)
}

// UpdateSubscribeLog 更新订阅活动日志 - 按照原项目实现
func (r *publicSubscriptionRepo) UpdateSubscribeLog(ctx context.Context, userSubscribe *subscriptionbiz.UserSubscribe, userAgent, clientIP string) error {
	if userSubscribe == nil {
		return nil
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"token":             userSubscribe.Token,
		"user_agent":        userAgent,
		"client_ip":         clientIP,
		"user_subscribe_id": userSubscribe.ID,
	})

	_, err := r.data.db.ProxySystemLog.Create().
		SetType(20).
		SetDate(time.Now().Format(time.DateOnly)).
		SetObjectID(userSubscribe.UserID).
		SetContent(string(payload)).
		Save(ctx)

	if err != nil {
		r.log.Errorf("Failed to insert subscribe log: %v", err)
		return err
	}

	return nil
}

// getInt64Value 获取int64指针的值，nil返回0
func getInt64Value(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// GetSubscribeApplications 获取所有订阅应用配置
func (r *publicSubscriptionRepo) GetSubscribeApplications(ctx context.Context) ([]*subscriptionbiz.SubscribeApplication, error) {
	apps, err := r.data.db.ProxySubscribeApplication.Query().
		Order(ent.Asc(proxysubscribeapplication.FieldID)).
		All(ctx)

	if err != nil {
		r.log.Errorf("Failed to query subscribe applications: %v", err)
		return nil, err
	}

	result := make([]*subscriptionbiz.SubscribeApplication, 0, len(apps))
	for _, app := range apps {
		result = append(result, &subscriptionbiz.SubscribeApplication{
			ID:                int64(app.ID),
			Name:              app.Name,
			Icon:              getStringValue(app.Icon),
			Description:       getStringValue(app.Description),
			Scheme:            app.Scheme,
			UserAgent:         app.UserAgent,
			IsDefault:         app.IsDefault,
			SubscribeTemplate: getStringValue(app.SubscribeTemplate),
			OutputFormat:      app.OutputFormat,
			DownloadLink:      app.DownloadLink,
		})
	}

	return result, nil
}
