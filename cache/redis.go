package cache

import (
	"time"

	"github.com/gotid/god/lib/gconv"
	"github.com/gotid/god/lib/store/kv"
)

// Redis 提供一个基于 Redis 的缓存。
type Redis struct {
	store kv.Store
}

// NewRedis 返回一个新的 Redis 缓存。
func NewRedis(store kv.Store) *Redis {
	return &Redis{
		store: store,
	}
}

var _ Cache = (*Redis)(nil)

func (r *Redis) Get(key string) interface{} {
	v, err := r.store.Get(key)
	if err != nil {
		return nil
	}
	if v == "" {
		return nil
	}
	return v
}

func (r *Redis) Set(key string, val interface{}, timeout time.Duration) error {
	err := r.store.SetEx(key, gconv.String(val), int(timeout/time.Second))
	if err != nil {
		return err
	}
	return nil
}

func (r *Redis) Exists(key string) bool {
	exists, _ := r.store.Exists(key)
	return exists
}

func (r *Redis) Delete(key string) error {
	_, err := r.store.Del(key)
	if err != nil {
		return err
	}
	return nil
}
