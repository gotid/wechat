package context

import (
	"encoding/json"
	"fmt"
	"github.com/gotid/god/lib/g"
	"github.com/gotid/wechat/cache"
	"github.com/gotid/wechat/util"
	"time"
)

const urlComponentAccessToken = "https://api.weixin.qq.com/cgi-bin/component/api_component_token"

// ComponentAccessToken 是一个第三方平台访问令牌。
type ComponentAccessToken struct {
	util.WechatError
	AccessToken string `json:"component_access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// SetComponentAccessToken 设置第三方平台访问令牌。
func (ctx *Context) SetComponentAccessToken(verifyTicket string) (*ComponentAccessToken, error) {
	data, err := util.PostJSON(urlComponentAccessToken, g.Map{
		"component_appid":         ctx.AppID,
		"component_appsecret":     ctx.AppSecret,
		"component_verify_ticket": verifyTicket,
	})
	if err != nil {
		return nil, err
	}

	token := &ComponentAccessToken{}
	if err = json.Unmarshal(data, token); err != nil {
		return nil, err
	}

	if token.ErrCode != 0 {
		return nil, fmt.Errorf("SetComponentAccessToken 错误，"+
			"errcode=%d, errmsg=%s", token.ErrCode, token.ErrMsg)
	}

	key := cache.KeyComponentAccessToken(ctx.AppID)
	timeout := time.Duration(token.ExpiresIn-1500) * time.Second
	err = ctx.Cache.Set(key, token.AccessToken, timeout)
	if err != nil {
		return nil, fmt.Errorf("SetComponentAccessToken 错误：%v", err)
	}

	return token, nil
}

// ComponentAccessToken 从缓存中获取第三方平台访问令牌。
func (ctx *Context) ComponentAccessToken() (token string, err error) {
	key := cache.KeyComponentAccessToken(ctx.AppID)
	val := ctx.Cache.Get(key)
	if v, ok := val.(string); ok {
		token = v
	}

	if token == "" {
		ticket, err := ctx.ComponentVerifyTicket()
		if err != nil {
			return "", err
		}
		at, err := ctx.SetComponentAccessToken(ticket)
		if err != nil {
			return "", err
		}
		token = at.AccessToken
	}

	if token == "" {
		return "", fmt.Errorf("平台令牌初始化中，请10分钟后再试")
	}

	return token, nil
}
