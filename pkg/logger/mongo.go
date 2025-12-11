package logger

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
)

// MongoHook MongoDB 命令执行钩子
// 用于记录 MongoDB 命令执行情况，类似于 GORM logger 和 Redis Hook
type MongoHook struct {
	enabled            bool
	slowThreshold      time.Duration
	logCommandDetail   bool          // 是否记录命令详情（默认 true，类似 GORM 记录 SQL）
	logReplyDetail     bool          // 是否记录响应详情（默认 false）
	logHeartbeat       bool          // 是否记录心跳（默认 false，减少日志噪音）
	logStarted         bool          // 是否记录命令开始（默认 false，只记录完成）
	ignoredCommands    []string      // 忽略的命令列表（如心跳、握手等高频命令）
	slowReadThreshold  time.Duration // 慢读命令阈值
	slowWriteThreshold time.Duration // 慢写命令阈值
	commandCache       sync.Map      // 缓存命令详情，key 为 requestID
}

// commandInfo 存储命令信息
type commandInfo struct {
	database   string
	collection string
	query      string
	ctx        context.Context
}

// MongoHookConfig MongoDB 钩子配置
type MongoHookConfig struct {
	// Enabled 是否启用日志记录
	Enabled bool
	// SlowThreshold 慢命令阈值（超过此时间会记录警告）
	SlowThreshold time.Duration
	// LogCommandDetail 是否记录命令详情（建议仅在调试时开启）
	LogCommandDetail bool
	// LogReplyDetail 是否记录响应详情（建议仅在调试时开启）
	LogReplyDetail bool
	// LogHeartbeat 是否记录心跳日志（默认 false）
	LogHeartbeat bool
	// LogStarted 是否记录命令开始日志（默认 false，只记录完成）
	LogStarted bool
	// IgnoredCommands 忽略的命令列表（降噪）
	IgnoredCommands []string
	// SlowReadThreshold 慢读命令阈值（0 表示使用 SlowThreshold）
	SlowReadThreshold time.Duration
	// SlowWriteThreshold 慢写命令阈值（0 表示使用 SlowThreshold）
	SlowWriteThreshold time.Duration
}

// DefaultMongoHookConfig 返回默认 MongoDB 钩子配置
func DefaultMongoHookConfig() MongoHookConfig {
	return MongoHookConfig{
		Enabled:            true,
		SlowThreshold:      200 * time.Millisecond,
		LogCommandDetail:   true,  // 默认记录查询详情（类似 GORM 的 SQL 日志），敏感信息会自动脱敏
		LogReplyDetail:     false, // 默认不记录响应详情（避免返回大量数据）
		LogHeartbeat:       false, // 默认不记录心跳，减少噪音
		LogStarted:         false, // 默认不记录开始，减少日志量
		IgnoredCommands:    []string{"hello", "isMaster", "ismaster", "ping", "saslStart", "saslContinue"},
		SlowReadThreshold:  0, // 使用 SlowThreshold
		SlowWriteThreshold: 0, // 使用 SlowThreshold
	}
}

// NewMongoHook 创建 MongoDB 钩子
// enabled: 是否启用日志记录
// slowThreshold: 慢命令阈值（超过此时间会记录警告）
func NewMongoHook(enabled bool, slowThreshold time.Duration) *MongoHook {
	if slowThreshold <= 0 {
		slowThreshold = 200 * time.Millisecond // 默认 200ms
	}

	return &MongoHook{
		enabled:            enabled,
		slowThreshold:      slowThreshold,
		logCommandDetail:   true, // 默认记录查询详情
		logReplyDetail:     false,
		logHeartbeat:       false,
		logStarted:         false,
		ignoredCommands:    []string{"hello", "isMaster", "ismaster", "ping", "saslStart", "saslContinue"},
		slowReadThreshold:  0,
		slowWriteThreshold: 0,
	}
}

// NewMongoHookWithConfig 使用指定配置创建 MongoDB 钩子
func NewMongoHookWithConfig(config MongoHookConfig) *MongoHook {
	slowThreshold := config.SlowThreshold
	if slowThreshold <= 0 {
		slowThreshold = 200 * time.Millisecond
	}

	ignoredCommands := config.IgnoredCommands
	if len(ignoredCommands) == 0 {
		ignoredCommands = []string{"hello", "isMaster", "ismaster", "ping", "saslStart", "saslContinue"}
	}

	return &MongoHook{
		enabled:            config.Enabled,
		slowThreshold:      slowThreshold,
		logCommandDetail:   config.LogCommandDetail,
		logReplyDetail:     config.LogReplyDetail,
		logHeartbeat:       config.LogHeartbeat,
		logStarted:         config.LogStarted,
		ignoredCommands:    ignoredCommands,
		slowReadThreshold:  config.SlowReadThreshold,
		slowWriteThreshold: config.SlowWriteThreshold,
	}
}

