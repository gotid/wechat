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
	platformFieldList             = builder.FieldList(&Platform{})
	platformFields                = strings.Join(platformFieldList, ",")
	platformFieldsAutoSet         = strings.Join(stringx.RemoveDBFields(platformFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), ",")
	platformFieldsWithPlaceHolder = strings.Join(stringx.RemoveDBFields(platformFieldList, "id", "created_at", "updated_at", "create_time", "update_time"), "=?,") + "=?"

	cacheWechatPlatformPlatformAppIdPrefix = "cache:wechatPlatform:platform:appId:"
	cacheWechatPlatformPlatformIdPrefix    = "cache:wechatPlatform:platform:id:"
)

type (
	Platform struct {
		Id              int64     `db:"id" json:"id"`                             // 自增id
		AppId           string    `db:"app_id" json:"appId"`                      // 平台 appid
		AppSecret       string    `db:"app_secret" json:"appSecret"`              // 平台 appsecret
		Token           string    `db:"token" json:"token"`                       // 平台 消息校验Token
		EncodingAesKey  string    `db:"encoding_aes_key" json:"encodingAesKey"`   // 平台 消息加解密Key
		ServerDomain    string    `db:"server_domain" json:"serverDomain"`        // 小程序服务器域名
		BizDomain       string    `db:"biz_domain" json:"bizDomain"`              // 小程序业务域名
		ApiHost         string    `db:"api_host" json:"apiHost"`                  // 平台接口部署主机
		AuthRedirectUrl string    `db:"auth_redirect_url" json:"authRedirectUrl"` // 用户授权成功回跳地址
		CreateTime      time.Time `db:"create_time" json:"createTime"`
		UpdateTime      time.Time `db:"update_time" json:"updateTime"`
	}

	PlatformModel struct {
		sqlx.CachedConn
		table string
	}
)

func NewPlatformModel(conn sqlx.Conn, clusterConf cache.ClusterConf) *PlatformModel {
	return &PlatformModel{
		CachedConn: sqlx.NewCachedConnWithCluster(conn, clusterConf),
		table:      "platform",
	}
}

