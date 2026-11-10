package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyserver"
	serverbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/server"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

// CreateRelayEntry 事务执行中转链路创建：
// 1. 将 biz 层克隆好的新协议实例（新服务端口 + 新倍率）追加到服务器协议列表
// 2. 创建隐藏的中间落地节点（指向新实例）
// 3. 创建 front 中转入口节点（target 指向中间落地节点）
func (r *adminNodeRepo) CreateRelayEntry(ctx context.Context, entry *serverbiz.RelayEntry) (*serverbiz.Node, error) {
	if entry == nil || entry.SourceNode == nil || entry.NewInstance == nil || entry.Params == nil {
		return nil, servermodel.ErrInvalidProtocolConfig
	}
	params := entry.Params
	serverID := entry.SourceNode.ServerID

	var front *ent.ProxyNode
	err := withTx(ctx, r.data.db, func(tx *ent.Tx) error {
		server, err := tx.ProxyServer.Query().Where(proxyserver.IDEQ(serverID)).Only(ctx)
		if err != nil {
			return err
		}
		protocols, err := servermodel.UnmarshalProtocols(server.Protocol)
		if err != nil {
			return err
		}
		// 事务内再校验一次服务端口冲突（防并发创建）
		for _, item := range protocols {
			if item == nil || !item.Enable {
				continue
			}
			if item.Port == int32(params.ServicePort) {
				return servermodel.ErrDuplicateProtocolPort
			}
		}

		protocols = append(protocols, entry.NewInstance)
		protocolsJSON, err := servermodel.MarshalProtocols(protocols)
		if err != nil {
			return err
		}
		if err := tx.ProxyServer.UpdateOneID(serverID).SetProtocol(protocolsJSON).Exec(ctx); err != nil {
			return err
		}

		// 中间落地节点：对用户隐藏（is_hidden），仅管理员可见，承载新协议实例
		landing, err := tx.ProxyNode.Create().
			SetName(params.Name + "-landing").
			SetPort(params.ServicePort).
			SetAddress(params.Address).
			SetServerID(serverID).
			SetProtocol(entry.SourceNode.Protocol).
			SetNodeType(serverbiz.NodeTypeLanding).
			SetIsHidden(true).
			SetEnabled(true).
			SetNodeGroupIds(params.NodeGroupIDs).
			Save(ctx)
		if err != nil {
			return err
		}

		// front 中转入口节点：连接地址/端口 + 指向中间落地
		front, err = tx.ProxyNode.Create().
			SetName(params.Name).
			SetTags(tool.StringSliceToString(params.Tags)).
			SetPort(params.Port).
			SetAddress(params.Address).
			SetServerID(serverID).
			SetProtocol(entry.SourceNode.Protocol).
			SetNodeType(serverbiz.NodeTypeFront).
			SetEnabled(true).
			SetTargetNodeID(landing.ID).
			SetNodeGroupIds(params.NodeGroupIDs).
			Save(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return adminNodeToBiz(front), nil
}

// CountLandingNodesByInstance 统计引用该协议实例（server+protocol+port）的启用落地节点数
func (r *adminNodeRepo) CountLandingNodesByInstance(ctx context.Context, serverID int64, protocol string, port uint16) (int, error) {
	return r.data.db.ProxyNode.Query().
		Where(
			proxynode.ServerIDEQ(serverID),
			proxynode.ProtocolEQ(strings.TrimSpace(protocol)),
			proxynode.PortEQ(port),
			proxynode.NodeTypeEQ(serverbiz.NodeTypeLanding),
			proxynode.EnabledEQ(true),
		).
		Count(ctx)
}

// UpdateProtocolRatio 修改服务器协议实例的倍率（按 type+port 定位）
func (r *adminNodeRepo) UpdateProtocolRatio(ctx context.Context, serverID int64, protocol string, port uint16, ratio float64) error {
	server, err := r.data.db.ProxyServer.Query().Where(proxyserver.IDEQ(serverID)).Only(ctx)
	if err != nil {
		return err
	}
	protocols, err := servermodel.UnmarshalProtocols(server.Protocol)
	if err != nil {
		return err
	}
	found := false
	for _, item := range protocols {
		if item == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Type), strings.TrimSpace(protocol)) && item.Port == int32(port) {
			item.Ratio = ratio
			found = true
			break
		}
	}
	if !found {
		return servermodel.ErrInvalidProtocolConfig
	}
	protocolsJSON, err := servermodel.MarshalProtocols(protocols)
	if err != nil {
		return err
	}
	return r.data.db.ProxyServer.UpdateOneID(serverID).SetProtocol(protocolsJSON).Exec(ctx)
}

// attachNodeRatios 为节点列表批量填充链路倍率（复用订阅下发的 delivery 解析，避免 N+1）
func (r *adminNodeRepo) attachNodeRatios(ctx context.Context, entNodes []*ent.ProxyNode, nodes []*serverbiz.Node) {
	deliveries, err := resolveDeliveryNodes(ctx, r.data.db, entNodes)
	if err != nil {
		r.log.Warnf("Failed to resolve node ratios: %v", err)
		return
	}
	ratioMap := make(map[int64]float64, len(deliveries))
	for _, d := range deliveries {
		if d == nil || d.protocol == nil || d.display == nil {
			continue
		}
		ratio := d.protocol.Ratio
		if ratio <= 0 {
			ratio = 1
		}
		ratioMap[d.display.ID] = ratio
	}
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if ratio, ok := ratioMap[node.ID]; ok {
			node.Ratio = ratio
		} else {
			node.Ratio = 1
		}
	}
}

// withTx 事务包装（出错自动回滚，成功提交）
func withTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()
	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("tx rollback: %v (original error: %w)", rollbackErr, err)
		}
		return err
	}
	return tx.Commit()
}
