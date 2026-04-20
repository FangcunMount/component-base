package keyspace

import "strings"

// Namespace 命名空间，用于将 Redis 键分组的一种机制。
type Namespace string

// 规范化命名空间，去除开头和结尾的冒号。
func NormalizeNamespace(ns string) string {
	return strings.Trim(ns, ":")
}

// 创建规范化命名空间。
func NewNamespace(ns string) Namespace {
	return Namespace(NormalizeNamespace(ns))
}

// 返回规范化命名空间。
func (n Namespace) String() string {
	return string(n)
}

// KeyPattern 键模式，用于 SCAN 和 purge 操作。
type KeyPattern string

// 创建规范化键模式。
func NewKeyPattern(pattern string) KeyPattern {
	return KeyPattern(pattern)
}

// 返回原始键模式字符串。
func (p KeyPattern) String() string {
	return string(p)
}

// Keyspace 键空间，用于将 Redis 键分组的一种机制。
type Keyspace struct {
	namespace Namespace
}

// 创建键空间。
func New(namespace string) Keyspace {
	return Keyspace{namespace: NewNamespace(namespace)}
}

// NewFromNamespace 从命名空间值对象创建 Keyspace。
func NewFromNamespace(namespace Namespace) Keyspace {
	return Keyspace{namespace: NewNamespace(namespace.String())}
}

// Namespace 返回当前 Keyspace 的命名空间。
func (k Keyspace) Namespace() Namespace {
	return k.namespace
}

// Child 在当前命名空间下继续派生子命名空间。
func (k Keyspace) Child(child string) Keyspace {
	childNamespace := NewNamespace(child)
	switch {
	case k.namespace.String() == "":
		return NewFromNamespace(childNamespace)
	case childNamespace.String() == "":
		return NewFromNamespace(k.namespace)
	default:
		return New(k.namespace.String() + ":" + childNamespace.String())
	}
}

// Prefix 为键附加命名空间前缀。
func (k Keyspace) Prefix(key string) string {
	if k.namespace.String() == "" {
		return key
	}
	if key == "" {
		return k.namespace.String()
	}
	return k.namespace.String() + ":" + key
}

// Pattern 返回带命名空间前缀的模式值对象。
func (k Keyspace) Pattern(pattern string) KeyPattern {
	return NewKeyPattern(k.Prefix(pattern))
}
