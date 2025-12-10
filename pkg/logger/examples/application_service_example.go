package examples

import (
	"context"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/logger"
)

// ApplicationServiceExample 演示在应用层服务中使用 logger
type ApplicationServiceExample struct {
	// 依赖注入...
}

// CreateUser 演示创建操作的日志记录
func (s *ApplicationServiceExample) CreateUser(ctx context.Context, username, email string) error {
	// 1. 从 context 获取请求范围的 Logger
	l := logger.L(ctx)

	// 2. 记录操作开始（Debug 级别，生产环境可关闭）
	l.Debugw("开始创建用户",
		"action", logger.ActionCreate,
		"resource", logger.ResourceUser,
		"username", username,
	)

	// 3. 执行业务逻辑...
	// user, err := s.repo.Create(ctx, ...)

	// 4. 如果失败，记录错误
	// if err != nil {
	//     l.Errorw("创建用户失败",
	//         "action", logger.ActionCreate,
	//         "resource", logger.ResourceUser,
	//         "error", err.Error(),
	//         "result", logger.ResultFailed,
	//     )
	//     return err
	// }

	// 5. 如果成功，记录成功
	l.Infow("用户创建成功",
		"action", logger.ActionCreate,
		"resource", logger.ResourceUser,
		"resource_id", "user-12345",
		"result", logger.ResultSuccess,
	)

	return nil
}

// Login 演示认证操作的日志记录
func (s *ApplicationServiceExample) Login(ctx context.Context, loginType, appCode string) error {
	l := logger.L(ctx)

	// 认证开始
	l.Debugw("开始认证",
		"action", logger.ActionLogin,
		"login_type", loginType,
		"app_code", appCode,
	)

	// ... 认证逻辑

	// 认证成功
	l.Infow("认证成功",
		"action", logger.ActionLogin,
		"user_id", "user-12345",
		"account_id", "account-67890",
		"result", logger.ResultSuccess,
	)

	return nil
}

// GetUserByID 演示查询操作的日志记录
func (s *ApplicationServiceExample) GetUserByID(ctx context.Context, userID string) error {
	l := logger.L(ctx)

	// 查询操作通常使用 Debug 级别
	l.Debugw("查询用户",
		"action", logger.ActionRead,
		"resource", logger.ResourceUser,
		"resource_id", userID,
	)

	// ... 查询逻辑

	return nil
}

// UpdateUser 演示更新操作的日志记录
func (s *ApplicationServiceExample) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error {
	l := logger.L(ctx)

	l.Debugw("开始更新用户",
		"action", logger.ActionUpdate,
		"resource", logger.ResourceUser,
		"resource_id", userID,
		"update_fields", updates,
	)

	// ... 更新逻辑

	l.Infow("用户更新成功",
		"action", logger.ActionUpdate,
		"resource", logger.ResourceUser,
		"resource_id", userID,
		"result", logger.ResultSuccess,
	)

	return nil
}

// DeleteUser 演示删除操作的日志记录
func (s *ApplicationServiceExample) DeleteUser(ctx context.Context, userID string) error {
	l := logger.L(ctx)

	// 删除是重要操作，应该记录 Info 级别
	l.Infow("开始删除用户",
		"action", logger.ActionDelete,
		"resource", logger.ResourceUser,
		"resource_id", userID,
	)

	// ... 删除逻辑

	l.Infow("用户删除成功",
		"action", logger.ActionDelete,
		"resource", logger.ResourceUser,
		"resource_id", userID,
		"result", logger.ResultSuccess,
	)

	return nil
}

// ProcessWithSubLogger 演示如何添加额外字段创建子 Logger
func (s *ApplicationServiceExample) ProcessWithSubLogger(ctx context.Context, orderID string) error {
	// 获取基础 Logger
	l := logger.L(ctx)

	// 为特定业务流程创建带额外字段的 Logger
	orderLogger := l.WithFields(
		log.String("order_id", orderID),
		log.String("resource", "order"),
	)

	// 后续所有日志都会自动带上 order_id
	orderLogger.Infow("开始处理订单")
	orderLogger.Debugw("验证订单状态")
	orderLogger.Infow("订单处理完成", "result", logger.ResultSuccess)

	return nil
}

// HandleError 演示错误处理时的日志记录
func (s *ApplicationServiceExample) HandleError(ctx context.Context) error {
	l := logger.L(ctx)

	// 业务错误 - 使用 Warn
	l.Warnw("用户不存在",
		"action", logger.ActionRead,
		"resource", logger.ResourceUser,
		"resource_id", "user-not-found",
		"result", logger.ResultFailed,
	)

	// 权限拒绝 - 使用 Warn
	l.Warnw("权限不足",
		"action", logger.ActionDelete,
		"resource", logger.ResourceUser,
		"result", logger.ResultDenied,
	)

	// 系统错误 - 使用 Error
	l.Errorw("数据库连接失败",
		"action", logger.ActionRead,
		"resource", logger.ResourceUser,
		"error", "connection refused",
		"result", logger.ResultFailed,
	)

	return nil
}

// ApplicationServiceLogOutput 说明应用层日志的格式
//
// 操作开始日志（Debug 级别）：
//
//	{
//	    "level": "debug",
//	    "ts": "2025-12-10T10:00:00.050Z",
//	    "msg": "开始创建用户",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001",
//	    "action": "create",
//	    "resource": "user",
//	    "username": "john_doe"
//	}
//
// 操作成功日志（Info 级别）：
//
//	{
//	    "level": "info",
//	    "ts": "2025-12-10T10:00:00.120Z",
//	    "msg": "用户创建成功",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001",
//	    "action": "create",
//	    "resource": "user",
//	    "resource_id": "user-12345",
//	    "result": "success"
//	}
//
// 业务错误日志（Warn 级别）：
//
//	{
//	    "level": "warn",
//	    "ts": "2025-12-10T10:00:00.080Z",
//	    "msg": "用户不存在",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001",
//	    "action": "read",
//	    "resource": "user",
//	    "resource_id": "user-not-found",
//	    "result": "failed"
//	}
//
// 系统错误日志（Error 级别）：
//
//	{
//	    "level": "error",
//	    "ts": "2025-12-10T10:00:00.090Z",
//	    "msg": "数据库连接失败",
//	    "trace_id": "abc123def456",
//	    "request_id": "req-001",
//	    "action": "read",
//	    "resource": "user",
//	    "error": "connection refused",
//	    "result": "failed"
//	}
func ApplicationServiceLogOutput() {}
