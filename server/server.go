package server

import (
	"github.com/gotid/god/lib/logx"
	"github.com/gotid/wechat/context"
	"github.com/gotid/wechat/msg"
)

// Server 微信消息管理服务器，支持开放平台、支付、客服消息。
type Server struct {
	*context.Context

	debug        bool                                // 是否调试
	openID       string                              // 用户 openid
	msgHandler   func(message msg.Msg) *msg.Response // 消息钩子
	payHandler   func() *msg.Response                // 支付钩子
	requestRaw   []byte                              // 微信请求原始数据
	requestMsg   msg.Msg                             // 解析后微信请求数据
	responseType msg.ResponseType                    // 相应类型 string|xml|json
	responseMsg  interface{}                         // 响应数据
	isSafeMode   bool                                // 是否为加密模式
	random       []byte                              // 密文中的随机值
	nonce        string
	timestamp    int64
}

// NewServer 返回一个新的消息管理服务器。
func NewServer(ctx *context.Context) *Server {
	return &Server{
		Context: ctx,
	}
}

// Debug 指示是否打印调试日志
func (s *Server) Debug(v bool) {
	s.debug = v
}

// SetMsgHandler 设置常规消息钩子
func (s *Server) SetMsgHandler(h func(m msg.Msg) *msg.Response) {
	s.msgHandler = h
}

// Serve 处理微信请求并响应
func (s *Server) Serve() error {
	// 处理测试字符串
	echostr, exists := s.GetQuery("echostr")
	if exists {
		s.String(echostr)
		return nil
	}

	// 处理微信请求
	reply, err := s.handleRequest()
	if err != nil {
		return err
	}

	// 打印原始请求信息
	if s.debug {
		logx.Debug("微信原始请求信息：", string(s.requestRaw))
	}

	// 构建微信响应体
	return s.buildResponse(reply)
}

// Send 发送响应
func (s *Server) Send() {
	// 打印调试信息
	if s.debug {
		logx.Debugf("待发送给微信的响应消息 => %v \n", s)
	}

	// 跳过空白响应
	if s.responseMsg == nil {
		return
	}

	// 根据响应类型提供输出流
	switch s.responseType {
	case msg.ResponseTypeJSON:
	case msg.ResponseTypeXML:
		s.XML(s.responseMsg)
	case msg.ResponseTypeString:
		if v, ok := s.responseMsg.(string); ok {
			s.String(v)
		}
	}
}

// OpenID 获取请求者 openID
func (s *Server) OpenID() string {
	return s.openID
}