func (m *PlatformModel) Insert(data Platform) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + platformFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?)`
	return m.ExecNoCache(query, data.AppId, data.AppSecret, data.Token, data.EncodingAesKey, data.ServerDomain, data.BizDomain, data.ApiHost, data.AuthRedirectUrl)
}

func (m *PlatformModel) TxInsert(tx sqlx.TxSession, data Platform) (sql.Result, error) {
	query := `insert into ` + m.table + ` (` + platformFieldsAutoSet + `) values (?, ?, ?, ?, ?, ?, ?, ?)`
	return tx.Exec(query, data.AppId, data.AppSecret, data.Token, data.EncodingAesKey, data.ServerDomain, data.BizDomain, data.ApiHost, data.AuthRedirectUrl)
}

func (m *PlatformModel) FindOne(id int64) (*Platform, error) {
	platformIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPlatformIdPrefix, id)
	var dest Platform
	err := m.Query(&dest, platformIdKey, func(conn sqlx.Conn, v interface{}) error {
		query := `select ` + platformFields + ` from ` + m.table + ` where id = ? limit 1`
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

func (m *PlatformModel) FindMany(ids []int64, workers ...int) (list []*Platform) {
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
		list = append(list, one.(*Platform))
	}

	sort.Slice(list, func(i, j int) bool {
		return gutil.IndexOf(list[i].Id, ids) < gutil.IndexOf(list[j].Id, ids)
	})

	return
}

func (m *PlatformModel) FindOneByAppId(appId string) (*Platform, error) {
	platformAppIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPlatformAppIdPrefix, appId)
	var dest Platform
	err := m.QueryIndex(&dest, platformAppIdKey, func(primary interface{}) string {
		// 主键的缓存键
		return fmt.Sprintf("%s%v", cacheWechatPlatformPlatformIdPrefix, primary)
	}, func(conn sqlx.Conn, v interface{}) (i interface{}, e error) {
		// 无索引建——主键对应缓存，通过索引键查目标行
		query := `select ` + platformFields + ` from ` + m.table + ` where app_id = ? limit 1`
		if err := conn.Query(&dest, query, appId); err != nil {
			return nil, err
		}
		return dest.Id, nil
	}, func(conn sqlx.Conn, v, primary interface{}) error {
		// 如果有索引建——主键对应缓存，则通过主键直接查目标航
		query := `select ` + platformFields + ` from ` + m.table + ` where id = ? limit 1`
		return conn.Query(v, query, primary)
	})
	if err == nil {
		return &dest, nil
	} else if err == sqlx.ErrNotFound {
		return nil, ErrNotFound
	} else {
		return nil, err
	}
}

func (m *PlatformModel) FindManyByAppIds(keys []string, workers ...int) (list []*Platform) {
	keys = gconv.Strings(garray.NewArrayFrom(gconv.Interfaces(keys), true).Unique())

	var nWorkers int
	if len(workers) > 0 {
		nWorkers = workers[0]
	} else {
		nWorkers = mathx.MinInt(10, len(keys))
	}

	channel := mr.Map(func(source chan<- interface{}) {
		for _, key := range keys {
			source <- key
		}
	}, func(item interface{}, writer mr.Writer) {
		key := item.(string)
		one, err := m.FindOneByAppId(key)
		if err == nil {
			writer.Write(one)
		} else {
			logx.Error(err)
		}
	}, mr.WithWorkers(nWorkers))

	for one := range channel {
		list = append(list, one.(*Platform))
	}

	sort.Slice(list, func(i, j int) bool {
		return gutil.IndexOf(list[i].AppId, keys) < gutil.IndexOf(list[j].AppId, keys)
	})

	return
}

func (m *PlatformModel) Update(data Platform) error {
	platformIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPlatformIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + platformFieldsWithPlaceHolder + ` where id = ?`
		return conn.Exec(query, data.AppId, data.AppSecret, data.Token, data.EncodingAesKey, data.ServerDomain, data.BizDomain, data.ApiHost, data.AuthRedirectUrl, data.Id)
	}, platformIdKey)
	return err
}

func (m *PlatformModel) UpdatePartial(ms ...g.Map) (err error) {
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

func (m *PlatformModel) updatePartial(data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(platformFieldList, data)
	if err != nil {
		return err
	}

	platformIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPlatformIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return conn.Exec(query, updateArgs.Args...)
	}, platformIdKey)
	return err
}

func (m *PlatformModel) TxUpdate(tx sqlx.TxSession, data Platform) error {
	platformIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPlatformIdPrefix, data.Id)
	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + platformFieldsWithPlaceHolder + ` where id = ?`
		return tx.Exec(query, data.AppId, data.AppSecret, data.Token, data.EncodingAesKey, data.ServerDomain, data.BizDomain, data.ApiHost, data.AuthRedirectUrl, data.Id)
	}, platformIdKey)
	return err
}

func (m *PlatformModel) TxUpdatePartial(tx sqlx.TxSession, ms ...g.Map) (err error) {
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

func (m *PlatformModel) txUpdatePartial(tx sqlx.TxSession, data g.Map) error {
	updateArgs, err := sqlx.ExtractUpdateArgs(platformFieldList, data)
	if err != nil {
		return err
	}

	platformIdKey := fmt.Sprintf("%s%v", cacheWechatPlatformPlatformIdPrefix, updateArgs.Id)
	_, err = m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := `update ` + m.table + ` set ` + updateArgs.Fields + ` where id = ` + updateArgs.Id
		return tx.Exec(query, updateArgs.Args...)
	}, platformIdKey)
	return err
}

func (m *PlatformModel) Delete(id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	datas := m.FindMany(id)
	keys := make([]string, len(id)*2)
	for i, v := range id {
		data := datas[i]
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformPlatformIdPrefix, v)
		keys[i+1] = fmt.Sprintf("%s%v", cacheWechatPlatformPlatformAppIdPrefix, data.AppId)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return conn.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}

func (m *PlatformModel) TxDelete(tx sqlx.TxSession, id ...int64) error {
	if len(id) == 0 {
		return nil
	}

	datas := m.FindMany(id)
	keys := make([]string, len(id)*2)
	for i, v := range id {
		data := datas[i]
		keys[i] = fmt.Sprintf("%s%v", cacheWechatPlatformPlatformIdPrefix, v)
		keys[i+1] = fmt.Sprintf("%s%v", cacheWechatPlatformPlatformAppIdPrefix, data.AppId)
	}

	_, err := m.Exec(func(conn sqlx.Conn) (result sql.Result, err error) {
		query := fmt.Sprintf(`delete from `+m.table+` where id in (%s)`, sqlx.In(len(id)))
		return tx.Exec(query, gconv.Interfaces(id)...)
	}, keys...)
	return err
}
