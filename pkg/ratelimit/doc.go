// Package ratelimit 提供通用限流机制。
//
// 本包定义限流策略、决策结果、Limiter 端口、本地 token bucket 限流器和
// 基于共享 Backend 的分布式限流器。它不绑定 HTTP、gRPC、Redis keyspace
// 或业务观测模型；项目层可以把 Decision 映射为自己的响应头、指标或治理事件。
package ratelimit
