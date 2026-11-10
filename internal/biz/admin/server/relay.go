package server

import (
	"context"
	"strings"

	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
)

// RelayEntryParams 中转入口链路向导参数（连接端口/服务端口分离 + 差异化倍率）
type RelayEntryParams struct {
	Name         string   // 中转入口名称
	Address      string   // 中转商地址（用户连接）
	Port         uint16   // 连接端口（用户连接中转商）
	SourceNodeID int64    // 源落地节点（复制其协议实例）
	ServicePort  uint16   // 服务端口（中转商转发到落地 VPS，新协议实例端口）
	Ratio        float64  // 链路倍率
	Tags         []string // front 节点标签
	NodeGroupIDs []int64  // front 节点分组
}

// RelayEntry 链路创建所需上下文（biz 校验并克隆实例后交给 repo 事务执行）
type RelayEntry struct {
	SourceNode  *Node                 // 源落地节点
	NewInstance *servermodel.Protocol // 克隆出的新协议实例（新端口+新倍率，repo 直接持久化）
	Params      *RelayEntryParams
}

// CreateRelayEntry 创建中转入口链路：
// 复制源落地节点的协议实例（新服务端口 + 新倍率）→ 创建隐藏中间落地节点 → 创建 front 中转入口。
// 倍率挂在独立协议实例上，流量计费与订阅展示天然一致。
func (uc *NodeUsecase) CreateRelayEntry(ctx context.Context, params *RelayEntryParams) (*Node, error) {
	if err := uc.validateRelayEntry(ctx, params); err != nil {
		return nil, err
	}
	source, _ := uc.repo.GetNodeByID(ctx, params.SourceNodeID)
	protocols, err := uc.repo.GetServerProtocols(ctx, int(source.ServerID))
	if err != nil {
		return nil, err
	}
	sourceInstance := findEnabledProtocolInstance(protocols, source.Protocol, source.Port)
	if sourceInstance == nil {
		return nil, servermodel.ErrInvalidProtocolConfig
	}
	instance := cloneProtocolInstance(sourceInstance)
	instance.Port = int32(params.ServicePort)
	instance.Ratio = params.Ratio
	instance.Enable = true
	node, err := uc.repo.CreateRelayEntry(ctx, &RelayEntry{
		SourceNode:  source,
		NewInstance: instance,
		Params:      params,
	})
	if err != nil {
		return nil, err
	}
	if err := uc.repo.ClearNodeCache(ctx, []int{int(source.ServerID)}); err != nil {
		uc.log.Warnf("Failed to clear node cache for server %d after creating relay entry: %v", source.ServerID, err)
	}
	return node, nil
}

// cloneProtocolInstance 深拷贝协议实例（切片字段独立，避免共享底层数组）
func cloneProtocolInstance(source *servermodel.Protocol) *servermodel.Protocol {
	if source == nil {
		return nil
	}
	clone := *source
	if source.Mc1CidrSegments != nil {
		clone.Mc1CidrSegments = append([]string(nil), source.Mc1CidrSegments...)
	}
	return &clone
}

func (uc *NodeUsecase) validateRelayEntry(ctx context.Context, params *RelayEntryParams) error {
	if params == nil || strings.TrimSpace(params.Name) == "" || strings.TrimSpace(params.Address) == "" ||
		params.Port == 0 || params.ServicePort == 0 || params.SourceNodeID <= 0 || params.Ratio <= 0 {
		return servermodel.ErrInvalidProtocolConfig
	}
	source, err := uc.repo.GetNodeByID(ctx, params.SourceNodeID)
	if err != nil {
		return err
	}
	if source == nil || normalizeNodeType(source.NodeType) != NodeTypeLanding || source.Enabled == nil || !*source.Enabled {
		return servermodel.ErrInvalidProtocolConfig
	}
	protocols, err := uc.repo.GetServerProtocols(ctx, int(source.ServerID))
	if err != nil {
		return err
	}
	if findEnabledProtocolInstance(protocols, source.Protocol, source.Port) == nil {
		return servermodel.ErrInvalidProtocolConfig
	}
	// 服务端口不能与该服务器上任何已启用实例的端口重复
	for _, item := range protocols {
		if item == nil || !item.Enable {
			continue
		}
		if item.Port == int32(params.ServicePort) {
			return servermodel.ErrDuplicateProtocolPort
		}
	}
	return nil
}

// UpdateNodeRatio 修改节点链路倍率：front 节点沿落地目标解析到协议实例后修改。
// 协议实例被多个落地节点共享时拒绝修改（无法按入口拆分流量归属）。
func (uc *NodeUsecase) UpdateNodeRatio(ctx context.Context, nodeID int64, ratio float64) error {
	if nodeID <= 0 || ratio <= 0 {
		return servermodel.ErrInvalidProtocolConfig
	}
	node, err := uc.repo.GetNodeByID(ctx, nodeID)
	if err != nil {
		return err
	}
	if node == nil {
		return servermodel.ErrInvalidProtocolConfig
	}
	target := node
	if normalizeNodeType(node.NodeType) == NodeTypeFront {
		if node.TargetNodeID == nil || *node.TargetNodeID <= 0 {
			return servermodel.ErrInvalidProtocolConfig
		}
		target, err = uc.repo.GetNodeByID(ctx, *node.TargetNodeID)
		if err != nil {
			return err
		}
		if target == nil || normalizeNodeType(target.NodeType) != NodeTypeLanding {
			return servermodel.ErrInvalidProtocolConfig
		}
	}
	if target.ServerID <= 0 || strings.TrimSpace(target.Protocol) == "" || target.Port == 0 {
		return servermodel.ErrInvalidProtocolConfig
	}
	count, err := uc.repo.CountLandingNodesByInstance(ctx, target.ServerID, target.Protocol, target.Port)
	if err != nil {
		return err
	}
	if count > 1 {
		return servermodel.ErrSharedProtocolInstance
	}
	if err := uc.repo.UpdateProtocolRatio(ctx, target.ServerID, target.Protocol, target.Port, ratio); err != nil {
		return err
	}
	if err := uc.repo.ClearNodeCache(ctx, []int{int(target.ServerID)}); err != nil {
		uc.log.Warnf("Failed to clear node cache for server %d after updating node ratio: %v", target.ServerID, err)
	}
	return nil
}

// findEnabledProtocolInstance 按 type（忽略大小写）+端口精确匹配启用的协议实例
func findEnabledProtocolInstance(protocols []*servermodel.Protocol, protocol string, port uint16) *servermodel.Protocol {
	for _, item := range protocols {
		if item == nil || !item.Enable {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Type), strings.TrimSpace(protocol)) && item.Port == int32(port) {
			return item
		}
	}
	return nil
}
