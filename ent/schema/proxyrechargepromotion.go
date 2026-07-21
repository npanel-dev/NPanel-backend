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

// ProxyRechargePromotion holds the schema definition for the ProxyRechargePromotion entity.
// 充值促销活动，支持预设档位 + 自定义金额按比例赠送 + 限时活动 + 多活动叠加/互斥。
// 时间字段 start_time/end_time 用 Unix 毫秒（与项目 marketing 模块约定一致），0 表示不限/长期。
type ProxyRechargePromotion struct {
	ent.Schema
}

func (ProxyRechargePromotion) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "recharge_promotion"},
		entsql.WithComments(true),
	}
}

func (ProxyRechargePromotion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Positive().Comment("活动ID"),
		field.String("name").NotEmpty().Comment("活动名称"),
		field.Text("description").Default("").Comment("活动描述 Markdown"),
		field.Int64("start_time").Default(0).Comment("开始时间 Unix毫秒 0=不限"),
		field.Int64("end_time").Default(0).Comment("结束时间 Unix毫秒 0=长期"),
		field.Bool("status").Default(false).Comment("启用状态 false=停用 true=启用"),
		field.Int32("priority").Default(0).Comment("优先级 DESC（大优先）"),
		field.Int8("stack_mode").Default(0).Comment("叠加策略 0=互斥 1=叠加"),
		field.Bool("custom_bonus_enabled").Default(false).Comment("是否对自定义金额赠送"),
		field.Int32("custom_bonus_percent").Default(0).Comment("自定义金额赠送比例 万分比 0-10000（500=5%）"),
		field.Int32("sort").Default(0).Comment("列表排序"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
	}
}

func (ProxyRechargePromotion) Indexes() []ent.Index {
	return []ent.Index{
		// ListActive 查询主力索引：按状态 + 时间窗口过滤
		index.Fields("status", "start_time", "end_time"),
	}
}

func (ProxyRechargePromotion) Edges() []ent.Edge {
	return []ent.Edge{
		// 主表 -> 档位子表，外键列通过子表的 field + .Field("promotion_id") 绑定
		edge.To("tiers", ProxyRechargePromotionTier.Type),
	}
}
