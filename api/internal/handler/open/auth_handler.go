package handler

import (
	"fmt"
	"net/http"

	"github.com/gotid/wechat/api/internal/logic/open"
	"github.com/gotid/wechat/api/internal/svc"
	"github.com/gotid/wechat/api/internal/types"

	"github.com/gotid/god/api/httpx"
)

func AuthHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PlatformReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := logic.NewAuthLogic(r.Context(), ctx, w, r)
		authURL, err := l.Auth(req)
		if err != nil {
			httpx.Error(w, err)
		} else {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(fmt.Sprintf("<script>location.href=\"%s\"</script>", authURL)))
		}
	}
}
