package server

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/npanel-dev/NPanel-backend/api/server/v1"
	serverBiz "github.com/npanel-dev/NPanel-backend/internal/biz/server"
)

func normalizeSimnetProtocolForResponse(protocol *serverBiz.Protocol) *serverBiz.Protocol {
	if protocol == nil || protocol.Type != "simnet" {
		return protocol
	}
	normalized := *protocol
	if strings.TrimSpace(normalized.SimnetPath) == "" {
		normalized.SimnetPath = "/simnet/session"
	}
	applySimnetResourceDefaultsForResponse(&normalized)
	if !normalized.SimnetFallbackEnabled || strings.TrimSpace(normalized.SimnetFallbackTargetHost) == "" {
		normalized.SimnetFallbackEnabled = false
		normalized.SimnetFallbackTargetScheme = ""
		normalized.SimnetFallbackTargetHost = ""
		normalized.SimnetFallbackTargetPort = 0
		normalized.SimnetFallbackHostHeader = ""
		normalized.SimnetFallbackTLSSNI = ""
	} else {
		normalized.SimnetFallbackTargetHost = strings.TrimSpace(normalized.SimnetFallbackTargetHost)
		normalized.SimnetFallbackHostHeader = strings.TrimSpace(normalized.SimnetFallbackHostHeader)
		normalized.SimnetFallbackTLSSNI = strings.TrimSpace(normalized.SimnetFallbackTLSSNI)
		switch strings.ToLower(strings.TrimSpace(normalized.SimnetFallbackTargetScheme)) {
		case "http", "https":
			normalized.SimnetFallbackTargetScheme = strings.ToLower(strings.TrimSpace(normalized.SimnetFallbackTargetScheme))
		default:
			normalized.SimnetFallbackTargetScheme = "https"
		}
	}
	if !normalized.SimnetAfEnabled {
		normalized.SimnetAfPathMode = ""
		normalized.SimnetAfPathPrefix = ""
		normalized.SimnetAfPathSuffix = ""
		normalized.SimnetAfMagicMode = ""
		normalized.SimnetAfResponseJitterMs = 0
		normalized.SimnetAfHandshakePolymorphism = false
		normalized.SimnetAfSettingsJitter = false
		normalized.SimnetAfFakeHeaderInjection = false
		return &normalized
	}
	if normalized.SimnetAfPathMode == "" {
		normalized.SimnetAfPathMode = "api"
	}
	if normalized.SimnetAfMagicMode == "" {
		normalized.SimnetAfMagicMode = "derived"
	}
	if normalized.SimnetAfResponseJitterMs == 0 {
		normalized.SimnetAfResponseJitterMs = 50
	}
	if !normalized.SimnetAfHandshakePolymorphism {
		normalized.SimnetAfHandshakePolymorphism = true
	}
	if !normalized.SimnetAfSettingsJitter {
		normalized.SimnetAfSettingsJitter = true
	}
	if !normalized.SimnetAfFakeHeaderInjection {
		normalized.SimnetAfFakeHeaderInjection = true
	}
	return &normalized
}

func applySimnetResourceDefaultsForResponse(protocol *serverBiz.Protocol) {
	if protocol.SimnetInboundMaxStreamsPerSession <= 0 {
		protocol.SimnetInboundMaxStreamsPerSession = 128
	}
	if protocol.SimnetInboundMaxUDPStreamsPerSession <= 0 {
		protocol.SimnetInboundMaxUDPStreamsPerSession = 64
	}
	if protocol.SimnetInboundMaxHandlerTasksPerSession <= 0 {
		protocol.SimnetInboundMaxHandlerTasksPerSession = 128
	}
	if protocol.SimnetStreamEventChannelCapacity <= 0 {
		protocol.SimnetStreamEventChannelCapacity = 256
	}
	if protocol.SimnetStreamDataChannelCapacity <= 0 {
		protocol.SimnetStreamDataChannelCapacity = 128
	}
	if protocol.SimnetTargetDialTimeoutMs <= 0 {
		protocol.SimnetTargetDialTimeoutMs = 12_000
	}
	if protocol.SimnetTargetMaxConcurrentDials <= 0 {
		protocol.SimnetTargetMaxConcurrentDials = 256
	}
	if protocol.SimnetSendWindow <= 0 {
		protocol.SimnetSendWindow = 4 * 1024 * 1024
	}
	if protocol.SimnetRecvWindow <= 0 {
		protocol.SimnetRecvWindow = 4 * 1024 * 1024
	}
	if protocol.SimnetMaxConcurrentStreams <= 0 {
		protocol.SimnetMaxConcurrentStreams = 100
	}
	if protocol.SimnetInitialWindowSize <= 0 {
		protocol.SimnetInitialWindowSize = 65_535
	}
	if protocol.SimnetMaxFrameSize <= 0 {
		protocol.SimnetMaxFrameSize = 16_384
	}
	if protocol.SimnetClientMaxConcurrentStreams <= 0 {
		protocol.SimnetClientMaxConcurrentStreams = 32
	}
	if protocol.SimnetClientMaxStreamsPerSession <= 0 {
		protocol.SimnetClientMaxStreamsPerSession = 512
	}
	if protocol.SimnetClientSessionIdleTimeoutSecs <= 0 {
		protocol.SimnetClientSessionIdleTimeoutSecs = 90
	}
	if protocol.SimnetClientMaxUDPSessions <= 0 {
		protocol.SimnetClientMaxUDPSessions = 64
	}
}

