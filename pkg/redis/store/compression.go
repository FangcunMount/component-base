package store

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// CompressionCodec 使用 gzip 为基础 codec 增加压缩能力。
type CompressionCodec[T any] struct {
	base ValueCodec[T]
}

// NewCompressionCodec 使用 gzip 包装一个基础 codec。
func NewCompressionCodec[T any](base ValueCodec[T]) CompressionCodec[T] {
	return CompressionCodec[T]{base: base}
}

func (c CompressionCodec[T]) Marshal(value T) ([]byte, error) {
	if c.base == nil {
		return nil, fmt.Errorf("base codec is nil")
	}
	payload, err := c.base.Marshal(value)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c CompressionCodec[T]) Unmarshal(payload []byte) (T, error) {
	var zero T
	if c.base == nil {
		return zero, fmt.Errorf("base codec is nil")
	}

	zr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return zero, err
	}
	defer func() { _ = zr.Close() }()

	decoded, err := io.ReadAll(zr)
	if err != nil {
		return zero, err
	}
	return c.base.Unmarshal(decoded)
}

func (c CompressionCodec[T]) Name() string {
	if c.base == nil {
		return "gzip"
	}
	return "gzip+" + c.base.Name()
}

// Compressed 返回当前 codec 是否输出压缩载荷。
func (CompressionCodec[T]) Compressed() bool {
	return true
}
