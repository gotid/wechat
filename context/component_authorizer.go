package context

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gotid/god/lib/g"
	"github.com/gotid/wechat/util"
)

const (
	urlQueryAuth       = "https://api.weixin.qq.com/cgi-bin/component/api_query_auth?component_access_token=%s"
	urlAuthorizerToken = "https://api.weixin.qq.com/cgi-bin/component/api_authorizer_token?component_access_token=%s"
	urlAuthorizerInfo  = "https://api.weixin.qq.com/cgi-bin/component/api_get_authorizer_info?component_access_token=%s"
)

type (
	// AuthorizerToken 授权方令牌
	AuthorizerToken struct {
		AppID        string `json:"authorizer_appid"`
		AccessToken  string `json:"authorizer_access_token"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"authorizer_refresh_token"`
	}

	// AuthorizerInfo 授权方的帐号信息
	// https://developers.weixin.qq.com/doc/oplatform/Third-party_Platforms/2.0/api/ThirdParty/token/api_get_authorizer_info.html
	AuthorizerInfo struct {
		NickName        string `json:"nick_name"`
		HeadImg         string `json:"head_img"`
		ServiceTypeInfo AuthID `json:"service_type_info"` // 公众号类型
		VerifyTypeInfo  AuthID `json:"verify_type_info"`  // 公众号认证类型
		UserName        string `json:"user_name"`
		Alias           string `json:"alias"`
		QrcodeURL       string `json:"qrcode_url"`
		PrincipalName   string `json:"principal_name"`
		Signature       string `json:"signature"` // 小程序名称
		BusinessInfo    struct {
			OpenStore string `json:"open_store"` // 门店
			OpenScan  string `json:"open_scan"`  // 扫商品
			OpenPay   string `json:"open_pay"`   // 支付
			OpenCard  string `json:"open_card"`  // 卡券
			OpenShake string `json:"open_shake"` // 摇一摇
		}
		MiniProgramInfo struct {
			Network struct {
				RequestDomain   []string
				WsRequestDomain []string
				UploadDomain    []string
				DownloadDomain  []string
				BizDomain       []string
				UDPDomain       []string
			} `json:"network"`
			Categories []struct {
				First  string `json:"first"`
				Second string `json:"second"`
			} `json:"categories"`
			VisitStatus int8 `json:"visit_status"`
		}
	}

	// AuthorizationInfo 授权方的授权信息
	AuthorizationInfo struct {
		AuthorizerToken
		FuncInfo []AuthFuncInfo `json:"func_info"`
	}

	// AuthFuncInfo 授权类目
	AuthFuncInfo struct {
		FuncscopeCategory AuthID `json:"funcscope_category"`
	}

	// AuthID 授权ID
	AuthID struct {
		ID int `json:"id"`
	}
)

// QueryAuth 使用授权码获取授权信息
func (ctx *Context) QueryAuth(authCode string) (*AuthorizationInfo, error) {
	accessToken, err := ctx.ComponentAccessToken()
	if err != nil {
		return nil, err
	}

	data, err := util.PostJSON(fmt.Sprintf(urlQueryAuth, accessToken), g.Map{
		"component_appid":    ctx.AppID,
		"authorization_code": authCode,
	})
	if err != nil {
		return nil, err
	}

	var ret struct {
		util.WechatError
		AuthInfo *AuthorizationInfo `json:"authorization_info"`
	}
	if err = json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}

	if ret.ErrCode != 0 {
		return nil, fmt.Errorf("QueryAuth 错误："+
			"errcode=%d, errmsg=%s", ret.ErrCode, ret.ErrMsg)
	}

	return ret.AuthInfo, nil
}

// RefreshAuthorizerToken 刷新授权方接口的调用令牌
func (ctx *Context) RefreshAuthorizerToken(appID, refreshToken string) (*AuthorizerToken, error) {
	accessToken, err := ctx.ComponentAccessToken()
	if err != nil {
		return nil, err
	}

	data, err := util.PostJSON(fmt.Sprintf(urlAuthorizerToken, accessToken), g.Map{
		"component_appid":          ctx.AppID,
		"authorizer_appid":         appID,
		"authorizer_refresh_token": refreshToken,
	})
	if err != nil {
		return nil, err
	}

	ret := &AuthorizerToken{}
	if err = json.Unmarshal(data, ret); err != nil {
		return nil, err
	}

	key := "authorizer_token_" + appID
	if err = ctx.Cache.Set(key, ret.AccessToken, 80*time.Minute); err != nil {
		return nil, err
	}

	return ret, nil
}

// AuthorizerAccessToken 从缓存中获取授权方的访问令牌。
func (ctx *Context) AuthorizerAccessToken(appID string) (string, error) {
	key := "authorizer_token_" + appID
	val := ctx.Cache.Get(key)
	if val == nil {
		return "", fmt.Errorf("无法获取授权方 %s 的令牌", appID)
	}
	return val.(string), nil
}

// AuthorizerInfo 网络获取授权方的帐号基本信息。
func (ctx *Context) AuthorizerInfo(appID string) (*AuthorizerInfo, *AuthorizationInfo, error) {
	accessToken, err := ctx.ComponentAccessToken()
	if err != nil {
		return nil, nil, err
	}

	data, err := util.PostJSON(fmt.Sprintf(urlAuthorizerInfo, accessToken), g.Map{
		"component_appid":  ctx.AppID,
		"authorizer_appid": appID,
	})
	if err != nil {
		return nil, nil, err
	}

	var ret struct {
		AuthorizerInfo    *AuthorizerInfo    `json:"authorizer_info"`
		AuthorizationInfo *AuthorizationInfo `json:"authorization_info"`
	}
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, nil, err
	}

	return ret.AuthorizerInfo, ret.AuthorizationInfo, nil
}
