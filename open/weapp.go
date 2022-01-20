package open

import (
	"encoding/json"
	"fmt"
	"github.com/gotid/wechat/context"
	"github.com/gotid/wechat/util"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// WeApp 代小程序控制器
type WeApp struct {
	*Open

	AppID        string // 授权方小程序ID
	RefreshToken string // 授权方接口刷新令牌
}

// 拉取代小程序网络请求
func (wa *WeApp) get(rawURL string, params map[string]string) (resp []byte, err error) {
	// 构建完整请求网址
	uri, err := wa.buildRequestURI(rawURL, params)
	if err != nil {
		return nil, err
	}

	// 拉取网络请求
	resp, err = util.HTTPGet(uri)
	return
}

// 拉取代小程序图片类数据
func (wa *WeApp) getImage(rawURL string, params map[string]string) (resp []byte, err error) {
	// 构建完整请求网址
	uri, err := wa.buildRequestURI(rawURL, params)
	if err != nil {
		return nil, err
	}

	// 拉取网络请求
	response, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// 判断响应状态
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("WeApp.getImage失败：网址=%s，状态码=%d", uri, response.StatusCode)
		return
	}

	// 读取响应数据
	body, err := ioutil.ReadAll(response.Body)
	contentType := response.Header.Get("Content-Type")

	// 根据内容类型返回响应
	if contentType == "image/jpeg" {
		return body, nil
	} else if strings.HasPrefix(contentType, "application/json") {
		var jsonErr util.WechatError
		err = json.Unmarshal(body, &jsonErr)
		if err == nil && jsonErr.ErrCode != 0 {
			err = fmt.Errorf("WeApp.getImage失败：[%d] %s", jsonErr.ErrCode, jsonErr.ErrMsg)
			return nil, err
		}
	} else {
		err = fmt.Errorf("WeApp.getImage失败，期待 image/jpeg，实际返回：%s", contentType)
		return nil, err
	}

	return
}

// 投递代小程序网络请求
func (wa *WeApp) post(rawURL string, body map[string]string) (resp []byte, err error) {
	// 构建完整请求网址
	uri, err := wa.buildRequestURI(rawURL, nil)
	if err != nil {
		return nil, err
	}

	// 拉取网络请求
	resp, err = util.PostJSON(uri, body)
	return
}

// 构建带参的完整请求网址
func (wa *WeApp) buildRequest(rawURL string, params map[string]string) (fullURL string, err error) {
	// 解析网址
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// 从缓存获取授权方访问令牌
	accessToken, err := wa.AuthorizerAccessToken(wa.AppID)
	// 缓存获取不到，则网络获取
	if err != nil {
		var token *context.AuthorizerToken
		token, err = wa.RefreshAuthorizerToken(wa.AppID, wa.RefreshToken)
		if err != nil {
			return
		}
		accessToken = token.AccessToken
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
