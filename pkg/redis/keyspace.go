package rediskit

import "strings"

// NormalizeNamespace trims trailing/leading separators from a Redis namespace.
func NormalizeNamespace(ns string) string {
	return strings.Trim(ns, ":")
}

// Keyspace prefixes raw Redis keys with an optional namespace.
type Keyspace struct {
	Namespace string
}

// NewKeyspace creates a namespaced keyspace helper.
func NewKeyspace(namespace string) Keyspace {
	return Keyspace{Namespace: NormalizeNamespace(namespace)}
}

// Child returns a nested keyspace under the current namespace.
func (k Keyspace) Child(child string) Keyspace {
	child = NormalizeNamespace(child)
	switch {
	case k.Namespace == "":
		return NewKeyspace(child)
	case child == "":
		return NewKeyspace(k.Namespace)
	default:
		return NewKeyspace(k.Namespace + ":" + child)
	}
}

// Prefix returns the key with namespace applied when configured.
func (k Keyspace) Prefix(key string) string {
	if k.Namespace == "" {
		return key
	}
	return k.Namespace + ":" + key
}
