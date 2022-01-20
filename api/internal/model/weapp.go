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
	weappFieldList             = builder.FieldList(&Weapp{})
	weappFields                = strings.Join(weappFieldList, ",")
	weappFieldsAutoSet         = strings.Join(stringx.RemoveDBFields(weappFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), ",")
	weappFieldsWithPlaceHolder = strings.Join(stringx.RemoveDBFields(weappFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), "=?,") + "=?"

	cacheWechatPlatformWeappIdPrefix = "cache:wechatPlatform:weapp:id:"
)

type (
	Weapp struct {
		Id             int64           `db:"id" json:"id"`                          // 自增id
		AppId          string          `db:"app_id" json:"appId"`                   // 小程序appid
		PlatformId     string          `db:"platform_id" json:"platformId"`         // 开放平台ID
		MchId          sqlx.NullString `db:"mch_id" json:"mchId"`                   // 支付商户号id
		OriginalId     string          `db:"original_id" json:"originalId"`         // 原始ID
		RefreshToken   string          `db:"refresh_token" json:"refreshToken"`     // 接口调用凭据刷新令牌
		Secret         string          `db:"secret" json:"secret"`                  // 小程序secret
		ExtConfig      string          `db:"ext_config" json:"extConfig"`           // 小程序扩展配置
		State          int64           `db:"state" json:"state"`                    // -1-授权失效 1授权成功，2审核中，3审核通过，4审核失败，5已发布 6已撤审
		Version        string          `db:"version" json:"version"`                // 当前版本
		NowTemplateId  sqlx.NullInt64  `db:"now_template_id" json:"nowTemplateId"`  // 当前模板ID
		TemplateListen string          `db:"template_listen" json:"templateListen"` // 模板监听开发小程序(appid)
		AuditId        int64           `db:"audit_id" json:"auditId"`               // 审核编号
		AutoAudit      int64           `db:"auto_audit" json:"autoAudit"`           // 自动提审(升级) -1否 1是
		AutoRelease    int64           `db:"auto_release" json:"autoRelease"`       // 自动发布-1否 1是
		CreateTime     time.Time       `db:"create_time" json:"createTime"`
		UpdateTime     time.Time       `db:"update_time" json:"updateTime"`
	}

	WeappModel struct {
		sqlx.CachedConn
		table string
	}
)

func NewWeappModel(conn sqlx.Conn, clusterConf cache.ClusterConf) *WeappModel {
	return &WeappModel{
		CachedConn: sqlx.NewCachedConnWithCluster(conn, clusterConf),
		table:      "weapp",
	}
}

func (m *WeappModel) Insert(data Weapp) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + weappFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return m.ExecNoCache(query, data.AppId, data.PlatformId, data.MchId, data.OriginalId, data.RefreshToken, data.Secret, data.ExtConfig, data.State, data.Version, data.NowTemplateId, data.TemplateListen, data.AuditId, data.AutoAudit, data.AutoRelease)
}

func (m *WeappModel) TxInsert(tx sqlx.TxSession, data Weapp) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + weappFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return tx.Exec(query, data.AppId, data.PlatformId, data.MchId, data.OriginalId, data.RefreshToken, data.Secret, data.ExtConfig, data.State, data.Version, data.NowTemplateId, data.TemplateListen, data.AuditId, data.AutoAudit, data.AutoRelease)
}

func (m *WeappModel) FindOne(id int64) (*Weapp, error) {
	weappIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappIdPrefix, id)
	var dest Weapp
	err := m.Query(&dest, weappIdKey, func(conn sqlx.Conn, v interface{}) error {
		query := `select ` + weappFields + ` from ` + m.table + ` where id = ? limit 1`
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

func (m *WeappModel) FindMany(ids []int64, workers ...int) (list []*Weapp) {
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
		list = append(list, one.(*Weapp))
	}

	sort.Slice(list, func(i, j int) bool {
		return gutil.IndexOf(list[i].Id, ids) < gutil.IndexOf(list[j].Id, ids)
	})

	return
}

func (m *WeappModel) Update(data Weapp) error {
	weappIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + weappFieldsWithPlaceHolder + ` where id = ?`
		return conn.Exec(query, data.AppId, data.PlatformId, data.MchId, data.OriginalId, data.RefreshToken, data.Secret, data.ExtConfig, data.State, data.Version, data.NowTemplateId, data.TemplateListen, data.AuditId, data.AutoAudit, data.AutoRelease, data.Id)
	}, weappIdKey)
	return err
}

func (m *WeappModel) UpdatePartial(ms ...g.Map) (err error) {
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

func (m *WeappModel) updatePartial(data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(weappFieldList, data)
	if err != nil {
		return err
	}

	weappIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return conn.Exec(query, updateArgs.Args...)
	}, weappIdKey)
	return err
}

func (m *WeappModel) TxUpdate(tx sqlx.TxSession, data Weapp) error {
	weappIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + weappFieldsWithPlaceHolder + ` where id = ?`
		return tx.Exec(query, data.AppId, data.PlatformId, data.MchId, data.OriginalId, data.RefreshToken, data.Secret, data.ExtConfig, data.State, data.Version, data.NowTemplateId, data.TemplateListen, data.AuditId, data.AutoAudit, data.AutoRelease, data.Id)
	}, weappIdKey)
	return err
}

func (m *WeappModel) TxUpdatePartial(tx sqlx.TxSession, ms ...g.Map) (err error) {
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

func (m *WeappModel) txUpdatePartial(tx sqlx.TxSession, data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(weappFieldList, data)
	if err != nil {
		return err
	}

	weappIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return tx.Exec(query, updateArgs.Args...)
	}, weappIdKey)
	return err
}

func (m *WeappModel) Delete(id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformWeappIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return conn.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}

func (m *WeappModel) TxDelete(tx sqlx.TxSession, id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformWeappIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return tx.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}
