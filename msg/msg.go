package msg

import "encoding/xml"

type (
	Type      string // 基本消息类型
	EventType string // 事件消息类型
	InfoType  string // 第三方平台授权事件类型
)

const (
	TypeText       Type = "text"                      // 文本消息
	TypeImage      Type = "image"                     // 图片消息
	TypeVoice      Type = "voice"                     // 音频消息
	TypeVideo      Type = "video"                     // 视频消息
	TypeMusic      Type = "music"                     // 音乐消息
	TypeNews       Type = "news"                      // 图文消息
	TypeTransferKf      = "transfer_customer_service" // 转发客服消息
)

const (
	InfoTypeVerifyTicket     InfoType = "component_verify_ticket" // 平台票据推送
	InfoTypeAuthorized       InfoType = "authorized"              // 授权
	InfoTypeUnauthorized     InfoType = "unauthorized"            // 取消授权
	InfoTypeUpdateAuthorized InfoType = "updateauthorized"        // 更新授权
)

type Msg struct {
	Base

	// === 第三方平台相关 ===
	InfoType                     InfoType `xml:"InfoType"`                     // 平台事件类型
	AppID                        string   `xml:"AppId"`                        // 平台 AppID
	ComponentVerifyTicket        string   `xml:"ComponentVerifyTicket"`        // 微信推送的平台票据
	PreAuthCode                  string   `xml:"PreAuthCode"`                  // 预授权码
	AuthorizerAppid              string   `xml:"AuthorizerAppid"`              // 授权者 AppID
	AuthorizationCode            string   `xml:"AuthorizationCode"`            // 授权码
	AuthorizationCodeExpiredTime int64    `xml:"AuthorizationCodeExpiredTime"` // 授权码过期时间
	Reason                       string   `xml:"Reason"`
	ScreenShot                   string   `xml:"ScreenShot"`
}

// EncryptedMsg 安全模式下的消息体。
type EncryptedMsg struct {
	XMLName    struct{} `xml:"xml" json:"-"`
	ToUserName string   `xml:"ToUserName" json:"ToUserName"`
	Encrypt    string   `xml:"Encrypt" json:"Encrypt"`
}

// EncryptedResponseMsg 安全模式下的加密响应消息体。
type EncryptedResponseMsg struct {
	XMLName      struct{} `xml:"xml" json:"-"`
	EncryptedMsg string   `xml:"Encrypt"      json:"Encrypt"`
	MsgSignature string   `xml:"MsgSignature" json:"MsgSignature"`
	Timestamp    int64    `xml:"TimeStamp"    json:"TimeStamp"`
	Nonce        string   `xml:"Nonce"        json:"Nonce"`
}

// Base 消息中通用的基础结构。
type Base struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDATA    `xml:"ToUserName"`
	FromUserName CDATA    `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      Type     `xml:"MsgType"`
}

// SetToUserName set ToUserName
func (msg *Base) SetToUserName(toUserName CDATA) {
	msg.ToUserName = toUserName
}

// SetFromUserName set FromUserName
func (msg *Base) SetFromUserName(fromUserName CDATA) {
	msg.FromUserName = fromUserName
}

// SetCreateTime set createTime
func (msg *Base) SetCreateTime(createTime int64) {
	msg.CreateTime = createTime
}

// SetMsgType set MsgType
func (msg *Base) SetMsgType(msgType Type) {
	msg.MsgType = msgType
}

// CDATA 使用该类型，在序列化为 xml 时文本会被忽略。
type CDATA string

func (c CDATA) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(
		struct {
			string `xml:",cdata"`
		}{
			string: string(c),
		},
		start,
	)
}
