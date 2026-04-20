package lease

import (
	"fmt"
	"time"
)

// LeaseKey 是不可变的锁键值对象。
type LeaseKey string

// NewLeaseKey 创建经过校验的租约锁键。
func NewLeaseKey(key string) (LeaseKey, error) {
	if key == "" {
		return "", fmt.Errorf("lock key is empty")
	}
	return LeaseKey(key), nil
}

// MustLeaseKey 创建租约锁键，输入非法时 panic。
func MustLeaseKey(key string) LeaseKey {
	leaseKey, err := NewLeaseKey(key)
	if err != nil {
		panic(err)
	}
	return leaseKey
}

// String 返回原始锁键字符串。
func (k LeaseKey) String() string {
	return string(k)
}

// LeaseToken 是不可变的锁 token 值对象。
type LeaseToken string

// NewLeaseToken 创建经过校验的租约 token。
func NewLeaseToken(token string) (LeaseToken, error) {
	if token == "" {
		return "", fmt.Errorf("lease token is empty")
	}
	return LeaseToken(token), nil
}

// MustLeaseToken 创建租约 token，输入非法时 panic。
func MustLeaseToken(token string) LeaseToken {
	leaseToken, err := NewLeaseToken(token)
	if err != nil {
		panic(err)
	}
	return leaseToken
}

// String 返回原始 token 字符串。
func (t LeaseToken) String() string {
	return string(t)
}

// LeaseOwner 是锁观测与调试使用的可选拥有者元数据。
type LeaseOwner struct {
	ID    string
	Label string
}

// Identity 返回最适合作为拥有者标识的字符串。
func (o *LeaseOwner) Identity() string {
	if o == nil {
		return ""
	}
	if o.ID != "" {
		return o.ID
	}
	return o.Label
}

// Lease 表示一个已获取的锁租约。
type Lease struct {
	Key        LeaseKey
	Token      LeaseToken
	TTL        time.Duration
	Owner      *LeaseOwner
	AcquiredAt time.Time
}

// LeaseAttempt 描述一次 acquire 是否成功。
type LeaseAttempt struct {
	Lease    Lease
	Acquired bool
}
