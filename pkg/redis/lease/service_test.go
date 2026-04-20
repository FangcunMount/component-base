package lease

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestServiceAcquireRenewCheckOwnershipAndRelease(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	service := NewService(client)
	ctx := context.Background()

	attempt, err := service.Acquire(ctx, MustLeaseKey("lock:test"), 5*time.Second, &LeaseOwner{ID: "owner-1"})
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if !attempt.Acquired {
		t.Fatalf("Acquire() should succeed")
	}

	owned, err := service.CheckOwnership(ctx, attempt.Lease)
	if err != nil {
		t.Fatalf("CheckOwnership() error = %v", err)
	}
	if !owned {
		t.Fatalf("CheckOwnership() should report ownership")
	}

	renewed, ok, err := service.Renew(ctx, attempt.Lease, 10*time.Second)
	if err != nil {
		t.Fatalf("Renew() error = %v", err)
	}
	if !ok || renewed.TTL != 10*time.Second {
		t.Fatalf("Renew() = (%v, %v), want renewed lease with updated ttl", renewed, ok)
	}

	if err := service.Release(ctx, renewed); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	owned, err = service.CheckOwnership(ctx, renewed)
	if err != nil {
		t.Fatalf("CheckOwnership() after release error = %v", err)
	}
	if owned {
		t.Fatalf("released lease should no longer own the key")
	}
}

func TestServiceRenewWithWrongTokenFailsGracefully(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	service := NewService(client)
	ctx := context.Background()

	attempt, err := service.Acquire(ctx, MustLeaseKey("lock:test"), 5*time.Second, nil)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if !attempt.Acquired {
		t.Fatalf("Acquire() should succeed")
	}

	wrongLease := attempt.Lease
	wrongLease.Token = MustLeaseToken("wrong-token")

	if _, ok, err := service.Renew(ctx, wrongLease, 5*time.Second); err != nil {
		t.Fatalf("Renew() error = %v", err)
	} else if ok {
		t.Fatalf("Renew() with wrong token should fail ownership check")
	}

	if err := service.Release(ctx, wrongLease); err != nil {
		t.Fatalf("Release() with wrong token should not error: %v", err)
	}

	owned, err := service.CheckOwnership(ctx, attempt.Lease)
	if err != nil {
		t.Fatalf("CheckOwnership() error = %v", err)
	}
	if !owned {
		t.Fatalf("original lease should still own the key")
	}
}
