package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/TejParker/bigdata-manager/internal/db"
	"github.com/TejParker/bigdata-manager/pkg/model"
)

// CreateService 创建新服务
func CreateService(c *gin.Context) {
	var service model.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 检查集群是否存在
	var clusterExists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM cluster WHERE id = ?)"
	err := db.DB.QueryRow(existsQuery, service.ClusterID).Scan(&clusterExists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询集群失败")
		return
	}
	if !clusterExists {
		ResponseError(c, http.StatusBadRequest, "指定的集群不存在")
		return
	}

	// 检查服务名是否已存在
	var serviceExists bool
	serviceExistsQuery := "SELECT EXISTS(SELECT 1 FROM service WHERE service_name = ? AND cluster_id = ?)"
	err = db.DB.QueryRow(serviceExistsQuery, service.ServiceName, service.ClusterID).Scan(&serviceExists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "检查服务名失败")
		return
	}
	if serviceExists {
		ResponseError(c, http.StatusBadRequest, "集群中已存在同名服务")
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "开始事务失败")
		return
	}

	// 插入服务记录
	query := `INSERT INTO service 
		(cluster_id, service_type, service_name, version, status) 
		VALUES (?, ?, ?, ?, ?)`
	result, err := tx.Exec(query, 
		service.ClusterID, 
		service.ServiceType, 
		service.ServiceName, 
		service.Version,
		"INSTALLING") // 初始状态为安装中
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "创建服务失败: "+err.Error())
		return
	}

	// 获取新服务ID
	serviceID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "获取服务ID失败")
		return
	}

	// 创建任务记录
	taskQuery := `INSERT INTO task 
		(task_type, related_id, related_type, status)
		VALUES (?, ?, ?, ?)`
	_, err = tx.Exec(taskQuery, 
		"INSTALL_SERVICE", 
		serviceID, 
		"SERVICE",
		"PENDING")
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "创建任务失败: "+err.Error())
		return
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	// 返回成功响应
	service.ID = int(serviceID)
	service.Status = "INSTALLING"
	ResponseSuccessWithMessage(c, "服务创建成功", service)
}

// GetServices 获取服务列表
func GetServices(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// 过滤参数
	clusterID := c.Query("cluster_id")
	serviceType := c.Query("service_type")
	status := c.Query("status")

	// 构建查询条件
	whereClause := ""
	args := []interface{}{}
	
	if clusterID != "" {
		whereClause += " WHERE s.cluster_id = ?"
		args = append(args, clusterID)
		
		if serviceType != "" {
			whereClause += " AND s.service_type = ?"
			args = append(args, serviceType)
		}
		
		if status != "" {
			whereClause += " AND s.status = ?"
			args = append(args, status)
		}
	} else {
		if serviceType != "" {
			whereClause += " WHERE s.service_type = ?"
			args = append(args, serviceType)
			
			if status != "" {
				whereClause += " AND s.status = ?"
				args = append(args, status)
			}
		} else if status != "" {
			whereClause += " WHERE s.status = ?"
			args = append(args, status)
		}
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM service s" + whereClause
	var total int
	err := db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务总数失败")
		return
	}

	// 分页参数
	query := `SELECT s.id, s.service_type, s.service_name, s.version, s.status,
		s.cluster_id, c.name as cluster_name, s.created_at, s.updated_at
		FROM service s
		LEFT JOIN cluster c ON s.cluster_id = c.id` + 
		whereClause + 
		" ORDER BY s.id DESC LIMIT ? OFFSET ?"
	
	args = append(args, pageSize, offset)
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务列表失败")
		return
	}
	defer rows.Close()

	// 构造结果集
	type ServiceWithCluster struct {
		model.Service
		ClusterName string `json:"cluster_name"`
	}
	
	var services []ServiceWithCluster
	for rows.Next() {
		var service ServiceWithCluster
		
		if err := rows.Scan(
			&service.ID, &service.ServiceType, &service.ServiceName, &service.Version, &service.Status,
			&service.ClusterID, &service.ClusterName, &service.CreatedAt, &service.UpdatedAt); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取服务数据失败")
			return
		}
		
		services = append(services, service)
	}

	if err = rows.Err(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "处理服务数据失败")
		return
	}

	// 返回分页结果
	ResponsePageSuccess(c, services, total, page, pageSize)
}

