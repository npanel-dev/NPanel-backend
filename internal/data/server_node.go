package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyservergroup"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
	"github.com/npanel-dev/NPanel-backend/ent/proxytrafficlog"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	serverBiz "github.com/npanel-dev/NPanel-backend/internal/biz/server"
	"github.com/npanel-dev/NPanel-backend/internal/queue/types"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
	"github.com/npanel-dev/NPanel-backend/pkg/uuidx"
	"github.com/redis/go-redis/v9"
)

type serverNodeConfigDNS struct {
	Proto   string   `json:"Proto"`
	Address string   `json:"Address"`
	Domains []string `json:"Domains"`
}

type serverNodeConfigOutbound struct {
	Name     string   `json:"Name"`
	Protocol string   `json:"Protocol"`
	Address  string   `json:"Address"`
	Port     int64    `json:"Port"`
	Password string   `json:"Password"`
	Rules    []string `json:"Rules"`
}

// serverNodeRepo 节点服务器数据仓储
type serverNodeRepo struct {
	data  *Data
	log   *log.Helper
	queue *asynq.Client
}

// NewServerNodeRepo 创建节点服务器数据仓储
func NewServerNodeRepo(data *Data, logger log.Logger) serverBiz.ServerNodeRepo {
	return &serverNodeRepo{
		data:  data,
		log:   log.NewHelper(logger),
		queue: data.queue,
	}
}

func (r *serverNodeRepo) GetNodeSecret(ctx context.Context) (string, error) {
	if appConf := r.data.AppConf(); appConf != nil && appConf.Node != nil {
		return appConf.Node.NodeSecret, nil
	}
	nodeConfig, err := LoadNodeConfigForServer(ctx, r.data, log.With(log.DefaultLogger, "module", "data/server_node"))
	if err != nil {
		r.log.Errorf("GetNodeSecret failed: %v", err)
		return "", err
	}
	r.log.Infof("GetNodeSecret loaded from admin node config path: %q", nodeConfig.NodeSecret)
	return nodeConfig.NodeSecret, nil
}

// GetServerConfig 获取服务器配置
func (r *serverNodeRepo) GetServerConfig(ctx context.Context, serverID int64, protocol string) (*serverBiz.ServerConfig, error) {
	// 查找服务器
	server, err := r.data.db.ProxyServer.Get(ctx, serverID)
	if err != nil {
		r.log.Errorf("GetServerConfig failed: %v", err)
		return nil, err
	}

	// 解析协议配置
	var protocolConfig map[string]interface{}
	if server.Protocol != "" {
		var protocols []map[string]interface{}
		if err := json.Unmarshal([]byte(server.Protocol), &protocols); err != nil {
			r.log.Errorf("Failed to unmarshal protocols: %v", err)
			return nil, err
		}
		// 查找指定协议的配置
		for _, p := range protocols {
			if p["type"] == protocol {
				protocolConfig = p
				break
			}
		}
	}

	// 获取协议配置的JSON字符串
	configJSON := "{}"
	if len(protocolConfig) > 0 {
		if b, err := json.Marshal(protocolConfig); err == nil {
			configJSON = string(b)
		}
	}

	pushInterval := int64(0)
	pullInterval := int64(0)
	if appConf := r.data.AppConf(); appConf != nil && appConf.Node != nil {
		pushInterval = appConf.Node.NodePushInterval
		pullInterval = appConf.Node.NodePullInterval
	} else {
		nodeConfig, err := LoadNodeConfigForServer(ctx, r.data, log.With(log.DefaultLogger, "module", "data/server_node"))
		if err != nil {
			r.log.Errorf("GetServerConfig load node config failed: %v", err)
			return nil, err
		}
		pushInterval = int64(nodeConfig.NodePushInterval)
		pullInterval = int64(nodeConfig.NodePullInterval)
	}

	return &serverBiz.ServerConfig{
		PushInterval: pushInterval,
		PullInterval: pullInterval,
		Protocol:     protocol,
		Config:       configJSON,
	}, nil
}

// GetServerUserList 获取服务器用户列表
func (r *serverNodeRepo) GetServerUserList(ctx context.Context, serverID int64, protocol string) ([]*serverBiz.ServerUser, error) {
	if _, err := r.data.db.ProxyServer.Get(ctx, serverID); err != nil {
		r.log.Errorf("GetServerUserList get server failed: %v", err)
		return nil, err
	}

	nodes, err := r.data.db.ProxyNode.Query().
		Where(
			proxynode.ServerIDEQ(serverID),
			proxynode.ProtocolEQ(protocol),
		).
		Order(ent.Asc(proxynode.FieldSort)).
		All(ctx)
	if err != nil {
		r.log.Errorf("GetServerUserList query nodes failed: %v", err)
		return nil, err
	}
	if len(nodes) == 0 {
		return []*serverBiz.ServerUser{{ID: 1, UUID: uuidx.NewUUID().String()}}, nil
	}

	nodeGroupMap := make(map[int64]struct{})
	nodeIDs := make([]int64, 0, len(nodes))
	nodeTags := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
		if node.Tags != "" {
			nodeTags = append(nodeTags, strings.Split(node.Tags, ",")...)
		}
		for _, groupID := range node.NodeGroupIds {
			if groupID > 0 {
				nodeGroupMap[groupID] = struct{}{}
			}
		}
	}

	nodeGroupIDs := make([]int64, 0, len(nodeGroupMap))
	for groupID := range nodeGroupMap {
		nodeGroupIDs = append(nodeGroupIDs, groupID)
	}

	plans, err := r.matchedSubscribePlans(ctx, nodeGroupIDs, nodeIDs, tool.RemoveDuplicateElements(nodeTags...))
	if err != nil {
		r.log.Errorf("GetServerUserList match subscribes failed: %v", err)
		return nil, err
	}
	if len(plans) == 0 {
		return []*serverBiz.ServerUser{{ID: 1, UUID: uuidx.NewUUID().String()}}, nil
	}

	users := make([]*serverBiz.ServerUser, 0)
	now := time.Now()
	for _, plan := range plans {
		userSubs, err := r.usersBySubscribeID(ctx, plan.ID)
		if err != nil {
			r.log.Errorf("GetServerUserList query subscribe users failed: %v", err)
			return nil, err
		}
		for _, userSub := range userSubs {
			if !r.shouldIncludeServerUser(ctx, userSub, nodeGroupIDs, now) {
				continue
			}
			users = append(users, &serverBiz.ServerUser{
				ID:          userSub.ID,
				UUID:        serverNodeStringValue(userSub.UUID),
				SpeedLimit:  r.effectiveSpeedLimit(ctx, plan, userSub, now),
				DeviceLimit: int64(plan.DeviceLimit),
			})
		}
	}

	if len(nodeGroupIDs) > 0 {
		expiredUsers, expiredSpeedLimit, err := r.expiredServerUsers(ctx, nodeGroupIDs)
		if err != nil {
			r.log.Errorf("GetServerUserList query expired users failed: %v", err)
			return nil, err
		}
		for i := range expiredUsers {
			if expiredSpeedLimit > 0 {
				expiredUsers[i].SpeedLimit = expiredSpeedLimit
			}
		}
		users = append(users, expiredUsers...)
	}

	if len(users) == 0 {
		return []*serverBiz.ServerUser{{ID: 1, UUID: uuidx.NewUUID().String()}}, nil
	}
	return users, nil
}

