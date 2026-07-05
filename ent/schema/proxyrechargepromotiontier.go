package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProxyRechargePromotionTier holds the schema definition for the ProxyRechargePromotionTier entity.
// 充值促销档位，属于某个活动。amount 为充值本金，bonus 为赠送金额（进 user.gift_amount）。
// 所有金额单位均为「分」。
type ProxyRechargePromotionTier struct {
	ent.Schema
}

func (ProxyRechargePromotionTier) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "recharge_promotion_tier"},
		entsql.WithComments(true),
	}
}

func (ProxyRechargePromotionTier) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Positive().Comment("档位ID"),
		field.Int64("promotion_id").Comment("所属活动ID"),
		field.Int64("amount").Default(0).Comment("充值档位金额 分"),
		field.Int64("bonus").Default(0).Comment("赠送金额 分"),
		field.String("label").Default("").Comment("档位标签（如 热门/推荐）"),
		field.Int32("sort").Default(0).Comment("档位排序 ASC"),
		field.Bool("enable").Default(true).Comment("单档启停"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
	}
}

func (ProxyRechargePromotionTier) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("promotion_id"),
		// 命中算法主力索引：按活动 + 金额 + 启用状态查档位
		index.Fields("promotion_id", "amount", "enable"),
	}
}

func (ProxyRechargePromotionTier) Edges() []ent.Edge {
	return []ent.Edge{
		// 子表 -> 主表，外键列通过 .Field("promotion_id") 绑定到上面的 field.Int64("promotion_id")
		edge.From("promotion", ProxyRechargePromotion.Type).
			Ref("tiers").
			Field("promotion_id").
			Unique().
			Required(),
	}
}
