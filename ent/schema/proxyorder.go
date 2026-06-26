package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// ProxyOrder holds the schema definition for the ProxyOrder entity.
type ProxyOrder struct {
	ent.Schema
}

func (ProxyOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "order"},
		entsql.WithComments(true),
	}
}

func (ProxyOrder) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Comment("订单ID"),
		field.Int64("parent_id").Optional().Comment("父订单ID"),
		field.Int64("user_id").Default(0).Comment("用户ID"),
		field.String("order_no").MaxLen(255).NotEmpty().Unique().Comment("订单号"),
		field.Int8("type").Default(1).Comment("订单类型"),
		field.Int32("quantity").Default(1).Comment("数量"),
		field.Int64("price").Default(0).Comment("原价"),
		field.Int64("amount").Default(0).Comment("实付金额"),
		field.Int64("gift_amount").Default(0).Comment("赠送金额"),
		field.Int64("discount").Default(0).Comment("折扣金额"),
		field.String("coupon").MaxLen(255).Optional().Comment("优惠券代码"),
		field.Int64("coupon_discount").Default(0).Comment("优惠券折扣金额"),
		field.Int64("commission").Default(0).Comment("佣金金额"),
		field.Int64("payment_id").Default(0).Comment("支付方式ID"),
		field.String("method").MaxLen(255).Default("").Comment("支付方式标识"),
		field.Int64("fee_amount").Default(0).Comment("手续费金额"),
		field.String("trade_no").MaxLen(255).Optional().Comment("第三方交易号"),
		field.Int8("status").Default(1).Comment("订单状态"),
		field.Int64("subscribe_id").Default(0).Comment("关联订阅ID"),
		field.Int64("price_option_id").Default(0).Comment("价格档位ID"),
		field.String("price_option_name").MaxLen(255).Default("").Comment("价格档位名称快照"),
		field.String("duration_unit").MaxLen(32).Default("").Comment("时长单位快照"),
		field.Int64("duration_value").Default(0).Comment("时长数值快照"),
		field.Int64("option_price").Default(0).Comment("价格档位售价快照"),
		field.String("subscribe_token").MaxLen(255).Optional().Comment("续费订阅Token"),
		field.Bool("is_new").Default(false).Comment("是否新订单"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
	}
}

func (ProxyOrder) Edges() []ent.Edge {
	return nil
}
