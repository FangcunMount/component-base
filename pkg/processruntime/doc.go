// Package processruntime 提供进程启动与关闭编排的基础机制。
//
// 该包只表达通用运行时生命周期能力，包括：
//   - 按注册顺序执行关闭钩子的 Lifecycle。
//   - 并发运行一组长跑服务并返回首个错误的 RunGroup。
//   - 按阶段顺序准备进程状态的泛型 Runner。
//
// 业务项目可以把具体的 HTTP/gRPC 服务、消息订阅器、调度器等装配为
// ServiceRunner 或 Stage，但不应把业务语义放入本包。
package processruntime
