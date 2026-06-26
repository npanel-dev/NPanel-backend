package subscribe

import (
	"context"
	"strings"
)

// SubscribeRepo Public Subscribe数据仓库接口
type SubscribeRepo interface {
	// QuerySubscribeList 查询订阅列表
	QuerySubscribeList(ctx context.Context, language string, categoryID int64) ([]*Subscribe, int32, error)
	QuerySubscribeCatalog(ctx context.Context, language string) (*SubscribeCatalog, error)
	QueryUserSubscribeNodeList(ctx context.Context, userID int64) ([]*UserSubscribeInfo, error)
}

// Subscribe 订阅信息
type Subscribe struct {
	ID                int64
	Name              string
	Language          string
	Description       string
	UnitPrice         int64
	UnitTime          string
	Discount          []*SubscribeDiscount
	Replacement       int64
	Inventory         int64
	Traffic           int64
	SpeedLimit        int64
	DeviceLimit       int64
	Quota             int64
	CategoryID        int64
	CategoryName      string
	Nodes             []int
	NodeTags          []string
	NodeGroupIds      []int64
	NodeGroupId       int64
	TrafficLimit      []*TrafficLimit
	Show              bool
	Sell              bool
	Sort              int64
	DeductionRatio    int64
	AllowDeduction    bool
	ResetCycle        int64
	RenewalReset      bool
	ShowOriginalPrice bool
	PriceOptions      []SubscribePriceOption
	CreatedAt         int64
	UpdatedAt         int64
}

type SubscribePriceOption struct {
	ID            int64
	SubscribeID   int64
	Name          string
	DurationUnit  string
	DurationValue int64
	Price         int64
	OriginalPrice int64
	Inventory     int64
	Show          bool
	Sell          bool
	IsDefault     bool
	Sort          int64
	CreatedAt     int64
	UpdatedAt     int64
}

type SubscribeCategory struct {
	ID          int64
	ParentID    int64
	Name        string
	Description string
	Language    string
	Show        bool
	Sort        int64
	List        []*Subscribe
	Children    []*SubscribeCategory
}

type SubscribeCatalog struct {
	Categories    []*SubscribeCategory
	Uncategorized []*Subscribe
	Total         int32
}

// SubscribeDiscount 订阅折扣
type SubscribeDiscount struct {
	Quantity int64
	Discount float64
}

type TrafficLimit struct {
	StatType     string
	StatValue    int64
	TrafficUsage int64
	SpeedLimit   int64
}

type UserSubscribeInfo struct {
	ID          int64
	UserID      int64
	OrderID     int64
	SubscribeID int64
	StartTime   int64
	ExpireTime  int64
	FinishedAt  int64
	ResetTime   int64
	Traffic     int64
	Download    int64
	Upload      int64
	Token       string
	Status      int32
	CreatedAt   int64
	UpdatedAt   int64
	IsTryOut    bool
	Nodes       []*UserSubscribeNodeInfo
}

type UserSubscribeNodeInfo struct {
	ID              int64
	Name            string
	Uuid            string
	Protocol        string
	Protocols       string
	Port            uint32
	Address         string
	Tags            []string
	Country         string
	City            string
	Longitude       string
	Latitude        string
	LatitudeCenter  string
	LongitudeCenter string
	CreatedAt       int64

	SNI                        string
	OmniflowCarrier            string
	OmniflowPath               string
	OmniflowContentType        string
	OmniflowProfileJson        string
	OmniflowCaCertPath         string
	OmniflowTargetMeta         string
	OmniflowSpkiPin            string
	OmniflowAdaptiveTlsEnabled bool
	OmniflowTlsFingerprint     string
	OmniflowSniMode            string
	OmniflowPaddingMode        string
	OmniflowAfEnabled          bool
	OmniflowAfPathMode         string
	OmniflowAfPathPrefix       string
	OmniflowAfPathSuffix       string
	OmniflowAfPathRotationSecs int
	OmniflowAfPathSkewSlots    int
}

// SubscribeUseCase Public Subscribe用例
type SubscribeUseCase struct {
	repo SubscribeRepo
}

// NewSubscribeUseCase 创建Public Subscribe用例
func NewSubscribeUseCase(repo SubscribeRepo) *SubscribeUseCase {
	return &SubscribeUseCase{repo: repo}
}

// QuerySubscribeList 查询订阅列表
func (uc *SubscribeUseCase) QuerySubscribeList(ctx context.Context, language string, categoryID int64) ([]*Subscribe, int32, error) {
	return uc.repo.QuerySubscribeList(ctx, language, categoryID)
}

func (uc *SubscribeUseCase) QuerySubscribeCatalog(ctx context.Context, language string) (*SubscribeCatalog, error) {
	return uc.repo.QuerySubscribeCatalog(ctx, language)
}

func (uc *SubscribeUseCase) QueryUserSubscribeNodeList(ctx context.Context, userID int64) ([]*UserSubscribeInfo, error) {
	return uc.repo.QueryUserSubscribeNodeList(ctx, userID)
}

// FilterExperimentalNodesForClient 按客户端 User-Agent 过滤实验性协议节点。
// simnet/omniflow 等新协议仅对自有客户端/SDK（UA 命中 omnxt、slag 或 slaglab）放行，
// 其它客户端（含开源客户端、空 UA、未知 UA）一律剔除，避免下发无法使用的配置。
func FilterExperimentalNodesForClient(list []*UserSubscribeInfo, userAgent string) {
	if len(list) == 0 || isOfficialClient(userAgent) {
		return
	}
	for _, item := range list {
		if item == nil {
			continue
		}
		filtered := make([]*UserSubscribeNodeInfo, 0, len(item.Nodes))
		for _, node := range item.Nodes {
			if node == nil || isExperimentalProtocol(node.Protocol) {
				continue
			}
			filtered = append(filtered, node)
		}
		item.Nodes = filtered
	}
}

// isExperimentalProtocol 判断单个 protocol 字符串是否为实验性协议。
func isExperimentalProtocol(protocol string) bool {
	lower := strings.ToLower(protocol)
	if lower == "" {
		return false
	}
	for _, keyword := range experimentalProtocolKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// isOfficialClient 判断请求是否来自自有客户端/SDK。
func isOfficialClient(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return false
	}
	for _, keyword := range officialClientKeywords {
		if strings.Contains(ua, keyword) {
			return true
		}
	}
	return false
}

// officialClientKeywords 自有客户端/SDK 的 UA 关键字白名单。
// 命中其一即可下发实验性协议。
var officialClientKeywords = []string{
	"omnxt",
	"slag/",
	"slaglab",
}

// experimentalProtocolKeywords 需要对非自有客户端隐藏的实验性协议关键字。
// simnet / omniflow 是新协议；omn 是 omniflow 的旧别名。
var experimentalProtocolKeywords = []string{
	"simnet",
	"omniflow",
	"omn",
}
