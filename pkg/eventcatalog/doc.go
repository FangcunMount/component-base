// Package eventcatalog 提供通用事件契约目录。
//
// 本包负责从 YAML 加载事件目录，并提供 topic、handler 和 delivery class 的查询视图。
// 它只表达事件运行时路由契约，不包含项目级事件名常量、消息中间件实现或业务 handler。
//
// 支持的 delivery class 包括 best_effort 和 durable_outbox。项目层可以基于
// DeliveryClassResolver 判断事件是否需要 outbox 等可靠投递机制。
package eventcatalog
