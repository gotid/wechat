package model

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/gotid/god/lib/container/garray"
	"github.com/gotid/god/lib/fx"
	"github.com/gotid/god/lib/g"
	"github.com/gotid/god/lib/gconv"
	"github.com/gotid/god/lib/gutil"
	"github.com/gotid/god/lib/logx"
	"github.com/gotid/god/lib/mathx"
	"github.com/gotid/god/lib/mr"
	"github.com/gotid/god/lib/store/cache"
	"github.com/gotid/god/lib/store/sqlx"
	"github.com/gotid/god/lib/stringx"
	"github.com/gotid/god/tools/god/mysql/builder"
)

var (
	payOrderFieldList             = builder.FieldList(&PayOrder{})
	payOrderFields                = strings.Join(payOrderFieldList, ",")
	payOrderFieldsAutoSet         = strings.Join(stringx.RemoveDBFields(payOrderFieldList, "created_at", "updated_at", "create_time", "update_time"), ",")
	payOrderFieldsWithPlaceHolder = strings.Join(stringx.RemoveDBFields(payOrderFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), "=?,") + "=?"

	cacheWechatPlatformPayOrderIdPrefix = "cache:wechatPlatform:payOrder:id:"
)

type (
	PayOrder struct {
		Id            int64           `db:"id" json:"id"`                        // 订单号，主键
		Uid           sqlx.NullInt64  `db:"uid" json:"uid"`                      // 购买的用户
		PayType       sqlx.NullInt64  `db:"pay_type" json:"payType"`             // 支付方式 2-支付宝 3-微信 4-现金收银
		BuyType       int64           `db:"buy_type" json:"buyType"`             // 购买商品的类型：1-购买商品 2-充值 3-发票 4-会员 5-分销升级
		Status        int64           `db:"status" json:"status"`                // 状态 0-待支付 1-成功
		Amount        sqlx.NullInt64  `db:"amount" json:"amount"`                // 商品金额，单位分
		BuyGoodsKey   int64           `db:"buy_goods_key" json:"buyGoodsKey"`    // 子订单号
		Extra         sqlx.NullString `db:"extra" json:"extra"`                  // 附加字段，备用
		TransactionId sqlx.NullString `db:"transaction_id" json:"transactionId"` // 支付平台交易号
		PayAppId      string          `db:"pay_app_id" json:"payAppId"`          // 商户账号
		CreateTime    int64           `db:"create_time" json:"createTime"`       // 创建订单的时间
		PaySuccTime   int64           `db:"pay_succ_time" json:"paySuccTime"`    // 支付成功的时间
	}

	PayOrderModel struct {
		sqlx.CachedConn
		table string
	}
)

func NewPayOrderModel(conn sqlx.Conn, clusterConf cache.ClusterConf) *PayOrderModel {
	return &PayOrderModel{
		CachedConn: sqlx.NewCachedConnWithCluster(conn, clusterConf),
		table:      "pay_order",
	}
}

func (m *PayOrderModel) Insert(data PayOrder) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + payOrderFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return m.ExecNoCache(query, data.Id, data.Uid, data.PayType, data.BuyType, data.Status, data.Amount, data.BuyGoodsKey, data.Extra, data.TransactionId, data.PayAppId, data.PaySuccTime)
}

func (m *PayOrderModel) TxInsert(tx sqlx.TxSession, data PayOrder) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + payOrderFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return tx.Exec(query, data.Id, data.Uid, data.PayType, data.BuyType, data.Status, data.Amount, data.BuyGoodsKey, data.Extra, data.TransactionId, data.PayAppId, data.PaySuccTime)
}

func (m *PayOrderModel) FindOne(id int64) (*PayOrder, error) {
	payOrderIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayOrderIdPrefix, id)
	var dest PayOrder
	err := m.Query(&dest, payOrderIdKey, func(conn sqlx.Conn, v interface{}) error {
		query := `select ` + payOrderFields + ` from ` + m.table + ` where id = ? limit 1`
		return conn.Query(v, query, id)
	})
	if err == nil {
		return &dest, nil
	} else if err == sqlx.ErrNotFound {
		return nil, ErrNotFound
	} else {
		return nil, err
	}
}

