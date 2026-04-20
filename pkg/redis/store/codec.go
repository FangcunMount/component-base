package store

import (
	"encoding/json"
	"fmt"
)

// ValueCodec 负责 typed value 的编解码。
type ValueCodec[T any] interface {
	Marshal(value T) ([]byte, error)
	Unmarshal(payload []byte) (T, error)
	Name() string
}

// ValueEnvelope 描述 typed store 的载荷与写入元数据。
type ValueEnvelope struct {
	Payload    []byte
	TTL        int64
	Encoding   string
	Compressed bool
}

// JSONCodec 为任意值提供 JSON 编解码。
type JSONCodec[T any] struct{}

func (JSONCodec[T]) Marshal(value T) ([]byte, error) {
	return json.Marshal(value)
}

func (JSONCodec[T]) Unmarshal(payload []byte) (T, error) {
	var value T
	if len(payload) == 0 {
		return value, fmt.Errorf("json payload is empty")
	}
	err := json.Unmarshal(payload, &value)
	return value, err
}

func (JSONCodec[T]) Name() string {
	return "json"
}

// StringCodec 用于存储原始字符串。
type StringCodec struct{}

func (StringCodec) Marshal(value string) ([]byte, error) {
	return []byte(value), nil
}

func (StringCodec) Unmarshal(payload []byte) (string, error) {
	return string(payload), nil
}

func (StringCodec) Name() string {
	return "string"
}

// BytesCodec 用于存储原始字节切片。
type BytesCodec struct{}

func (BytesCodec) Marshal(value []byte) ([]byte, error) {
	return append([]byte(nil), value...), nil
}

func (BytesCodec) Unmarshal(payload []byte) ([]byte, error) {
	return append([]byte(nil), payload...), nil
}

func (BytesCodec) Name() string {
	return "bytes"
}