func normalizeMundoProtocolForResponse(protocol *serverBiz.Protocol) *serverBiz.Protocol {
	if protocol == nil {
		return protocol
	}
	if protocol.Type != "mx" || !isMundoTransport(protocol.Transport) {
		if strings.TrimSpace(protocol.MundoUsername) == "" &&
			strings.TrimSpace(protocol.MundoCertificateFingerprint) == "" &&
			strings.TrimSpace(protocol.MundoFakeTitle) == "" &&
			strings.TrimSpace(protocol.MundoFakeMessage) == "" &&
			!protocol.MundoAcceptProxyProtocol &&
			!protocol.MundoUseTLSCertificate {
			return protocol
		}
		normalized := *protocol
		normalized.MundoUsername = ""
		normalized.MundoCertificateFingerprint = ""
		normalized.MundoFakeTitle = ""
		normalized.MundoFakeMessage = ""
		normalized.MundoAcceptProxyProtocol = false
		normalized.MundoUseTLSCertificate = false
		return &normalized
	}
	if strings.TrimSpace(protocol.MundoUsername) != "" {
		return protocol
	}
	normalized := *protocol
	normalized.MundoUsername = "MundoUser"
	return &normalized
}

func isMundoTransport(transport string) bool {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "mundordp", "mundosql":
		return true
	default:
		return false
	}
}

// ServerService 节点服务器服务
type ServerService struct {
	v1.UnimplementedServerServer

	uc  *serverBiz.ServerNodeUsecase
	log *log.Helper
}

