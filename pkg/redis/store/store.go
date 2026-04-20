package store

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/FangcunMount/component-base/pkg/redis/observability"
)

// StoreKey 是不可变的 typed store 键值对象。
type StoreKey string

// NewStoreKey 创建经过校验的 store 键。
func NewStoreKey(key string) (StoreKey, error) {
	if key == "" {
		return "", fmt.Errorf("store key is empty")
	}
	return StoreKey(key), nil
}

// MustStoreKey 创建 store 键，输入非法时 panic。
func MustStoreKey(key string) StoreKey {
	storeKey, err := NewStoreKey(key)
	if err != nil {
		panic(err)
	}
	return storeKey
}

// String 返回原始 Redis 键字符串。
func (k StoreKey) String() string {
	return string(k)
}

// ValueStore 是最小可复用的 typed Redis value store。
type ValueStore[T any] struct {
	client   goredis.UniversalClient
	codec    ValueCodec[T]
	observer observability.StoreObserver
}

type Option[T any] func(*ValueStore[T])

// WithObserver 配置 typed store 观测器。
func WithObserver[T any](observer observability.StoreObserver) Option[T] {
	return func(store *ValueStore[T]) {
		if observer != nil {
			store.observer = observer
		}
	}
}

// NewValueStore 创建 typed Redis value store。
func NewValueStore[T any](client goredis.UniversalClient, codec ValueCodec[T], opts ...Option[T]) *ValueStore[T] {
	store := &ValueStore[T]{
		client:   client,
		codec:    codec,
		observer: observability.NopStoreObserver{},
	}
	for _, opt := range opts {
		opt(store)
	}
	return store
}

// Get 读取并解码一个 typed value。
func (s *ValueStore[T]) Get(ctx context.Context, key StoreKey) (T, bool, error) {
	var zero T
	if s == nil || s.client == nil {
		return zero, false, fmt.Errorf("redis client is nil")
	}
	if s.codec == nil {
		return zero, false, fmt.Errorf("value codec is nil")
	}
	if key.String() == "" {
		return zero, false, fmt.Errorf("store key is empty")
	}

	payload, err := s.client.Get(ctx, key.String()).Bytes()
	if err == goredis.Nil {
		s.observer.OnStore(ctx, observability.StoreEvent{
			Operation: "get",
			Key:       key.String(),
			Codec:     s.codec.Name(),
			Hit:       false,
		})
		return zero, false, nil
	}
	if err != nil {
		s.observer.OnStore(ctx, observability.StoreEvent{
			Operation: "get",
			Key:       key.String(),
			Codec:     s.codec.Name(),
			Err:       err,
		})
		return zero, false, err
	}

	value, err := s.codec.Unmarshal(payload)
	s.observer.OnStore(ctx, observability.StoreEvent{
		Operation: "get",
		Key:       key.String(),
		Codec:     s.codec.Name(),
		Hit:       err == nil,
		Size:      len(payload),
		Err:       err,
	})
	if err != nil {
		return zero, false, err
	}
	return value, true, nil
}

// Set 编码并写入一个 typed value。
func (s *ValueStore[T]) Set(ctx context.Context, key StoreKey, value T, ttl time.Duration) error {
	return s.set(ctx, key, value, ttl, false)
}

// SetIfAbsent 仅在键不存在时写入 typed value。
func (s *ValueStore[T]) SetIfAbsent(ctx context.Context, key StoreKey, value T, ttl time.Duration) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	if s.codec == nil {
		return false, fmt.Errorf("value codec is nil")
	}
	if key.String() == "" {
		return false, fmt.Errorf("store key is empty")
	}

	payload, err := s.codec.Marshal(value)
	if err != nil {
		s.observer.OnStore(ctx, observability.StoreEvent{
			Operation: "set_if_absent",
			Key:       key.String(),
			Codec:     s.codec.Name(),
			Err:       err,
		})
		return false, err
	}

	ok, err := s.client.SetNX(ctx, key.String(), payload, ttl).Result()
	s.observer.OnStore(ctx, observability.StoreEvent{
		Operation: "set_if_absent",
		Key:       key.String(),
		Codec:     s.codec.Name(),
		Hit:       ok,
		TTL:       ttl,
		Size:      len(payload),
		Err:       err,
	})
	return ok, err
}

// Delete 从 Redis 中删除一个键。
func (s *ValueStore[T]) Delete(ctx context.Context, key StoreKey) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if key.String() == "" {
		return fmt.Errorf("store key is empty")
	}
	_, err := s.client.Del(ctx, key.String()).Result()
	s.observer.OnStore(ctx, observability.StoreEvent{
		Operation: "delete",
		Key:       key.String(),
		Codec:     codecName(s.codec),
		Err:       err,
	})
	return err
}

// Exists 返回键是否存在。
func (s *ValueStore[T]) Exists(ctx context.Context, key StoreKey) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	if key.String() == "" {
		return false, fmt.Errorf("store key is empty")
	}
	count, err := s.client.Exists(ctx, key.String()).Result()
	exists := count > 0
	s.observer.OnStore(ctx, observability.StoreEvent{
		Operation: "exists",
		Key:       key.String(),
		Codec:     codecName(s.codec),
		Hit:       exists,
		Err:       err,
	})
	return exists, err
}

func (s *ValueStore[T]) set(ctx context.Context, key StoreKey, value T, ttl time.Duration, onlyIfAbsent bool) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if s.codec == nil {
		return fmt.Errorf("value codec is nil")
	}
	if key.String() == "" {
		return fmt.Errorf("store key is empty")
	}

	payload, err := s.codec.Marshal(value)
	if err != nil {
		s.observer.OnStore(ctx, observability.StoreEvent{
			Operation: "set",
			Key:       key.String(),
			Codec:     s.codec.Name(),
			Err:       err,
		})
		return err
	}
	err = s.client.Set(ctx, key.String(), payload, ttl).Err()
	s.observer.OnStore(ctx, observability.StoreEvent{
		Operation: "set",
		Key:       key.String(),
		Codec:     s.codec.Name(),
		TTL:       ttl,
		Size:      len(payload),
		Hit:       !onlyIfAbsent,
		Err:       err,
	})
	return err
}

func codecName[T any](codec ValueCodec[T]) string {
	if codec == nil {
		return ""
	}
	return codec.Name()
}
