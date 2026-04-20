package rediskit

import rediskeyspace "github.com/FangcunMount/component-base/pkg/redis/keyspace"

// Namespace 是 Foundation 层命名空间值对象的兼容别名。
type Namespace = rediskeyspace.Namespace

// KeyPattern 是带命名空间模式值对象的兼容别名。
type KeyPattern = rediskeyspace.KeyPattern

// Keyspace 是 Foundation 层 Keyspace 的兼容别名。
type Keyspace = rediskeyspace.Keyspace

// NormalizeNamespace 去掉 Redis 命名空间两端的分隔符。
func NormalizeNamespace(ns string) string {
	return rediskeyspace.NormalizeNamespace(ns)
}

// NewKeyspace 创建带命名空间的 Keyspace 兼容入口。
func NewKeyspace(namespace string) Keyspace {
	return rediskeyspace.New(namespace)
}
