package context

import (
	"github.com/gotid/wechat/cache"
	"net/http"
)

// Context 微信上下文结构
type Context struct {
	// 开放平台、客服消息公用部分
	AppID          string // 小程序/平台 APPID
	AppSecret      string // 小程序/平台 AppSecret
	Token          string // 消息校验Token
	EncodingAESKey string // 消息加解密Key

	// 支付商户部分
	PayMchID     string // 商户ID
	PayNotifyURL string // 微信支付结果通知的接口地址
	PayKey       string // 商户后台设置的支付 key
	P12          []byte // 商户证书文件

	// 微信接口请求响应
	Writer  http.ResponseWriter
	Request *http.Request

	// 令牌等信息缓存
	Cache cache.Cache
}
