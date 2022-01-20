package handler

import (
	"github.com/gotid/god/lib/logx"
	"net/http"

	"github.com/gotid/wechat/api/internal/logic/open"
	"github.com/gotid/wechat/api/internal/svc"
	"github.com/gotid/wechat/api/internal/types"

	"github.com/gotid/god/api/httpx"
)

func NotifyHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PlatformReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := logic.NewNotifyLogic(r.Context(), ctx, w, r)
		err := l.Notify(req)
		if err != nil {
			logx.Errorf("第三方平台授权事件通知处理失败：%v", err)
			return
		}
	}
}