// GetServiceById 根据ID获取服务详情
func GetServiceById(c *gin.Context) {
	serviceID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的服务ID")
		return
	}

	// 查询服务基本信息
	query := `SELECT s.id, s.service_type, s.service_name, s.version, s.status,
		s.cluster_id, c.name as cluster_name, s.created_at, s.updated_at
		FROM service s
		LEFT JOIN cluster c ON s.cluster_id = c.id
		WHERE s.id = ?`
	
	type ServiceWithCluster struct {
		model.Service
		ClusterName string `json:"cluster_name"`
	}
	
	var service ServiceWithCluster
	err = db.DB.QueryRow(query, serviceID).Scan(
		&service.ID, &service.ServiceType, &service.ServiceName, &service.Version, &service.Status,
		&service.ClusterID, &service.ClusterName, &service.CreatedAt, &service.UpdatedAt)
	
	if err == sql.ErrNoRows {
		ResponseError(c, http.StatusNotFound, "服务不存在")
		return
	}
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务信息失败")
		return
	}

	// 查询服务组件
	componentsQuery := `SELECT id, service_id, component_type, desired_instances, created_at, updated_at
		FROM service_component
		WHERE service_id = ?`
	
	componentRows, err := db.DB.Query(componentsQuery, serviceID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务组件失败")
		return
	}
	defer componentRows.Close()
	
	var components []model.ServiceComponent
	for componentRows.Next() {
		var component model.ServiceComponent
		if err := componentRows.Scan(
			&component.ID, &component.ServiceID, &component.ComponentType, 
			&component.DesiredInstances, &component.CreatedAt, &component.UpdatedAt); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取组件数据失败")
			return
		}
		components = append(components, component)
	}

	// 查询服务配置
	configQuery := `SELECT id, scope_type, scope_id, config_key, config_value, version, is_current, created_at, updated_at
		FROM config
		WHERE scope_type = 'SERVICE' AND scope_id = ? AND is_current = true`
	
	configRows, err := db.DB.Query(configQuery, serviceID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务配置失败")
		return
	}
	defer configRows.Close()
	
	var configs []model.Config
	for configRows.Next() {
		var config model.Config
		var isCurrent bool
		if err := configRows.Scan(
			&config.ID, &config.ScopeType, &config.ScopeID, &config.ConfigKey, 
			&config.ConfigValue, &config.Version, &isCurrent, &config.CreatedAt, &config.UpdatedAt); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取配置数据失败")
			return
		}
		config.IsCurrent = isCurrent
		configs = append(configs, config)
	}

	// 返回结果
	ResponseSuccess(c, gin.H{
		"service":    service,
		"components": components,
		"configs":    configs,
	})
}

// UpdateService 更新服务信息
func UpdateService(c *gin.Context) {
	serviceID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的服务ID")
		return
	}

	var service model.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 检查服务是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM service WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, serviceID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "服务不存在")
		return
	}

	// 更新服务信息
	updateQuery := `UPDATE service SET 
		service_name = ?,
		version = ?
		WHERE id = ?`
	_, err = db.DB.Exec(updateQuery, 
		service.ServiceName,
		service.Version,
		serviceID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "更新服务失败: "+err.Error())
		return
	}

	ResponseSuccessWithMessage(c, "服务更新成功", nil)
}

// StartService 启动服务
func StartService(c *gin.Context) {
	serviceID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的服务ID")
		return
	}

	// 检查服务是否存在
	var status string
	statusQuery := "SELECT status FROM service WHERE id = ?"
	err = db.DB.QueryRow(statusQuery, serviceID).Scan(&status)
	if err == sql.ErrNoRows {
		ResponseError(c, http.StatusNotFound, "服务不存在")
		return
	}
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务状态失败")
		return
	}

	if status != "STOPPED" {
		ResponseError(c, http.StatusBadRequest, "只有已停止的服务才能启动")
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "开始事务失败")
		return
	}

	// 更新服务状态
	_, err = tx.Exec("UPDATE service SET status = 'RUNNING' WHERE id = ?", serviceID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "更新服务状态失败")
		return
	}

	// 创建启动任务
	_, err = tx.Exec(
		"INSERT INTO task (task_type, related_id, related_type, status) VALUES (?, ?, ?, ?)",
		"START_SERVICE", serviceID, "SERVICE", "PENDING")
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "创建启动任务失败")
		return
	}

	if err = tx.Commit(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	ResponseSuccessWithMessage(c, "服务启动命令已发送", nil)
}

