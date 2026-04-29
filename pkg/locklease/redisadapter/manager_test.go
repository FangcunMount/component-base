package redisadapter

import (
	"context"
	"testing"
	"time"

	"github.com/FangcunMount/component-base/pkg/locklease"
	"github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
)

func TestManagerAcquireReleaseAndContention(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	manager := NewManager(client, func(key string) string { return "cache:lock:" + key })
	identity := locklease.Identity{Name: "leader", Key: "scheduler:leader"}

	lease1, acquired, err := manager.Acquire(context.Background(), identity, 30*time.Second)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	if !acquired || lease1 == nil {
		t.Fatalf("expected first acquire success, got acquired=%v lease=%+v", acquired, lease1)
	}
	if lease1.Key != "cache:lock:scheduler:leader" {
		t.Fatalf("lease key = %q, want concrete keyspace key", lease1.Key)
	}

	if _, acquired, err := manager.Acquire(context.Background(), identity, 30*time.Second); err != nil {
		t.Fatalf("second acquire failed: %v", err)
	} else if acquired {
		t.Fatal("expected lock contention on second acquire")
	}

	if err := manager.Release(context.Background(), identity, &locklease.Lease{
		Key:   lease1.Key,
		Token: "wrong-token",
	}); err != nil {
		t.Fatalf("release with wrong token returned error: %v", err)
	}
	if _, acquired, err := manager.Acquire(context.Background(), identity, 30*time.Second); err != nil {
		t.Fatalf("acquire after wrong release failed: %v", err)
	} else if acquired {
		t.Fatal("expected wrong-token release to keep lock held")
	}

	if err := manager.Release(context.Background(), identity, lease1); err != nil {
		t.Fatalf("release failed: %v", err)
	}
	if _, acquired, err := manager.Acquire(context.Background(), identity, 30*time.Second); err != nil {
		t.Fatalf("acquire after release failed: %v", err)
	} else if !acquired {
		t.Fatal("expected acquire success after release")
	}
}

func TestManagerAcquireSpecUsesSpecTTLAndExpires(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	spec := locklease.Spec{Name: "submit", DefaultTTL: 5 * time.Second}
	manager := NewManager(client, func(key string) string { return "locks:" + key })

	lease, acquired, err := manager.AcquireSpec(context.Background(), spec, "submit:req-1")
	if err != nil {
		t.Fatalf("AcquireSpec() error = %v", err)
	}
	if !acquired || lease == nil {
		t.Fatalf("AcquireSpec() got acquired=%v lease=%+v, want acquired lock", acquired, lease)
	}
	if ttl := mr.TTL(lease.Key); ttl != spec.DefaultTTL {
		t.Fatalf("ttl = %s, want %s", ttl, spec.DefaultTTL)
	}

	mr.FastForward(6 * time.Second)
	if _, acquired, err := manager.AcquireSpec(context.Background(), spec, "submit:req-1"); err != nil {
		t.Fatalf("AcquireSpec() after expiry error = %v", err)
	} else if !acquired {
		t.Fatal("expected acquire success after ttl expiry")
	}
}

func TestManagerRejectsInvalidSpecAndIdentity(t *testing.T) {
	manager := NewManager(nil, nil)

	if _, acquired, err := manager.Acquire(context.Background(), locklease.Identity{}, time.Second); err == nil {
		t.Fatal("expected empty identity name to be rejected")
	} else if acquired {
		t.Fatal("expected invalid identity to not acquire lock")
	}

	if _, acquired, err := manager.AcquireSpec(context.Background(), locklease.Spec{DefaultTTL: time.Second}, "key"); err == nil {
		t.Fatal("expected empty spec name to be rejected")
	} else if acquired {
		t.Fatal("expected invalid spec to not acquire lock")
	}

	if _, acquired, err := manager.AcquireSpec(context.Background(), locklease.Spec{Name: "invalid_ttl"}, "key"); err == nil {
		t.Fatal("expected empty ttl to be rejected")
	} else if acquired {
		t.Fatal("expected invalid ttl to not acquire lock")
	}
}