// PushUserTraffic 推送用户流量（创建流量统计任务）
func (r *serverNodeRepo) PushUserTraffic(ctx context.Context, req *serverBiz.PushUserTrafficRequest) error {
	// 验证服务器是否存在
	_, err := r.data.db.ProxyServer.Get(ctx, req.ServerID)
	if err != nil {
		r.log.Errorf("PushUserTraffic failed: server not found: %v", err)
		return fmt.Errorf("server not found")
	}

	// 更新服务器最后上报时间
	now := time.Now()
	err = r.data.db.ProxyServer.UpdateOneID(req.ServerID).
		SetLastReportedAt(now).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("Update server last_reported_at failed: %v", err)
	}

	// 创建流量统计任务并推送到队列
	// 转换请求格式到队列任务格式
	userTrafficLogs := make([]types.UserTraffic, 0, len(req.Traffic))
	for _, traffic := range req.Traffic {
		userTrafficLogs = append(userTrafficLogs, types.UserTraffic{
			SID:      traffic.SID,
			Upload:   traffic.Upload,
			Download: traffic.Download,
		})
	}

	payload := types.TrafficStatistics{
		ServerID: req.ServerID,
		Protocol: req.Protocol,
		Logs:     userTrafficLogs,
	}

	// 序列化任务负载
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		r.log.Errorf("PushUserTraffic: Failed to marshal payload: %v", err)
		return fmt.Errorf("failed to marshal traffic payload: %w", err)
	}

	// 创建任务并立即执行
	task := asynq.NewTask(types.ForthwithTrafficStatistics, payloadBytes, asynq.MaxRetry(3))

	// 入队任务（立即执行）
	_, err = r.queue.Enqueue(task)
	if err != nil {
		r.log.Errorf("PushUserTraffic: Failed to enqueue task: %v", err)
		return fmt.Errorf("failed to enqueue traffic task: %w", err)
	}

	r.log.Infof("PushUserTraffic: serverID=%d, protocol=%s, traffic count=%d, task enqueued",
		req.ServerID, req.Protocol, len(req.Traffic))

	return nil
}

// PushServerStatus 推送服务器状态
func (r *serverNodeRepo) PushServerStatus(ctx context.Context, req *serverBiz.PushServerStatusRequest) error {
	// 验证服务器是否存在
	_, err := r.data.db.ProxyServer.Get(ctx, req.ServerID)
	if err != nil {
		r.log.Errorf("PushServerStatus failed: server not found: %v", err)
		return fmt.Errorf("server not found")
	}

	// 更新服务器最后上报时间
	now := time.Now()
	err = r.data.db.ProxyServer.UpdateOneID(req.ServerID).
		SetLastReportedAt(now).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("Update server last_reported_at failed: %v", err)
		return err
	}

	if r.data.rdb != nil {
		statusPayload := map[string]interface{}{
			"cpu":        req.CPU,
			"mem":        req.Mem,
			"disk":       req.Disk,
			"updated_at": req.UpdatedAt,
		}
		payloadBytes, marshalErr := json.Marshal(statusPayload)
		if marshalErr != nil {
			r.log.Errorf("PushServerStatus marshal status failed: %v", marshalErr)
			return marshalErr
		}
		if err = r.data.rdb.Set(ctx, fmt.Sprintf("node:status:%d", req.ServerID), payloadBytes, 5*time.Minute).Err(); err != nil {
			r.log.Errorf("PushServerStatus cache status failed: %v", err)
			return err
		}
	}

	r.log.Infof("PushServerStatus: serverID=%d, cpu=%.2f, mem=%.2f, disk=%.2f",
		req.ServerID, req.CPU, req.Mem, req.Disk)

	return nil
}

// PushOnlineUsers 推送在线用户
func (r *serverNodeRepo) PushOnlineUsers(ctx context.Context, req *serverBiz.PushOnlineUsersRequest) error {
	if req == nil || req.ServerID <= 0 || len(req.Users) == 0 {
		return fmt.Errorf("invalid request parameters")
	}
	for _, user := range req.Users {
		if user == nil || user.SID <= 0 || user.IP == "" {
			if user == nil {
				return fmt.Errorf("invalid user data: uid=%d, ip=%s", 0, "")
			}
			return fmt.Errorf("invalid user data: uid=%d, ip=%s", user.SID, user.IP)
		}
	}

	// 验证服务器是否存在
	_, err := r.data.db.ProxyServer.Get(ctx, req.ServerID)
	if err != nil {
		r.log.Errorf("PushOnlineUsers failed: server not found: %v", err)
		return fmt.Errorf("server not found")
	}

	if r.data.rdb == nil {
		return nil
	}

	// 构建在线用户映射 map[subscribeID][]IP
	onlineUsers := make(map[int64][]string)
	for _, user := range req.Users {
		onlineUsers[user.SID] = append(onlineUsers[user.SID], user.IP)
	}

	// 存储到Redis缓存
	// 格式：node:online:subscribe:{serverID}:{protocol}
	key := fmt.Sprintf("node:online:subscribe:%d:%s", req.ServerID, req.Protocol)

	if len(onlineUsers) == 0 {
		if err := r.data.rdb.Del(ctx, key).Err(); err != nil && err != redis.Nil {
			r.log.Errorf("Redis Del failed: %v", err)
			return err
		}
	} else {
		// 序列化在线用户数据
		data, err := json.Marshal(onlineUsers)
		if err != nil {
			r.log.Errorf("Marshal online users failed: %v", err)
			return err
		}

		// 存储到Redis（设置过期时间为5分钟）
		err = r.data.rdb.Set(ctx, key, data, 5*time.Minute).Err()
		if err != nil {
			r.log.Errorf("Redis Set failed: %v", err)
			return err
		}
	}

	if err := UpdateOnlineUserSubscribeGlobalCache(ctx, r.data.rdb, onlineUsers); err != nil {
		r.log.Errorf("PushOnlineUsers update global online cache failed: %v", err)
		return err
	}

	r.log.Infof("PushOnlineUsers: serverID=%d, protocol=%s, online users=%d",
		req.ServerID, req.Protocol, len(onlineUsers))

	return nil
}

