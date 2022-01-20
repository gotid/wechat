package logic

import (
	"context"
	"github.com/gotid/wechat/api/internal/logic"
	"net/http"

	"github.com/gotid/wechat/api/internal/svc"
	"github.com/gotid/wechat/api/internal/types"

	"github.com/gotid/god/lib/logx"
)

type NotifyLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	writer  http.ResponseWriter
	request *http.Request
}

func NewNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext,
	w http.ResponseWriter, r *http.Request) NotifyLogic {
	return NotifyLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		writer:  w,
		request: r,
	}
}

func (l *NotifyLogic) Notify(req types.PlatformReq) error {
	// 获取微信控制器
	wc, _, err := logic.GetWeChat(l.svcCtx, req.PlatformID)
	if err != nil {
		return err
	}

	// 获取当前微信请求的消息管理器
	server := wc.Server(l.writer, l.request)
	server.Debug(true)

	// 设置常规消息钩子
	server.SetMsgHandler(logic.MsgHandler)

	// 处理请求、构建响应
	err = server.Serve()
	if err != nil {
		return err
	}

	// 发送响应
	server.Send()

	return nil
}
