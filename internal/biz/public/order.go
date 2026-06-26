package public

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// OrderRepo is the interface for order data access
type OrderRepo interface {
	// CloseOrder closes an order
	CloseOrder(ctx context.Context, userID int, orderNo string) error

	// QueryOrderDetail queries order detail
	QueryOrderDetail(ctx context.Context, userID int, orderNo string) (*OrderDetail, error)

	// QueryOrderList queries order list
	QueryOrderList(ctx context.Context, userID int, page, size int, status, orderType int32) ([]*OrderDetail, int32, error)

	// PreCreateOrder validates and calculates order price
	PreCreateOrder(ctx context.Context, req *PreCreateOrderParams) (*PreCreateOrderResult, error)

	// Purchase creates a purchase order
	Purchase(ctx context.Context, req *PurchaseParams) (*OrderResult, error)

	// Recharge creates a recharge order
	Recharge(ctx context.Context, req *RechargeParams) (*OrderResult, error)

	// Renewal creates a renewal order
	Renewal(ctx context.Context, req *RenewalParams) (*OrderResult, error)

	// ResetTraffic creates a reset traffic order
	ResetTraffic(ctx context.Context, req *ResetTrafficParams) (*OrderResult, error)
}

// PaymentMethod represents payment method
type PaymentMethod struct {
	ID          int64
	Name        string
	Platform    string
	Description string
	Icon        string
	FeeMode     int32
	FeePercent  int64
	FeeAmount   int64
}

// SubscribeDiscount represents subscribe discount
type SubscribeDiscount struct {
	Quantity int64
	Discount int64
}

// Subscribe represents subscribe
type Subscribe struct {
	ID                int64
	Name              string
	Language          string
	Description       string
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
	Show              bool
	Sell              bool
	Sort              int64
	DeductionRatio    int64
	AllowDeduction    bool
	NodeGroupIds      []string
	NodeGroupId       string
	ResetCycle        int64
	RenewalReset      bool
	ShowOriginalPrice bool
	PriceOptions      []SubscribePriceOption
	CreatedAt         int64
	UpdatedAt         int64
}

// SubscribePriceOption represents a public sellable price/duration option.
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

// OrderDetail represents order detail
type OrderDetail struct {
	ID              int64
	ParentID        int64
	UserID          int64
	OrderNo         string
	Type            int32
	Quantity        int64
	Price           int64
	Amount          int64
	GiftAmount      int64
	Discount        int64
	Coupon          string
	CouponDiscount  int64
	Commission      int64
	Payment         *PaymentMethod // Nested payment method object
	Method          string
	FeeAmount       int64
	TradeNo         string
	Status          int32
	SubscribeID     int64
	PriceOptionID   int64
	PriceOptionName string
	DurationUnit    string
	DurationValue   int64
	OptionPrice     int64
	SubscribeToken  string
	IsNew           bool
	CreatedAt       int64
	UpdatedAt       int64
	Subscribe       *Subscribe // Nested subscribe object
	// Additional fields for backward compatibility
	SubscribeName string
	PaymentName   string
	StatusText    string
	TypeText      string
}

// PreCreateOrderParams represents pre-create order parameters
type PreCreateOrderParams struct {
	UserID           int64
	Type             int32
	SubscribeID      int64
	SubscribeGroupID int64
	Quantity         int64
	PriceOptionID    int64
	Coupon           string
	Payment          int64
}

// PreCreateOrderResult represents pre-create order result
type PreCreateOrderResult struct {
	Price          int64
	Amount         int64
	Discount       int64
	CouponDiscount int64
	FeeAmount      int64
	Commission     int64
	GiftAmount     int64
	Valid          bool
	Message        string
}

// PurchaseParams represents purchase parameters
type PurchaseParams struct {
	UserID        int64
	SubscribeID   int64
	PriceOptionID int64
	Quantity      int64
	Coupon        string
	Payment       int64
}

// RechargeParams represents recharge parameters
type RechargeParams struct {
	UserID  int64
	Amount  int64
	Payment int64
}

// RenewalParams represents renewal parameters
type RenewalParams struct {
	UserID          int64
	UserSubscribeID int64
	PriceOptionID   int64
	Quantity        int64
	Coupon          string
	Payment         int64
}

// ResetTrafficParams represents reset traffic parameters
type ResetTrafficParams struct {
	UserID          int64
	UserSubscribeID int64
	Payment         int64
}

// OrderResult represents order creation result
type OrderResult struct {
	OrderID    int64
	OrderNo    string
	Amount     int64
	PaymentURL string
	QRCode     string
}

// OrderUsecase is the usecase for public order operations
type OrderUsecase struct {
	repo   OrderRepo
	logger *log.Helper
}

// NewOrderUsecase creates a new OrderUsecase
func NewOrderUsecase(repo OrderRepo, logger log.Logger) *OrderUsecase {
	return &OrderUsecase{
		repo:   repo,
		logger: log.NewHelper(logger),
	}
}

// CloseOrder closes an order
func (uc *OrderUsecase) CloseOrder(ctx context.Context, userID int, orderNo string) error {
	return uc.repo.CloseOrder(ctx, userID, orderNo)
}

// QueryOrderDetail queries order detail
func (uc *OrderUsecase) QueryOrderDetail(ctx context.Context, userID int, orderNo string) (*OrderDetail, error) {
	return uc.repo.QueryOrderDetail(ctx, userID, orderNo)
}

// QueryOrderList queries order list
func (uc *OrderUsecase) QueryOrderList(ctx context.Context, userID int, page, size int, status, orderType int32) ([]*OrderDetail, int32, error) {
	return uc.repo.QueryOrderList(ctx, userID, page, size, status, orderType)
}

// PreCreateOrder validates and calculates order price
func (uc *OrderUsecase) PreCreateOrder(ctx context.Context, req *PreCreateOrderParams) (*PreCreateOrderResult, error) {
	return uc.repo.PreCreateOrder(ctx, req)
}

// Purchase creates a purchase order
func (uc *OrderUsecase) Purchase(ctx context.Context, req *PurchaseParams) (*OrderResult, error) {
	return uc.repo.Purchase(ctx, req)
}

// Recharge creates a recharge order
func (uc *OrderUsecase) Recharge(ctx context.Context, req *RechargeParams) (*OrderResult, error) {
	return uc.repo.Recharge(ctx, req)
}

// Renewal creates a renewal order
func (uc *OrderUsecase) Renewal(ctx context.Context, req *RenewalParams) (*OrderResult, error) {
	return uc.repo.Renewal(ctx, req)
}

// ResetTraffic creates a reset traffic order
func (uc *OrderUsecase) ResetTraffic(ctx context.Context, req *ResetTrafficParams) (*OrderResult, error) {
	return uc.repo.ResetTraffic(ctx, req)
}
