// Package rediskit 提供 component-base Redis Foundation 的兼容门面。
//
// 新的 Foundation 子包位于：
//   - pkg/redis/runtime
//   - pkg/redis/keyspace
//   - pkg/redis/ops
//   - pkg/redis/store
//   - pkg/redis/lease
//   - pkg/redis/observability
//
// 当前根包刻意保留旧 helper 的稳定性，并将实现委托给新的
// Foundation 子包。需要新能力的调用方应优先直接导入子包。
package rediskit
