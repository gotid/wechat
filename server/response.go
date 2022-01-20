package server

import (
	"encoding/xml"

	"reflect"
	"strconv"
	"time"

	"github.com/gotid/god/lib/logx"
	"github.com/gotid/wechat/msg"
	"github.com/gotid/wechat/util"
)

// 根据回复场景构建响应体
func (s *Server) buildResponse(resp *msg.Response) (err error) {
	// 跳过空白响应
	if resp == nil {
		return nil
	}

	switch resp.Scene {
	case msg.ResponseSceneOpen:
		err = s.buildOpenResponse(resp)
	case msg.ResponseScenePay:
		err = s.buildPayResponse(resp)
	case msg.ResponseSceneKefu:
		err = s.buildKefuResponse(resp)
	}

	return err
}

// 构建开放平台场景的响应类型和消息
func (s *Server) buildOpenResponse(resp *msg.Response) error {
	if s.debug {
		logx.Debugf("开放平台回复体 => %#v \n", resp)
	}

	// 在发送回复前，记录微信推送的平台验证票据
	/// 微信每10分钟推送1次
	if s.requestMsg.InfoType == msg.InfoTypeVerifyTicket {
		s.SetComponentVerifyTicket(s.requestMsg.ComponentVerifyTicket)
	}

	// 设置回复类型和消息
	if resp.Type == "" {
		resp.Type = msg.ResponseTypeString
	}
	if resp.Msg == nil {
		resp.Msg = "success"
	}
	s.responseType = resp.Type
	s.responseMsg = resp.Msg

	return nil
}

// 构建支付场景的响应类型和消息
func (s *Server) buildPayResponse(resp *msg.Response) error {
	// 设置默认回复消息
	if resp.Msg == nil {
		s.responseMsg = msg.PayNotifyResponse{
			ReturnCode: "SUCCESS",
			ReturnMsg:  "OK",
		}
	}

	s.responseType = resp.Type
	s.responseMsg = resp.Msg

	return nil
}

// 构建客服场景的响应类型和消息
func (s *Server) buildKefuResponse(resp *msg.Response) error {
	// 判断响应消息值是否为指针类型
	respMsg := resp.Msg
	value := reflect.ValueOf(respMsg)
	if value.Kind().String() != "ptr" {
		return msg.ErrUnsupportedResponse
	}

	// 设置默认响应类型
	if resp.Type == "" {
		resp.Type = msg.ResponseTypeXML
	}

	// 设置基础回复信息(反射方法调用入参切片)
	in := make([]reflect.Value, 1)
	in[0] = reflect.ValueOf(s.requestMsg.FromUserName)
	value.MethodByName("SetToUserName").Call(in)

	in[0] = reflect.ValueOf(s.requestMsg.ToUserName)
	value.MethodByName("SetFromUserName").Call(in)

	in[0] = reflect.ValueOf(time.Now().Unix())
	value.MethodByName("SetCreateTime").Call(in)

	s.responseMsg = respMsg

	// 安全模式加密响应消息
	if s.isSafeMode {
		// XML转字节切片
		bs, err := xml.Marshal(s.responseMsg)
		if err != nil {
			return err
		}

		// 加密消息
		encryptedMsg, err := util.EncryptMsg(s.random, bs, s.AppID, s.EncodingAESKey)
		if err != nil {
			return err
		}

		// 对消息签名
		timestamp := strconv.FormatInt(s.timestamp, 10)
		signed := util.Signature(s.Token, timestamp, s.nonce, string(encryptedMsg))
		s.responseMsg = msg.EncryptedResponseMsg{
			XMLName:      struct{}{},
			EncryptedMsg: string(encryptedMsg),
			MsgSignature: signed,
			Timestamp:    s.timestamp,
			Nonce:        s.nonce,
		}
	}

	return nil
}
