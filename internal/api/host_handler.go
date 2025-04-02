package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/TejParker/bigdata-manager/internal/db"
	"github.com/TejParker/bigdata-manager/pkg/model"
)

// CreateHost 创建新主机
func CreateHost(c *gin.Context) {
	var host model.Host
	if err := c.ShouldBindJSON(&host); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 检查集群是否存在
	var clusterExists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM cluster WHERE id = ?)"
	err := db.DB.QueryRow(existsQuery, host.ClusterID).Scan(&clusterExists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询集群失败")
		return
	}
	if !clusterExists {
		ResponseError(c, http.StatusBadRequest, "指定的集群不存在")
		return
	}

	// 检查主机名是否已存在
	var hostExists bool
	hostExistsQuery := "SELECT EXISTS(SELECT 1 FROM host WHERE hostname = ? AND cluster_id = ?)"
	err = db.DB.QueryRow(hostExistsQuery, host.Hostname, host.ClusterID).Scan(&hostExists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "检查主机名失败")
		return
	}
	if hostExists {
		ResponseError(c, http.StatusBadRequest, "集群中已存在同名主机")
		return
	}

	// 插入主机记录
	query := `INSERT INTO host 
		(hostname, ip, cluster_id, status, cpu_cores, memory_size) 
		VALUES (?, ?, ?, ?, ?, ?)`
	result, err := db.DB.Exec(query, 
		host.Hostname, 
		host.IP, 
		host.ClusterID, 
		"OFFLINE", // 初始状态为离线
		host.CPUCores, 
		host.MemorySize)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "创建主机失败: "+err.Error())
		return
	}

	// 获取新主机ID
	hostID, err := result.LastInsertId()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "获取主机ID失败")
		return
	}

	// 返回成功响应
	host.ID = int(hostID)
	host.Status = "OFFLINE"
	ResponseSuccessWithMessage(c, "主机添加成功", host)
}

// GetHosts 获取主机列表
func GetHosts(c *gin.Context) {
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
	status := c.Query("status")

	// 构建查询条件
	whereClause := ""
	args := []interface{}{}
	
	if clusterID != "" {
		whereClause += " WHERE cluster_id = ?"
		args = append(args, clusterID)
		
		if status != "" {
			whereClause += " AND status = ?"
			args = append(args, status)
		}
	} else if status != "" {
		whereClause += " WHERE status = ?"
		args = append(args, status)
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM host" + whereClause
	var total int
	err := db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机总数失败")
		return
	}

	// 分页参数
	query := `SELECT h.id, h.hostname, h.ip, h.cluster_id, c.name as cluster_name,
		h.cpu_cores, h.memory_size, h.status, h.agent_version, h.last_heartbeat,
		h.created_at, h.updated_at
		FROM host h
		LEFT JOIN cluster c ON h.cluster_id = c.id` + 
		whereClause + 
		" ORDER BY h.id DESC LIMIT ? OFFSET ?"
	
	args = append(args, pageSize, offset)
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机列表失败")
		return
	}
	defer rows.Close()

	// 构造结果集
	type HostWithCluster struct {
		model.Host
		ClusterName string `json:"cluster_name"`
	}
	
	var hosts []HostWithCluster
	for rows.Next() {
		var host HostWithCluster
		var lastHeartbeat sql.NullTime
		var agentVersion sql.NullString
		
		if err := rows.Scan(
			&host.ID, &host.Hostname, &host.IP, &host.ClusterID, &host.ClusterName,
			&host.CPUCores, &host.MemorySize, &host.Status, &agentVersion, &lastHeartbeat,
			&host.CreatedAt, &host.UpdatedAt); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取主机数据失败")
			return
		}
		
		if lastHeartbeat.Valid {
			host.LastHeartbeat = lastHeartbeat.Time
		}
		
		if agentVersion.Valid {
			host.AgentVersion = agentVersion.String
		}
		
		hosts = append(hosts, host)
	}

	if err = rows.Err(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "处理主机数据失败")
		return
	}

	// 返回分页结果
	ResponsePageSuccess(c, hosts, total, page, pageSize)
}

