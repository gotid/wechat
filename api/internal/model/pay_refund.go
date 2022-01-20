package model

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

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
	payRefundFieldList             = builder.FieldList(&PayRefund{})
	payRefundFields                = strings.Join(payRefundFieldList, ",")
	payRefundFieldsAutoSet         = strings.Join(stringx.RemoveDBFields(payRefundFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), ",")
	payRefundFieldsWithPlaceHolder = strings.Join(stringx.RemoveDBFields(payRefundFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), "=?,") + "=?"

	cacheWechatPlatformPayRefundIdPrefix = "cache:wechatPlatform:payRefund:id:"
)

type (
	PayRefund struct {
		Id            int64           `db:"id" json:"id"`
		Uid           int64           `db:"uid" json:"uid"`                      // 用户id
		RefundStatus  int64           `db:"refund_status" json:"refundStatus"`   // 退款状态 -1-失败 0-退款中 1-成功
		RefundMoney   int64           `db:"refund_money" json:"refundMoney"`     // 当前退款单退款金额
		RefundMsg     sqlx.NullString `db:"refund_msg" json:"refundMsg"`         // 退款备注
		RefundTotal   int64           `db:"refund_total" json:"refundTotal"`     // 已退款金额(不包含此笔退款)
		PayMoney      int64           `db:"pay_money" json:"payMoney"`           // 此笔交易金额
		PayType       int64           `db:"pay_type" json:"payType"`             // 支付类型 2-支付宝  3-微信
		PayId         string          `db:"pay_id" json:"payId"`                 // 支付单号
		PayAppId      sqlx.NullString `db:"pay_app_id" json:"payAppId"`          // 应用id
		TransactionId string          `db:"transaction_id" json:"transactionId"` // 第三方流水号
		RefundFrom    string          `db:"refund_from" json:"refundFrom"`       // 退款来源
		ResultLog     string          `db:"result_log" json:"resultLog"`         // 返回结果日志
		CreateTime    time.Time       `db:"create_time" json:"createTime"`
		UpdateTime    time.Time       `db:"update_time" json:"updateTime"`
	}

	PayRefundModel struct {
		sqlx.CachedConn
		table string
	}
)

func NewPayRefundModel(conn sqlx.Conn, clusterConf cache.ClusterConf) *PayRefundModel {
	return &PayRefundModel{
		CachedConn: sqlx.NewCachedConnWithCluster(conn, clusterConf),
		table:      "pay_refund",
	}
}

func (m *PayRefundModel) Insert(data PayRefund) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + payRefundFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return m.ExecNoCache(query, data.Uid, data.RefundStatus, data.RefundMoney, data.RefundMsg, data.RefundTotal, data.PayMoney, data.PayType, data.PayId, data.PayAppId, data.TransactionId, data.RefundFrom, data.ResultLog)
}

func (m *PayRefundModel) TxInsert(tx sqlx.TxSession, data PayRefund) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + payRefundFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return tx.Exec(query, data.Uid, data.RefundStatus, data.RefundMoney, data.RefundMsg, data.RefundTotal, data.PayMoney, data.PayType, data.PayId, data.PayAppId, data.TransactionId, data.RefundFrom, data.ResultLog)
}

func (m *PayRefundModel) FindOne(id int64) (*PayRefund, error) {
	payRefundIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayRefundIdPrefix, id)
	var dest PayRefund
	err := m.Query(&dest, payRefundIdKey, func(conn sqlx.Conn, v interface{}) error {
		query := `select ` + payRefundFields + ` from ` + m.table + ` where id = ? limit 1`
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

func (m *PayRefundModel) FindMany(ids []int64, workers ...int) (list []*PayRefund) {
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
		list = append(list, one.(*PayRefund))
	}

	sort.Slice(list, func(i, j int) bool {
		return gutil.IndexOf(list[i].Id, ids) < gutil.IndexOf(list[j].Id, ids)
	})

	return
}

func (m *PayRefundModel) Update(data PayRefund) error {
	payRefundIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayRefundIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + payRefundFieldsWithPlaceHolder + ` where id = ?`
		return conn.Exec(query, data.Uid, data.RefundStatus, data.RefundMoney, data.RefundMsg, data.RefundTotal, data.PayMoney, data.PayType, data.PayId, data.PayAppId, data.TransactionId, data.RefundFrom, data.ResultLog, data.Id)
	}, payRefundIdKey)
	return err
}

func (m *PayRefundModel) UpdatePartial(ms ...g.Map) (err error) {
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

func (m *PayRefundModel) updatePartial(data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(payRefundFieldList, data)
	if err != nil {
		return err
	}

	payRefundIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayRefundIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return conn.Exec(query, updateArgs.Args...)
	}, payRefundIdKey)
	return err
}

func (m *PayRefundModel) TxUpdate(tx sqlx.TxSession, data PayRefund) error {
	payRefundIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayRefundIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + payRefundFieldsWithPlaceHolder + ` where id = ?`
		return tx.Exec(query, data.Uid, data.RefundStatus, data.RefundMoney, data.RefundMsg, data.RefundTotal, data.PayMoney, data.PayType, data.PayId, data.PayAppId, data.TransactionId, data.RefundFrom, data.ResultLog, data.Id)
	}, payRefundIdKey)
	return err
}

func (m *PayRefundModel) TxUpdatePartial(tx sqlx.TxSession, ms ...g.Map) (err error) {
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

func (m *PayRefundModel) txUpdatePartial(tx sqlx.TxSession, data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(payRefundFieldList, data)
	if err != nil {
		return err
	}

	payRefundIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayRefundIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return tx.Exec(query, updateArgs.Args...)
	}, payRefundIdKey)
	return err
}

func (m *PayRefundModel) Delete(id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformPayRefundIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return conn.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}

func (m *PayRefundModel) TxDelete(tx sqlx.TxSession, id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformPayRefundIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return tx.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}
