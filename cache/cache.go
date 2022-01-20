// Package cache 提供微信票据/令牌等信息的快速存取。
package cache

import (
	"fmt"
	"time"
)

const (
	// 开放平台票据
	keyComponentVerifyTicket = "component_verify_ticket_%s"
	// 开放平台令牌
	keyComponentAccessToken = "component_access_token_%s"
)

type Cache interface {
	// Get 获取指定键对应的值。
	Get(key string) interface{}
	// Set 设置键值对缓存。
	Set(key string, val interface{}, timeout time.Duration) error
	// Exists 判断指定的键值是否存在。
	Exists(key string) bool
	// Delete 删除指定的键值。
	Delete(key string) error
}

// KeyComponentVerifyTicket 获取开放平台票据缓存键
func KeyComponentVerifyTicket(appID string) string {
	return fmt.Sprintf(keyComponentVerifyTicket, appID)
}

// KeyComponentAccessToken 获取开放平台令牌缓存键
func KeyComponentAccessToken(appID string) string {
	return fmt.Sprintf(keyComponentAccessToken, appID)
}
