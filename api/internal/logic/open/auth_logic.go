package logic

import (
	"context"
	"fmt"
	"github.com/gotid/wechat/api/internal/logic"
	"github.com/gotid/wechat/util"
	"net/http"

	"github.com/gotid/wechat/api/internal/svc"
	"github.com/gotid/wechat/api/internal/types"

	"github.com/gotid/god/lib/logx"
)

type AuthLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	writer  http.ResponseWriter
	request *http.Request
}

func NewAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext,
	w http.ResponseWriter, r *http.Request) AuthLogic {
	return AuthLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		writer:  w,
		request: r,
	}
}

func (l *AuthLogic) Auth(req types.PlatformReq) (string, error) {
	// 获取微信控制器
	wc, platform, err := logic.GetWeChat(l.svcCtx, req.PlatformID)
	if err != nil {
		return "", err
	}

	apiHost := platform.ApiHost
	redirect := fmt.Sprintf("%s/api/wechat/open/%s/redirect", apiHost, platform.AppId)
	inMicro := util.InMicroMessenger(l.request.UserAgent())

	uri, err := wc.OpenPlatform().AuthURL(inMicro, redirect)
	if err != nil {
		return "", err
	}

	return uri, nil
}