// GetHostById 根据ID获取主机详情
func GetHostById(c *gin.Context) {
	hostID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	// 查询主机基本信息
	query := `SELECT h.id, h.hostname, h.ip, h.cluster_id, c.name as cluster_name,
		h.cpu_cores, h.memory_size, h.status, h.agent_version, h.last_heartbeat,
		h.created_at, h.updated_at
		FROM host h
		LEFT JOIN cluster c ON h.cluster_id = c.id
		WHERE h.id = ?`
	
	type HostWithCluster struct {
		model.Host
		ClusterName string `json:"cluster_name"`
	}
	
	var host HostWithCluster
	var lastHeartbeat sql.NullTime
	var agentVersion sql.NullString
	
	err = db.DB.QueryRow(query, hostID).Scan(
		&host.ID, &host.Hostname, &host.IP, &host.ClusterID, &host.ClusterName,
		&host.CPUCores, &host.MemorySize, &host.Status, &agentVersion, &lastHeartbeat,
		&host.CreatedAt, &host.UpdatedAt)
	
	if err == sql.ErrNoRows {
		ResponseError(c, http.StatusNotFound, "主机不存在")
		return
	}
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机信息失败")
		return
	}
	
	if lastHeartbeat.Valid {
		host.LastHeartbeat = lastHeartbeat.Time
	}
	
	if agentVersion.Valid {
		host.AgentVersion = agentVersion.String
	}

	// 查询部署的组件数量
	var componentCount int
	componentCountQuery := `
		SELECT COUNT(*) FROM host_component hc
		WHERE hc.host_id = ?
	`
	err = db.DB.QueryRow(componentCountQuery, hostID).Scan(&componentCount)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询组件数量失败")
		return
	}

	// 查询最近的指标数据
	var cpuUsage, memoryUsage, diskUsage float64
	metricQuery := `
		SELECT m.value FROM metric m
		WHERE m.host_id = ? AND m.metric_name = ?
		ORDER BY m.timestamp DESC LIMIT 1
	`
	
	_ = db.DB.QueryRow(metricQuery, hostID, "cpu_usage").Scan(&cpuUsage)
	_ = db.DB.QueryRow(metricQuery, hostID, "memory_usage").Scan(&memoryUsage)
	_ = db.DB.QueryRow(metricQuery, hostID, "disk_usage").Scan(&diskUsage)

	// 返回结果
	ResponseSuccess(c, gin.H{
		"host":            host,
		"component_count": componentCount,
		"metrics": gin.H{
			"cpu_usage":    cpuUsage,
			"memory_usage": memoryUsage,
			"disk_usage":   diskUsage,
		},
	})
}

// UpdateHost 更新主机信息
func UpdateHost(c *gin.Context) {
	hostID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	var host model.Host
	if err := c.ShouldBindJSON(&host); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 检查主机是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM host WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, hostID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "主机不存在")
		return
	}

	// 更新主机信息
	updateQuery := `
		UPDATE host SET 
		hostname = ?,
		ip = ?,
		cpu_cores = ?,
		memory_size = ?
		WHERE id = ?
	`
	_, err = db.DB.Exec(updateQuery, 
		host.Hostname,
		host.IP,
		host.CPUCores,
		host.MemorySize,
		hostID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "更新主机失败: "+err.Error())
		return
	}

	ResponseSuccessWithMessage(c, "主机更新成功", nil)
}

// DeleteHost 删除主机
func DeleteHost(c *gin.Context) {
	hostID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	// 检查主机是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM host WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, hostID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "主机不存在")
		return
	}

	// 检查是否有部署的组件
	var componentCount int
	componentCountQuery := "SELECT COUNT(*) FROM host_component WHERE host_id = ?"
	err = db.DB.QueryRow(componentCountQuery, hostID).Scan(&componentCount)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询组件数量失败")
		return
	}
	if componentCount > 0 {
		ResponseError(c, http.StatusBadRequest, "主机上部署了组件，无法删除")
		return
	}

	// 删除主机
	tx, err := db.DB.Begin()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "开始事务失败")
		return
	}

	// 删除主机相关的指标数据
	_, err = tx.Exec("DELETE FROM metric WHERE host_id = ?", hostID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "删除主机指标数据失败")
		return
	}

	// 删除主机相关的日志
	_, err = tx.Exec("DELETE FROM log_record WHERE host_id = ?", hostID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "删除主机日志失败")
		return
	}

	// 删除主机本身
	_, err = tx.Exec("DELETE FROM host WHERE id = ?", hostID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "删除主机失败")
		return
	}

	if err = tx.Commit(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	ResponseSuccessWithMessage(c, "主机删除成功", nil)
}

