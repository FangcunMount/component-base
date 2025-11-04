// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("ID 验证示例")
	fmt.Println("========================================")
	fmt.Println()

	// 1. 生成合法的 ID
	fmt.Println("1️⃣  生成和验证合法 ID")
	fmt.Println("========================================")
	validID := idutil.GetIntID()
	fmt.Printf("生成的 ID: %d\n", validID)
	fmt.Printf("基本验证: %v\n", idutil.ValidateIntID(validID))
	fmt.Printf("严格验证: %v\n", idutil.ValidateIntIDStrict(validID))
	fmt.Println()

	// 2. 解析 ID 信息
	fmt.Println("2️⃣  解析 ID 信息")
	fmt.Println("========================================")
	timestamp := idutil.GetIDTimestamp(validID)
	machineID := idutil.GetIDMachineID(validID)
	sequence := idutil.GetIDSequence(validID)

	fmt.Printf("ID: %d\n", validID)
	fmt.Printf("生成时间: %s\n", timestamp.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("机器 ID: %d (0x%04X)\n", machineID, machineID)
	fmt.Printf("序列号: %d\n", sequence)
	fmt.Printf("距离现在: %v\n", time.Since(timestamp))
	fmt.Println()

	// 3. 测试无效的 ID
	fmt.Println("3️⃣  测试无效 ID")
	fmt.Println("========================================")

	invalidIDs := map[string]uint64{
		"零值 ID":   0,
		"超大值 ID":  uint64(1<<63 + 1000),
		"未来时间 ID": uint64(1) << 50, // 远未来的时间戳
		"过去时间 ID": uint64(1) << 20, // 2014年之前
		"随机无效 ID": 12345,           // 时间戳太旧
	}

	for name, id := range invalidIDs {
		valid := idutil.ValidateIntID(id)
		fmt.Printf("%-12s ID=%20d 验证结果=%v\n", name, id, valid)
		if id > 0 && id < uint64(1<<63) {
			ts := idutil.GetIDTimestamp(id)
			fmt.Printf("             解析时间: %s\n", ts.Format("2006-01-02 15:04:05"))
		}
	}
	fmt.Println()

	// 4. 时间范围验证
	fmt.Println("4️⃣  时间范围验证")
	fmt.Println("========================================")

	// 生成一个旧 ID
	time.Sleep(100 * time.Millisecond)
	oldID := idutil.GetIntID()

	time.Sleep(100 * time.Millisecond)
	newID := idutil.GetIntID()

	// 验证最近 1 秒内的 ID
	start := time.Now().Add(-1 * time.Second)
	end := time.Now()

	fmt.Printf("时间范围: %s 到 %s\n", start.Format("15:04:05.000"), end.Format("15:04:05.000"))
	fmt.Printf("旧 ID (%d) 在范围内: %v\n", oldID, idutil.IsValidIDRange(oldID, start, end))
	fmt.Printf("新 ID (%d) 在范围内: %v\n", newID, idutil.IsValidIDRange(newID, start, end))

	// 验证过去 7 天的 ID
	start7Days := time.Now().Add(-7 * 24 * time.Hour)
	endNow := time.Now()
	fmt.Printf("\n最近7天范围内: %v\n", idutil.IsValidIDRange(newID, start7Days, endNow))
	fmt.Println()

	// 5. 批量 ID 验证
	fmt.Println("5️⃣  批量 ID 验证")
	fmt.Println("========================================")

	// 生成一批 ID
	var ids []uint64
	for i := 0; i < 5; i++ {
		ids = append(ids, idutil.GetIntID())
		time.Sleep(1 * time.Millisecond)
	}

	// 验证所有 ID
	validCount := 0
	for i, id := range ids {
		valid := idutil.ValidateIntID(id)
		if valid {
			validCount++
		}
		ts := idutil.GetIDTimestamp(id)
		fmt.Printf("ID #%d: %d | 有效=%v | 时间=%s\n",
			i+1, id, valid, ts.Format("15:04:05.000"))
	}
	fmt.Printf("\n有效 ID 数量: %d/%d\n", validCount, len(ids))
	fmt.Println()

	// 6. 实际应用场景
	fmt.Println("6️⃣  实际应用场景示例")
	fmt.Println("========================================")

	// 模拟从 HTTP 请求中获取用户 ID
	userIDFromRequest := validID
	fmt.Printf("接收到用户 ID: %d\n", userIDFromRequest)

	// 验证 ID
	if !idutil.ValidateIntID(userIDFromRequest) {
		fmt.Println("❌ 错误: 无效的用户 ID")
	} else {
		fmt.Println("✅ ID 验证通过")

		// 检查 ID 的生成时间
		idTime := idutil.GetIDTimestamp(userIDFromRequest)
		age := time.Since(idTime)

		fmt.Printf("   ID 生成于: %s (距今 %v)\n", idTime.Format("2006-01-02 15:04:05"), age)

		// 检查是否是最近生成的
		if age < 30*time.Minute {
			fmt.Println("   ℹ️  这是最近生成的 ID")
		} else if age < 24*time.Hour {
			fmt.Println("   ℹ️  这是今天生成的 ID")
		} else {
			fmt.Printf("   ℹ️  这是 %d 天前生成的 ID\n", int(age.Hours()/24))
		}

		// 严格验证（用于敏感操作）
		if idutil.ValidateIntIDStrict(userIDFromRequest) {
			fmt.Println("   ✅ 严格验证通过（适合敏感操作）")
		} else {
			fmt.Println("   ⚠️  严格验证失败（ID 可能太旧）")
		}
	}
	fmt.Println()

	// 7. 性能测试
	fmt.Println("7️⃣  性能测试")
	fmt.Println("========================================")

	count := 100000
	startTime := time.Now()
	for i := 0; i < count; i++ {
		idutil.ValidateIntID(validID)
	}
	duration := time.Since(startTime)

	fmt.Printf("验证 %d 次 ID 耗时: %v\n", count, duration)
	fmt.Printf("平均每次验证: %v\n", duration/time.Duration(count))
	fmt.Printf("每秒可验证: %.0f 次\n", float64(count)/duration.Seconds())
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("示例完成！")
	fmt.Println("========================================")
}
