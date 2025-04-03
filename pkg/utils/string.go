package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
)

// MD5 计算字符串的MD5哈希值
func MD5(text string) string {
	hash := md5.New()
	io.WriteString(hash, text)
	return hex.EncodeToString(hash.Sum(nil))
}

// SHA1 计算字符串的SHA1哈希值
func SHA1(text string) string {
	hash := sha1.New()
	io.WriteString(hash, text)
	return hex.EncodeToString(hash.Sum(nil))
}

// SHA256 计算字符串的SHA256哈希值
func SHA256(text string) string {
	hash := sha256.New()
	io.WriteString(hash, text)
	return hex.EncodeToString(hash.Sum(nil))
}

// IsEmailValid 验证邮箱地址是否有效
func IsEmailValid(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

// IsIPv4Valid 验证IPv4地址是否有效
func IsIPv4Valid(ip string) bool {
	pattern := `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	match, _ := regexp.MatchString(pattern, ip)
	return match
}

// IsURLValid 验证URL是否有效
func IsURLValid(url string) bool {
	pattern := `^(http|https)://[a-zA-Z0-9]+([\-\.]{1}[a-zA-Z0-9]+)*\.[a-zA-Z]{2,}(:[0-9]{1,5})?(\/.*)?$`
	match, _ := regexp.MatchString(pattern, url)
	return match
}

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成随机字符串失败: %v", err)
	}
	for i, b := range bytes {
		bytes[i] = chars[b%byte(len(chars))]
	}
	return string(bytes), nil
}

// TruncateString 截断字符串并添加省略号
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// RemoveExtraSpaces 移除字符串中的多余空格
func RemoveExtraSpaces(s string) string {
	// 将连续的空白字符替换为单个空格
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(s, " "))
}

// CamelCaseToSnakeCase 驼峰命名转下划线命名
func CamelCaseToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// SnakeCaseToCamelCase 下划线命名转驼峰命名
func SnakeCaseToCamelCase(s string) string {
	var result strings.Builder
	upper := false
	for _, r := range s {
		if r == '_' {
			upper = true
		} else {
			if upper {
				result.WriteRune(unicode.ToUpper(r))
				upper = false
			} else {
				result.WriteRune(r)
			}
		}
	}
	return result.String()
}

// SnakeCaseToPascalCase 下划线命名转帕斯卡命名（首字母大写的驼峰命名）
func SnakeCaseToPascalCase(s string) string {
	camel := SnakeCaseToCamelCase(s)
	if len(camel) == 0 {
		return ""
	}
	return strings.ToUpper(camel[:1]) + camel[1:]
}

// MaskEmail 对邮箱地址进行部分掩码处理
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	domain := parts[1]

	if len(username) <= 3 {
		return username[:1] + strings.Repeat("*", len(username)-1) + "@" + domain
	}

	return username[:2] + strings.Repeat("*", len(username)-3) + username[len(username)-1:] + "@" + domain
}

// MaskPhone 对手机号进行部分掩码处理
func MaskPhone(phone string) string {
	// 去除可能存在的非数字字符
	re := regexp.MustCompile(`\D`)
	cleanPhone := re.ReplaceAllString(phone, "")

	if len(cleanPhone) < 7 {
		return phone // 太短，不处理
	}

	// 保留前3位和后4位
	return cleanPhone[:3] + strings.Repeat("*", len(cleanPhone)-7) + cleanPhone[len(cleanPhone)-4:]
}
