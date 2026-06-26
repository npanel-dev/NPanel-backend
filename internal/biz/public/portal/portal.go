package portal

import (
	"context"
	"errors"
	"time"
)

// Portal模块错误定义
var (
	ErrUserAlreadyExists = errors.New("user already exists")
)

// PortalRepo Portal数据仓库接口
type PortalRepo interface {
	// CheckUserExists 检查用户是否已存在（通过AuthType和Identifier）
	CheckUserExists(ctx context.Context, authType, identifier string) (bool, error)

	// GetSubscribeList 获取订阅列表
	// language: 语言过滤（可选），如果为空且defaultLanguage=true则返回默认语言（language=''）
	GetSubscribeList(ctx context.Context, language string, categoryID int64) ([]*SubscribeInfo, error)
	GetSubscribeCatalog(ctx context.Context, language string) (*SubscribeCatalog, error)

	// CalculateOrderPrice 计算订单价格（含折扣、优惠券、手续费）
	// paymentID: 可选，用于计算手续费
	CalculateOrderPrice(ctx context.Context, subscribeID, priceOptionID, quantity int64, coupon *string, paymentID *int64) (*PriceInfo, error)

	// CreatePortalOrder 创建Portal订单（userId=0，is_new=true）+ 保存临时订单 + 入队延迟关闭任务
	// ⚠️ 包含保存临时订单到Redis的逻辑（复刻原项目事务逻辑）
	CreatePortalOrder(ctx context.Context, req *CreateOrderRequest) (string, error)

	// GetOrderByNo 根据订单号查询订单
	GetOrderByNo(ctx context.Context, orderNo string) (*OrderInfo, error)

	// GetTempOrderInfo 获取临时订单信息
	GetTempOrderInfo(ctx context.Context, orderNo string) (*TempOrderInfo, error)

	// GetAvailablePaymentMethods 获取可用支付方式
	GetAvailablePaymentMethods(ctx context.Context) ([]*PaymentMethod, error)

	// CreatePayment 创建支付记录并返回支付信息
	CreatePayment(ctx context.Context, orderNo string, returnURL string) (*PaymentInfo, error)

	// CheckOrderStatus 检查订单状态（含临时订单处理、token生成）
	CheckOrderStatus(ctx context.Context, orderNo string, authType, identifier string) (*OrderStatusInfo, string, error)
}

// SubscribeInfo 订阅信息
// ⚠️ 完全复刻原项目（server-master/internal/types/types.go:Subscribe）
type SubscribeInfo struct {
	ID                int64
	Name              string
	Language          string
	Description       *string
	UnitPrice         int64
	UnitTime          string
	Discount          []SubscribeDiscount
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
	NodeGroupIds      []string
	NodeGroupId       string
	TrafficLimit      []TrafficLimit
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
	List        []*SubscribeInfo
	Children    []*SubscribeCategory
}

type SubscribeCatalog struct {
	Categories    []*SubscribeCategory
	Uncategorized []*SubscribeInfo
	Total         int32
}

// SubscribeDiscount 订阅折扣配置
type SubscribeDiscount struct {
	Quantity int `json:"quantity"` // 购买数量
	Discount int `json:"discount"` // 折扣值（百分比 0-100）
}

type TrafficLimit struct {
	StatType     string `json:"stat_type"`
	StatValue    int64  `json:"stat_value"`
	TrafficUsage int64  `json:"traffic_usage"`
	SpeedLimit   int64  `json:"speed_limit"`
}

// PriceInfo 价格信息（预览）
// ⚠️ 完全复刻原项目（server-master/internal/types/types.go:PrePurchaseOrderResponse）
type PriceInfo struct {
	Price          int    // 原价
	Amount         int    // 实际支付金额（含手续费）⚠️
	Discount       int    // 折扣金额
	Coupon         string // 优惠券代码
	CouponDiscount int    // 优惠券折扣金额
	FeeAmount      int    // 手续费金额
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	SubscribeID   int64
	PriceOptionID int64
	Quantity      int64
	PaymentID     int // 支付方式ID（用于计算手续费）
	Coupon        *string
	Identifier    string // 认证标识符（邮箱/Telegram ID等）
	AuthType      string // 认证类型（email/telegram等）
	Password      string // 密码（明文，将在创建用户时加密）
	InviteCode    *string
}

// TempOrderInfo 临时订单信息（存储到Redis）
type TempOrderInfo struct {
	OrderNo    string
	Identifier string
	AuthType   string
	Password   string // 已加密
	InviteCode string
}

