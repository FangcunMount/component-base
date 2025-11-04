package idutil

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/FangcunMount/component-base/pkg/util/iputil"
	"github.com/FangcunMount/component-base/pkg/util/stringutil"
	"github.com/sony/sonyflake"
	hashids "github.com/speps/go-hashids"
)

// 62进制字母表
const (
	Alphabet62 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	Alphabet36 = "abcdefghijklmnopqrstuvwxyz1234567890"
)

// 雪花算法实例
var sf *sonyflake.Sonyflake

// 初始化雪花算法
func init() {
	var st sonyflake.Settings
	st.MachineID = func() (uint16, error) {
		ip := iputil.GetLocalIP()

		return uint16([]byte(ip)[2])<<8 + uint16([]byte(ip)[3]), nil
	}

	sf = sonyflake.NewSonyflake(st)
	// 如果初始化失败(例如在测试环境中),使用 nil 标记,后续使用降级方案
	if sf == nil {
		// 将在 GetIntID 中使用降级方案
	}
}

// GetIntID 获取雪花算法生成的唯一ID
// 在测试环境或雪花算法不可用时,返回基于时间戳的ID
func GetIntID() uint64 {
	if sf == nil {
		// 降级方案:使用纳秒时间戳作为ID(仅用于测试)
		return uint64(time.Now().UnixNano())
	}

	id, err := sf.NextID()
	if err != nil {
		// 降级方案:使用纳秒时间戳
		return uint64(time.Now().UnixNano())
	}

	return id
}

// GetInstanceID 获取实例ID
func GetInstanceID(uid uint64, prefix string) string {
	hd := hashids.NewData()
	hd.Alphabet = Alphabet36
	hd.MinLength = 6
	hd.Salt = "x20k5x"

	h, err := hashids.NewWithData(hd)
	if err != nil {
		panic(err)
	}

	i, err := h.Encode([]int{int(uid)})
	if err != nil {
		panic(err)
	}

	return prefix + stringutil.Reverse(i)
}

// GetUUID36 获取36进制ID
func GetUUID36(prefix string) string {
	id := GetIntID()
	hd := hashids.NewData()
	hd.Alphabet = Alphabet36

	h, err := hashids.NewWithData(hd)
	if err != nil {
		panic(err)
	}

	i, err := h.Encode([]int{int(id)})
	if err != nil {
		panic(err)
	}

	return prefix + stringutil.Reverse(i)
}

// randString 随机字符串
func randString(letters string, n int) string {
	output := make([]byte, n)

	// We will take n bytes, one byte for each character of output.
	randomness := make([]byte, n)

	// read all random
	_, err := rand.Read(randomness)
	if err != nil {
		panic(err)
	}

	l := len(letters)
	// fill output
	for pos := range output {
		// get random item
		random := randomness[pos]

		// random % 64
		randomPos := random % uint8(l)

		// put into output
		output[pos] = letters[randomPos]
	}

	return string(output)
}

// 生成36位随机字符串
func NewSecretID() string {
	return randString(Alphabet62, 36)
}

// 生成32位随机字符串
func NewSecretKey() string {
	return randString(Alphabet62, 32)
}

// NewTraceID 生成追踪 ID（32位十六进制字符串）
// 格式：trace-{timestamp}-{random}
func NewTraceID() string {
	return randString("0123456789abcdef", 32)
}

// NewSpanID 生成 Span ID（16位十六进制字符串）
func NewSpanID() string {
	return randString("0123456789abcdef", 16)
}

// NewRequestID 生成请求 ID
// 格式：req-{timestamp}-{random}
func NewRequestID() string {
	timestamp := time.Now().UnixNano() / 1000000 // 毫秒时间戳
	randomPart := randString(Alphabet36, 8)
	return fmt.Sprintf("req-%d-%s", timestamp, randomPart)
}

// ValidateIntID 验证 uint64 类型的 ID 是否合法
// 基于 Sonyflake 的特性进行验证
func ValidateIntID(id uint64) bool {
	// 1. ID 不能为 0
	if id == 0 {
		return false
	}

	// 2. Sonyflake 使用 39 位时间戳（从 2014-09-01 00:00:00 开始，单位 10ms）
	// 最大时间戳约为 174 年，即到 2188 年左右
	// Sonyflake 的最大值约为 2^63-1（不使用符号位）
	// 这里检查是否超过了合理范围
	const maxSonyflakeID = uint64(1<<63 - 1)
	if id > maxSonyflakeID {
		return false
	}

	// 3. 检查时间戳部分是否合理
	// Sonyflake 格式: [39位时间戳][8位序列号][16位机器ID]
	// 提取时间戳（右移 24 位）
	timestamp := id >> 24

	// Sonyflake 的起始时间是 2014-09-01 00:00:00 UTC
	sonyflakeEpoch := time.Date(2014, 9, 1, 0, 0, 0, 0, time.UTC)

	// 计算 ID 对应的时间（timestamp * 10ms）
	idTime := sonyflakeEpoch.Add(time.Duration(timestamp) * 10 * time.Millisecond)

	// 4. ID 的时间不能早于 Sonyflake 起始时间
	if idTime.Before(sonyflakeEpoch) {
		return false
	}

	// 5. ID 的时间不能晚于当前时间太多（允许 1 小时的时钟偏差）
	futureThreshold := time.Now().Add(1 * time.Hour)
	if idTime.After(futureThreshold) {
		return false
	}

	return true
}

// ValidateIntIDStrict 严格验证 uint64 类型的 ID
// 只接受由当前系统生成的 ID（时间范围更严格）
func ValidateIntIDStrict(id uint64) bool {
	if !ValidateIntID(id) {
		return false
	}

	// 提取时间戳
	timestamp := id >> 24
	sonyflakeEpoch := time.Date(2014, 9, 1, 0, 0, 0, 0, time.UTC)
	idTime := sonyflakeEpoch.Add(time.Duration(timestamp) * 10 * time.Millisecond)

	// 严格模式：ID 的时间必须在过去 30 天到未来 1 分钟之间
	now := time.Now()
	pastThreshold := now.Add(-30 * 24 * time.Hour)
	futureThreshold := now.Add(1 * time.Minute)

	if idTime.Before(pastThreshold) || idTime.After(futureThreshold) {
		return false
	}

	return true
}

// IsValidIDRange 验证 ID 是否在指定的时间范围内
func IsValidIDRange(id uint64, start, end time.Time) bool {
	if !ValidateIntID(id) {
		return false
	}

	timestamp := id >> 24
	sonyflakeEpoch := time.Date(2014, 9, 1, 0, 0, 0, 0, time.UTC)
	idTime := sonyflakeEpoch.Add(time.Duration(timestamp) * 10 * time.Millisecond)

	return !idTime.Before(start) && !idTime.After(end)
}

// GetIDTimestamp 从 ID 中提取时间戳
// 返回 ID 生成的大致时间
func GetIDTimestamp(id uint64) time.Time {
	timestamp := id >> 24
	sonyflakeEpoch := time.Date(2014, 9, 1, 0, 0, 0, 0, time.UTC)
	return sonyflakeEpoch.Add(time.Duration(timestamp) * 10 * time.Millisecond)
}

// GetIDMachineID 从 ID 中提取机器 ID
func GetIDMachineID(id uint64) uint16 {
	// 机器 ID 占低 16 位
	return uint16(id & 0xFFFF)
}

// GetIDSequence 从 ID 中提取序列号
func GetIDSequence(id uint64) uint8 {
	// 序列号占中间 8 位（第 16-23 位）
	return uint8((id >> 16) & 0xFF)
}
