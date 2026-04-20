package store

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

type sampleValue struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestValueStoreJSONRoundTrip(t *testing.T) {
	client, cleanup := newTestClient(t)
	defer cleanup()

	valueStore := NewValueStore[sampleValue](client, JSONCodec[sampleValue]{})
	key := MustStoreKey("sample:json")
	input := sampleValue{Name: "alpha", Count: 2}

	if err := valueStore.Set(context.Background(), key, input, time.Minute); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, ok, err := valueStore.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatalf("Get() should hit")
	}
	if got != input {
		t.Fatalf("Get() = %+v, want %+v", got, input)
	}

	exists, err := valueStore.Exists(context.Background(), key)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Fatalf("Exists() should return true")
	}

	if err := valueStore.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	exists, err = valueStore.Exists(context.Background(), key)
	if err != nil {
		t.Fatalf("Exists() after delete error = %v", err)
	}
	if exists {
		t.Fatalf("Exists() should return false after delete")
	}
}

func TestValueStoreSetIfAbsentAndCompression(t *testing.T) {
	client, cleanup := newTestClient(t)
	defer cleanup()

	valueStore := NewValueStore[sampleValue](client, NewCompressionCodec(JSONCodec[sampleValue]{}))
	key := MustStoreKey("sample:compressed")
	input := sampleValue{Name: "beta", Count: 4}

	ok, err := valueStore.SetIfAbsent(context.Background(), key, input, time.Minute)
	if err != nil {
		t.Fatalf("SetIfAbsent() error = %v", err)
	}
	if !ok {
		t.Fatalf("first SetIfAbsent() should succeed")
	}

	ok, err = valueStore.SetIfAbsent(context.Background(), key, sampleValue{Name: "gamma", Count: 5}, time.Minute)
	if err != nil {
		t.Fatalf("second SetIfAbsent() error = %v", err)
	}
	if ok {
		t.Fatalf("second SetIfAbsent() should fail")
	}

	got, hit, err := valueStore.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !hit || got != input {
		t.Fatalf("Get() = (%+v, %v), want (%+v, true)", got, hit, input)
	}
}

func TestValueStoreMissingAndInvalidPayload(t *testing.T) {
	client, cleanup := newTestClient(t)
	defer cleanup()

	valueStore := NewValueStore[sampleValue](client, JSONCodec[sampleValue]{})
	key := MustStoreKey("sample:missing")

	if _, hit, err := valueStore.Get(context.Background(), key); err != nil {
		t.Fatalf("Get() missing error = %v", err)
	} else if hit {
		t.Fatalf("Get() missing should not hit")
	}

	if err := client.Set(context.Background(), "sample:missing", []byte("not-json"), 0).Err(); err != nil {
		t.Fatalf("seed invalid payload failed: %v", err)
	}
	if _, _, err := valueStore.Get(context.Background(), key); err == nil {
		t.Fatalf("Get() should fail on invalid payload")
	}
}

func newTestClient(t *testing.T) (*goredis.Client, func()) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return client, func() {
		_ = client.Close()
	}
}
