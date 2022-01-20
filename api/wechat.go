package main

import (
	"flag"
	"fmt"

	"github.com/gotid/wechat/api/internal/config"
	"github.com/gotid/wechat/api/internal/handler"
	"github.com/gotid/wechat/api/internal/svc"

	"github.com/gotid/god/api"
	"github.com/gotid/god/lib/conf"
)

var configFile = flag.String("f", "etc/wechat-api.yaml", "配置文件")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)
	server := api.MustNewServer(c.ServerConf)
	defer server.Stop()

	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