func (m *PayOrderModel) FindMany(ids []int64, workers ...int) (list []*PayOrder) {
	ids = gconv.Int64s(garray.NewArrayFrom(gconv.Interfaces(ids), true).Unique())

	var nWorkers int
	if len(workers) > 0 {
		nWorkers = workers[0]
	} else {
		nWorkers = mathx.MinInt(10, len(ids))
	}

	channel := mr.Map(func(source chan<- interface{}) {
		for _, id := range ids {
			source <- id
		}
	}, func(item interface{}, writer mr.Writer) {
		id := item.(int64)
		one, err := m.FindOne(id)
		if err == nil {
			writer.Write(one)
		} else {
			logx.Error(err)
		}
	}, mr.WithWorkers(nWorkers))

	for one := range channel {
		list = append(list, one.(*PayOrder))
	}

	sort.Slice(list, func(i, j int) bool {
		return gutil.IndexOf(list[i].Id, ids) < gutil.IndexOf(list[j].Id, ids)
	})

	return
}

func (m *PayOrderModel) Update(data PayOrder) error {
	payOrderIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayOrderIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + payOrderFieldsWithPlaceHolder + ` where id = ?`
		return conn.Exec(query, data.Uid, data.PayType, data.BuyType, data.Status, data.Amount, data.BuyGoodsKey, data.Extra, data.TransactionId, data.PayAppId, data.PaySuccTime, data.Id)
	}, payOrderIdKey)
	return err
}

func (m *PayOrderModel) UpdatePartial(ms ...g.Map) (err error) {
	okNum := 0
	fx.From(func(source chan<- interface{}) {
		for _, data := range ms {
			source <- data
		}
	}).Parallel(func(item interface{}) {
		err = m.updatePartial(item.(g.Map))
		if err != nil {
			return
		}
		okNum++
	})

	if err == nil && okNum != len(ms) {
		err = fmt.Errorf("部分局部更新失败！待更新(%d) != 实际更新(%d)", len(ms), okNum)
	}

	return err
}

func (m *PayOrderModel) updatePartial(data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(payOrderFieldList, data)
	if err != nil {
		return err
	}

	payOrderIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayOrderIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return conn.Exec(query, updateArgs.Args...)
	}, payOrderIdKey)
	return err
}

func (m *PayOrderModel) TxUpdate(tx sqlx.TxSession, data PayOrder) error {
	payOrderIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayOrderIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + payOrderFieldsWithPlaceHolder + ` where id = ?`
		return tx.Exec(query, data.Uid, data.PayType, data.BuyType, data.Status, data.Amount, data.BuyGoodsKey, data.Extra, data.TransactionId, data.PayAppId, data.PaySuccTime, data.Id)
	}, payOrderIdKey)
	return err
}

func (m *PayOrderModel) TxUpdatePartial(tx sqlx.TxSession, ms ...g.Map) (err error) {
	okNum := 0
	fx.From(func(source chan<- interface{}) {
		for _, data := range ms {
			source <- data
		}
	}).Parallel(func(item interface{}) {
		err = m.txUpdatePartial(tx, item.(g.Map))
		if err != nil {
			return
		}
		okNum++
	})

	if err == nil && okNum != len(ms) {
		err = fmt.Errorf("部分事务型局部更新失败！待更新(%d) != 实际更新(%d)", len(ms), okNum)
	}
	return err
}

func (m *PayOrderModel) txUpdatePartial(tx sqlx.TxSession, data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(payOrderFieldList, data)
	if err != nil {
		return err
	}

	payOrderIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayOrderIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return tx.Exec(query, updateArgs.Args...)
	}, payOrderIdKey)
	return err
}

func (m *PayOrderModel) Delete(id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformPayOrderIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return conn.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}

func (m *PayOrderModel) TxDelete(tx sqlx.TxSession, id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformPayOrderIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return tx.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}
