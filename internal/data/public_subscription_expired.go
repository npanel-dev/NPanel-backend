package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyserver"
	"github.com/npanel-dev/NPanel-backend/ent/proxyservergroup"
	subscriptionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/subscription"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

// createExpiredNodesFromDB 按照原项目逻辑从数据库获取过期节点组
func (r *publicSubscriptionRepo) createExpiredNodesFromDB(ctx context.Context, userSubscribe *subscriptionbiz.UserSubscribe) []*subscriptionbiz.NodeInfo {
	// 1. 查询过期节点组
	expiredGroup, err := r.data.db.ProxyServerGroup.Query().
		Where(proxyservergroup.IsExpiredGroupEQ(true)).
		First(ctx)

	if err != nil {
		r.log.Debugf("No expired node group configured: %v", err)
		return r.createExpiredNodesDefault()
	}
	if !isSubscriptionNodeGroupTypeAccessible(expiredGroup.GroupType, subscriptionNodeGroupAccessSubscribe) {
		r.log.Debugf("Expired node group %d is not accessible for subscribe output, type=%s", expiredGroup.ID, expiredGroup.GroupType)
		return r.createExpiredNodesDefault()
	}

	// 2. 检查用户是否在过期天数限制内
	if userSubscribe.ExpireTime == 0 {
		r.log.Debug("User subscription has no expire time")
		return r.createExpiredNodesDefault()
	}

	expireTime := time.UnixMilli(userSubscribe.ExpireTime)
	expiredDays := int(time.Since(expireTime).Hours() / 24)

	if expiredDays > expiredGroup.ExpiredDaysLimit {
		r.log.Debugf("User subscription expired %d days, exceeds limit %d days", expiredDays, expiredGroup.ExpiredDaysLimit)
		return nil
	}

	// 3. 检查用户已使用流量是否超过限制(仅使用过期期间的流量)
	if expiredGroup.MaxTrafficGBExpired != nil && *expiredGroup.MaxTrafficGBExpired > 0 {
		expiredDownload := userSubscribe.ExpiredDownload
		expiredUpload := userSubscribe.ExpiredUpload

		usedTrafficGB := float64(expiredDownload+expiredUpload) / (1024 * 1024 * 1024)
		if usedTrafficGB >= float64(*expiredGroup.MaxTrafficGBExpired) {
			r.log.Debugf("User expired traffic %.2f GB, exceeds expired group limit %.2f GB",
				usedTrafficGB, float64(*expiredGroup.MaxTrafficGBExpired))
			return nil
		}
	}

	// 4. 过期节点也必须满足“已启用且未隐藏”，并且只能来自过期分组。
	nodes, err := r.data.db.ProxyNode.Query().
		Where(
			proxynode.EnabledEQ(true),
			proxynode.IsHiddenEQ(false),
		).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP("JSON_CONTAINS("+proxynode.FieldNodeGroupIds+", ?)", fmt.Sprintf("%d", expiredGroup.ID)))
		}).
		Order(ent.Asc(proxynode.FieldSort)).
		Limit(1000).
		All(ctx)

	if err != nil {
		r.log.Errorf("Failed to query expired group nodes: %v", err)
		return r.createExpiredNodesDefault()
	}

	if len(nodes) == 0 {
		r.log.Debug("No nodes found in expired group")
		return r.createExpiredNodesDefault()
	}

	// 5. 转换为NodeInfo
	nodeInfos := make([]*subscriptionbiz.NodeInfo, 0, len(nodes))

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
		for _, protocol := range protocols {
			if protocol != nil && protocol.Type == node.Protocol {
				matched = protocol
				break
			}
		}
		if matched == nil {
			continue
		}

		nodeInfo := &subscriptionbiz.NodeInfo{
			ID:                                 int64(node.ID),
			Sort:                               int(node.Sort),
			Name:                               node.Name,
			Server:                             node.Address,
			Port:                               node.Port,
			Type:                               node.Protocol,
			Tags:                               tool.StringToStringSlice(node.Tags),
			NodeGroupID:                        expiredGroup.ID,
			Security:                           matched.Security,
			SNI:                                matched.SNI,
			AllowInsecure:                      matched.AllowInsecure,
			Fingerprint:                        matched.Fingerprint,
			Method:                             matched.Cipher,
			ServerKey:                          matched.ServerKey,
			Flow:                               matched.Flow,
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
			RealityServerAddr:                  matched.RealityServerAddr,
			RealityServerPort:                  int(matched.RealityServerPort),
			RealityPrivateKey:                  matched.RealityPrivateKey,
			RealityPublicKey:                   matched.RealityPublicKey,
			RealityShortId:                     matched.RealityShortId,
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

	r.log.Infof("Returned %d nodes from expired group for user %d (expired %d days)",
		len(nodeInfos), userSubscribe.UserID, expiredDays)

	return nodeInfos
}

// createExpiredNodesDefault 创建默认的过期提示节点
func (r *publicSubscriptionRepo) createExpiredNodesDefault() []*subscriptionbiz.NodeInfo {
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
