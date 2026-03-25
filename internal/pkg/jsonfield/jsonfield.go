package jsonfield

import (
	"encoding/json"
	"fmt"
)

// Unmarshal 解析 JSON 字符串字段为指定类型，空字符串返回错误
func Unmarshal[T any](raw string, typeName string) (*T, error) {
	if raw == "" {
		return nil, fmt.Errorf("%s为空", typeName)
	}
	var v T
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, fmt.Errorf("解析%s失败: %w", typeName, err)
	}
	return &v, nil
}

// UnmarshalOrDefault 解析 JSON 字符串字段，空字符串返回零值
func UnmarshalOrDefault[T any](raw string, typeName string) (*T, error) {
	if raw == "" {
		return new(T), nil
	}
	var v T
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, fmt.Errorf("解析%s失败: %w", typeName, err)
	}
	return &v, nil
}

// Marshal 序列化值为 JSON 字符串
func Marshal[T any](v *T, typeName string) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("序列化%s失败: %w", typeName, err)
	}
	return string(data), nil
}

// MarshalOrClear 序列化值为 JSON 字符串，当 isEmpty 返回 true 时返回空字符串
func MarshalOrClear[T any](v *T, isEmpty func(*T) bool, typeName string) (string, error) {
	if v == nil || isEmpty(v) {
		return "", nil
	}
	return Marshal(v, typeName)
}
