// Package logger 提供请求范围的日志工具
//
// 该包提供了一套完整的日志解决方案，包括：
// - RequestLogger: 请求范围的日志记录器，支持通过 context 传递
// - 标准字段定义: 统一的日志字段命名规范
// - GORM 日志适配器: 将 GORM 日志输出到结构化日志系统
// - Redis 日志钩子: 记录 Redis 命令执行情况
// - HTTP 日志中间件: Gin 框架的请求日志中间件
// - gRPC 日志拦截器: gRPC 服务端日志拦截器
//
// 设计目标:
// 1. 上下文传递: 通过 context.Context 传递 Logger，确保追踪信息不丢失
// 2. 结构化日志: 所有日志必须是 key-value 结构，便于机器解析
// 3. 字段标准化: 统一字段命名，提供常量定义
// 4. 分层日志: 不同层级（HTTP/gRPC/Service/DB）有明确的日志策略
// 5. 零侵入性: 业务代码只需调用 logger.L(ctx)，无需关心底层实现
//
// 基本用法:
//
//	func (s *userService) CreateUser(ctx context.Context, dto CreateUserDTO) (*User, error) {
//	    l := logger.L(ctx)
//	    l.Infow("开始创建用户",
//	        "action", logger.ActionCreate,
//	        "resource", logger.ResourceUser,
//	    )
//	    // ...
//	}
//
// 该包设计为可独立使用，未来可迁移到 component-base 供多个项目共享。
package logger