// QueryServerProtocolConfig 查询服务器协议配置
func (r *serverNodeRepo) QueryServerProtocolConfig(ctx context.Context, serverID int64) (*serverBiz.ProtocolConfig, error) {
	// 查找服务器
	server, err := r.data.db.ProxyServer.Get(ctx, serverID)
	if err != nil {
		r.log.Errorf("QueryServerProtocolConfig failed: %v", err)
		return nil, err
	}

	// 解析协议配置
	var protocols []map[string]interface{}
	if server.Protocol != "" {
		if err := json.Unmarshal([]byte(server.Protocol), &protocols); err != nil {
			r.log.Errorf("Failed to unmarshal protocols: %v", err)
			return nil, err
		}
	}

	dnsConfigs := make([]*serverBiz.DNSConfig, 0)
	block := make([]string, 0)
	outboundConfigs := make([]*serverBiz.OutboundConfig, 0)
	trafficReportThreshold := int64(0)
	ipStrategy := ""
	if appConf := r.data.AppConf(); appConf != nil && appConf.Node != nil {
		trafficReportThreshold = appConf.Node.TrafficReportThreshold
		ipStrategy = appConf.Node.IpStrategy
		block = serverNodeSanitizeStringList(append([]string(nil), appConf.Node.Block...))
		for _, dns := range appConf.Node.Dns {
			if dns == nil {
				continue
			}
			domains := serverNodeSanitizeStringList(append([]string(nil), dns.Domains...))
			if strings.TrimSpace(dns.Proto) == "" && strings.TrimSpace(dns.Address) == "" && len(domains) == 0 {
				continue
			}
			dnsConfigs = append(dnsConfigs, &serverBiz.DNSConfig{
				Server: strings.TrimSpace(dns.Address),
				Domain: strings.Join(domains, ","),
				Port:   0,
			})
		}
		for _, outbound := range appConf.Node.Outbound {
			if outbound == nil {
				continue
			}
			rules := serverNodeSanitizeStringList(append([]string(nil), outbound.Rules...))
			name := strings.TrimSpace(outbound.Name)
			protocol := strings.TrimSpace(outbound.Protocol)
			address := strings.TrimSpace(outbound.Address)
			password := strings.TrimSpace(outbound.Password)
			if name == "" && protocol == "" && address == "" && outbound.Port == 0 && password == "" && len(rules) == 0 {
				continue
			}
			settings := map[string]string{
				"address":  address,
				"port":     fmt.Sprintf("%d", outbound.Port),
				"password": password,
			}
			if len(rules) > 0 {
				settings["rules"] = strings.Join(rules, ",")
			}
			outboundConfigs = append(outboundConfigs, &serverBiz.OutboundConfig{
				Tag:      name,
				Protocol: protocol,
				Settings: settings,
			})
		}
	} else {
		nodeConfig, err := LoadNodeConfigForServer(ctx, r.data, log.With(log.DefaultLogger, "module", "data/server_node"))
		if err != nil {
			r.log.Errorf("QueryServerProtocolConfig load node config failed: %v", err)
			return nil, err
		}
		trafficReportThreshold = int64(nodeConfig.TrafficReportThreshold)
		ipStrategy = nodeConfig.IPStrategy
		if raw := strings.TrimSpace(nodeConfig.Block); raw != "" {
			if err := json.Unmarshal([]byte(raw), &block); err != nil {
				r.log.Warnf("QueryServerProtocolConfig unmarshal block failed: %v", err)
				block = nil
			}
			block = serverNodeSanitizeStringList(block)
		}
		if raw := strings.TrimSpace(nodeConfig.DNS); raw != "" {
			var dnsList []serverNodeConfigDNS
			if err := json.Unmarshal([]byte(raw), &dnsList); err != nil {
				r.log.Warnf("QueryServerProtocolConfig unmarshal dns failed: %v", err)
			} else {
				for _, dns := range dnsList {
					dns.Proto = strings.TrimSpace(dns.Proto)
					dns.Address = strings.TrimSpace(dns.Address)
					dns.Domains = serverNodeSanitizeStringList(dns.Domains)
					if dns.Proto == "" && dns.Address == "" && len(dns.Domains) == 0 {
						continue
					}
					dnsConfigs = append(dnsConfigs, &serverBiz.DNSConfig{
						Server: dns.Address,
						Domain: strings.Join(dns.Domains, ","),
						Port:   0,
					})
				}
			}
		}
		if raw := strings.TrimSpace(nodeConfig.Outbound); raw != "" {
			var outboundList []serverNodeConfigOutbound
			if err := json.Unmarshal([]byte(raw), &outboundList); err != nil {
				r.log.Warnf("QueryServerProtocolConfig unmarshal outbound failed: %v", err)
			} else {
				for _, outbound := range outboundList {
					outbound.Name = strings.TrimSpace(outbound.Name)
					outbound.Protocol = strings.TrimSpace(outbound.Protocol)
					outbound.Address = strings.TrimSpace(outbound.Address)
					outbound.Password = strings.TrimSpace(outbound.Password)
					outbound.Rules = serverNodeSanitizeStringList(outbound.Rules)
					if outbound.Name == "" && outbound.Protocol == "" && outbound.Address == "" && outbound.Port == 0 && outbound.Password == "" && len(outbound.Rules) == 0 {
						continue
					}
					settings := map[string]string{
						"address":  outbound.Address,
						"port":     fmt.Sprintf("%d", outbound.Port),
						"password": outbound.Password,
					}
					if len(outbound.Rules) > 0 {
						settings["rules"] = strings.Join(outbound.Rules, ",")
					}
					outboundConfigs = append(outboundConfigs, &serverBiz.OutboundConfig{
						Tag:      outbound.Name,
						Protocol: outbound.Protocol,
						Settings: settings,
					})
				}
			}
		}
	}

	// 构建协议配置响应
	protocolConfigs := make([]*serverBiz.Protocol, 0, len(protocols))
	for _, p := range protocols {
		protocolConfigs = append(protocolConfigs, &serverBiz.Protocol{
			Type:                                   serverNodeMapString(p["type"]),
			Port:                                   int32(serverNodeMapInt64(p["port"])),
			Enable:                                 serverNodeMapBool(p["enable"]),
			Security:                               serverNodeMapString(p["security"]),
			SNI:                                    serverNodeMapString(p["sni"]),
			AllowInsecure:                          serverNodeMapBool(p["allow_insecure"]),
			Fingerprint:                            serverNodeMapString(p["fingerprint"]),
			RealityServerAddr:                      serverNodeMapString(p["reality_server_addr"]),
			RealityServerPort:                      int32(serverNodeMapInt64(p["reality_server_port"])),
			RealityPrivateKey:                      serverNodeMapString(p["reality_private_key"]),
			RealityPublicKey:                       serverNodeMapString(p["reality_public_key"]),
			RealityShortId:                         serverNodeMapString(p["reality_short_id"]),
			Transport:                              serverNodeMapString(p["transport"]),
			Host:                                   serverNodeMapString(p["host"]),
			Path:                                   serverNodeMapString(p["path"]),
			ServiceName:                            serverNodeMapString(p["service_name"]),
			Mc1Mode:                                serverNodeFirstString(p, "mc1_mode", "mc1Mode", "mode"),
			Mc1CidrSegments:                        serverNodeFirstStringSlice(p, "mc1_cidr_segments", "mc1CidrSegments", "cidrSegments"),
			MundoUsername:                          serverNodeFirstString(p, "mundo_username", "mundoUsername", "username"),
			MundoCertificateFingerprint:            serverNodeFirstString(p, "mundo_certificate_fingerprint", "mundoCertificateFingerprint", "certificateFingerprint"),
			MundoFakeTitle:                         serverNodeFirstString(p, "mundo_fake_title", "mundoFakeTitle", "fakeTitle"),
			MundoFakeMessage:                       serverNodeFirstString(p, "mundo_fake_message", "mundoFakeMessage", "fakeMessage"),
			MundoAcceptProxyProtocol:               serverNodeFirstBool(p, "mundo_accept_proxy_protocol", "mundoAcceptProxyProtocol", "acceptProxyProtocol"),
			MundoUseTLSCertificate:                 serverNodeFirstBool(p, "mundo_use_tls_certificate", "mundoUseTLSCertificate", "useTLSCertificate"),
			Cipher:                                 serverNodeMapString(p["cipher"]),
			ServerKey:                              serverNodeMapString(p["server_key"]),
			Flow:                                   serverNodeMapString(p["flow"]),
			HopPorts:                               serverNodeMapString(p["hop_ports"]),
			HopInterval:                            int32(serverNodeMapInt64(p["hop_interval"])),
			ObfsPassword:                           serverNodeMapString(p["obfs_password"]),
			DisableSNI:                             serverNodeMapBool(p["disable_sni"]),
			ReduceRtt:                              serverNodeMapBool(p["reduce_rtt"]),
			UDPRelayMode:                           serverNodeMapString(p["udp_relay_mode"]),
			CongestionController:                   serverNodeMapString(p["congestion_controller"]),
			Multiplex:                              serverNodeMapString(p["multiplex"]),
			PaddingScheme:                          serverNodeMapString(p["padding_scheme"]),
			UpMbps:                                 int32(serverNodeMapInt64(p["up_mbps"])),
			DownMbps:                               int32(serverNodeMapInt64(p["down_mbps"])),
			Obfs:                                   serverNodeMapString(p["obfs"]),
			ObfsHost:                               serverNodeMapString(p["obfs_host"]),
			ObfsPath:                               serverNodeMapString(p["obfs_path"]),
			XhttpMode:                              serverNodeMapString(p["xhttp_mode"]),
			XhttpExtra:                             serverNodeMapString(p["xhttp_extra"]),
			Encryption:                             serverNodeMapString(p["encryption"]),
			EncryptionMode:                         serverNodeMapString(p["encryption_mode"]),
			EncryptionRtt:                          serverNodeMapString(p["encryption_rtt"]),
			EncryptionTicket:                       serverNodeMapString(p["encryption_ticket"]),
			EncryptionServerPadding:                serverNodeMapString(p["encryption_server_padding"]),
			EncryptionPrivateKey:                   serverNodeMapString(p["encryption_private_key"]),
			EncryptionClientPadding:                serverNodeMapString(p["encryption_client_padding"]),
			EncryptionPassword:                     serverNodeMapString(p["encryption_password"]),
			Ratio:                                  serverNodeMapFloat64(p["ratio"]),
			CertMode:                               serverNodeMapString(p["cert_mode"]),
			CertDNSProvider:                        serverNodeMapString(p["cert_dns_provider"]),
			CertDNSEnv:                             serverNodeMapString(p["cert_dns_env"]),
			SimnetPsk:                              serverNodeMapString(p["simnet_psk"]),
			SimnetKeyID:                            int32(serverNodeMapInt64(p["simnet_key_id"])),
			SimnetTicketID:                         serverNodeMapString(p["simnet_ticket_id"]),
			SimnetPath:                             serverNodeMapString(p["simnet_path"]),
			SimnetCarrier:                          serverNodeMapString(p["simnet_carrier"]),
			SimnetAfEnabled:                        serverNodeMapBool(p["simnet_af_enabled"]),
			SimnetAfPathMode:                       serverNodeMapString(p["simnet_af_path_mode"]),
			SimnetAfPathPrefix:                     serverNodeMapString(p["simnet_af_path_prefix"]),
			SimnetAfPathSuffix:                     serverNodeMapString(p["simnet_af_path_suffix"]),
			SimnetAfMagicMode:                      serverNodeMapString(p["simnet_af_magic_mode"]),
			SimnetAfResponseJitterMs:               int32(serverNodeMapInt64(p["simnet_af_response_jitter_ms"])),
			SimnetAfHandshakePolymorphism:          serverNodeMapBool(p["simnet_af_handshake_polymorphism"]),
			SimnetAfSettingsJitter:                 serverNodeMapBool(p["simnet_af_settings_jitter"]),
			SimnetAfFakeHeaderInjection:            serverNodeMapBool(p["simnet_af_fake_header_injection"]),
			SimnetReverseEnabled:                   serverNodeMapBool(p["simnet_reverse_enabled"]),
			SimnetReverseListenAddr:                serverNodeMapString(p["simnet_reverse_listen_addr"]),
			SimnetReverseListenPort:                int32(serverNodeMapInt64(p["simnet_reverse_listen_port"])),
			SimnetReverseTargetHost:                serverNodeMapString(p["simnet_reverse_target_host"]),
			SimnetReverseTargetPort:                int32(serverNodeMapInt64(p["simnet_reverse_target_port"])),
			SimnetFallbackEnabled:                  serverNodeMapBool(p["simnet_fallback_enabled"]),
			SimnetFallbackTargetScheme:             serverNodeMapString(p["simnet_fallback_target_scheme"]),
			SimnetFallbackTargetHost:               serverNodeMapString(p["simnet_fallback_target_host"]),
			SimnetFallbackTargetPort:               int32(serverNodeMapInt64(p["simnet_fallback_target_port"])),
			SimnetFallbackHostHeader:               serverNodeMapString(p["simnet_fallback_host_header"]),
			SimnetFallbackTLSSNI:                   serverNodeMapString(p["simnet_fallback_tls_sni"]),
			SimnetInboundMaxStreamsPerSession:      int32(serverNodeMapInt64(p["simnet_inbound_max_streams_per_session"])),
			SimnetInboundMaxUDPStreamsPerSession:   int32(serverNodeMapInt64(p["simnet_inbound_max_udp_streams_per_session"])),
			SimnetInboundMaxHandlerTasksPerSession: int32(serverNodeMapInt64(p["simnet_inbound_max_handler_tasks_per_session"])),
			SimnetStreamEventChannelCapacity:       int32(serverNodeMapInt64(p["simnet_stream_event_channel_capacity"])),
			SimnetStreamDataChannelCapacity:        int32(serverNodeMapInt64(p["simnet_stream_data_channel_capacity"])),
			SimnetTargetDialTimeoutMs:              int32(serverNodeMapInt64(p["simnet_target_dial_timeout_ms"])),
			SimnetTargetMaxConcurrentDials:         int32(serverNodeMapInt64(p["simnet_target_max_concurrent_dials"])),
			SimnetEgressBlockLoopback:              serverNodeMapBool(p["simnet_egress_block_loopback"]),
			SimnetEgressBlockPrivate:               serverNodeMapBool(p["simnet_egress_block_private"]),
			SimnetEgressBlockLinkLocal:             serverNodeMapBool(p["simnet_egress_block_link_local"]),
			SimnetEgressBlockMetadata:              serverNodeMapBool(p["simnet_egress_block_metadata"]),
			SimnetSendWindow:                       int32(serverNodeMapInt64(p["simnet_send_window"])),
			SimnetRecvWindow:                       int32(serverNodeMapInt64(p["simnet_recv_window"])),
			SimnetMaxConcurrentStreams:             int32(serverNodeMapInt64(p["simnet_max_concurrent_streams"])),
			SimnetInitialWindowSize:                int32(serverNodeMapInt64(p["simnet_initial_window_size"])),
			SimnetMaxFrameSize:                     int32(serverNodeMapInt64(p["simnet_max_frame_size"])),
			SimnetClientMaxConcurrentStreams:       int32(serverNodeMapInt64(p["simnet_client_max_concurrent_streams"])),
			SimnetClientMaxStreamsPerSession:       int32(serverNodeMapInt64(p["simnet_client_max_streams_per_session"])),
			SimnetClientSessionIdleTimeoutSecs:     int32(serverNodeMapInt64(p["simnet_client_session_idle_timeout_secs"])),
			SimnetClientMaxUDPSessions:             int32(serverNodeMapInt64(p["simnet_client_max_udp_sessions"])),
			// OmniFlow
			OmniflowCarrier:                    serverNodeMapString(p["omniflow_carrier"]),
			OmniflowPath:                       serverNodeMapString(p["omniflow_path"]),
			OmniflowContentType:                serverNodeMapString(p["omniflow_content_type"]),
			OmniflowProfilePath:                serverNodeMapString(p["omniflow_profile_path"]),
			OmniflowProfileJson:                serverNodeMapString(p["omniflow_profile_json"]),
			OmniflowServerHost:                 serverNodeMapString(p["omniflow_server_host"]),
			OmniflowServerPort:                 int32(serverNodeMapInt64(p["omniflow_server_port"])),
			OmniflowCaCertPath:                 serverNodeMapString(p["omniflow_ca_cert_path"]),
			OmniflowTargetMeta:                 serverNodeMapString(p["omniflow_target_meta"]),
			OmniflowSpkiPin:                    serverNodeMapString(p["omniflow_spki_pin"]),
			OmniflowH3FallbackEnabled:          serverNodeMapBool(p["omniflow_h3_fallback_enabled"]),
			OmniflowH3FallbackPolicy:           serverNodeMapString(p["omniflow_h3_fallback_policy"]),
			OmniflowH3FallbackTimeoutMs:        int32(serverNodeMapInt64(p["omniflow_h3_fallback_timeout_ms"])),
			OmniflowH3FallbackRetryBudget:      int32(serverNodeMapInt64(p["omniflow_h3_fallback_retry_budget"])),
			OmniflowH3FallbackSmokeEnabled:     serverNodeMapBool(p["omniflow_h3_fallback_smoke_enabled"]),
			OmniflowH3FallbackSmokeIntervalSec: int32(serverNodeMapInt64(p["omniflow_h3_fallback_smoke_interval_sec"])),
			OmniflowH3FallbackSmokeTimeoutMs:   int32(serverNodeMapInt64(p["omniflow_h3_fallback_smoke_timeout_ms"])),
			OmniflowMaxAgeSec:                  int32(serverNodeMapInt64(p["omniflow_max_age_sec"])),
			OmniflowIdleTimeoutSec:             int32(serverNodeMapInt64(p["omniflow_idle_timeout_sec"])),
			OmniflowMaxConnections:             int32(serverNodeMapInt64(p["omniflow_max_connections"])),
			OmniflowAdaptiveTlsEnabled:         serverNodeMapBool(p["omniflow_adaptive_tls_enabled"]),
			OmniflowTlsFingerprint:             serverNodeMapString(p["omniflow_tls_fingerprint"]),
			OmniflowSniMode:                    serverNodeMapString(p["omniflow_sni_mode"]),
			OmniflowPaddingMode:                serverNodeMapString(p["omniflow_padding_mode"]),
			OmniflowTrafficShapingEnabled:      serverNodeMapBool(p["omniflow_traffic_shaping_enabled"]),
			OmniflowAfEnabled:                  serverNodeMapBool(p["omniflow_af_enabled"]),
			OmniflowAfPathMode:                 serverNodeMapString(p["omniflow_af_path_mode"]),
			OmniflowAfPathPrefix:               serverNodeMapString(p["omniflow_af_path_prefix"]),
			OmniflowAfPathSuffix:               serverNodeMapString(p["omniflow_af_path_suffix"]),
			OmniflowAfPathRotationSecs:         int32(serverNodeMapInt64(p["omniflow_af_path_rotation_secs"])),
			OmniflowAfPathSkewSlots:            int32(serverNodeMapInt64(p["omniflow_af_path_skew_slots"])),
			OmniflowFallbackEnabled:            serverNodeMapBool(p["omniflow_fallback_enabled"]),
			OmniflowFallbackTargetScheme:       serverNodeMapString(p["omniflow_fallback_target_scheme"]),
			OmniflowFallbackTargetHost:         serverNodeMapString(p["omniflow_fallback_target_host"]),
			OmniflowFallbackTargetPort:         int32(serverNodeMapInt64(p["omniflow_fallback_target_port"])),
			OmniflowFallbackHostHeader:         serverNodeMapString(p["omniflow_fallback_host_header"]),
			OmniflowFallbackTLSSNI:             serverNodeMapString(p["omniflow_fallback_tls_sni"]),
			OmniflowFallbackCarrierEnabled:     serverNodeMapBool(p["omniflow_fallback_carrier_enabled"]),
			OmniflowFallbackConnectTunnel:      serverNodeMapBool(p["omniflow_fallback_connect_tunnel"]),
			OmniflowFallbackWssEnabled:         serverNodeMapBool(p["omniflow_fallback_wss_enabled"]),
		})
	}

	return &serverBiz.ProtocolConfig{
		TrafficReportThreshold: trafficReportThreshold,
		IPStrategy:             ipStrategy,
		DNS:                    dnsConfigs,
		Block:                  block,
		Outbound:               outboundConfigs,
		Protocols:              protocolConfigs,
		Total:                  int32(len(protocolConfigs)),
	}, nil
}

