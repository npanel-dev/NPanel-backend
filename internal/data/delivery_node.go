package data

import (
	"context"
	"strings"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyserver"
	subscriptionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/subscription"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

const deliveryNodeTypeFront = "front"

type deliveryNode struct {
	display  *ent.ProxyNode
	landing  *ent.ProxyNode
	server   *ent.ProxyServer
	protocol *servermodel.Protocol
}

func resolveDeliveryNodes(ctx context.Context, db *ent.Client, nodes []*ent.ProxyNode) ([]*deliveryNode, error) {
	if len(nodes) == 0 {
		return []*deliveryNode{}, nil
	}

	targetIDs := make([]int64, 0)
	targetSeen := make(map[int64]struct{})
	for _, node := range nodes {
		if node == nil || !isFrontDeliveryNode(node) || node.TargetNodeID == nil || *node.TargetNodeID <= 0 {
			continue
		}
		if _, ok := targetSeen[*node.TargetNodeID]; ok {
			continue
		}
		targetSeen[*node.TargetNodeID] = struct{}{}
		targetIDs = append(targetIDs, *node.TargetNodeID)
	}

	targetMap := make(map[int64]*ent.ProxyNode, len(targetIDs))
	if len(targetIDs) > 0 {
		targets, err := db.ProxyNode.Query().
			Where(proxynode.IDIn(targetIDs...)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, target := range targets {
			targetMap[target.ID] = target
		}
	}

	serverIDs := make([]int64, 0)
	serverSeen := make(map[int64]struct{})
	landings := make([]*ent.ProxyNode, 0, len(nodes))
	for _, node := range nodes {
		landing := resolveLandingNode(node, targetMap)
		if landing == nil || landing.ServerID <= 0 {
			landings = append(landings, nil)
			continue
		}
		landings = append(landings, landing)
		if _, ok := serverSeen[landing.ServerID]; ok {
			continue
		}
		serverSeen[landing.ServerID] = struct{}{}
		serverIDs = append(serverIDs, landing.ServerID)
	}

	serverMap := make(map[int64]*ent.ProxyServer, len(serverIDs))
	if len(serverIDs) > 0 {
		servers, err := db.ProxyServer.Query().
			Where(proxyserver.IDIn(serverIDs...)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, server := range servers {
			serverMap[server.ID] = server
		}
	}

	result := make([]*deliveryNode, 0, len(nodes))
	for i, node := range nodes {
		landing := landings[i]
		if node == nil || landing == nil {
			continue
		}
		server := serverMap[landing.ServerID]
		if server == nil {
			continue
		}
		protocols, err := servermodel.UnmarshalProtocols(server.Protocol)
		if err != nil {
			continue
		}
		matched, _, _ := matchNodeProtocolConfig(protocols, landing.Protocol, landing.Port)
		if matched == nil {
			continue
		}
		matched.NormalizeOmniflow()
		result = append(result, &deliveryNode{
			display:  node,
			landing:  landing,
			server:   server,
			protocol: matched,
		})
	}
	return result, nil
}

func resolveLandingNode(node *ent.ProxyNode, targetMap map[int64]*ent.ProxyNode) *ent.ProxyNode {
	if node == nil {
		return nil
	}
	if !isFrontDeliveryNode(node) {
		return node
	}
	if node.TargetNodeID == nil || *node.TargetNodeID <= 0 {
		return nil
	}
	target := targetMap[*node.TargetNodeID]
	if target == nil || !target.Enabled || isFrontDeliveryNode(target) {
		return nil
	}
	return target
}

func isFrontDeliveryNode(node *ent.ProxyNode) bool {
	return node != nil && strings.EqualFold(strings.TrimSpace(node.NodeType), deliveryNodeTypeFront)
}

func deliveryNodeToSubscriptionNodeInfo(item *deliveryNode, effectiveNodeGroupID int64) *subscriptionbiz.NodeInfo {
	if item == nil || item.display == nil || item.protocol == nil {
		return nil
	}
	node := item.display
	matched := item.protocol
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
	return nodeInfo
}
