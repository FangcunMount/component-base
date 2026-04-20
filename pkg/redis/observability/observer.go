package observability

import (
	"context"
	"time"
)

// CommandEvent 表示一次底层 Redis 命令观测事件。
type CommandEvent struct {
	Operation string
	Profile   string
	Key       string
	Duration  time.Duration
	Err       error
}

// CommandObserver 接收底层 Redis 命令观测事件。
type CommandObserver interface {
	OnCommand(ctx context.Context, event CommandEvent)
}

// ProfileEvent 表示 profile 健康状态与可用性切换事件。
type ProfileEvent struct {
	Profile     string
	State       string
	Err         error
	NextRetryAt time.Time
}

// ProfileObserver 接收运行时 profile 状态变化事件。
type ProfileObserver interface {
	OnProfileStatus(ctx context.Context, event ProfileEvent)
}

// LeaseEvent 表示租约锁相关操作事件。
type LeaseEvent struct {
	Operation string
	Key       string
	Token     string
	Owner     string
	TTL       time.Duration
	Acquired  bool
	Owned     bool
	Err       error
}

// LeaseObserver 接收租约锁观测事件。
type LeaseObserver interface {
	OnLease(ctx context.Context, event LeaseEvent)
}

// StoreEvent 表示 typed store 操作事件。
type StoreEvent struct {
	Operation string
	Key       string
	Codec     string
	Hit       bool
	TTL       time.Duration
	Size      int
	Err       error
}

// StoreObserver 接收 typed value store 观测事件。
type StoreObserver interface {
	OnStore(ctx context.Context, event StoreEvent)
}

// NopCommandObserver 是空实现命令观测器。
type NopCommandObserver struct{}

func (NopCommandObserver) OnCommand(context.Context, CommandEvent) {}

// NopProfileObserver 是空实现 profile 观测器。
type NopProfileObserver struct{}

func (NopProfileObserver) OnProfileStatus(context.Context, ProfileEvent) {}

// NopLeaseObserver 是空实现租约锁观测器。
type NopLeaseObserver struct{}

func (NopLeaseObserver) OnLease(context.Context, LeaseEvent) {}

// NopStoreObserver 是空实现 store 观测器。
type NopStoreObserver struct{}

func (NopStoreObserver) OnStore(context.Context, StoreEvent) {}
