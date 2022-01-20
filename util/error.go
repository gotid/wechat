package util

import (
	"encoding/json"
	"fmt"
)

// WechatError 是微信接口通用错误结构体。
type WechatError struct {
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg,omitempty"`
}

// UnknownError 返回一个未知错误信息。
func UnknownError(err error) *WechatError {
	if err != nil {
		return &WechatError{
			ErrCode: -99,
			ErrMsg:  err.Error(),
		}
	}
	return nil
}

// TryDecodeError 尝试解码响应错误。
func TryDecodeError(data []byte, apiName string) (err error) {
	var commonErr WechatError
	err = json.Unmarshal(data, &commonErr)
	if err != nil {
		return err
	}

	if commonErr.ErrCode != 0 {
		return fmt.Errorf("微信接口 %s 错误：errcode=%d, errmsg=%s",
			apiName, commonErr.ErrCode, commonErr.ErrMsg)
	}

	return nil
}

func (e *WechatError) Success() bool {
	if e.ErrCode == 0 {
		return true
	}
	return false
}
