package main

import (
	"fmt"
	"github.com/gotid/wechat"
	"github.com/gotid/wechat/cache"
	"github.com/gotid/wechat/context"
	"github.com/gotid/wechat/msg"
	"net/http"
)

var ctx *context.Context

func init() {
	ctx = &context.Context{
		AppID:          "xxx",
		AppSecret:      "xxx",
		Token:          "xxx",
		EncodingAESKey: "xxx",
		Cache:          cache.NewMemory(),
	}
}

func main() {
	http.HandleFunc("/favicon.ico", favorite)
	http.HandleFunc("/", index)
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		fmt.Printf("启动服务器错误，错误=%v", err)
	}
}

func favorite(w http.ResponseWriter, r *http.Request) {
	fmt.Println(w, r)
}

func index(w http.ResponseWriter, r *http.Request) {
	wc := wechat.Get(ctx)
	server := wc.Server(w, r)

	// 设置常规消息钩子
	server.SetMsgHandler(func(ctx *context.Context, m msg.Msg) *msg.Response {
		return &msg.Response{
			Scene: msg.ResponseSceneKefu,
			Type:  msg.ResponseTypeXML,
			Msg:   "hello world",
		}
	})

	// 处理请求、构建响应
	err := server.Serve()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 发送响应
	server.Send()
}
