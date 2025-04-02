package api

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// SetupRouter 设置API路由
func SetupRouter() *gin.Engine {
	// 创建路由
	r := gin.Default()
	
	// 应用中间件
	r.Use(CORSMiddleware())
	
	// API前缀
	apiPrefix := viper.GetString("server.api_prefix")
	apiGroup := r.Group(apiPrefix)
	
	// 注册各模块路由
	RegisterAuthRoutes(apiGroup)
	RegisterClusterRoutes(apiGroup)
	RegisterHostRoutes(apiGroup)
	RegisterServiceRoutes(apiGroup)
	RegisterMonitorRoutes(apiGroup)
	RegisterLogRoutes(apiGroup)
	
	return r
} 