package log

import (
	"encoding/json"
)

type Type uint8

/*
Log Types:
	1X Message Logs
	2X Subscription Logs
	3X User Logs
	4X Traffic Ranking Logs
*/

const (
	TypeEmailMessage      Type = 10 // Message log
	TypeMobileMessage     Type = 11 // Mobile message log
	TypeSubscribe         Type = 20 // Subscription log
	TypeSubscribeTraffic  Type = 21 // Subscription traffic log
	TypeServerTraffic     Type = 22 // Server traffic log
	TypeResetSubscribe    Type = 23 // Reset subscription log
	TypeLogin             Type = 30 // Login log
	TypeRegister          Type = 31 // Registration log
	TypeBalance           Type = 32 // Balance log
	TypeCommission        Type = 33 // Commission log
	TypeGift              Type = 34 // Gift log
	TypeUserTrafficRank   Type = 40 // Top 10 User traffic rank log
	TypeServerTrafficRank Type = 41 // Top 10 Server traffic rank log
	TypeTrafficStat       Type = 42 // Daily traffic statistics log
)

const (
	ResetSubscribeTypeAuto        uint16 = 231 // Auto reset
	ResetSubscribeTypeAdvance     uint16 = 232 // Advance reset
	ResetSubscribeTypePaid        uint16 = 233 // Paid reset
	ResetSubscribeTypeQuota       uint16 = 234 // Quota reset
	BalanceTypeRecharge           uint16 = 321 // Recharge
	BalanceTypeWithdraw           uint16 = 322 // Withdraw
	BalanceTypePayment            uint16 = 323 // Payment
	BalanceTypeRefund             uint16 = 324 // Refund
	BalanceTypeReward             uint16 = 325 // Reward
	BalanceTypeAdjust             uint16 = 326 // Admin Adjust
	BalanceTypeCommissionTransfer uint16 = 327 // Commission Transfer
	CommissionTypePurchase        uint16 = 331 // Purchase
	CommissionTypeRenewal         uint16 = 332 // Renewal
	CommissionTypeRefund          uint16 = 333 // Refund
	CommissionTypeWithdraw        uint16 = 334 // Withdraw
	CommissionTypeAdjust          uint16 = 335 // Admin Adjust
	CommissionTypeConvertBalance  uint16 = 336 // Convert to Balance
	GiftTypeIncrease              uint16 = 341 // Increase
	GiftTypeReduce                uint16 = 342 // Reduce
)

// Uint8 converts Type to uint8
func (t Type) Uint8() uint8 {
	return uint8(t)
}

// Login represents a login log entry
type Login struct {
	Method    string `json:"method"`
	LoginIP   string `json:"login_ip"`
	UserAgent string `json:"user_agent"`
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Login
func (l *Login) Marshal() ([]byte, error) {
	return json.Marshal(l)
}

// Unmarshal implements the json.Unmarshaler interface for Login
func (l *Login) Unmarshal(data []byte) error {
	return json.Unmarshal(data, l)
}

// Balance represents a balance log entry
type Balance struct {
	Type      uint16 `json:"type"`
	Amount    int64  `json:"amount"`
	OrderNo   string `json:"order_no,omitempty"`
	Balance   int64  `json:"balance"`
	Timestamp int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Balance
func (b *Balance) Marshal() ([]byte, error) {
	return json.Marshal(b)
}

// Unmarshal implements the json.Unmarshaler interface for Balance
func (b *Balance) Unmarshal(data []byte) error {
	return json.Unmarshal(data, b)
}

// Commission represents a commission log entry
type Commission struct {
	Type      uint16 `json:"type"`
	Amount    int64  `json:"amount"`
	OrderNo   string `json:"order_no"`
	Timestamp int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Commission
func (c *Commission) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// Unmarshal implements the json.Unmarshaler interface for Commission
func (c *Commission) Unmarshal(data []byte) error {
	return json.Unmarshal(data, c)
}

// Gift represents a gift log entry
type Gift struct {
	Type        uint16 `json:"type"`
	OrderNo     string `json:"order_no"`
	SubscribeId int64  `json:"subscribe_id"`
	Amount      int64  `json:"amount"`
	Balance     int64  `json:"balance"`
	Remark      string `json:"remark,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Gift
func (g *Gift) Marshal() ([]byte, error) {
	return json.Marshal(g)
}

// Unmarshal implements the json.Unmarshaler interface for Gift
func (g *Gift) Unmarshal(data []byte) error {
	return json.Unmarshal(data, g)
}

// Message represents a message log entry (email/mobile)
type Message struct {
	To       string                 `json:"to"`
	Subject  string                 `json:"subject,omitempty"`
	Content  map[string]interface{} `json:"content"`
	Platform string                 `json:"platform"`
	Template string                 `json:"template"`
	Status   uint8                  `json:"status"` // 1: Sent, 2: Failed
}

// Marshal implements the json.Marshaler interface for Message
func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// Unmarshal implements the json.Unmarshaler interface for Message
func (m *Message) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

// Traffic represents a subscription/server traffic log entry
type Traffic struct {
	Download int64 `json:"download"`
	Upload   int64 `json:"upload"`
}

// Marshal implements the json.Marshaler interface for Traffic
func (t *Traffic) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

// Unmarshal implements the json.Unmarshaler interface for Traffic
func (t *Traffic) Unmarshal(data []byte) error {
	return json.Unmarshal(data, t)
}

// Register represents a registration log entry
type Register struct {
	AuthMethod string `json:"auth_method"`
	Identifier string `json:"identifier"`
	RegisterIP string `json:"register_ip"`
	UserAgent  string `json:"user_agent"`
	Timestamp  int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for Register
func (r *Register) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// Unmarshal implements the json.Unmarshaler interface for Register
func (r *Register) Unmarshal(data []byte) error {
	return json.Unmarshal(data, r)
}

// Subscribe represents a subscription log entry
type Subscribe struct {
	Token           string `json:"token"`
	UserAgent       string `json:"user_agent"`
	ClientIP        string `json:"client_ip"`
	UserSubscribeId int64  `json:"user_subscribe_id"`
}

// Marshal implements the json.Marshaler interface for Subscribe
func (s *Subscribe) Marshal() ([]byte, error) {
	return json.Marshal(s)
}

// Unmarshal implements the json.Unmarshaler interface for Subscribe
func (s *Subscribe) Unmarshal(data []byte) error {
	return json.Unmarshal(data, s)
}

// ResetSubscribe represents a reset subscription log entry
type ResetSubscribe struct {
	Type            uint16 `json:"type"`
	UserId          int64  `json:"user_id"`
	UserSubscribeId int64  `json:"user_subscribe_id"`
	OrderNo         string `json:"order_no,omitempty"`
	Timestamp       int64  `json:"timestamp"`
}

// Marshal implements the json.Marshaler interface for ResetSubscribe
func (r *ResetSubscribe) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// Unmarshal implements the json.Unmarshaler interface for ResetSubscribe
func (r *ResetSubscribe) Unmarshal(data []byte) error {
	return json.Unmarshal(data, r)
}
