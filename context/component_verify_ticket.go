package context

import (
	"fmt"
	"git.zc0901.com/go/god/lib/logx"
	"github.com/gotid/wechat/cache"
)

// SetComponentVerifyTicket 保存每 10 分钟推送一次的第三方平台票据
func (ctx *Context) SetComponentVerifyTicket(v string) {
	err := ctx.Cache.Set(cache.KeyComponentVerifyTicket(ctx.AppID), v, 0)
	if err != nil {
		logx.Errorf("保存开放平台票据失败：%v", err)
	}
}

// ComponentVerifyTicket 获取第三方平台票据
func (ctx *Context) ComponentVerifyTicket() (string, error) {
	err := fmt.Errorf("无法从缓存获取 component verify ticket")

	val := ctx.Cache.Get(cache.KeyComponentVerifyTicket(ctx.AppID))
	if val == nil {
		return "", err
	}
	if ticket := val.(string); ticket != "" {
		return ticket, nil
	}

	return "", err
}