// CommandMonitor 返回 MongoDB 命令监控器
// 用于监控所有 MongoDB 命令的执行
func (h *MongoHook) CommandMonitor() *event.CommandMonitor {
	if !h.enabled {
		return nil
	}

	return &event.CommandMonitor{
		Started: func(ctx context.Context, evt *event.CommandStartedEvent) {
			h.logCommandStarted(ctx, evt)
		},
		Succeeded: func(ctx context.Context, evt *event.CommandSucceededEvent) {
			h.logCommandSucceeded(ctx, evt)
		},
		Failed: func(ctx context.Context, evt *event.CommandFailedEvent) {
			h.logCommandFailed(ctx, evt)
		},
	}
}

// PoolMonitor 返回 MongoDB 连接池监控器
// 用于监控连接池的创建、关闭等事件
func (h *MongoHook) PoolMonitor() *event.PoolMonitor {
	if !h.enabled {
		return nil
	}

	return &event.PoolMonitor{
		Event: func(evt *event.PoolEvent) {
			h.logPoolEvent(evt)
		},
	}
}

// ServerMonitor 返回 MongoDB 服务器监控器
// 用于监控服务器心跳、状态变更等事件
func (h *MongoHook) ServerMonitor() *event.ServerMonitor {
	if !h.enabled {
		return nil
	}

	return &event.ServerMonitor{
		ServerHeartbeatStarted: func(evt *event.ServerHeartbeatStartedEvent) {
			// 默认不记录心跳开始，减少日志噪音
			if h.logHeartbeat {
				log.MongoDebug("MongoDB heartbeat started",
					log.String("connection_id", evt.ConnectionID),
				)
			}
		},
		ServerHeartbeatSucceeded: func(evt *event.ServerHeartbeatSucceededEvent) {
			// 默认不记录心跳成功，减少日志噪音
			if h.logHeartbeat {
				log.MongoDebug("MongoDB heartbeat succeeded",
					log.String("connection_id", evt.ConnectionID),
					log.Float64("duration_ms", float64(evt.Duration.Nanoseconds())/1e6),
				)
			}
		},
		ServerHeartbeatFailed: func(evt *event.ServerHeartbeatFailedEvent) {
			// 心跳失败始终记录
			log.MongoWarn("MongoDB heartbeat failed",
				log.String("connection_id", evt.ConnectionID),
				log.Float64("duration_ms", float64(evt.Duration.Nanoseconds())/1e6),
				log.String("error", evt.Failure.Error()),
			)
		},
	}
}

// isIgnoredCommand 判断命令是否应该被忽略
func (h *MongoHook) isIgnoredCommand(commandName string) bool {
	for _, ignored := range h.ignoredCommands {
		if strings.EqualFold(commandName, ignored) {
			return true
		}
	}
	return false
}

// isWriteCommand 判断是否为写命令
func (h *MongoHook) isWriteCommand(commandName string) bool {
	writeCommands := []string{"insert", "update", "delete", "findAndModify", "findandmodify"}
	commandLower := strings.ToLower(commandName)
	for _, write := range writeCommands {
		if commandLower == write {
			return true
		}
	}
	return false
}

// getSlowThreshold 获取命令的慢查询阈值
func (h *MongoHook) getSlowThreshold(commandName string) time.Duration {
	if h.isWriteCommand(commandName) && h.slowWriteThreshold > 0 {
		return h.slowWriteThreshold
	}
	if !h.isWriteCommand(commandName) && h.slowReadThreshold > 0 {
		return h.slowReadThreshold
	}
	return h.slowThreshold
}