// StopService 停止服务
func StopService(c *gin.Context) {
	serviceID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的服务ID")
		return
	}

	// 检查服务是否存在
	var status string
	statusQuery := "SELECT status FROM service WHERE id = ?"
	err = db.DB.QueryRow(statusQuery, serviceID).Scan(&status)
	if err == sql.ErrNoRows {
		ResponseError(c, http.StatusNotFound, "服务不存在")
		return
	}
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务状态失败")
		return
	}

	if status != "RUNNING" {
		ResponseError(c, http.StatusBadRequest, "只有运行中的服务才能停止")
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "开始事务失败")
		return
	}

	// 更新服务状态
	_, err = tx.Exec("UPDATE service SET status = 'STOPPED' WHERE id = ?", serviceID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "更新服务状态失败")
		return
	}

	// 创建停止任务
	_, err = tx.Exec(
		"INSERT INTO task (task_type, related_id, related_type, status) VALUES (?, ?, ?, ?)",
		"STOP_SERVICE", serviceID, "SERVICE", "PENDING")
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "创建停止任务失败")
		return
	}

	if err = tx.Commit(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	ResponseSuccessWithMessage(c, "服务停止命令已发送", nil)
}

// DeleteService 删除服务
func DeleteService(c *gin.Context) {
	serviceID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的服务ID")
		return
	}

	// 检查服务是否存在
	var status string
	statusQuery := "SELECT status FROM service WHERE id = ?"
	err = db.DB.QueryRow(statusQuery, serviceID).Scan(&status)
	if err == sql.ErrNoRows {
		ResponseError(c, http.StatusNotFound, "服务不存在")
		return
	}
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务状态失败")
		return
	}

	if status == "RUNNING" {
		ResponseError(c, http.StatusBadRequest, "不能删除运行中的服务，请先停止服务")
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "开始事务失败")
		return
	}

	// 删除服务相关的配置
	_, err = tx.Exec("DELETE FROM config WHERE scope_type = 'SERVICE' AND scope_id = ?", serviceID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "删除服务配置失败")
		return
	}

	// 删除服务相关的组件
	_, err = tx.Exec("DELETE FROM service_component WHERE service_id = ?", serviceID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "删除服务组件失败")
		return
	}

	// 删除服务相关的指标数据
	_, err = tx.Exec("DELETE FROM metric WHERE service_id = ?", serviceID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "删除服务指标数据失败")
		return
	}

	// 删除服务
	_, err = tx.Exec("DELETE FROM service WHERE id = ?", serviceID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "删除服务失败")
		return
	}

	if err = tx.Commit(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	ResponseSuccessWithMessage(c, "服务删除成功", nil)
}

// ServiceComponents 管理服务组件
func AddServiceComponent(c *gin.Context) {
	serviceID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的服务ID")
		return
	}

	var component model.ServiceComponent
	if err := c.ShouldBindJSON(&component); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 检查服务是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM service WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, serviceID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询服务失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "服务不存在")
		return
	}

	// 检查是否已存在相同类型的组件
	var componentExists bool
	componentExistsQuery := "SELECT EXISTS(SELECT 1 FROM service_component WHERE service_id = ? AND component_type = ?)"
	err = db.DB.QueryRow(componentExistsQuery, serviceID, component.ComponentType).Scan(&componentExists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "检查组件失败")
		return
	}
	if componentExists {
		ResponseError(c, http.StatusBadRequest, "服务中已存在相同类型的组件")
		return
	}

	// 插入组件
	component.ServiceID = serviceID
	insertQuery := "INSERT INTO service_component (service_id, component_type, desired_instances) VALUES (?, ?, ?)"
	result, err := db.DB.Exec(insertQuery, component.ServiceID, component.ComponentType, component.DesiredInstances)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "添加组件失败: "+err.Error())
		return
	}

	componentID, _ := result.LastInsertId()
	component.ID = int(componentID)

	ResponseSuccessWithMessage(c, "组件添加成功", component)
}

// UpdateServiceComponent 更新服务组件
func UpdateServiceComponent(c *gin.Context) {
	componentID, err := strconv.Atoi(c.Param("component_id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的组件ID")
		return
	}

	var component model.ServiceComponent
	if err := c.ShouldBindJSON(&component); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 检查组件是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM service_component WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, componentID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询组件失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "组件不存在")
		return
	}

	// 更新组件
	updateQuery := "UPDATE service_component SET desired_instances = ? WHERE id = ?"
	_, err = db.DB.Exec(updateQuery, component.DesiredInstances, componentID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "更新组件失败: "+err.Error())
		return
	}

	ResponseSuccessWithMessage(c, "组件更新成功", nil)
}

