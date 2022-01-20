package wechat

import (
	"github.com/gotid/wechat/context"
	"github.com/gotid/wechat/open"
	"github.com/gotid/wechat/server"
	"net/http"
)

// WeChat 微信接口控制器
type WeChat struct {
	Context *context.Context
}

// New 返回微信控制器
func New(context *context.Context) *WeChat {
	return &WeChat{Context: context}
}

// NewServer 返回消息管理服务器
func (wc *WeChat) NewServer(w http.ResponseWriter, r *http.Request) *server.Server {
	wc.Context.Writer = w
	wc.Context.Request = r
	return server.NewServer(wc.Context)
}

// Open 返回开放平台控制器
func (wc *WeChat) Open() *open.Open {
	return open.New(wc.Context)
}
