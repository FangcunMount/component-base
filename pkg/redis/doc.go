// Package rediskit provides reusable Redis primitives shared across services.
//
// It intentionally stays below business adapters and key schemas: callers bring
// their own Redis clients, naming conventions, TTL policies, and repositories.
// The package only offers low-level helpers such as keyspace prefixing, TTL
// jitter, batched scan/delete, lease locks, and single-key atomic consume.
//
// For pattern deletion, callers should prefer DefaultDeleteByPatternOptions()
// instead of assembling their own zero-value option struct.
package rediskit