func serverNodeMapString(value interface{}) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func serverNodeMapStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return serverNodeSanitizeStringList(v)
	case string:
		return serverNodeSanitizeStringList(strings.Split(v, ","))
	case []interface{}:
		values := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				values = append(values, s)
			}
		}
		return serverNodeSanitizeStringList(values)
	default:
		return nil
	}
}

func serverNodeFirstString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(serverNodeMapString(values[key])); value != "" {
			return value
		}
	}
	return ""
}

func serverNodeFirstStringSlice(values map[string]interface{}, keys ...string) []string {
	for _, key := range keys {
		if value := serverNodeMapStringSlice(values[key]); len(value) > 0 {
			return value
		}
	}
	return nil
}

func serverNodeFirstBool(values map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if serverNodeMapBool(values[key]) {
			return true
		}
	}
	return false
}

func serverNodeMapBool(value interface{}) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	return false
}

func serverNodeMapInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func serverNodeMapFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		n, _ := v.Float64()
		return n
	default:
		return 0
	}
}

func (r *serverNodeRepo) matchedSubscribePlans(ctx context.Context, nodeGroupIDs, nodeIDs []int64, nodeTags []string) ([]*ent.ProxySubscribe, error) {
	plans, err := r.data.db.ProxySubscribe.Query().
		Order(ent.Asc(proxysubscribe.FieldSort)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*ent.ProxySubscribe, 0, len(plans))
	for _, plan := range plans {
		if len(nodeGroupIDs) > 0 {
			if subscribeMatchesNodeGroups(plan, nodeGroupIDs) {
				result = append(result, plan)
			}
			continue
		}
		if subscribeMatchesNodesAndTags(plan, nodeIDs, nodeTags) {
			result = append(result, plan)
		}
	}
	return result, nil
}

