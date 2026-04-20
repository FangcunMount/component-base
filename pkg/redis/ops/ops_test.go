package ops

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestDeleteByPatternDryRunKeepsKeysAndReportsBatches(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	for _, key := range []string{"cache:a", "cache:b", "other:c"} {
		if err := client.Set(ctx, key, "1", 0).Err(); err != nil {
			t.Fatalf("seed key %s failed: %v", key, err)
		}
	}

	var batches []DeleteBatch
	deleted, err := DeleteByPattern(ctx, client, "cache:*", DeleteByPatternOptions{
		ScanCount: 1,
		BatchSize: 1,
		DryRun:    true,
		OnBatch: func(batch DeleteBatch) {
			batches = append(batches, batch)
		},
	})
	if err != nil {
		t.Fatalf("DeleteByPattern() error = %v", err)
	}
	if deleted != 2 {
		t.Fatalf("DeleteByPattern() dry-run count = %d, want 2", deleted)
	}
	if len(batches) != 2 {
		t.Fatalf("DeleteByPattern() batches = %d, want 2", len(batches))
	}

	for _, key := range []string{"cache:a", "cache:b", "other:c"} {
		exists, err := client.Exists(ctx, key).Result()
		if err != nil {
			t.Fatalf("Exists(%s) error = %v", key, err)
		}
		if exists == 0 {
			t.Fatalf("dry-run should not delete key %s", key)
		}
	}
}
