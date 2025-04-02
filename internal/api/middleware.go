package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/TejParker/bigdata-manager/internal/auth"
	"github.com/TejParker/bigdata-manager/pkg/model"
)

// JWTAuthMiddleware JWT认证中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			ResponseError(c, http.StatusUnauthorized, "未提供认证令牌")
			c.Abort()
			return
		}

		// 验证token格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			ResponseError(c, http.StatusUnauthorized, "无效的认证格式")
			c.Abort()
			return
		}

		// 解析token
		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			ResponseError(c, http.StatusUnauthorized, "无效或过期的令牌")
			c.Abort()
			return
		}

		// 将用户信息保存到请求上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// PrivilegeMiddleware 权限检查中间件
func PrivilegeMiddleware(requiredPrivilege string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			ResponseError(c, http.StatusUnauthorized, "未认证的用户")
			c.Abort()
			return
		}

		// 检查用户是否有权限
		hasPrivilege, err := auth.CheckUserPrivilege(userID.(int), requiredPrivilege)
		if err != nil {
			ResponseError(c, http.StatusInternalServerError, "权限检查失败")
			c.Abort()
			return
		}

		if !hasPrivilege {
			ResponseError(c, http.StatusForbidden, "无权执行此操作")
			c.Abort()
			return
		}

		c.Next()
	}
}

// CORSMiddleware 跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ResponseError 返回错误响应
func ResponseError(c *gin.Context, code int, message string) {
	c.JSON(code, model.Response{
		Success: false,
		Message: message,
	})
}

// ResponseSuccess 返回成功响应
func ResponseSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "操作成功",
		Data:    data,
	})
}

// ResponseSuccessWithMessage 返回带自定义消息的成功响应
func ResponseSuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// ResponsePageSuccess 返回分页成功响应
func ResponsePageSuccess(c *gin.Context, data interface{}, total, page, pageSize int) {
	c.JSON(http.StatusOK, model.PageResponse{
		Success:  true,
		Message:  "操作成功",
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
} 