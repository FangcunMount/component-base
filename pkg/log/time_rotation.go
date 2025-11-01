/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TimeRotationWriter 按时间轮转的日志写入器
type TimeRotationWriter struct {
	filename    string      // 基础文件名（不含日期后缀）
	timeFormat  string      // 时间格式，如 "2006-01-02"
	currentDate string      // 当前日期
	currentFile *os.File    // 当前打开的文件
	mu          sync.Mutex  // 保护并发写入
	maxAge      int         // 保留文件的最大天数
	compress    bool        // 是否压缩旧文件
	fileMode    os.FileMode // 文件权限
}

// NewTimeRotationWriter 创建一个按时间轮转的写入器
func NewTimeRotationWriter(filename, timeFormat string, maxAge int, compress bool) *TimeRotationWriter {
	return &TimeRotationWriter{
		filename:   filename,
		timeFormat: timeFormat,
		maxAge:     maxAge,
		compress:   compress,
		fileMode:   0644,
	}
}

// Write 实现 io.Writer 接口
func (w *TimeRotationWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查是否需要轮转
	currentDate := time.Now().Format(w.timeFormat)
	if w.currentFile == nil || w.currentDate != currentDate {
		if err := w.rotate(currentDate); err != nil {
			return 0, err
		}
	}

	return w.currentFile.Write(p)
}

// Sync 实现 zapcore.WriteSyncer 接口
func (w *TimeRotationWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile != nil {
		return w.currentFile.Sync()
	}
	return nil
}

// Close 关闭当前文件
func (w *TimeRotationWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}

// rotate 执行日志轮转
func (w *TimeRotationWriter) rotate(newDate string) error {
	// 关闭当前文件
	if w.currentFile != nil {
		if err := w.currentFile.Close(); err != nil {
			return err
		}
	}

	// 创建新文件名
	dir := filepath.Dir(w.filename)
	ext := filepath.Ext(w.filename)
	base := w.filename[:len(w.filename)-len(ext)]

	newFilename := fmt.Sprintf("%s.%s%s", base, newDate, ext)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 打开新文件
	file, err := os.OpenFile(newFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, w.fileMode)
	if err != nil {
		return err
	}

	w.currentFile = file
	w.currentDate = newDate

	// 清理旧文件
	go w.cleanOldFiles()

	return nil
}

// cleanOldFiles 清理过期的日志文件
func (w *TimeRotationWriter) cleanOldFiles() {
	if w.maxAge <= 0 {
		return
	}

	dir := filepath.Dir(w.filename)
	ext := filepath.Ext(w.filename)
	base := filepath.Base(w.filename[:len(w.filename)-len(ext)])

	cutoff := time.Now().AddDate(0, 0, -w.maxAge)

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// 检查文件名是否匹配模式
		filename := filepath.Base(path)
		if len(filename) < len(base) || filename[:len(base)] != base {
			return nil
		}

		// 检查文件是否过期
		if info.ModTime().Before(cutoff) {
			os.Remove(path)
		}

		return nil
	})
}

// getCurrentFilename 获取当前日志文件名
func (w *TimeRotationWriter) getCurrentFilename() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile != nil {
		return w.currentFile.Name()
	}

	currentDate := time.Now().Format(w.timeFormat)
	ext := filepath.Ext(w.filename)
	base := w.filename[:len(w.filename)-len(ext)]

	return fmt.Sprintf("%s.%s%s", base, currentDate, ext)
}