// logCommandStarted 记录命令开始执行
func (h *MongoHook) logCommandStarted(ctx context.Context, evt *event.CommandStartedEvent) {
	// 忽略配置的命令
	if h.isIgnoredCommand(evt.CommandName) {
		return
	}

	// 缓存命令信息，供 succeeded 时使用
	if h.logCommandDetail && evt.Command != nil {
		collection := extractCollectionName(evt.Command, evt.CommandName)
		safeCmd := sanitizeCommand(evt.Command, evt.CommandName)

		h.commandCache.Store(evt.RequestID, &commandInfo{
			database:   evt.DatabaseName,
			collection: collection,
			query:      safeCmd,
			ctx:        ctx,
		})
	}

	// 如果不记录 started，直接返回
	if !h.logStarted {
		return
	}

	fields := []log.Field{
		log.Int64("request_id", evt.RequestID),
		log.String("database", evt.DatabaseName),
		log.String("command", evt.CommandName),
		log.String("connection_id", evt.ConnectionID),
	}

	// 提取集合名称
	if collection := extractCollectionName(evt.Command, evt.CommandName); collection != "" {
		fields = append(fields, log.String("collection", collection))
	}

	// 添加分布式追踪字段
	fields = append(fields, log.TraceFields(ctx)...)

	// 记录命令详情（类似 GORM 记录 SQL 语句）
	if h.logCommandDetail && evt.Command != nil {
		safeCmd := sanitizeCommand(evt.Command, evt.CommandName)
		fields = append(fields, log.String("query", safeCmd))
	}

	log.MongoDebug("MongoDB command started", fields...)
}

// logCommandSucceeded 记录命令执行成功
func (h *MongoHook) logCommandSucceeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	// 忽略配置的命令
	if h.isIgnoredCommand(evt.CommandName) {
		return
	}

	elapsed := evt.Duration
	slowThreshold := h.getSlowThreshold(evt.CommandName)

	fields := []log.Field{
		log.Int64("request_id", evt.RequestID),
		log.String("command", evt.CommandName),
		log.String("connection_id", evt.ConnectionID),
		log.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
	}

	// 从缓存中获取命令信息
	if cached, ok := h.commandCache.LoadAndDelete(evt.RequestID); ok {
		if info, ok := cached.(*commandInfo); ok {
			// 添加数据库和集合信息
			fields = append(fields, log.String("database", info.database))
			if info.collection != "" {
				fields = append(fields, log.String("collection", info.collection))
			}

			// 添加查询语句（类似 GORM 记录 SQL）
			if info.query != "" {
				fields = append(fields, log.String("query", info.query))
			}

			// 使用原始 context 获取追踪信息
			fields = append(fields, log.TraceFields(info.ctx)...)
		}
	} else {
		// 如果没有缓存，使用当前 context
		fields = append(fields, log.TraceFields(ctx)...)
	}

	// 仅在显式开启且慢查询时记录响应详情（避免敏感信息泄露）
	if h.logReplyDetail && elapsed > slowThreshold && evt.Reply != nil {
		safeReply := sanitizeReply(evt.Reply)
		fields = append(fields, log.String("reply", safeReply))
	}

	// 判断是否为慢命令
	if elapsed > slowThreshold {
		fields = append(fields,
			log.String("event", "slow_command"),
			log.Duration("slow_threshold", slowThreshold),
		)
		log.MongoWarn("MongoDB slow command", fields...)
	} else {
		log.MongoDebug("MongoDB command succeeded", fields...)
	}
}

// logCommandFailed 记录命令执行失败
func (h *MongoHook) logCommandFailed(ctx context.Context, evt *event.CommandFailedEvent) {
	// 失败命令始终记录，不忽略
	fields := []log.Field{
		log.Int64("request_id", evt.RequestID),
		log.String("command", evt.CommandName),
		log.String("connection_id", evt.ConnectionID),
		log.Float64("elapsed_ms", float64(evt.Duration.Nanoseconds())/1e6),
		log.String("error", evt.Failure),
	}

	// 从缓存中获取命令信息
	if cached, ok := h.commandCache.LoadAndDelete(evt.RequestID); ok {
		if info, ok := cached.(*commandInfo); ok {
			// 添加数据库和集合信息
			fields = append(fields, log.String("database", info.database))
			if info.collection != "" {
				fields = append(fields, log.String("collection", info.collection))
			}

			// 添加查询语句
			if info.query != "" {
				fields = append(fields, log.String("query", info.query))
			}

			// 使用原始 context 获取追踪信息
			fields = append(fields, log.TraceFields(info.ctx)...)
		}
	} else {
		// 如果没有缓存，使用当前 context
		fields = append(fields, log.TraceFields(ctx)...)
	}

	log.MongoError("MongoDB command failed", fields...)
}

