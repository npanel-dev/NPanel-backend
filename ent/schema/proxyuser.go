package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ProxyUser holds the schema definition for the ProxyUser entity.
type ProxyUser struct {
	ent.Schema
}

func (ProxyUser) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user"},
		entsql.WithComments(true),
	}
}

func (ProxyUser) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Positive().Comment("用户ID"),
		field.String("password").MaxLen(100).NotEmpty().Comment("用户密码"),
		field.String("algo").MaxLen(20).Default("default").Comment("加密算法"),
		field.String("salt").MaxLen(20).Optional().Nillable().Comment("密码盐值"),
		field.String("source_panel").MaxLen(64).Default("npanel").Comment("账号来源面板"),
		field.Text("avatar").Optional().Nillable().Comment("用户头像"),
		field.Int64("balance").Optional().Nillable().Default(0).Comment("用户余额"),
		field.Int64("telegram").Optional().Nillable().Comment("Telegram 账号"),
		field.String("refer_code").MaxLen(20).Optional().Nillable().Comment("推荐码"),
		field.Int64("referer_id").Optional().Nillable().Comment("推荐人ID"),
		field.Int64("commission").Optional().Nillable().Default(0).Comment("佣金"),
		field.Int8("referral_percentage").Default(0).Comment("推荐百分比"),
		field.Bool("only_first_purchase").Default(true).Comment("仅首次购买"),
		field.Int64("gift_amount").Optional().Nillable().Default(0).Comment("礼品金额"),
		field.Bool("enable").Default(true).Comment("账号是否启用"),
		field.Bool("is_admin").Default(false).Comment("是否管理员"),
		field.Bool("valid_email").Default(false).Comment("邮箱是否已验证"),
		field.Bool("enable_email_notify").Default(false).Comment("启用邮件通知"),
		field.Bool("enable_telegram_notify").Default(false).Comment("启用 Telegram 通知"),
		field.Bool("enable_balance_notify").Default(false).Comment("启用余额变动通知"),
		field.Bool("enable_login_notify").Default(false).Comment("启用登录通知"),
		field.Bool("enable_subscribe_notify").Default(false).Comment("启用订阅通知"),
		field.Bool("enable_trade_notify").Default(false).Comment("启用交易通知"),
		field.String("rules").Optional().Nillable().Comment("用户规则"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
		field.Time("deleted_at").Optional().Nillable().Comment("删除时间"),
		field.Uint64("is_del").Optional().Nillable().Comment("1: 正常 0: 删除"),
	}
}

func (ProxyUser) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("redemption_records", ProxyRedemptionRecord.Type),
		edge.To("withdrawals", ProxyUserWithdrawal.Type),
	}
}
