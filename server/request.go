package server

import (
	"encoding/xml"
	"fmt"
	"github.com/gotid/wechat/msg"
	"github.com/gotid/wechat/util"
	"io/ioutil"
	"strconv"
)

func (s *Server) handleRequest() (reply *msg.Response, err error) {
	s.requestRaw, err = ioutil.ReadAll(s.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("读取微信请求体失败，错误：%v", err)
	}

	req := requestModel{}
	err = xml.Unmarshal(s.requestRaw, &req)
	if err != nil {
		err = fmt.Errorf("解析微信XML请求体失败：data=%s, err=%v", s.requestRaw, err)
		return
	}

	if req.IsPay() {
		reply, err = s.handlePay()
	} else {
		reply, err = s.handleMsg()
	}

	return
}

// 处理并回复常规请求消息
func (s *Server) handleMsg() (reply *msg.Response, err error) {
	// 填充openid和安全模式
	s.openID = s.Query("openid")
	s.isSafeMode = s.Query("encrypt_type") == "aes"

	// 校验消息签名
	sign := util.Signature(s.Token, s.Query("timestamp"), s.Query("nonce"))
	if !s.debug && s.Query("signature") == sign {
		err = fmt.Errorf("请求校验失败")
		return
	}

	// 解密
	if s.isSafeMode {
		// 二进制转XML
		var encryptedMessage msg.EncryptedMsg
		err = xml.Unmarshal(s.requestRaw, &encryptedMessage)
		if err != nil {
			err = fmt.Errorf("解析微信加密请求体失败，错误=%v", err)
			return
		}

		// 验证消息签名
		timestamp := s.Query("timestamp")
		s.timestamp, err = strconv.ParseInt(timestamp, 10, 32)
		if err != nil {
			return
		}
		nonce := s.Query("nonce")
		signed := util.Signature(s.Token, timestamp, nonce, encryptedMessage.Encrypt)
		if signed != s.Query("msg_signature") {
			err = fmt.Errorf("微信加密体签名不匹配")
			return
		}

		// 解密
		s.random, s.requestRaw, err = util.DecryptMsg(s.AppID, encryptedMessage.Encrypt, s.EncodingAESKey)
		if err != nil {
			err = fmt.Errorf("微信加密体解密失败，错误=%v", err)
			return
		}
	}

	// 调用自定义消息钩子生成回复内容
	err = xml.Unmarshal(s.requestRaw, &s.requestMsg)
	reply = s.msgHandler(s.Context, s.requestMsg)
	return
}

// 处理并回复支付请求消息
func (s *Server) handlePay() (reply *msg.Response, err error) {
	return nil, fmt.Errorf("暂未实现支付消息处理逻辑")
}

// 微信请求特征模型
type requestModel struct {
	ReturnCode string `xml:"return_code"`
	ReturnMsg  string `xml:"return_msg"`
	AppID      string `xml:"appid"`
	MchID      string `xml:"mch_id"`
}

// IsPay 是否为微信支付请求
func (m *requestModel) IsPay() bool {
	return m.ReturnCode == "" && m.MchID != ""
}

// IsMsg 是否为常规消息体
func (m *requestModel) IsMsg() bool {
	return !m.IsPay()
}