// UpdateHostStatus 更新主机状态
func UpdateHostStatus(c *gin.Context) {
	hostID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证状态值
	validStatuses := map[string]bool{
		"ONLINE":      true,
		"OFFLINE":     true,
		"MAINTENANCE": true,
	}
	
	if !validStatuses[req.Status] {
		ResponseError(c, http.StatusBadRequest, "无效的状态值")
		return
	}

	// 检查主机是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM host WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, hostID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "主机不存在")
		return
	}

	// 更新主机状态
	updateQuery := "UPDATE host SET status = ? WHERE id = ?"
	_, err = db.DB.Exec(updateQuery, req.Status, hostID)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "更新主机状态失败")
		return
	}

	ResponseSuccessWithMessage(c, "主机状态更新成功", nil)
}

// ProcessHeartbeat 处理Agent心跳
func ProcessHeartbeat(c *gin.Context) {
	var req model.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的心跳请求")
		return
	}

	// 检查主机是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM host WHERE id = ?)"
	err := db.DB.QueryRow(existsQuery, req.HostID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询主机失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "主机不存在")
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "开始事务失败")
		return
	}

	// 更新主机状态和心跳时间
	now := time.Now()
	updateQuery := "UPDATE host SET status = 'ONLINE', last_heartbeat = ? WHERE id = ?"
	_, err = tx.Exec(updateQuery, now, req.HostID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "更新心跳失败")
		return
	}

	// 存储基础指标
	basicMetrics := []struct {
		name  string
		value float64
	}{
		{"cpu_usage", req.CPUUsage},
		{"memory_usage", req.MemoryUsage},
		{"disk_usage", req.DiskUsage},
	}

	for _, m := range basicMetrics {
		_, err = tx.Exec(
			"INSERT INTO metric (host_id, metric_name, timestamp, value) VALUES (?, ?, ?, ?)",
			req.HostID, m.name, now, m.value,
		)
		if err != nil {
			tx.Rollback()
			ResponseError(c, http.StatusInternalServerError, "存储基础指标失败")
			return
		}
	}

	// 存储其他指标
	if len(req.Metrics) > 0 {
		for _, metric := range req.Metrics {
			_, err = tx.Exec(
				"INSERT INTO metric (host_id, metric_name, timestamp, value) VALUES (?, ?, ?, ?)",
				req.HostID, metric.Name, metric.Timestamp, metric.Value,
			)
			if err != nil {
				tx.Rollback()
				ResponseError(c, http.StatusInternalServerError, "存储自定义指标失败")
				return
			}
		}
	}

	// 更新组件状态
	if len(req.Components) > 0 {
		for _, comp := range req.Components {
			_, err = tx.Exec(
				"UPDATE host_component SET status = ?, process_id = ? WHERE host_id = ? AND component_id = ?",
				comp.Status, comp.ProcessID, req.HostID, comp.ComponentID,
			)
			if err != nil {
				tx.Rollback()
				ResponseError(c, http.StatusInternalServerError, "更新组件状态失败")
				return
			}
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	// 返回响应，可以包含命令给Agent执行
	ResponseSuccess(c, gin.H{
		"timestamp": time.Now(),
		"commands":  []model.AgentCommand{}, // 这里可以返回需要Agent执行的命令
	})
}

// RegisterHostRoutes 注册主机相关路由
func RegisterHostRoutes(router *gin.RouterGroup) {
	// 心跳接口不需要认证
	router.POST("/agent/heartbeat", ProcessHeartbeat)
	
	// 以下路由需要认证
	authRouter := router.Group("/")
	authRouter.Use(JWTAuthMiddleware())
	
	// 需要主机查看权限的接口
	viewRouter := authRouter.Group("/")
	viewRouter.Use(PrivilegeMiddleware("VIEW_HOST"))
	{
		viewRouter.GET("/hosts", GetHosts)
		viewRouter.GET("/hosts/:id", GetHostById)
	}
	
	// 需要主机管理权限的接口
	manageRouter := authRouter.Group("/")
	manageRouter.Use(PrivilegeMiddleware("MANAGE_HOST"))
	{
		manageRouter.POST("/hosts", CreateHost)
		manageRouter.PUT("/hosts/:id", UpdateHost)
		manageRouter.DELETE("/hosts/:id", DeleteHost)
		manageRouter.PUT("/hosts/:id/status", UpdateHostStatus)
	}
} 