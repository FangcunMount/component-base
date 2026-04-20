package ops

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	defaultScanCount       int64 = 100
	defaultDeleteBatchSize       = 500
)

// DeleteBatch contains a single scan/delete batch result.
type DeleteBatch struct {
	Pattern string
	Keys    []string
	Matched int
	Deleted int
	DryRun  bool
}

// DeleteByPatternOptions controls batched key deletion.
type DeleteByPatternOptions struct {
	ScanCount     int64
	BatchSize     int
	UseUnlink     bool
	DeleteTimeout time.Duration
	DryRun        bool
	OnBatch       func(DeleteBatch)
}

// DefaultDeleteByPatternOptions returns the canonical pattern-deletion defaults.
func DefaultDeleteByPatternOptions() DeleteByPatternOptions {
	return DeleteByPatternOptions{
		ScanCount: defaultScanCount,
		BatchSize: defaultDeleteBatchSize,
		UseUnlink: true,
	}
}

// ScanKeys collects keys using SCAN for the provided pattern.
func ScanKeys(ctx context.Context, client goredis.UniversalClient, pattern string, count int64) ([]string, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if count <= 0 {
		count = defaultScanCount
	}

	keys := make([]string, 0)
	var cursor uint64
	for {
		batch, nextCursor, err := client.Scan(ctx, cursor, pattern, count).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// DeleteByPattern scans and deletes keys in batches.
// When DryRun is enabled, it only counts matches and does not delete them.
func DeleteByPattern(ctx context.Context, client goredis.UniversalClient, pattern string, opts DeleteByPatternOptions) (int, error) {
	if client == nil {
		return 0, fmt.Errorf("redis client is nil")
	}

	opts = normalizeDeleteByPatternOptions(opts)

	var (
		cursor  uint64
		deleted int
	)
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, opts.ScanCount).Result()
		if err != nil {
			return deleted, err
		}
		for start := 0; start < len(keys); start += opts.BatchSize {
			end := start + opts.BatchSize
			if end > len(keys) {
				end = len(keys)
			}
			batch := append([]string(nil), keys[start:end]...)
			if len(batch) == 0 {
				continue
			}

			batchDeleted, err := deleteBatch(ctx, client, batch, opts)
			if err != nil {
				return deleted, err
			}
			deleted += batchDeleted

			if opts.OnBatch != nil {
				opts.OnBatch(DeleteBatch{
					Pattern: pattern,
					Keys:    batch,
					Matched: len(batch),
					Deleted: batchDeleted,
					DryRun:  opts.DryRun,
				})
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return deleted, nil
}

func deleteBatch(ctx context.Context, client goredis.UniversalClient, batch []string, opts DeleteByPatternOptions) (int, error) {
	if opts.DryRun {
		return len(batch), nil
	}

	opCtx := ctx
	cancel := func() {}
	if opts.DeleteTimeout > 0 {
		opCtx, cancel = context.WithTimeout(ctx, opts.DeleteTimeout)
	}
	defer cancel()

	var (
		count int64
		err   error
	)
	if opts.UseUnlink {
		count, err = client.Unlink(opCtx, batch...).Result()
	} else {
		count, err = client.Del(opCtx, batch...).Result()
	}
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func normalizeDeleteByPatternOptions(opts DeleteByPatternOptions) DeleteByPatternOptions {
	if opts.ScanCount <= 0 {
		opts.ScanCount = defaultScanCount
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaultDeleteBatchSize
	}
	return opts
}
