package msg

import "errors"

var ErrUnsupportedResponse = errors.New("不支持的消息类型")

// ResponseType 响应类型
type ResponseType string

const (
	ResponseTypeString ResponseType = "string"
	ResponseTypeXML    ResponseType = "xml"
	ResponseTypeJSON   ResponseType = "json"
)

// ResponseScene 响应场景
type ResponseScene string

const (
	ResponseSceneOpen ResponseScene = "open" // 开放平台
	ResponseSceneKefu ResponseScene = "kefu" // 客服场景
	ResponseScenePay  ResponseScene = "pay"  // 支付场景
)

// Response 常规消息响应体
type Response struct {
	Scene ResponseScene // 响应场景
	Type  ResponseType  // 响应类型
	Msg   interface{}   // 响应消息
}

// PayNotifyResponse 支付通知响应体
type PayNotifyResponse struct {
	ReturnCode string `xml:"return_code"`
	ReturnMsg  string `xml:"return_msg"`
}
