package runtime

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestRegistryBindFallsBackButByProfileDoesNot(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port := splitAddr(t, mr.Addr())

	registry := New(&Config{
		Host: host,
		Port: port,
	}, map[string]*Config{
		"static_cache": {
			Database: 2,
		},
	})
	if err := registry.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = registry.Close() })

	defaultHandle, err := registry.Default()
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	boundHandle, err := registry.Bind("missing_profile")
	if err != nil {
		t.Fatalf("Bind(missing_profile) error = %v", err)
	}
	if defaultHandle.Client() != boundHandle.Client() {
		t.Fatalf("Bind(missing_profile) should fall back to default client")
	}

	if _, err := registry.ByProfile("missing_profile"); err == nil {
		t.Fatalf("ByProfile(missing_profile) should return error")
	}
}

func TestRegistryProfilesExposeMergedConfig(t *testing.T) {
	registry := New(&Config{
		Host:         "127.0.0.1",
		Port:         6379,
		MaxActive:    16,
		MaxIdle:      8,
		MinIdleConns: 2,
	}, map[string]*Config{
		"query_cache": {
			Database: 3,
		},
	})

	profiles := registry.Profiles()
	if len(profiles) != 1 {
		t.Fatalf("Profiles() len = %d, want 1", len(profiles))
	}
	profile := profiles[0]
	if profile.Name != "query_cache" {
		t.Fatalf("profile name = %q, want query_cache", profile.Name)
	}
	if profile.Config.Host != "127.0.0.1" || profile.Config.Port != 6379 {
		t.Fatalf("profile config did not inherit default host/port: %+v", profile.Config)
	}
	if profile.Config.Database != 3 {
		t.Fatalf("profile database = %d, want 3", profile.Config.Database)
	}
	if profile.Config.MaxActive != 16 || profile.Config.MaxIdle != 8 || profile.Config.MinIdleConns != 2 {
		t.Fatalf("profile config did not inherit pool settings: %+v", profile.Config)
	}
}

func TestRegistryClientReturnsNamedProfile(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port := splitAddr(t, mr.Addr())

	registry := New(&Config{
		Host: host,
		Port: port,
	}, map[string]*Config{
		"static_cache": {
			Host:     host,
			Port:     port,
			Database: 2,
		},
	})
	if err := registry.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = registry.Close() })

	defaultClient, err := registry.Client("")
	if err != nil {
		t.Fatalf("Client(default) error = %v", err)
	}
	staticClient, err := registry.Client("static_cache")
	if err != nil {
		t.Fatalf("Client(static_cache) error = %v", err)
	}

	if err := defaultClient.Set(context.Background(), "shared:key", "default", 0).Err(); err != nil {
		t.Fatalf("set default key failed: %v", err)
	}
	if err := staticClient.Set(context.Background(), "shared:key", "static", 0).Err(); err != nil {
		t.Fatalf("set static key failed: %v", err)
	}

	if got, _ := defaultClient.Get(context.Background(), "shared:key").Result(); got != "default" {
		t.Fatalf("default db value = %q, want default", got)
	}
	if got, _ := staticClient.Get(context.Background(), "shared:key").Result(); got != "static" {
		t.Fatalf("static db value = %q, want static", got)
	}
}

func TestRegistryRecoversUnavailableProfileOnHealthCheck(t *testing.T) {
	host := "127.0.0.1"
	port := reserveLocalPort(t)
	registry := New(nil, map[string]*Config{
		"static_cache": {
			Host: host,
			Port: port,
		},
	})

	if err := registry.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = registry.Close() })

	status := registry.Status("static_cache")
	if status.State != ProfileStateUnavailable {
		t.Fatalf("static_cache profile state = %q, want unavailable", status.State)
	}

	mr := miniredis.RunT(t)
	realHost, realPort := splitAddr(t, mr.Addr())
	profile := registry.profiles["static_cache"]
	profile.handle.config.Host = realHost
	profile.handle.config.Port = realPort
	profile.nextRetryAt = time.Now().Add(-time.Second)

	if err := registry.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	status = registry.Status("static_cache")
	if status.State != ProfileStateAvailable {
		t.Fatalf("static_cache profile state after recovery = %q, want available", status.State)
	}

	client, err := registry.Client("static_cache")
	if err != nil {
		t.Fatalf("Client(static_cache) error = %v", err)
	}
	if err := client.Set(context.Background(), "recover:key", "ok", 0).Err(); err != nil {
		t.Fatalf("set recovered key failed: %v", err)
	}
}

func splitAddr(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, port, ok := strings.Cut(addr, ":")
	if !ok {
		t.Fatalf("unexpected addr %q", addr)
	}
	parsed, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse port failed: %v", err)
	}
	return host, parsed
}

func reserveLocalPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserveLocalPort() listen error = %v", err)
	}
	defer func() { _ = ln.Close() }()

	host, port, ok := strings.Cut(ln.Addr().String(), ":")
	if !ok || host == "" {
		t.Fatalf("unexpected listener addr %q", ln.Addr().String())
	}
	parsed, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse reserved port failed: %v", err)
	}
	return parsed
}