func subscribeMatchesNodeGroups(plan *ent.ProxySubscribe, nodeGroupIDs []int64) bool {
	if plan == nil || len(nodeGroupIDs) == 0 {
		return false
	}
	if plan.NodeGroupID != nil && tool.Contains(nodeGroupIDs, *plan.NodeGroupID) {
		return true
	}
	for _, groupID := range plan.NodeGroupIds {
		if tool.Contains(nodeGroupIDs, groupID) {
			return true
		}
	}
	return false
}

func subscribeMatchesNodesAndTags(plan *ent.ProxySubscribe, nodeIDs []int64, nodeTags []string) bool {
	if plan == nil {
		return false
	}
	if len(nodeIDs) == 0 && len(nodeTags) == 0 {
		return false
	}
	if len(nodeIDs) > 0 {
		planNodeIDs := tool.StringToInt64Slice(plan.Nodes)
		matched := false
		for _, nodeID := range nodeIDs {
			if tool.Contains(planNodeIDs, nodeID) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(nodeTags) > 0 && !serverNodeMatchesTags(plan.NodeTags, nodeTags) {
		return false
	}
	return true
}

func (r *serverNodeRepo) usersBySubscribeID(ctx context.Context, subscribeID int64) ([]*ent.ProxyUserSubscribe, error) {
	userSubs, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.SubscribeIDEQ(subscribeID),
			proxyusersubscribe.StatusIn(int8(0), int8(1)),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := r.data.db.ProxyUserSubscribe.Update().
		Where(
			proxyusersubscribe.SubscribeIDEQ(subscribeID),
			proxyusersubscribe.StatusEQ(int8(0)),
		).
		SetStatus(int8(1)).
		Save(ctx); err != nil {
		return nil, err
	}
	return userSubs, nil
}

func (r *serverNodeRepo) shouldIncludeServerUser(ctx context.Context, userSub *ent.ProxyUserSubscribe, nodeGroupIDs []int64, now time.Time) bool {
	if userSub == nil {
		return false
	}
	if serverNodeIsLegacyUnlimitedTime(userSub.ExpireTime) {
		return true
	}
	if userSub.ExpireTime != nil && userSub.ExpireTime.After(now) {
		return true
	}
	return r.canUseExpiredNodeGroup(ctx, userSub, nodeGroupIDs, now)
}

func (r *serverNodeRepo) expiredServerUsers(ctx context.Context, serverNodeGroupIDs []int64) ([]*serverBiz.ServerUser, int64, error) {
	expiredGroup, err := r.data.db.ProxyServerGroup.Query().
		Where(proxyservergroup.IsExpiredGroupEQ(true)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	if !tool.Contains(serverNodeGroupIDs, expiredGroup.ID) {
		return nil, 0, nil
	}

	userSubs, err := r.data.db.ProxyUserSubscribe.Query().
		Where(proxyusersubscribe.StatusEQ(int8(3))).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	users := make([]*serverBiz.ServerUser, 0)
	seen := make(map[int64]struct{})
	now := time.Now()
	for _, userSub := range userSubs {
		if !expiredUserEligible(userSub, expiredGroup, now) {
			continue
		}
		if _, ok := seen[userSub.ID]; ok {
			continue
		}
		seen[userSub.ID] = struct{}{}
		users = append(users, &serverBiz.ServerUser{
			ID:   userSub.ID,
			UUID: serverNodeStringValue(userSub.UUID),
		})
	}
	return users, int64(expiredGroup.SpeedLimit), nil
}

func expiredUserEligible(userSub *ent.ProxyUserSubscribe, expiredGroup *ent.ProxyServerGroup, now time.Time) bool {
	if userSub == nil || expiredGroup == nil || userSub.ExpireTime == nil {
		return false
	}
	expiredDays := int(now.Sub(*userSub.ExpireTime).Hours() / 24)
	if expiredDays > expiredGroup.ExpiredDaysLimit {
		return false
	}
	if expiredGroup.MaxTrafficGBExpired != nil && *expiredGroup.MaxTrafficGBExpired > 0 {
		usedTrafficGB := float64(serverNodeInt64Value(userSub.ExpiredDownload)+serverNodeInt64Value(userSub.ExpiredUpload)) / (1024 * 1024 * 1024)
		if usedTrafficGB >= float64(*expiredGroup.MaxTrafficGBExpired) {
			return false
		}
	}
	return true
}

func (r *serverNodeRepo) canUseExpiredNodeGroup(ctx context.Context, userSub *ent.ProxyUserSubscribe, nodeGroupIDs []int64, now time.Time) bool {
	expiredGroup, err := r.data.db.ProxyServerGroup.Query().
		Where(proxyservergroup.IsExpiredGroupEQ(true)).
		First(ctx)
	if err != nil {
		return false
	}
	if !tool.Contains(nodeGroupIDs, expiredGroup.ID) {
		return false
	}
	return expiredUserEligible(userSub, expiredGroup, now)
}

func (r *serverNodeRepo) effectiveSpeedLimit(ctx context.Context, subscribePlan *ent.ProxySubscribe, userSub *ent.ProxyUserSubscribe, now time.Time) int64 {
	if subscribePlan == nil || userSub == nil {
		return 0
	}
	baseSpeedLimit := int64(subscribePlan.SpeedLimit)
	if subscribePlan.TrafficLimit == nil || strings.TrimSpace(*subscribePlan.TrafficLimit) == "" {
		return baseSpeedLimit
	}

	var rules []trafficLimitRule
	if err := json.Unmarshal([]byte(*subscribePlan.TrafficLimit), &rules); err != nil {
		return baseSpeedLimit
	}

	for _, rule := range rules {
		var startTime time.Time
		var endTime time.Time
		switch rule.StatType {
		case "hour":
			if rule.StatValue <= 0 {
				continue
			}
			startTime = now.Add(-time.Duration(rule.StatValue) * time.Hour)
			endTime = now
		case "day":
			if rule.StatValue <= 0 {
				continue
			}
			startTime = now.AddDate(0, 0, -int(rule.StatValue))
			endTime = now
		default:
			continue
		}

		logs, err := r.data.db.ProxyTrafficLog.Query().
			Where(
				proxytrafficlog.UserIDEQ(userSub.UserID),
				proxytrafficlog.SubscribeIDEQ(userSub.ID),
				proxytrafficlog.TimestampGTE(startTime),
				proxytrafficlog.TimestampLT(endTime),
			).
			All(ctx)
		if err != nil {
			continue
		}

		var usedTraffic int64
		for _, item := range logs {
			usedTraffic += item.Upload + item.Download
		}
		usedGB := float64(usedTraffic) / (1024 * 1024 * 1024)
		if usedGB >= float64(rule.TrafficUsage) && rule.SpeedLimit > 0 {
			if baseSpeedLimit == 0 || rule.SpeedLimit < baseSpeedLimit {
				return rule.SpeedLimit
			}
		}
	}
	return baseSpeedLimit
}

type trafficLimitRule struct {
	StatType     string `json:"stat_type"`
	StatValue    int64  `json:"stat_value"`
	TrafficUsage int64  `json:"traffic_usage"`
	SpeedLimit   int64  `json:"speed_limit"`
}

func serverNodeMatchesTags(nodeTags string, tags []string) bool {
	for _, tag := range tags {
		for _, item := range strings.Split(nodeTags, ",") {
			if strings.TrimSpace(item) == tag {
				return true
			}
		}
	}
	return false
}

func serverNodeStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func serverNodeInt64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func serverNodeIsLegacyUnlimitedTime(value *time.Time) bool {
	return value != nil && value.Unix() == 0
}

func serverNodeSanitizeStringList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// sessionCheckScript is the Lua script for atomic device session check
var sessionCheckScript = redis.NewScript(`
-- KEYS[1] = device:online:{user_id}
-- KEYS[2] = device:session:{user_id}:{identifier}
-- ARGV[1] = device_limit
-- ARGV[2] = server_id
-- ARGV[3] = protocol
-- ARGV[4] = connected_at (unix timestamp string)

-- Existing session (reconnect / same IP reuse) -> allow directly
if redis.call('EXISTS', KEYS[2]) == 1 then
    redis.call('HSET', KEYS[2], 'server_id', ARGV[2], 'protocol', ARGV[3], 'last_seen', ARGV[4])
    redis.call('EXPIRE', KEYS[2], 120)
    redis.call('EXPIRE', KEYS[1], 120)
    local current = tonumber(redis.call('GET', KEYS[1])) or 0
    return {1, current}
end

-- New device/connection -> check limit
local current = tonumber(redis.call('GET', KEYS[1])) or 0
local limit = tonumber(ARGV[1])
if limit > 0 and current >= limit then
    return {0, current}
end

-- Allow -> increment count, create session
redis.call('INCR', KEYS[1])
redis.call('EXPIRE', KEYS[1], 120)
redis.call('HSET', KEYS[2], 'server_id', ARGV[2], 'protocol', ARGV[3], 'connected_at', ARGV[4])
redis.call('EXPIRE', KEYS[2], 120)
return {1, current + 1}
`)

// SessionCheck performs atomic device admission check using Redis Lua script
func (r *serverNodeRepo) SessionCheck(ctx context.Context, serverID int64, userID int64, identifier string, deviceLimit int64) (allowed bool, current int64, err error) {
	if r.data.rdb == nil {
		return true, 0, nil
	}

	onlineKey := fmt.Sprintf("%s%d", DeviceOnlineKeyPrefix, userID)
	sessionKey := fmt.Sprintf("%s%d:%s", DeviceSessionKeyPrefix, userID, identifier)

	nowStr := strconv.FormatInt(time.Now().Unix(), 10)
	serverIDStr := strconv.FormatInt(serverID, 10)
	limitStr := strconv.FormatInt(deviceLimit, 10)

	result, err := sessionCheckScript.Run(ctx, r.data.rdb,
		[]string{onlineKey, sessionKey},
		limitStr, serverIDStr, "", nowStr,
	).Int64Slice()
	if err != nil {
		r.log.Errorf("SessionCheck Lua script failed: %v", err)
		return false, 0, err
	}

	if len(result) < 2 {
		return false, 0, fmt.Errorf("unexpected Lua script result length: %d", len(result))
	}

	return result[0] == 1, result[1], nil
}

// SessionRelease releases a device session and decrements the online counter
func (r *serverNodeRepo) SessionRelease(ctx context.Context, userID int64, identifier string) error {
	if r.data.rdb == nil {
		return nil
	}

	sessionKey := fmt.Sprintf("%s%d:%s", DeviceSessionKeyPrefix, userID, identifier)
	exists, err := r.data.rdb.Exists(ctx, sessionKey).Result()
	if err != nil {
		r.log.Errorf("SessionRelease check exists failed: %v", err)
		return err
	}
	if exists > 0 {
		// Delete session key
		r.data.rdb.Del(ctx, sessionKey)
		// Decrement online count (ensure not below 0)
		onlineKey := fmt.Sprintf("%s%d", DeviceOnlineKeyPrefix, userID)
		r.data.rdb.Decr(ctx, onlineKey)
		// If count becomes 0 or negative, delete the key
		val, _ := r.data.rdb.Get(ctx, onlineKey).Int64()
		if val <= 0 {
			r.data.rdb.Del(ctx, onlineKey)
		}
	}
	return nil
}

// GetDeviceCountMode reads device_count_mode from system config
func (r *serverNodeRepo) GetDeviceCountMode(ctx context.Context) (string, error) {
	if r.data.rdb != nil {
		// Try Redis cache first
		val, err := r.data.rdb.Get(ctx, DeviceCountModeKey).Result()
		if err == nil && val != "" {
			return val, nil
		}
	}

	// Fall back to database
	entry, err := r.data.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("server"),
			proxysystem.KeyIn("device_count_mode", "DeviceCountMode"),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// Default to "ip" mode
			if r.data.rdb != nil {
				r.data.rdb.Set(ctx, DeviceCountModeKey, "ip", 5*time.Minute)
			}
			return "ip", nil
		}
		r.log.Errorf("GetDeviceCountMode query failed: %v", err)
		return "ip", err
	}

	mode := strings.TrimSpace(entry.Value)
	if mode == "" {
		mode = "ip"
	}

	// Cache in Redis
	if r.data.rdb != nil {
		r.data.rdb.Set(ctx, DeviceCountModeKey, mode, 5*time.Minute)
	}

	return mode, nil
}

// GetDeviceAdmissionEnabled reads device_admission_enabled from system config
func (r *serverNodeRepo) GetDeviceAdmissionEnabled(ctx context.Context) (bool, error) {
	if r.data.rdb != nil {
		// Try Redis cache first
		val, err := r.data.rdb.Get(ctx, DeviceAdmissionEnabledKey).Result()
		if err == nil && val != "" {
			return val == "true", nil
		}
	}

	// Fall back to database
	entry, err := r.data.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("server"),
			proxysystem.KeyIn("device_admission_enabled", "DeviceAdmissionEnabled"),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// Default to false (disabled)
			if r.data.rdb != nil {
				r.data.rdb.Set(ctx, DeviceAdmissionEnabledKey, "false", 5*time.Minute)
			}
			return false, nil
		}
		r.log.Errorf("GetDeviceAdmissionEnabled query failed: %v", err)
		return false, err
	}

	value := strings.TrimSpace(entry.Value)
	enabled := value == "true"

	// Cache in Redis
	if r.data.rdb != nil {
		r.data.rdb.Set(ctx, DeviceAdmissionEnabledKey, value, 5*time.Minute)
	}

	return enabled, nil
}

// GetUserDeviceLimit queries the device_limit for a user from their active subscription plan
func (r *serverNodeRepo) GetUserDeviceLimit(ctx context.Context, userID int64) (int64, error) {
	// Find user's active subscription
	userSub, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.UserIDEQ(userID),
			proxyusersubscribe.StatusIn(int8(0), int8(1)),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// No active subscription, no limit
			return 0, nil
		}
		return 0, fmt.Errorf("query user subscribe failed: %w", err)
	}

	// Get subscribe plan's device_limit
	plan, err := r.data.db.ProxySubscribe.Get(ctx, userSub.SubscribeID)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("query subscribe plan failed: %w", err)
	}

	return int64(plan.DeviceLimit), nil
}
