package api

import (
	"net/http"
	"strconv"

	"github.com/TejParker/bigdata-manager/internal/deploy"
	"github.com/TejParker/bigdata-manager/pkg/model"
	
)

// RegisterComponent 注册组件
func RegisterComponent(c *gin.Context) {
	var component model.Component
	if err := c.ShouldBindJSON(&component); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证参数
	if component.Name == "" {
		ResponseError(c, http.StatusBadRequest, "组件名称不能为空")
		return
	}

	if component.Version == "" {
		ResponseError(c, http.StatusBadRequest, "组件版本不能为空")
		return
	}

	if component.PackageURL == "" {
		ResponseError(c, http.StatusBadRequest, "组件包地址不能为空")
		return
	}

	// 注册组件
	deployService := deploy.GetDeployService()
	deployService.RegisterComponent(component)

	// 返回结果
	ResponseSuccess(c, component)
}

// GetComponents 获取所有组件
func GetComponents(c *gin.Context) {
	deployService := deploy.GetDeployService()
	components := deployService.GetAllComponents()
	ResponseSuccess(c, components)
}

// DeployComponent 部署组件
func DeployComponent(c *gin.Context) {
	var req struct {
		HostID      int `json:"host_id" binding:"required"`
		ComponentID int `json:"component_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 部署组件
	deployService := deploy.GetDeployService()
	deploymentID, err := deployService.Deploy(req.HostID, req.ComponentID)
	if err != nil {
		if err == deploy.ErrComponentNotFound {
			ResponseError(c, http.StatusNotFound, "组件不存在")
		} else {
			ResponseError(c, http.StatusInternalServerError, "部署组件失败: "+err.Error())
		}
		return
	}

	// 返回结果
	ResponseSuccess(c, gin.H{
		"deployment_id": deploymentID,
		"message":       "组件部署任务已提交",
	})
}

// GetDeployments 获取主机的部署记录
func GetDeployments(c *gin.Context) {
	hostIDStr := c.DefaultQuery("host_id", "0")
	hostID, err := strconv.Atoi(hostIDStr)
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	// 获取部署记录
	deployService := deploy.GetDeployService()
	deployments := deployService.GetDeployments(hostID)

	// 返回结果
	ResponseSuccess(c, deployments)
}

// StartComponent 启动组件
func StartComponent(c *gin.Context) {
	var req struct {
		HostID      int `json:"host_id" binding:"required"`
		ComponentID int `json:"component_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 启动组件
	deployService := deploy.GetDeployService()
	commandID, err := deployService.StartComponent(req.HostID, req.ComponentID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "启动组件失败: "+err.Error())
		return
	}

	// 返回结果
	ResponseSuccess(c, gin.H{
		"command_id": commandID,
		"message":    "组件启动命令已发送",
	})
}

// StopComponent 停止组件
func StopComponent(c *gin.Context) {
	var req struct {
		HostID      int `json:"host_id" binding:"required"`
		ComponentID int `json:"component_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 停止组件
	deployService := deploy.GetDeployService()
	commandID, err := deployService.StopComponent(req.HostID, req.ComponentID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "停止组件失败: "+err.Error())
		return
	}

	// 返回结果
	ResponseSuccess(c, gin.H{
		"command_id": commandID,
		"message":    "组件停止命令已发送",
	})
}

// ConfigureComponent 配置组件
func ConfigureComponent(c *gin.Context) {
	var req struct {
		HostID      int                    `json:"host_id" binding:"required"`
		ComponentID int                    `json:"component_id" binding:"required"`
		Config      map[string]interface{} `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 配置组件
	deployService := deploy.GetDeployService()
	commandID, err := deployService.ConfigureComponent(req.HostID, req.ComponentID, req.Config)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "配置组件失败: "+err.Error())
		return
	}

	// 返回结果
	ResponseSuccess(c, gin.H{
		"command_id": commandID,
		"message":    "组件配置命令已发送",
	})
}

// RegisterDeployRoutes 注册部署相关路由
func RegisterDeployRoutes(router *gin.RouterGroup) {
	router.POST("/components", RegisterComponent)
	router.GET("/components", GetComponents)
	router.POST("/deployments", DeployComponent)
	router.GET("/deployments", GetDeployments)
	router.POST("/components/start", StartComponent)
	router.POST("/components/stop", StopComponent)
	router.POST("/components/configure", ConfigureComponent)
}
