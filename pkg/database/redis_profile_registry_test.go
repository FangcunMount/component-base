package database

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestNamedRedisRegistryFallsBackToDefault(t *testing.T) {
	mr := miniredis.RunT(t)

	host, port := splitMiniredisAddr(t, mr.Addr())
	registry := NewNamedRedisRegistry(&RedisConfig{
		Host: host,
		Port: port,
	}, map[string]*RedisConfig{
		"static_cache": {
			Host:     host,
			Port:     port,
			Database: 2,
		},
	})

	if err := registry.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = registry.Close()
	})

	defaultClient, err := registry.GetClient("")
	if err != nil {
		t.Fatalf("GetClient(default) error = %v", err)
	}
	if err := defaultClient.Set(context.Background(), "default:key", "1", 0).Err(); err != nil {
		t.Fatalf("set default key failed: %v", err)
	}

	queryClient, err := registry.GetClient("query_cache")
	if err != nil {
		t.Fatalf("GetClient(query_cache) error = %v", err)
	}
	if got, err := queryClient.Get(context.Background(), "default:key").Result(); err != nil || got != "1" {
		t.Fatalf("fallback client should read default db key, got value=%q err=%v", got, err)
	}

	if status := registry.ProfileStatus("query_cache"); status.State != RedisProfileStateMissing {
		t.Fatalf("query_cache profile state = %q, want missing", status.State)
	}
}

func TestNamedRedisRegistryUsesNamedProfiles(t *testing.T) {
	mr := miniredis.RunT(t)

	host, port := splitMiniredisAddr(t, mr.Addr())
	registry := NewNamedRedisRegistry(&RedisConfig{
		Host: host,
		Port: port,
	}, map[string]*RedisConfig{
		"static_cache": {
			Host:     host,
			Port:     port,
			Database: 2,
		},
		"query_cache": {
			Host:     host,
			Port:     port,
			Database: 3,
		},
	})

	if err := registry.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = registry.Close()
	})

	ctx := context.Background()
	defaultClient, err := registry.GetClient("")
	if err != nil {
		t.Fatalf("GetClient(default) error = %v", err)
	}
	staticClient, err := registry.GetClient("static_cache")
	if err != nil {
		t.Fatalf("GetClient(static_cache) error = %v", err)
	}
	queryClient, err := registry.GetClient("query_cache")
	if err != nil {
		t.Fatalf("GetClient(query_cache) error = %v", err)
	}

	if err := defaultClient.Set(ctx, "shared:key", "default", 0).Err(); err != nil {
		t.Fatalf("set default key failed: %v", err)
	}
	if err := staticClient.Set(ctx, "shared:key", "static", 0).Err(); err != nil {
		t.Fatalf("set static key failed: %v", err)
	}
	if err := queryClient.Set(ctx, "shared:key", "query", 0).Err(); err != nil {
		t.Fatalf("set query key failed: %v", err)
	}

	if got, _ := defaultClient.Get(ctx, "shared:key").Result(); got != "default" {
		t.Fatalf("default db value = %q, want default", got)
	}
	if got, _ := staticClient.Get(ctx, "shared:key").Result(); got != "static" {
		t.Fatalf("static db value = %q, want static", got)
	}
	if got, _ := queryClient.Get(ctx, "shared:key").Result(); got != "query" {
		t.Fatalf("query db value = %q, want query", got)
	}

	if err := registry.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	if status := registry.ProfileStatus("static_cache"); status.State != RedisProfileStateAvailable {
		t.Fatalf("static_cache profile state = %q, want available", status.State)
	}
	if status := registry.ProfileStatus("query_cache"); status.State != RedisProfileStateAvailable {
		t.Fatalf("query_cache profile state = %q, want available", status.State)
	}
}

func TestNamedRedisRegistryMarksUnavailableProfilesWithoutBreakingDefault(t *testing.T) {
	mr := miniredis.RunT(t)

	host, port := splitMiniredisAddr(t, mr.Addr())
	registry := NewNamedRedisRegistry(&RedisConfig{
		Host: host,
		Port: port,
	}, map[string]*RedisConfig{
		"static_cache": {
			Host: host,
			Port: 63999,
		},
	})

	if err := registry.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = registry.Close()
	})

	status := registry.ProfileStatus("static_cache")
	if status.State != RedisProfileStateUnavailable {
		t.Fatalf("static_cache profile state = %q, want unavailable", status.State)
	}
	if status.Err == nil {
		t.Fatalf("static_cache profile should retain connect error")
	}

	if _, err := registry.GetClient("static_cache"); err == nil {
		t.Fatalf("GetClient(static_cache) unexpectedly succeeded")
	}

	defaultClient, err := registry.GetClient("")
	if err != nil {
		t.Fatalf("GetClient(default) error = %v", err)
	}
	if err := defaultClient.Set(context.Background(), "default:key", "1", 0).Err(); err != nil {
		t.Fatalf("set default key failed: %v", err)
	}

	queryClient, err := registry.GetClient("query_cache")
	if err != nil {
		t.Fatalf("GetClient(query_cache) error = %v", err)
	}
	if got, err := queryClient.Get(context.Background(), "default:key").Result(); err != nil || got != "1" {
		t.Fatalf("fallback client should read default db key, got value=%q err=%v", got, err)
	}

	if err := registry.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}
}

func splitMiniredisAddr(t *testing.T, addr string) (string, int) {
	t.Helper()

	host, portStr, ok := strings.Cut(addr, ":")
	if !ok {
		t.Fatalf("unexpected miniredis addr %q", addr)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse miniredis port failed: %v", err)
	}
	return host, port
}
