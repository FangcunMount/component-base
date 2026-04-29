// Package redisadapter 提供基于 Redis 的 locklease.Manager 实现。
//
// 该 adapter 复用 component-base 的 redis/lease 基础能力，负责把通用
// locklease.Spec 和 locklease.Identity 映射为 Redis SET NX 租约锁。
// 调用方可以通过 KeyFunc 注入自己的 keyspace 规则，从而避免把项目级
// Redis key 设计泄漏到基础库。
package redisadapter
