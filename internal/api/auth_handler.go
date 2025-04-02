package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/TejParker/bigdata-manager/internal/auth"
	"github.com/TejParker/bigdata-manager/internal/db"
	"github.com/TejParker/bigdata-manager/pkg/model"
)

// Login 用户登录处理
func Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证用户名和密码
	userID, err := auth.ValidateUser(req.Username, req.Password)
	if err != nil {
		ResponseError(c, http.StatusUnauthorized, err.Error())
		return
	}

	// 生成JWT令牌
	token, err := auth.GenerateToken(userID, req.Username)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "生成令牌失败")
		return
	}

	// 返回令牌
	ResponseSuccess(c, model.LoginResponse{
		Token:  token,
		UserID: userID,
	})
}

// GetUserInfo 获取当前用户信息
func GetUserInfo(c *gin.Context) {
	userID := c.GetInt("userID")
	username := c.GetString("username")

	var user model.User
	query := "SELECT id, username, email, phone, status, created_at, updated_at FROM user WHERE id = ?"
	err := db.DB.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Phone,
		&user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "获取用户信息失败")
		return
	}

	// 获取用户角色
	var roles []string
	roleQuery := `
		SELECT r.name
		FROM role r
		JOIN user_role ur ON r.id = ur.role_id
		WHERE ur.user_id = ?
	`
	rows, err := db.DB.Query(roleQuery, userID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "获取用户角色失败")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取角色数据失败")
			return
		}
		roles = append(roles, role)
	}

	// 获取用户权限
	privileges, err := auth.GetUserPrivileges(userID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "获取用户权限失败")
		return
	}

	// 返回用户信息
	ResponseSuccess(c, gin.H{
		"user":       user,
		"roles":      roles,
		"privileges": privileges,
	})
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	userID := c.GetInt("userID")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证旧密码
	var (
		username     string
		passwordHash string
	)
	
	query := "SELECT username, password_hash FROM user WHERE id = ?"
	err := db.DB.QueryRow(query, userID).Scan(&username, &passwordHash)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "获取用户信息失败")
		return
	}

	if err := auth.CompareHashAndPassword([]byte(passwordHash), []byte(req.OldPassword)); err != nil {
		ResponseError(c, http.StatusBadRequest, "原密码错误")
		return
	}

	// 生成新密码哈希
	newHashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "密码加密失败")
		return
	}

	// 更新密码
	updateQuery := "UPDATE user SET password_hash = ? WHERE id = ?"
	_, err = db.DB.Exec(updateQuery, newHashedPassword, userID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "更新密码失败")
		return
	}

	ResponseSuccessWithMessage(c, "密码修改成功", nil)
}

// RegisterAuthRoutes 注册认证相关路由
func RegisterAuthRoutes(router *gin.RouterGroup) {
	router.POST("/login", Login)
	
	// 以下路由需要认证
	authRouter := router.Group("/")
	authRouter.Use(JWTAuthMiddleware())
	{
		authRouter.GET("/user/info", GetUserInfo)
		authRouter.POST("/user/change-password", ChangePassword)
	}
} 