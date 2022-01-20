package svc

import (
	"github.com/gotid/god/lib/store/kv"
	"github.com/gotid/god/lib/store/sqlx"
	"github.com/gotid/wechat/api/internal/config"
	"github.com/gotid/wechat/api/internal/model"
)

type ServiceContext struct {
	Config config.Config
	Cache  kv.Store

	PlatformModel   *model.PlatformModel
	WeappModel      *model.WeappModel
	WeappAuditModel *model.WeappAuditModel
	PayModel        *model.PayModel
	PayOrderModel   *model.PayOrderModel
	PayRefundModel  *model.PayRefundModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMySQL(c.MySQL)

	return &ServiceContext{
		Config: c,
		Cache:  kv.NewStore(c.Cache),

		PlatformModel:   model.NewPlatformModel(conn, c.Cache),
		WeappModel:      model.NewWeappModel(conn, c.Cache),
		WeappAuditModel: model.NewWeappAuditModel(conn, c.Cache),
		PayModel:        model.NewPayModel(conn, c.Cache),
		PayOrderModel:   model.NewPayOrderModel(conn, c.Cache),
		PayRefundModel:  model.NewPayRefundModel(conn, c.Cache),
	}
}
