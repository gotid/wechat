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
	payFieldList             = builder.FieldList(&Pay{})
	payFields                = strings.Join(payFieldList, ",")
	payFieldsAutoSet         = strings.Join(stringx.RemoveDBFields(payFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), ",")
	payFieldsWithPlaceHolder = strings.Join(stringx.RemoveDBFields(payFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), "=?,") + "=?"

	cacheWechatPlatformPayIdPrefix = "cache:wechatPlatform:pay:id:"
)

type (
	Pay struct {
		Id           int64     `db:"id" json:"id"`                       // 自增id
		MchId        string    `db:"mch_id" json:"mchId"`                // 支付商户号id
		Token        string    `db:"token" json:"token"`                 // 支付密钥
		Cert         string    `db:"cert" json:"cert"`                   // 支付证书
		PayNotifyUrl string    `db:"pay_notify_url" json:"payNotifyUrl"` // 支付回调
		PayRefundUrl string    `db:"pay_refund_url" json:"payRefundUrl"` // 退款回调
		CreateTime   time.Time `db:"create_time" json:"createTime"`
		UpdateTime   time.Time `db:"update_time" json:"updateTime"`
	}

	PayModel struct {
		sqlx.CachedConn
		table string
	}
)

func NewPayModel(conn sqlx.Conn, clusterConf cache.ClusterConf) *PayModel {
	return &PayModel{
		CachedConn: sqlx.NewCachedConnWithCluster(conn, clusterConf),
		table:      "pay",
	}
}

func (m *PayModel) Insert(data Pay) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + payFieldsAutoSet + `) values (?, ?, ?, ?, ?)`
	return m.ExecNoCache(query, data.MchId, data.Token, data.Cert, data.PayNotifyUrl, data.PayRefundUrl)
}

func (m *PayModel) TxInsert(tx sqlx.TxSession, data Pay) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + payFieldsAutoSet + `) values (?, ?, ?, ?, ?)`
	return tx.Exec(query, data.MchId, data.Token, data.Cert, data.PayNotifyUrl, data.PayRefundUrl)
}

func (m *PayModel) FindOne(id int64) (*Pay, error) {
	payIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayIdPrefix, id)
	var dest Pay
	err := m.Query(&dest, payIdKey, func(conn sqlx.Conn, v interface{}) error {
		query := `select ` + payFields + ` from ` + m.table + ` where id = ? limit 1`
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

func (m *PayModel) FindMany(ids []int64, workers ...int) (list []*Pay) {
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
		list = append(list, one.(*Pay))
	}

	sort.Slice(list, func(i, j int) bool {
		return gutil.IndexOf(list[i].Id, ids) < gutil.IndexOf(list[j].Id, ids)
	})

	return
}

func (m *PayModel) Update(data Pay) error {
	payIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + payFieldsWithPlaceHolder + ` where id = ?`
		return conn.Exec(query, data.MchId, data.Token, data.Cert, data.PayNotifyUrl, data.PayRefundUrl, data.Id)
	}, payIdKey)
	return err
}

func (m *PayModel) UpdatePartial(ms ...g.Map) (err error) {
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

func (m *PayModel) updatePartial(data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(payFieldList, data)
	if err != nil {
		return err
	}

	payIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return conn.Exec(query, updateArgs.Args...)
	}, payIdKey)
	return err
}

func (m *PayModel) TxUpdate(tx sqlx.TxSession, data Pay) error {
	payIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + payFieldsWithPlaceHolder + ` where id = ?`
		return tx.Exec(query, data.MchId, data.Token, data.Cert, data.PayNotifyUrl, data.PayRefundUrl, data.Id)
	}, payIdKey)
	return err
}

func (m *PayModel) TxUpdatePartial(tx sqlx.TxSession, ms ...g.Map) (err error) {
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

func (m *PayModel) txUpdatePartial(tx sqlx.TxSession, data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(payFieldList, data)
	if err != nil {
		return err
	}

	payIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPayIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return tx.Exec(query, updateArgs.Args...)
	}, payIdKey)
	return err
}

func (m *PayModel) Delete(id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformPayIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return conn.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}

func (m *PayModel) TxDelete(tx sqlx.TxSession, id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformPayIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return tx.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}
