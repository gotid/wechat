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
	weappAuditFieldList             = builder.FieldList(&WeappAudit{})
	weappAuditFields                = strings.Join(weappAuditFieldList, ",")
	weappAuditFieldsAutoSet         = strings.Join(stringx.RemoveDBFields(weappAuditFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), ",")
	weappAuditFieldsWithPlaceHolder = strings.Join(stringx.RemoveDBFields(weappAuditFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), "=?,") + "=?"

	cacheWechatPlatformWeappAuditIdPrefix = "cache:wechatPlatform:weappAudit:id:"
)

type (
	WeappAudit struct {
		Id                   int64     `db:"id" json:"id"`                                       // ID
		AppId                string    `db:"app_id" json:"appId"`                                // 小程序appId
		OriginalId           string    `db:"original_id" json:"originalId"`                      // 小程序原始id
		AuditId              int64     `db:"audit_id" json:"auditId"`                            // 审核编号
		State                int64     `db:"state" json:"state"`                                 // 审核状态，-1-撤销审核 1为审核中，2为审核成功，3为审核失败
		Reason               string    `db:"reason" json:"reason"`                               // 当status=1，审核被拒绝时，返回的拒绝原因
		ScreenShot           string    `db:"screen_shot" json:"screenShot"`                      // 附件素材
		TemplateId           int64     `db:"template_id" json:"templateId"`                      // 最新提交审核或者发布的模板id
		TemplateAppId        string    `db:"template_app_id" json:"templateAppId"`               // 模板开发小程序ID
		TemplateAppName      string    `db:"template_app_name" json:"templateAppName"`           // 开发小程序名
		TemplateAppDeveloper string    `db:"template_app_developer" json:"templateAppDeveloper"` // 开发者
		TemplateDesc         string    `db:"template_desc" json:"templateDesc"`                  // 模板描述
		TemplateVersion      string    `db:"template_version" json:"templateVersion"`            // 模板版本号
		CreateTime           time.Time `db:"create_time" json:"createTime"`
		UpdateTime           time.Time `db:"update_time" json:"updateTime"`
	}

	WeappAuditModel struct {
		sqlx.CachedConn
		table string
	}
)

func NewWeappAuditModel(conn sqlx.Conn, clusterConf cache.ClusterConf) *WeappAuditModel {
	return &WeappAuditModel{
		CachedConn: sqlx.NewCachedConnWithCluster(conn, clusterConf),
		table:      "weapp_audit",
	}
}

func (m *WeappAuditModel) Insert(data WeappAudit) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + weappAuditFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return m.ExecNoCache(query, data.AppId, data.OriginalId, data.AuditId, data.State, data.Reason, data.ScreenShot, data.TemplateId, data.TemplateAppId, data.TemplateAppName, data.TemplateAppDeveloper, data.TemplateDesc, data.TemplateVersion)
}

func (m *WeappAuditModel) TxInsert(tx sqlx.TxSession, data WeappAudit) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + weappAuditFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return tx.Exec(query, data.AppId, data.OriginalId, data.AuditId, data.State, data.Reason, data.ScreenShot, data.TemplateId, data.TemplateAppId, data.TemplateAppName, data.TemplateAppDeveloper, data.TemplateDesc, data.TemplateVersion)
}

func (m *WeappAuditModel) FindOne(id int64) (*WeappAudit, error) {
	weappAuditIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappAuditIdPrefix, id)
	var dest WeappAudit
	err := m.Query(&dest, weappAuditIdKey, func(conn sqlx.Conn, v interface{}) error {
		query := `select ` + weappAuditFields + ` from ` + m.table + ` where id = ? limit 1`
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

func (m *WeappAuditModel) FindMany(ids []int64, workers ...int) (list []*WeappAudit) {
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
		list = append(list, one.(*WeappAudit))
	}

	sort.Slice(list, func(i, j int) bool {
		return gutil.IndexOf(list[i].Id, ids) < gutil.IndexOf(list[j].Id, ids)
	})

	return
}

func (m *WeappAuditModel) Update(data WeappAudit) error {
	weappAuditIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappAuditIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + weappAuditFieldsWithPlaceHolder + ` where id = ?`
		return conn.Exec(query, data.AppId, data.OriginalId, data.AuditId, data.State, data.Reason, data.ScreenShot, data.TemplateId, data.TemplateAppId, data.TemplateAppName, data.TemplateAppDeveloper, data.TemplateDesc, data.TemplateVersion, data.Id)
	}, weappAuditIdKey)
	return err
}

func (m *WeappAuditModel) UpdatePartial(ms ...g.Map) (err error) {
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

func (m *WeappAuditModel) updatePartial(data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(weappAuditFieldList, data)
	if err != nil {
		return err
	}

	weappAuditIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappAuditIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return conn.Exec(query, updateArgs.Args...)
	}, weappAuditIdKey)
	return err
}

func (m *WeappAuditModel) TxUpdate(tx sqlx.TxSession, data WeappAudit) error {
	weappAuditIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappAuditIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + weappAuditFieldsWithPlaceHolder + ` where id = ?`
		return tx.Exec(query, data.AppId, data.OriginalId, data.AuditId, data.State, data.Reason, data.ScreenShot, data.TemplateId, data.TemplateAppId, data.TemplateAppName, data.TemplateAppDeveloper, data.TemplateDesc, data.TemplateVersion, data.Id)
	}, weappAuditIdKey)
	return err
}

func (m *WeappAuditModel) TxUpdatePartial(tx sqlx.TxSession, ms ...g.Map) (err error) {
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

func (m *WeappAuditModel) txUpdatePartial(tx sqlx.TxSession, data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(weappAuditFieldList, data)
	if err != nil {
		return err
	}

	weappAuditIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformWeappAuditIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return tx.Exec(query, updateArgs.Args...)
	}, weappAuditIdKey)
	return err
}

func (m *WeappAuditModel) Delete(id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformWeappAuditIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return conn.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}

func (m *WeappAuditModel) TxDelete(tx sqlx.TxSession, id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	keys := make([]string, len(id))
	for i, v := range id {
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformWeappAuditIdPrefix, v)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return tx.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}
