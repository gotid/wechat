package config

import (
	"github.com/gotid/god/api"
	"github.com/gotid/god/lib/store/cache"
)

type Config struct {
	api.ServerConf

	MySQL string
	Cache cache.ClusterConf
}
