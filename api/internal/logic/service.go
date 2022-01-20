package logic

import (
	"github.com/gotid/god/lib/logx"
	"github.com/gotid/wechat"
	"github.com/gotid/wechat/api/internal/model"
	"github.com/gotid/wechat/api/internal/svc"
	"github.com/gotid/wechat/cache"
	"github.com/gotid/wechat/context"
	"github.com/gotid/wechat/msg"
)

// GetWeChat 获取指定平台的微信控制器。
func GetWeChat(svcCtx *svc.ServiceContext, platformID string) (*wechat.WeChat, *model.Platform, error) {
	// 获取平台信息
	platform, err := svcCtx.PlatformModel.FindOneByAppId(platformID)
	if err != nil {
		return nil, nil, err
	}

	// 组装此次微信请求上下文
	ctx := &context.Context{
		AppID:          platform.AppId,
		AppSecret:      platform.AppSecret,
		Token:          platform.Token,
		EncodingAESKey: platform.EncodingAesKey,

		Cache: cache.NewRedis(svcCtx.Cache),
	}

	// 获取平台微信控制器
	wc := wechat.Get(ctx)
	return wc, platform, nil
}

type msgHandler msg.Msg

// MsgHandler 定义业务方默认消息钩子
func MsgHandler(ctx *context.Context, m msg.Msg) (resp *msg.Response) {
	resp = &msg.Response{
		Scene: msg.ResponseSceneOpen,
		Type:  msg.ResponseTypeString,
	}

	wc := wechat.Get(ctx)

	handle := msgHandler(m)
	switch m.InfoType {
	case msg.InfoTypeVerifyTicket:
		handle.CheckTicket(wc, resp)
	}

	return
}

// CheckTicket 检查验证票据是否已保存成功
func (h *msgHandler) CheckTicket(wc *wechat.WeChat, _ *msg.Response) {
	go func(id string) {
		_, err := wc.OpenPlatform().ComponentVerifyTicket()
		if err != nil {
			logx.Errorf("应取到验证票据，但是出错：%v", err)
			return
		}
	}(h.AppID)
}