// DeleteServiceComponent 删除服务组件
func DeleteServiceComponent(c *gin.Context) {
	componentID, err := strconv.Atoi(c.Param("component_id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的组件ID")
		return
	}

	// 检查组件是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM service_component WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, componentID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询组件失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "组件不存在")
		return
	}

	// 检查是否有主机-组件映射
	var hostComponentCount int
	countQuery := "SELECT COUNT(*) FROM host_component WHERE component_id = ?"
	err = db.DB.QueryRow(countQuery, componentID).Scan(&hostComponentCount)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机组件映射失败")
		return
	}
	if hostComponentCount > 0 {
		ResponseError(c, http.StatusBadRequest, "组件已部署到主机上，不能删除")
		return
	}

	// 删除组件
	deleteQuery := "DELETE FROM service_component WHERE id = ?"
	_, err = db.DB.Exec(deleteQuery, componentID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "删除组件失败: "+err.Error())
		return
	}

	ResponseSuccessWithMessage(c, "组件删除成功", nil)
}

// DeployComponent 将组件部署到主机
func DeployComponent(c *gin.Context) {
	componentID, err := strconv.Atoi(c.Param("component_id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的组件ID")
		return
	}

	var req struct {
		HostIDs []int `json:"host_ids" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 检查组件是否存在
	var (
		serviceID     int
		componentType string
	)
	componentQuery := "SELECT service_id, component_type FROM service_component WHERE id = ?"
	err = db.DB.QueryRow(componentQuery, componentID).Scan(&serviceID, &componentType)
	if err == sql.ErrNoRows {
		ResponseError(c, http.StatusNotFound, "组件不存在")
		return
	}
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询组件失败")
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "开始事务失败")
		return
	}

	// 为每个主机创建主机-组件映射和安装任务
	for _, hostID := range req.HostIDs {
		// 检查主机是否存在
		var hostExists bool
		hostExistsQuery := "SELECT EXISTS(SELECT 1 FROM host WHERE id = ?)"
		err = tx.QueryRow(hostExistsQuery, hostID).Scan(&hostExists)
		if err != nil {
			tx.Rollback()
			ResponseError(c, http.StatusInternalServerError, "检查主机失败")
			return
		}
		if !hostExists {
			tx.Rollback()
			ResponseError(c, http.StatusBadRequest, fmt.Sprintf("主机ID %d 不存在", hostID))
			return
		}

		// 检查是否已存在相同的部署
		var deployExists bool
		deployExistsQuery := "SELECT EXISTS(SELECT 1 FROM host_component WHERE host_id = ? AND component_id = ?)"
		err = tx.QueryRow(deployExistsQuery, hostID, componentID).Scan(&deployExists)
		if err != nil {
			tx.Rollback()
			ResponseError(c, http.StatusInternalServerError, "检查部署失败")
			return
		}
		if deployExists {
			tx.Rollback()
			ResponseError(c, http.StatusBadRequest, fmt.Sprintf("组件已部署到主机ID %d", hostID))
			return
		}

		// 创建主机-组件映射
		_, err = tx.Exec(
			"INSERT INTO host_component (host_id, component_id, status) VALUES (?, ?, ?)",
			hostID, componentID, "INSTALLING")
		if err != nil {
			tx.Rollback()
			ResponseError(c, http.StatusInternalServerError, "创建组件部署失败")
			return
		}

		// 创建安装任务
		_, err = tx.Exec(
			"INSERT INTO task (task_type, related_id, related_type, status) VALUES (?, ?, ?, ?)",
			"INSTALL_COMPONENT", componentID, "COMPONENT", "PENDING")
		if err != nil {
			tx.Rollback()
			ResponseError(c, http.StatusInternalServerError, "创建安装任务失败")
			return
		}
	}

	if err = tx.Commit(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	ResponseSuccessWithMessage(c, "组件部署已开始", nil)
}

// RegisterServiceRoutes 注册服务相关路由
func RegisterServiceRoutes(router *gin.RouterGroup) {
	router.Use(JWTAuthMiddleware())
	
	// 需要服务查看权限的接口
	viewRouter := router.Group("/")
	viewRouter.Use(PrivilegeMiddleware("VIEW_SERVICE"))
	{
		viewRouter.GET("/services", GetServices)
		viewRouter.GET("/services/:id", GetServiceById)
	}
	
	// 需要服务管理权限的接口
	manageRouter := router.Group("/")
	manageRouter.Use(PrivilegeMiddleware("MANAGE_SERVICE"))
	{
		// 服务管理
		manageRouter.POST("/services", CreateService)
		manageRouter.PUT("/services/:id", UpdateService)
		manageRouter.DELETE("/services/:id", DeleteService)
		manageRouter.POST("/services/:id/start", StartService)
		manageRouter.POST("/services/:id/stop", StopService)
		
		// 组件管理
		manageRouter.POST("/services/:id/components", AddServiceComponent)
		manageRouter.PUT("/components/:component_id", UpdateServiceComponent)
		manageRouter.DELETE("/components/:component_id", DeleteServiceComponent)
		manageRouter.POST("/components/:component_id/deploy", DeployComponent)
	}
} 