package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/TejParker/bigdata-manager/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// Claims JWT声明结构
type Claims struct {
	UserID int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// ValidateUser 验证用户名和密码
func ValidateUser(username, password string) (int, error) {
	var (
		userID       int
		passwordHash string
	)

	query := "SELECT id, password_hash FROM user WHERE username = ? AND status = 'ACTIVE'"
	err := db.DB.QueryRow(query, username).Scan(&userID, &passwordHash)
	if err != nil {
		return 0, fmt.Errorf("用户不存在或已禁用: %v", err)
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return 0, errors.New("密码错误")
	}

	return userID, nil
}

// GenerateToken 生成JWT令牌
func GenerateToken(userID int, username string) (string, error) {
	// 获取配置
	secret := viper.GetString("server.jwt_secret")
	expHours := viper.GetInt("server.jwt_expiration")

	// 创建令牌声明
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "bigdata-manager",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	// 生成令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseToken 解析JWT令牌
func ParseToken(tokenString string) (*Claims, error) {
	secret := viper.GetString("server.jwt_secret")

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的令牌")
}

// HashPassword 密码加密
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CompareHashAndPassword 比较密码哈希
func CompareHashAndPassword(hashedPassword, password []byte) error {
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}

// GetUserPrivileges 获取用户权限列表
func GetUserPrivileges(userID int) ([]string, error) {
	query := `
		SELECT DISTINCT p.name
		FROM privilege p
		JOIN role_privilege rp ON p.id = rp.privilege_id
		JOIN user_role ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ?
	`

	rows, err := db.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var privileges []string
	for rows.Next() {
		var privilege string
		if err := rows.Scan(&privilege); err != nil {
			return nil, err
		}
		privileges = append(privileges, privilege)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return privileges, nil
}

// CheckUserPrivilege 检查用户是否拥有特定权限
func CheckUserPrivilege(userID int, requiredPrivilege string) (bool, error) {
	privileges, err := GetUserPrivileges(userID)
	if err != nil {
		return false, err
	}

	for _, privilege := range privileges {
		if privilege == requiredPrivilege {
			return true, nil
		}
	}

	return false, nil
} 