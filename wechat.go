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

// Get 返回可复用的微信控制器
func Get(ctx *context.Context) *WeChat {
	return &WeChat{Context: ctx}
}

// Server 返回消息管理服务器
func (wc *WeChat) Server(w http.ResponseWriter, r *http.Request) *server.Server {
	wc.Context.Writer = w
	wc.Context.Request = r
	return server.NewServer(wc.Context)
}

// OpenPlatform 返回开放平台控制器
func (wc *WeChat) OpenPlatform() *open.Open {
	return open.NewPlatform(wc.Context)
}
