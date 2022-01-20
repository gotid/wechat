package context

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"git.zc0901.com/go/god/lib/g"
	"github.com/gotid/wechat/util"
)

// AuthType 要授权的帐号类型
type AuthType int

func (at AuthType) Int() int {
	return int(at)
}

const (
	urlCreatePreAuthCode  = "https://api.weixin.qq.com/cgi-bin/component/api_create_preauthcode?component_access_token=%s"
	urlComponentLoginPage = "https://mp.weixin.qq.com/cgi-bin/componentloginpage?component_appid=%s&pre_auth_code=%s&redirect_uri=%s&auth_type=2"
	urlBindComponent      = "https://mp.weixin.qq.com/safe/bindcomponent?action=bindcomponent&auth_type=2&no_scan=1&component_appid=%s&pre_auth_code=%s&redirect_uri=%s#wechat_redirect"

	AuthTypeMP  AuthType = 1 // 商户点击链接后，手机端仅展示公众号
	AuthTypeWA  AuthType = 2 // 仅展示小程序
	AuthTypeAll AuthType = 3 // 公众号和小程序都展示
)

// Auth 跳转至授权网页。
// 自动判断是否在微信内部打开。
func (ctx *Context) Auth(w http.ResponseWriter, r *http.Request, redirectURI string) error {
	uri, err := ctx.AuthURL(util.InMicroMessenger(r.UserAgent()), redirectURI)
	if err != nil {
		return err
	}

	http.Redirect(w, r, uri, 302)
	return nil
}

// AuthURL 获取PC端/移动端授权链接
func (ctx *Context) AuthURL(isMobile bool, redirectURI string) (string, error) {
	preAuthCode, err := ctx.PreAuthCode()
	if err != nil {
		return "", err
	}
	uri := url.QueryEscape(redirectURI)
	if isMobile {
		return fmt.Sprintf(urlBindComponent, ctx.AppID, preAuthCode, uri), nil
	}
	return fmt.Sprintf(urlComponentLoginPage, ctx.AppID, preAuthCode, uri), nil
}

// PreAuthCode 获取预授权码。
func (ctx *Context) PreAuthCode() (string, error) {
	accessToken, err := ctx.ComponentAccessToken()
	if err != nil {
		return "", err
	}

	data, err := util.PostJSON(fmt.Sprintf(urlCreatePreAuthCode, accessToken), g.Map{
		"component_appid": ctx.AppID,
	})
	if err != nil {
		return "", err
	}

	var ret struct {
		PreAuthCode string `json:"pre_auth_code"`
	}
	if err = json.Unmarshal(data, &ret); err != nil {
		return "", err
	}

	return ret.PreAuthCode, nil
}