// OrderInfo 订单信息
type OrderInfo struct {
	ID         int64
	OrderNo    string
	Type       int32
	Price      int64
	Amount     int64
	Discount   int64
	Commission int64
	Method     string
	Status     int32
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PaymentMethod 支付方式
type PaymentMethod struct {
	ID          int64
	Name        string
	Platform    string
	Description string
	Icon        string
	FeeMode     int32 // 费用模式：0=无费用 1=百分比 2=固定金额 3=百分比+固定金额
	FeePercent  int   // 费用百分比
	FeeAmount   int   // 固定费用金额
}

// PaymentInfo 支付信息
// ⚠️ 完全复刻原项目（server-master/internal/types/types.go:CheckoutOrderResponse）
type PaymentInfo struct {
	Type        string         // 支付类型：url/stripe/qr/balance
	CheckoutURL string         // 支付URL（可选）
	Stripe      *StripePayment // Stripe支付信息（可选）
}

// StripePayment Stripe支付信息
type StripePayment struct {
	PublishableKey string // Stripe公钥
	ClientSecret   string // 客户端密钥
	Method         string // 支付方式
}

// OrderStatusInfo 订单状态信息
type OrderStatusInfo struct {
	OrderNo         string
	Subscribe       *SubscribeInfo
	Quantity        int64
	Price           int64
	Amount          int64
	Discount        int64
	Coupon          string
	CouponDiscount  int64
	FeeAmount       int64
	Payment         *PaymentMethod
	Status          int32
	PriceOptionID   int64
	PriceOptionName string
	DurationUnit    string
	DurationValue   int64
	OptionPrice     int64
	CreatedAt       time.Time
}

// PortalUseCase Portal用例
type PortalUseCase struct {
	repo PortalRepo
}

// NewPortalUseCase 创建Portal用例
func NewPortalUseCase(repo PortalRepo) *PortalUseCase {
	return &PortalUseCase{repo: repo}
}

// GetSubscribeList 获取订阅列表
func (uc *PortalUseCase) GetSubscribeList(ctx context.Context, language string, categoryID int64) ([]*SubscribeInfo, error) {
	return uc.repo.GetSubscribeList(ctx, language, categoryID)
}

func (uc *PortalUseCase) GetSubscribeCatalog(ctx context.Context, language string) (*SubscribeCatalog, error) {
	return uc.repo.GetSubscribeCatalog(ctx, language)
}

// PrePurchaseOrder 预购买订单（计算价格）
func (uc *PortalUseCase) PrePurchaseOrder(ctx context.Context, subscribeID, priceOptionID, quantity int64, coupon *string, paymentID *int64) (*PriceInfo, error) {
	return uc.repo.CalculateOrderPrice(ctx, subscribeID, priceOptionID, quantity, coupon, paymentID)
}

// Purchase 购买（创建订单）
// ⚠️ 完整复刻原项目逻辑（purchaseLogic.go）
// ⚠️ CreatePortalOrder内部会保存临时订单到Redis（复刻原项目事务逻辑）
func (uc *PortalUseCase) Purchase(ctx context.Context, req *CreateOrderRequest) (string, error) {
	// 1. 检查用户是否已存在（通过AuthType和Identifier）
	exists, err := uc.repo.CheckUserExists(ctx, req.AuthType, req.Identifier)
	if err != nil {
		return "", err
	}
	if exists {
		return "", ErrUserAlreadyExists
	}

	// 2. 创建Portal订单
	// ⚠️ 内部逻辑（复刻原项目purchaseLogic.go line 130-151）：
	//    - 先保存临时订单到Redis（15分钟过期）
	//    - 再创建订单到数据库
	//    - 最后入队延迟关闭任务（15分钟）
	orderNo, err := uc.repo.CreatePortalOrder(ctx, req)
	if err != nil {
		return "", err
	}

	return orderNo, nil
}

// GetAvailablePaymentMethods 获取可用支付方式
func (uc *PortalUseCase) GetAvailablePaymentMethods(ctx context.Context) ([]*PaymentMethod, error) {
	return uc.repo.GetAvailablePaymentMethods(ctx)
}

// PurchaseCheckout 购买结账（获取支付信息）
func (uc *PortalUseCase) PurchaseCheckout(ctx context.Context, orderNo string, returnURL string) (*PaymentInfo, error) {
	return uc.repo.CreatePayment(ctx, orderNo, returnURL)
}

// QueryPurchaseOrder 查询购买订单状态
// 返回订单状态和token（如果订单已完成且是临时订单）
func (uc *PortalUseCase) QueryPurchaseOrder(ctx context.Context, orderNo, authType, identifier string) (*OrderStatusInfo, string, error) {
	return uc.repo.CheckOrderStatus(ctx, orderNo, authType, identifier)
}
