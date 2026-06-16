package utils

import (
	"encoding/json"
	"math/rand"
	"strings"
)

// GenerateUUID 生成简单UUID（v4格式）
func GenerateUUID() string {
	hex := "0123456789abcdef"
	b := make([]byte, 36)
	for i := range b {
		switch i {
		case 8, 13, 18, 23:
			b[i] = '-'
		case 14:
			b[i] = '4'
		case 19:
			b[i] = hex[rand.Intn(4)+8]
		default:
			b[i] = hex[rand.Intn(16)]
		}
	}
	return string(b)
}

// ToJSON 将任意对象转为JSON字符串
func ToJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ContainsString 判断字符串列表是否包含目标字符串
func ContainsString(slice []string, target string) bool {
	for _, s := range slice {
		if strings.Contains(strings.ToUpper(s), strings.ToUpper(target)) {
			return true
		}
	}
	return false
}
