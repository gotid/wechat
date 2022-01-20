package open

import (
	"github.com/gotid/wechat/context"
	"github.com/gotid/wechat/util"
	"net/url"
)

// Open 微信开放平台控制器
type Open struct {
	*context.Context
}

// New 返回一个新的开放平台控制器
func New(ctx *context.Context) *Open {
	return &Open{ctx}
}

// GetWeApp 获取指定的代小程序
func (o *Open) GetWeApp(appID string, refreshToken string) *WeApp {
	if appID == "" || refreshToken == "" {
		return nil
	}

	return &WeApp{
		Open:         o,
		AppID:        appID,
		RefreshToken: refreshToken,
	}
}

// 拉取开放平台网络请求
func (o *Open) get(rawURL string, params map[string]string) (resp []byte, err error) {
	// 构建完整请求网址
	uri, err := o.buildRequestURI(rawURL, params)
	if err != nil {
		return nil, err
	}

	// 拉取网络请求
	resp, err = util.HTTPGet(uri)
	return
}

// 投递开放平台网络请求
func (o *Open) post(rawURL string, body map[string]string) (resp []byte, err error) {
	// 构建完整请求网址
	uri, err := o.buildRequestURI(rawURL, nil)
	if err != nil {
		return nil, err
	}

	// 拉取网络请求
	resp, err = util.PostJSON(uri, body)
	return
}

// 构建带参的完整请求网址
func (o *Open) buildRequestURI(rawURL string, params map[string]string) (fullURL string, err error) {
	// 解析网址
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// 从缓存获取微信推送的平台访问令牌
	accessToken, err := o.ComponentAccessToken()
	if err != nil {
		return "", err
	}

	// 增加请求参数
	query := parsedURL.Query()
	query.Add("access_token", accessToken)
	if params != nil {
		for k, v := range params {
			query.Set(k, v)
		}
	}

	// 返回完整网址
	parsedURL.RawQuery = query.Encode()
	fullURL = parsedURL.String()
	return
}
