// Package backpressure 提供通用并发背压机制。
//
// 本包用 semaphore 限制同一资源的最大 in-flight 操作数，并可配置等待超时。
// Acquire 返回原始业务 context 和 release 函数，等待超时只影响排队过程，不会给
// 下游业务操作额外加 deadline。
//
// 包内事件和快照是机制级模型，业务项目可以在外层 adapter 中把它们映射为自己的
// metrics、resilience 事件或治理响应。
package backpressure