// NewServerService 创建节点服务器服务
func NewServerService(uc *serverBiz.ServerNodeUsecase, logger log.Logger) *ServerService {
	return &ServerService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

// GetServerConfig 获取服务器配置
func (s *ServerService) GetServerConfig(ctx context.Context, req *v1.GetServerConfigRequest) (*v1.GetServerConfigReply, error) {
	config, err := s.uc.GetServerConfig(ctx, req.ServerId, req.Protocol, req.SecretKey)
	if err != nil {
		return nil, err
	}

	// 解析config JSON字符串为map
	var configMap map[string]string
	if err := json.Unmarshal([]byte(config.Config), &configMap); err != nil {
		s.log.Errorf("Failed to parse config: %v", err)
		configMap = make(map[string]string)
	}

	// 获取设备计数模式和准入控制开关
	deviceCountMode, _ := s.uc.GetDeviceCountMode(ctx)
	deviceAdmissionEnabled, _ := s.uc.GetDeviceAdmissionEnabled(ctx)

	return &v1.GetServerConfigReply{
		Code:    0,
		Message: "success",
		Basic: &v1.ServerBasic{
			PushInterval:           config.PushInterval,
			PullInterval:           config.PullInterval,
			DeviceCountMode:        deviceCountMode,
			DeviceAdmissionEnabled: deviceAdmissionEnabled,
		},
		Protocol: config.Protocol,
		Config:   configMap,
	}, nil
}

// GetServerUserList 获取服务器用户列表
func (s *ServerService) GetServerUserList(ctx context.Context, req *v1.GetServerUserListRequest) (*v1.GetServerUserListReply, error) {
	users, err := s.uc.GetServerUserList(ctx, req.ServerId, req.Protocol, req.SecretKey)
	if err != nil {
		return nil, err
	}

	userList := make([]*v1.ServerUser, 0, len(users))
	for _, user := range users {
		userList = append(userList, &v1.ServerUser{
			Id:          user.ID,
			Uuid:        user.UUID,
			SpeedLimit:  user.SpeedLimit,
			DeviceLimit: user.DeviceLimit,
		})
	}

	return &v1.GetServerUserListReply{
		Code:    0,
		Message: "success",
		Users:   userList,
	}, nil
}

// PushUserTraffic 推送用户流量
func (s *ServerService) PushUserTraffic(ctx context.Context, req *v1.PushUserTrafficRequest) (*v1.PushUserTrafficReply, error) {
	// 转换Traffic数据
	traffic := make([]*serverBiz.UserTraffic, 0, len(req.Traffic))
	for _, t := range req.Traffic {
		traffic = append(traffic, &serverBiz.UserTraffic{
			SID:      t.Sid,
			Upload:   t.Upload,
			Download: t.Download,
		})
	}

	bizReq := &serverBiz.PushUserTrafficRequest{
		ServerID:  req.ServerId,
		Protocol:  req.Protocol,
		SecretKey: req.SecretKey,
		Traffic:   traffic,
	}

	err := s.uc.PushUserTraffic(ctx, bizReq)
	if err != nil {
		return nil, err
	}

	return &v1.PushUserTrafficReply{
		Code:    0,
		Message: "success",
	}, nil
}

// PushServerStatus 推送服务器状态
func (s *ServerService) PushServerStatus(ctx context.Context, req *v1.PushServerStatusRequest) (*v1.PushServerStatusReply, error) {
	bizReq := &serverBiz.PushServerStatusRequest{
		ServerID:  req.ServerId,
		Protocol:  req.Protocol,
		SecretKey: req.SecretKey,
		CPU:       req.Cpu,
		Mem:       req.Mem,
		Disk:      req.Disk,
		UpdatedAt: req.UpdatedAt,
	}

	err := s.uc.PushServerStatus(ctx, bizReq)
	if err != nil {
		return nil, err
	}

	return &v1.PushServerStatusReply{
		Code:    0,
		Message: "success",
	}, nil
}

// PushOnlineUsers 推送在线用户
func (s *ServerService) PushOnlineUsers(ctx context.Context, req *v1.PushOnlineUsersRequest) (*v1.PushOnlineUsersReply, error) {
	// 转换Users数据
	users := make([]*serverBiz.OnlineUser, 0, len(req.Users))
	for _, u := range req.Users {
		users = append(users, &serverBiz.OnlineUser{
			SID: u.Sid,
			IP:  u.Ip,
		})
	}

	bizReq := &serverBiz.PushOnlineUsersRequest{
		ServerID:  req.ServerId,
		Protocol:  req.Protocol,
		SecretKey: req.SecretKey,
		Users:     users,
	}

	err := s.uc.PushOnlineUsers(ctx, bizReq)
	if err != nil {
		return nil, err
	}

	return &v1.PushOnlineUsersReply{
		Code:    0,
		Message: "success",
	}, nil
}

// QueryServerProtocolConfig 查询服务器协议配置
func (s *ServerService) QueryServerProtocolConfig(ctx context.Context, req *v1.QueryServerProtocolConfigRequest) (*v1.QueryServerProtocolConfigReply, error) {
	s.log.Infof(
		"[QueryServerProtocolConfig] request received server_id=%d secret_present=%t protocols=%v",
		req.ServerId,
		req.SecretKey != "",
		req.Protocols,
	)
	config, err := s.uc.QueryServerProtocolConfig(ctx, req.ServerId, req.SecretKey, req.Protocols)
	if err != nil {
		s.log.Errorf("[QueryServerProtocolConfig] usecase failed server_id=%d err=%v", req.ServerId, err)
		return nil, err
	}
	s.log.Infof(
		"[QueryServerProtocolConfig] usecase returned server_id=%d total=%d dns=%d outbound=%d protocols=%d",
		req.ServerId,
		config.Total,
		len(config.DNS),
		len(config.Outbound),
		len(config.Protocols),
	)

	// 转换DNS配置
	dnsConfigs := make([]*v1.NodeDNS, 0, len(config.DNS))
	for _, dns := range config.DNS {
		dnsConfigs = append(dnsConfigs, &v1.NodeDNS{
			Server: dns.Server,
			Domain: dns.Domain,
			Port:   dns.Port,
		})
	}

	// 转换Outbound配置
	outboundConfigs := make([]*v1.NodeOutbound, 0, len(config.Outbound))
	for _, outbound := range config.Outbound {
		outboundConfigs = append(outboundConfigs, &v1.NodeOutbound{
			Tag:      outbound.Tag,
			Protocol: outbound.Protocol,
			Settings: outbound.Settings,
		})
	}

	// 转换Protocol配置
	protocolConfigs := make([]*v1.Protocol, 0, len(config.Protocols))
	for _, protocol := range config.Protocols {
		if !protocol.Enable {
			continue
		}
		protocol = normalizeSimnetProtocolForResponse(protocol)
		protocol = normalizeMundoProtocolForResponse(protocol)
		protocolConfigs = append(protocolConfigs, &v1.Protocol{
			Type:                                   protocol.Type,
			Port:                                   protocol.Port,
			Enable:                                 protocol.Enable,
			Security:                               protocol.Security,
			Sni:                                    protocol.SNI,
			AllowInsecure:                          protocol.AllowInsecure,
			Fingerprint:                            protocol.Fingerprint,
			RealityServerAddr:                      protocol.RealityServerAddr,
			RealityServerPort:                      protocol.RealityServerPort,
			RealityPrivateKey:                      protocol.RealityPrivateKey,
			RealityPublicKey:                       protocol.RealityPublicKey,
			RealityShortId:                         protocol.RealityShortId,
			Transport:                              protocol.Transport,
			Host:                                   protocol.Host,
			Path:                                   protocol.Path,
			ServiceName:                            protocol.ServiceName,
			Mc1Mode:                                protocol.Mc1Mode,
			Mc1CidrSegments:                        protocol.Mc1CidrSegments,
			MundoUsername:                          protocol.MundoUsername,
			MundoCertificateFingerprint:            protocol.MundoCertificateFingerprint,
			MundoFakeTitle:                         protocol.MundoFakeTitle,
			MundoFakeMessage:                       protocol.MundoFakeMessage,
			MundoAcceptProxyProtocol:               protocol.MundoAcceptProxyProtocol,
			MundoUseTlsCertificate:                 protocol.MundoUseTLSCertificate,
			Cipher:                                 protocol.Cipher,
			ServerKey:                              protocol.ServerKey,
			Flow:                                   protocol.Flow,
			HopPorts:                               protocol.HopPorts,
			HopInterval:                            protocol.HopInterval,
			ObfsPassword:                           protocol.ObfsPassword,
			DisableSni:                             protocol.DisableSNI,
			ReduceRtt:                              protocol.ReduceRtt,
			UdpRelayMode:                           protocol.UDPRelayMode,
			CongestionController:                   protocol.CongestionController,
			Multiplex:                              protocol.Multiplex,
			PaddingScheme:                          protocol.PaddingScheme,
			UpMbps:                                 protocol.UpMbps,
			DownMbps:                               protocol.DownMbps,
			Obfs:                                   protocol.Obfs,
			ObfsHost:                               protocol.ObfsHost,
			ObfsPath:                               protocol.ObfsPath,
			XhttpMode:                              protocol.XhttpMode,
			XhttpExtra:                             protocol.XhttpExtra,
			Encryption:                             protocol.Encryption,
			EncryptionMode:                         protocol.EncryptionMode,
			EncryptionRtt:                          protocol.EncryptionRtt,
			EncryptionTicket:                       protocol.EncryptionTicket,
			EncryptionServerPadding:                protocol.EncryptionServerPadding,
			EncryptionPrivateKey:                   protocol.EncryptionPrivateKey,
			EncryptionClientPadding:                protocol.EncryptionClientPadding,
			EncryptionPassword:                     protocol.EncryptionPassword,
			Ratio:                                  protocol.Ratio,
			CertMode:                               protocol.CertMode,
			CertDnsProvider:                        protocol.CertDNSProvider,
			CertDnsEnv:                             protocol.CertDNSEnv,
			SimnetPsk:                              protocol.SimnetPsk,
			SimnetKeyId:                            protocol.SimnetKeyID,
			SimnetTicketId:                         protocol.SimnetTicketID,
			SimnetPath:                             protocol.SimnetPath,
			SimnetCarrier:                          protocol.SimnetCarrier,
			SimnetAfEnabled:                        protocol.SimnetAfEnabled,
			SimnetAfPathMode:                       protocol.SimnetAfPathMode,
			SimnetAfPathPrefix:                     protocol.SimnetAfPathPrefix,
			SimnetAfPathSuffix:                     protocol.SimnetAfPathSuffix,
			SimnetAfMagicMode:                      protocol.SimnetAfMagicMode,
			SimnetAfResponseJitterMs:               protocol.SimnetAfResponseJitterMs,
			SimnetAfHandshakePolymorphism:          protocol.SimnetAfHandshakePolymorphism,
			SimnetAfSettingsJitter:                 protocol.SimnetAfSettingsJitter,
			SimnetAfFakeHeaderInjection:            protocol.SimnetAfFakeHeaderInjection,
			SimnetReverseEnabled:                   protocol.SimnetReverseEnabled,
			SimnetReverseListenAddr:                protocol.SimnetReverseListenAddr,
			SimnetReverseListenPort:                protocol.SimnetReverseListenPort,
			SimnetReverseTargetHost:                protocol.SimnetReverseTargetHost,
			SimnetReverseTargetPort:                protocol.SimnetReverseTargetPort,
			SimnetFallbackEnabled:                  protocol.SimnetFallbackEnabled,
			SimnetFallbackTargetScheme:             protocol.SimnetFallbackTargetScheme,
			SimnetFallbackTargetHost:               protocol.SimnetFallbackTargetHost,
			SimnetFallbackTargetPort:               protocol.SimnetFallbackTargetPort,
			SimnetFallbackHostHeader:               protocol.SimnetFallbackHostHeader,
			SimnetFallbackTlsSni:                   protocol.SimnetFallbackTLSSNI,
			SimnetInboundMaxStreamsPerSession:      protocol.SimnetInboundMaxStreamsPerSession,
			SimnetInboundMaxUdpStreamsPerSession:   protocol.SimnetInboundMaxUDPStreamsPerSession,
			SimnetInboundMaxHandlerTasksPerSession: protocol.SimnetInboundMaxHandlerTasksPerSession,
			SimnetStreamEventChannelCapacity:       protocol.SimnetStreamEventChannelCapacity,
			SimnetStreamDataChannelCapacity:        protocol.SimnetStreamDataChannelCapacity,
			SimnetTargetDialTimeoutMs:              protocol.SimnetTargetDialTimeoutMs,
			SimnetTargetMaxConcurrentDials:         protocol.SimnetTargetMaxConcurrentDials,
			SimnetEgressBlockLoopback:              protocol.SimnetEgressBlockLoopback,
			SimnetEgressBlockPrivate:               protocol.SimnetEgressBlockPrivate,
			SimnetEgressBlockLinkLocal:             protocol.SimnetEgressBlockLinkLocal,
			SimnetEgressBlockMetadata:              protocol.SimnetEgressBlockMetadata,
			SimnetSendWindow:                       protocol.SimnetSendWindow,
			SimnetRecvWindow:                       protocol.SimnetRecvWindow,
			SimnetMaxConcurrentStreams:             protocol.SimnetMaxConcurrentStreams,
			SimnetInitialWindowSize:                protocol.SimnetInitialWindowSize,
			SimnetMaxFrameSize:                     protocol.SimnetMaxFrameSize,
			SimnetClientMaxConcurrentStreams:       protocol.SimnetClientMaxConcurrentStreams,
			SimnetClientMaxStreamsPerSession:       protocol.SimnetClientMaxStreamsPerSession,
			SimnetClientSessionIdleTimeoutSecs:     protocol.SimnetClientSessionIdleTimeoutSecs,
			SimnetClientMaxUdpSessions:             protocol.SimnetClientMaxUDPSessions,
			// OmniFlow
			OmniflowCarrier:                    protocol.OmniflowCarrier,
			OmniflowPath:                       protocol.OmniflowPath,
			OmniflowContentType:                protocol.OmniflowContentType,
			OmniflowProfilePath:                protocol.OmniflowProfilePath,
			OmniflowProfileJson:                protocol.OmniflowProfileJson,
			OmniflowServerHost:                 protocol.OmniflowServerHost,
			OmniflowServerPort:                 protocol.OmniflowServerPort,
			OmniflowCaCertPath:                 protocol.OmniflowCaCertPath,
			OmniflowTargetMeta:                 protocol.OmniflowTargetMeta,
			OmniflowSpkiPin:                    protocol.OmniflowSpkiPin,
			OmniflowH3FallbackEnabled:          protocol.OmniflowH3FallbackEnabled,
			OmniflowH3FallbackPolicy:           protocol.OmniflowH3FallbackPolicy,
			OmniflowH3FallbackTimeoutMs:        protocol.OmniflowH3FallbackTimeoutMs,
			OmniflowH3FallbackRetryBudget:      protocol.OmniflowH3FallbackRetryBudget,
			OmniflowH3FallbackSmokeEnabled:     protocol.OmniflowH3FallbackSmokeEnabled,
			OmniflowH3FallbackSmokeIntervalSec: protocol.OmniflowH3FallbackSmokeIntervalSec,
			OmniflowH3FallbackSmokeTimeoutMs:   protocol.OmniflowH3FallbackSmokeTimeoutMs,
			OmniflowMaxAgeSec:                  protocol.OmniflowMaxAgeSec,
			OmniflowIdleTimeoutSec:             protocol.OmniflowIdleTimeoutSec,
			OmniflowMaxConnections:             protocol.OmniflowMaxConnections,
			OmniflowAdaptiveTlsEnabled:         protocol.OmniflowAdaptiveTlsEnabled,
			OmniflowTlsFingerprint:             protocol.OmniflowTlsFingerprint,
			OmniflowSniMode:                    protocol.OmniflowSniMode,
			OmniflowPaddingMode:                protocol.OmniflowPaddingMode,
			OmniflowTrafficShapingEnabled:      protocol.OmniflowTrafficShapingEnabled,
			OmniflowAfEnabled:                  protocol.OmniflowAfEnabled,
			OmniflowAfPathMode:                 protocol.OmniflowAfPathMode,
			OmniflowAfPathPrefix:               protocol.OmniflowAfPathPrefix,
			OmniflowAfPathSuffix:               protocol.OmniflowAfPathSuffix,
			OmniflowAfPathRotationSecs:         protocol.OmniflowAfPathRotationSecs,
			OmniflowAfPathSkewSlots:            protocol.OmniflowAfPathSkewSlots,
			OmniflowFallbackEnabled:            protocol.OmniflowFallbackEnabled,
			OmniflowFallbackTargetScheme:       protocol.OmniflowFallbackTargetScheme,
			OmniflowFallbackTargetHost:         protocol.OmniflowFallbackTargetHost,
			OmniflowFallbackTargetPort:         protocol.OmniflowFallbackTargetPort,
			OmniflowFallbackHostHeader:         protocol.OmniflowFallbackHostHeader,
			OmniflowFallbackTlsSni:             protocol.OmniflowFallbackTLSSNI,
			OmniflowFallbackCarrierEnabled:     protocol.OmniflowFallbackCarrierEnabled,
			OmniflowFallbackConnectTunnel:      protocol.OmniflowFallbackConnectTunnel,
			OmniflowFallbackWssEnabled:         protocol.OmniflowFallbackWssEnabled,
		})
	}

	reply := &v1.QueryServerProtocolConfigReply{
		Code:                   0,
		Message:                "success",
		TrafficReportThreshold: config.TrafficReportThreshold,
		IpStrategy:             config.IPStrategy,
		Dns:                    dnsConfigs,
		Block:                  config.Block,
		Outbound:               outboundConfigs,
		Protocols:              protocolConfigs,
		Total:                  int32(config.Total),
	}
	s.log.Infof(
		"[QueryServerProtocolConfig] reply ready server_id=%d total=%d dns=%d outbound=%d protocols=%d",
		req.ServerId,
		reply.Total,
		len(reply.Dns),
		len(reply.Outbound),
		len(reply.Protocols),
	)
	return reply, nil
}

// SessionCheck 会话准入检查
func (s *ServerService) SessionCheck(ctx context.Context, req *v1.SessionCheckRequest) (*v1.SessionCheckResponse, error) {
	return s.uc.SessionCheck(ctx, req)
}

// SessionRelease 会话释放
func (s *ServerService) SessionRelease(ctx context.Context, req *v1.SessionReleaseRequest) (*v1.SessionReleaseResponse, error) {
	return s.uc.SessionRelease(ctx, req)
}
