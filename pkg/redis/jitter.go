package rediskit

import (
	"time"

	redisops "github.com/FangcunMount/component-base/pkg/redis/ops"
)

// JitterTTL 按比例为 TTL 增加对称抖动。
func JitterTTL(ttl time.Duration, ratio float64) time.Duration {
	return redisops.JitterTTL(ttl, ratio)
}
