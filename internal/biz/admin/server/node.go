package server

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
)

const (
	NodeTypeLanding = "landing"
	NodeTypeFront   = "front"
)

type NodeUsecase struct {
	repo NodeRepo
	log  *log.Helper
}

func NewNodeUsecase(repo NodeRepo, logger log.Logger) *NodeUsecase {
	return &NodeUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *NodeUsecase) CreateNode(ctx context.Context, name string, tags []string, port uint16, address string, serverID int64, protocol string, enabled *bool, nodeType string, isHidden *bool, nodeGroupIDs []int64, targetNodeID *int64) (*Node, error) {
	nodeType = normalizeNodeType(nodeType)
	if err := uc.validateNodeConfig(ctx, 0, serverID, protocol, port, nodeType, targetNodeID); err != nil {
		return nil, err
	}
	return uc.repo.CreateNode(ctx, &Node{
		Name:         name,
		Tags:         tags,
		Port:         port,
		Address:      address,
		ServerID:     serverID,
		Protocol:     protocol,
		Enabled:      enabled,
		NodeType:     nodeType,
		IsHidden:     isHidden,
		TargetNodeID: normalizedTargetNodeID(nodeType, targetNodeID),
		NodeGroupIDs: nodeGroupIDs,
	})
}

func (uc *NodeUsecase) UpdateNode(ctx context.Context, id int, name string, tags []string, port uint16, address string, serverID int64, protocol string, enabled *bool, nodeType string, isHidden *bool, nodeGroupIDs []int64, targetNodeID *int64) (*Node, error) {
	nodeType = normalizeNodeType(nodeType)
	oldNode, _ := uc.repo.GetNodeByID(ctx, int64(id))
	if err := uc.validateNodeConfig(ctx, int64(id), serverID, protocol, port, nodeType, targetNodeID); err != nil {
		return nil, err
	}
	node := &Node{
		ID:           int64(id),
		Name:         name,
		Tags:         tags,
		Port:         port,
		Address:      address,
		ServerID:     serverID,
		Protocol:     protocol,
		Enabled:      enabled,
		NodeType:     nodeType,
		IsHidden:     isHidden,
		TargetNodeID: normalizedTargetNodeID(nodeType, targetNodeID),
		NodeGroupIDs: nodeGroupIDs,
	}
	updatedNode, err := uc.repo.UpdateNode(ctx, node)
	if err != nil {
		return nil, err
	}
	serverIDs := uc.nodeServerIDsForCache(ctx, oldNode, updatedNode)
	if err := uc.repo.ClearNodeCache(ctx, serverIDs); err != nil {
		uc.log.Warnf("Failed to clear node cache for node %d servers %v: %v", id, serverIDs, err)
	}
	return updatedNode, nil
}

func normalizeNodeType(nodeType string) string {
	if strings.EqualFold(strings.TrimSpace(nodeType), NodeTypeFront) {
		return NodeTypeFront
	}
	return NodeTypeLanding
}

func normalizedTargetNodeID(nodeType string, targetNodeID *int64) *int64 {
	if nodeType != NodeTypeFront || targetNodeID == nil || *targetNodeID <= 0 {
		return nil
	}
	return targetNodeID
}

func (uc *NodeUsecase) validateNodeConfig(ctx context.Context, id int64, serverID int64, protocol string, port uint16, nodeType string, targetNodeID *int64) error {
	if nodeType == NodeTypeFront {
		return uc.validateFrontNode(ctx, id, port, targetNodeID)
	}
	if targetNodeID != nil && *targetNodeID > 0 {
		return servermodel.ErrInvalidProtocolConfig
	}
	return uc.validateNodeProtocolInstance(ctx, serverID, protocol, port)
}

func (uc *NodeUsecase) validateFrontNode(ctx context.Context, id int64, port uint16, targetNodeID *int64) error {
	if port == 0 || targetNodeID == nil || *targetNodeID <= 0 || *targetNodeID == id {
		return servermodel.ErrInvalidProtocolConfig
	}
	target, err := uc.repo.GetNodeByID(ctx, *targetNodeID)
	if err != nil {
		return err
	}
	if target == nil || normalizeNodeType(target.NodeType) != NodeTypeLanding || target.Enabled == nil || !*target.Enabled {
		return servermodel.ErrInvalidProtocolConfig
	}
	return uc.validateNodeProtocolInstance(ctx, target.ServerID, target.Protocol, target.Port)
}

func (uc *NodeUsecase) validateNodeProtocolInstance(ctx context.Context, serverID int64, protocol string, port uint16) error {
	if serverID <= 0 || strings.TrimSpace(protocol) == "" || port == 0 {
		return servermodel.ErrInvalidProtocolConfig
	}
	protocols, err := uc.repo.GetServerProtocols(ctx, int(serverID))
	if err != nil {
		return err
	}
	for _, item := range protocols {
		if item == nil || !item.Enable {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Type), strings.TrimSpace(protocol)) && item.Port == int32(port) {
			return nil
		}
	}
	return servermodel.ErrInvalidProtocolConfig
}

func (uc *NodeUsecase) DeleteNode(ctx context.Context, id int) error {
	oldNode, _ := uc.repo.GetNodeByID(ctx, int64(id))
	count, err := uc.repo.CountFrontNodesByTarget(ctx, int64(id))
	if err != nil {
		return err
	}
	if count > 0 {
		return servermodel.ErrInvalidProtocolConfig
	}
	if err := uc.repo.DeleteNode(ctx, id); err != nil {
		return err
	}
	serverIDs := uc.nodeServerIDsForCache(ctx, oldNode)
	if err := uc.repo.ClearNodeCache(ctx, serverIDs); err != nil {
		uc.log.Warnf("Failed to clear node cache for deleted node %d servers %v: %v", id, serverIDs, err)
	}
	return nil
}

func (uc *NodeUsecase) FilterNodeList(ctx context.Context, page, size int32, search string, nodeGroupID *int64) (int32, []*Node, error) {
	return uc.repo.FilterNodeList(ctx, page, size, search, nodeGroupID)
}

func (uc *NodeUsecase) ToggleNodeStatus(ctx context.Context, id int, enable *bool) (*Node, error) {
	node, err := uc.repo.ToggleNodeStatus(ctx, id, enable)
	if err != nil {
		return nil, err
	}
	serverIDs := uc.nodeServerIDsForCache(ctx, node)
	if err := uc.repo.ClearNodeCache(ctx, serverIDs); err != nil {
		uc.log.Warnf("Failed to clear node cache for node %d servers %v: %v", id, serverIDs, err)
	}
	return node, nil
}

func (uc *NodeUsecase) QueryNodeTags(ctx context.Context) ([]string, error) {
	return uc.repo.QueryNodeTags(ctx)
}
func (uc *NodeUsecase) ResetNodeSort(ctx context.Context, sortItems []*SortItem) error {
	return uc.repo.ResetNodeSort(ctx, sortItems)
}

func (uc *NodeUsecase) nodeServerIDsForCache(ctx context.Context, nodes ...*Node) []int {
	seen := make(map[int]struct{})
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.ServerID > 0 {
			seen[int(node.ServerID)] = struct{}{}
		}
		if node.TargetNodeID != nil && *node.TargetNodeID > 0 {
			target, err := uc.repo.GetNodeByID(ctx, *node.TargetNodeID)
			if err != nil || target == nil || target.ServerID <= 0 {
				continue
			}
			seen[int(target.ServerID)] = struct{}{}
		}
	}
	serverIDs := make([]int, 0, len(seen))
	for id := range seen {
		serverIDs = append(serverIDs, id)
	}
	return serverIDs
}
