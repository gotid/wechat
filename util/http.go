package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// PostJSON 发送 JSON 数据请求。
func PostJSON(url string, object interface{}) ([]byte, error) {
	body := new(bytes.Buffer)
	encoder := json.NewEncoder(body)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(object); err != nil {
		return nil, err
	}
	resp, err := http.Post(url, "application/json;charset=utf-8", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PostJSON 错误：网址=%v, 状态码：%v", url, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

// HTTPGet 网络拉取请求
func HTTPGet(uri string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("网络拉取错误：网址=%s, 状态码=%d", uri, resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}

// HTTPPost 网络投递请求
func HTTPPost(uri string, data string) ([]byte, error) {
	bytes.NewBuffer([]byte(data))
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("网络投递错误：网址=%s, 状态码=%d", uri, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return body, nil
}

// InMicroMessenger 判断是否在微信内部打开
func InMicroMessenger(userAgent string) bool {
	if len(userAgent) == 0 {
		return false
	}

	in := false
	userAgent = strings.ToLower(userAgent)
	vs := []string{"micromessenger"}

	for i := 0; i < len(vs); i++ {
		if strings.Contains(userAgent, vs[i]) {
			in = true
			break
		}
	}

	return in
}
