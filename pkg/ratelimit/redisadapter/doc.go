// Package redisadapter 提供 ratelimit.Backend 的 Redis 实现。
//
// 该实现使用 Redis Lua 脚本维护共享 token bucket，并通过 KeyFunc 接收
// 外部 keyspace 映射规则。包内不定义项目级 key 名称、限流策略或降级语义。
package redisadapter