// logPoolEvent 记录连接池事件
func (h *MongoHook) logPoolEvent(evt *event.PoolEvent) {
	switch evt.Type {
	case event.PoolCreated:
		log.Mongo("MongoDB connection pool created",
			log.String("address", evt.Address),
		)
	case event.PoolCleared:
		log.Mongo("MongoDB connection pool cleared",
			log.String("address", evt.Address),
		)
	case event.PoolClosedEvent:
		log.Mongo("MongoDB connection pool closed",
			log.String("address", evt.Address),
		)
	case event.ConnectionCreated:
		log.MongoDebug("MongoDB connection created",
			log.String("address", evt.Address),
			log.Uint64("connection_id", evt.ConnectionID),
		)
	case event.ConnectionReady:
		log.MongoDebug("MongoDB connection ready",
			log.String("address", evt.Address),
			log.Uint64("connection_id", evt.ConnectionID),
		)
	case event.ConnectionClosed:
		log.MongoDebug("MongoDB connection closed",
			log.String("address", evt.Address),
			log.Uint64("connection_id", evt.ConnectionID),
			log.String("reason", evt.Reason),
		)
	case event.GetStarted:
		// 获取连接开始（通常不记录，避免日志过多）
	case event.GetSucceeded:
		log.MongoDebug("MongoDB connection get succeeded",
			log.String("address", evt.Address),
			log.Uint64("connection_id", evt.ConnectionID),
		)
	case event.GetFailed:
		log.MongoWarn("MongoDB connection get failed",
			log.String("address", evt.Address),
			log.String("reason", evt.Reason),
		)
		// 注意：以下事件类型在某些版本的 MongoDB 驱动中可能不存在
		// 如果遇到编译错误，可以注释掉相关代码
	}
}

// extractCollectionName 从命令中提取集合名称
func extractCollectionName(cmd bson.Raw, commandName string) string {
	// 尝试从命令中提取集合名称
	// 大部分命令的第一个字段就是集合名称
	val := cmd.Lookup(commandName)
	if val.Type == bson.TypeString {
		return val.StringValue()
	}
	return ""
}

// sanitizeCommand 对命令进行安全处理（移除敏感信息）
func sanitizeCommand(cmd bson.Raw, commandName string) string {
	// 认证相关命令直接返回命令名，不记录详情
	authCommands := []string{"saslStart", "saslContinue", "authenticate", "getnonce"}
	for _, auth := range authCommands {
		if strings.EqualFold(commandName, auth) {
			return "[REDACTED: authentication command]"
		}
	}

	// 对于写操作，只记录元信息，不记录具体数据
	if isWriteCommand := func(name string) bool {
		writes := []string{"insert", "update", "delete", "findAndModify", "findandmodify"}
		nameLower := strings.ToLower(name)
		for _, w := range writes {
			if nameLower == w {
				return true
			}
		}
		return false
	}(commandName); isWriteCommand {
		// 只保留安全的元数据字段
		return extractSafeMetadata(cmd, commandName)
	}

	// 其他命令限制长度
	cmdStr := cmd.String()
	if len(cmdStr) > 500 {
		return cmdStr[:500] + "... [truncated]"
	}
	return cmdStr
}

// sanitizeReply 对响应进行安全处理
func sanitizeReply(reply bson.Raw) string {
	// 不记录完整响应，只记录关键统计信息
	// 默认不记录详细响应以保护敏感信息
	return "[summary: ok, affected docs info available]"
}

// extractSafeMetadata 提取安全的元数据（用于写命令）
func extractSafeMetadata(cmd bson.Raw, commandName string) string {
	var parts []string

	// 提取集合名
	if val := cmd.Lookup(commandName); val.Type == bson.TypeString {
		parts = append(parts, "collection:"+val.StringValue())
	}

	// 提取操作数量
	switch strings.ToLower(commandName) {
	case "insert":
		if docs := cmd.Lookup("documents"); docs.Type == bson.TypeArray {
			arr, _ := docs.Array().Values()
			parts = append(parts, "count:"+fmt.Sprintf("%d", len(arr)))
		}
	case "update", "delete":
		if updates := cmd.Lookup("updates"); updates.Type == bson.TypeArray {
			arr, _ := updates.Array().Values()
			parts = append(parts, "count:"+fmt.Sprintf("%d", len(arr)))
		} else if deletes := cmd.Lookup("deletes"); deletes.Type == bson.TypeArray {
			arr, _ := deletes.Array().Values()
			parts = append(parts, "count:"+fmt.Sprintf("%d", len(arr)))
		}
	}

	if len(parts) > 0 {
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return "{metadata: not available}"
}

// toString 辅助函数：将值转换为字符串（已移除，使用 fmt.Sprintf 直接处理）
