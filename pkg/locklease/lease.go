package locklease

import (
	"context"
	"time"
)

// Spec 描述一个具有语义的租约锁工作负载。
type Spec struct {
	Name        string
	Description string
	DefaultTTL  time.Duration
}

// Identity 根据业务 key 构造一个具体锁身份。
func (s Spec) Identity(key string) Identity {
	return Identity{
		Name: s.Name,
		Key:  key,
	}
}

// Identity 描述一个具体锁实例。
type Identity struct {
	Name string
	Key  string
}

// Lease 表示一个已成功获取的锁租约。
type Lease struct {
	Key   string
	Token string
}

// Manager 是应用代码依赖的、与 Redis 无关的租约锁端口。
type Manager interface {
	AcquireSpec(ctx context.Context, spec Spec, key string, ttlOverride ...time.Duration) (*Lease, bool, error)
	ReleaseSpec(ctx context.Context, spec Spec, key string, lease *Lease) error
}
