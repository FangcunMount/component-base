// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

/*
Package idutil 提供各种 ID 生成和验证工具

# ID 验证

## uint64 类型 ID 验证

基本验证（推荐用于外部输入）：

	valid := idutil.ValidateIntID(id)
	if !valid {
		return errors.New("invalid ID")
	}

验证规则：
  - ID 不能为 0
  - ID 不能超过最大值（2^63-1）
  - 时间戳不能早于 2014-09-01（Sonyflake 起始时间）
  - 时间戳不能晚于当前时间 1 小时（考虑时钟偏差）

严格验证（用于系统内部生成的 ID）：

	valid := idutil.ValidateIntIDStrict(id)

严格规则：
  - 包含基本验证的所有规则
  - ID 时间必须在过去 30 天到未来 1 分钟之间

时间范围验证：

	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()
	valid := idutil.IsValidIDRange(id, start, end)

# ID 解析

从 uint64 ID 中提取信息：

	// 提取生成时间
	timestamp := idutil.GetIDTimestamp(id)

	// 提取机器 ID
	machineID := idutil.GetIDMachineID(id)

	// 提取序列号
	sequence := idutil.GetIDSequence(id)

# 使用示例

验证用户提交的 ID：

	func GetUser(idStr string) (*User, error) {
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			return nil, errors.New("invalid ID format")
		}

		if !idutil.ValidateIntID(id) {
			return nil, errors.New("invalid ID")
		}

		// 查询数据库...
	}
*/
package idutil
