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

// GetMetrics 获取指标数据
func GetMetrics(c *gin.Context) {
	// 获取查询参数
	hostID := c.Query("host_id")
	serviceID := c.Query("service_id")
	metricName := c.Query("metric_name")
	startTimeStr := c.DefaultQuery("start_time", "")
	endTimeStr := c.DefaultQuery("end_time", "")
	limitStr := c.DefaultQuery("limit", "100")

	// 验证必要参数
	if metricName == "" {
		ResponseError(c, http.StatusBadRequest, "必须指定metric_name参数")
		return
	}

	// 解析时间范围
	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的开始时间格式，请使用RFC3339格式")
			return
		}
	} else {
		// 默认最近24小时
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的结束时间格式，请使用RFC3339格式")
			return
		}
	} else {
		endTime = time.Now()
	}

	// 解析limit
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000 // 限制最大查询数量
	}

	// 构建查询
	query := "SELECT id, host_id, service_id, metric_name, timestamp, value FROM metric WHERE metric_name = ? AND timestamp BETWEEN ? AND ? "
	args := []interface{}{metricName, startTime, endTime}

	if hostID != "" {
		query += "AND host_id = ? "
		hostIDInt, err := strconv.Atoi(hostID)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的host_id参数")
			return
		}
		args = append(args, hostIDInt)
	}

	if serviceID != "" {
		query += "AND service_id = ? "
		serviceIDInt, err := strconv.Atoi(serviceID)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的service_id参数")
			return
		}
		args = append(args, serviceIDInt)
	}

	// 添加排序和限制
	query += "ORDER BY timestamp ASC LIMIT ?"
	args = append(args, limit)

	// 执行查询
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询指标数据失败: "+err.Error())
		return
	}
	defer rows.Close()

	// 处理结果
	var metrics []model.Metric
	for rows.Next() {
		var metric model.Metric
		var hostIDNull, serviceIDNull sql.NullInt32
		
		err := rows.Scan(
			&metric.ID,
			&hostIDNull,
			&serviceIDNull,
			&metric.MetricName,
			&metric.Timestamp,
			&metric.Value,
		)
		if err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取指标数据失败: "+err.Error())
			return
		}
		
		if hostIDNull.Valid {
			metric.HostID = int(hostIDNull.Int32)
		}
		
		if serviceIDNull.Valid {
			metric.ServiceID = int(serviceIDNull.Int32)
		}
		
		metrics = append(metrics, metric)
	}

	if err = rows.Err(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "处理指标数据失败: "+err.Error())
		return
	}

	// 获取指标定义
	var definition model.MetricDefinition
	definitionQuery := "SELECT id, metric_name, display_name, category, unit, description FROM metric_definition WHERE metric_name = ?"
	err = db.DB.QueryRow(definitionQuery, metricName).Scan(
		&definition.ID,
		&definition.MetricName,
		&definition.DisplayName,
		&definition.Category,
		&definition.Unit,
		&definition.Description,
	)
	if err != nil && err != sql.ErrNoRows {
		ResponseError(c, http.StatusInternalServerError, "查询指标定义失败: "+err.Error())
		return
	}

	ResponseSuccess(c, gin.H{
		"metrics":    metrics,
		"definition": definition,
		"start_time": startTime,
		"end_time":   endTime,
	})
}

// GetMetricDefinitions 获取指标定义列表
func GetMetricDefinitions(c *gin.Context) {
	category := c.Query("category")

	var query string
	var args []interface{}

	if category != "" {
		query = "SELECT id, metric_name, display_name, category, unit, description, created_at, updated_at FROM metric_definition WHERE category = ? ORDER BY display_name"
		args = []interface{}{category}
	} else {
		query = "SELECT id, metric_name, display_name, category, unit, description, created_at, updated_at FROM metric_definition ORDER BY display_name"
	}

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询指标定义失败: "+err.Error())
		return
	}
	defer rows.Close()

	var definitions []model.MetricDefinition
	for rows.Next() {
		var definition model.MetricDefinition
		err := rows.Scan(
			&definition.ID,
			&definition.MetricName,
			&definition.DisplayName,
			&definition.Category,
			&definition.Unit,
			&definition.Description,
			&definition.CreatedAt,
			&definition.UpdatedAt,
		)
		if err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取指标定义失败: "+err.Error())
			return
		}
		definitions = append(definitions, definition)
	}

	if err = rows.Err(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "处理指标定义失败: "+err.Error())
		return
	}

	ResponseSuccess(c, definitions)
}

// CreateAlertRule 创建告警规则
func CreateAlertRule(c *gin.Context) {
	var alert model.Alert
	if err := c.ShouldBindJSON(&alert); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证指标是否存在
	var metricExists bool
	metricQuery := "SELECT EXISTS(SELECT 1 FROM metric_definition WHERE metric_name = ?)"
	err := db.DB.QueryRow(metricQuery, alert.MetricName).Scan(&metricExists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "验证指标失败: "+err.Error())
		return
	}
	if !metricExists {
		ResponseError(c, http.StatusBadRequest, "指定的指标不存在")
		return
	}

	// 验证告警级别
	if alert.Severity != "INFO" && alert.Severity != "WARNING" && alert.Severity != "CRITICAL" {
		ResponseError(c, http.StatusBadRequest, "无效的告警级别，必须是INFO、WARNING或CRITICAL")
		return
	}

	// 插入告警规则
	query := `INSERT INTO alert 
		(name, metric_name, condition, threshold, duration, severity, notification_method) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := db.DB.Exec(query,
		alert.Name,
		alert.MetricName,
		alert.Condition,
		alert.Threshold,
		alert.Duration,
		alert.Severity,
		alert.NotificationMethod,
	)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "创建告警规则失败: "+err.Error())
		return
	}

	alertID, err := result.LastInsertId()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "获取告警规则ID失败")
		return
	}

	alert.ID = int(alertID)
	ResponseSuccessWithMessage(c, "告警规则创建成功", alert)
}

// GetAlertRules 获取告警规则列表
func GetAlertRules(c *gin.Context) {
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
	metricName := c.Query("metric_name")
	severity := c.Query("severity")

	// 构建查询条件
	whereClause := ""
	args := []interface{}{}
	
	if metricName != "" {
		whereClause += " WHERE metric_name = ?"
		args = append(args, metricName)
		
		if severity != "" {
			whereClause += " AND severity = ?"
			args = append(args, severity)
		}
	} else if severity != "" {
		whereClause += " WHERE severity = ?"
		args = append(args, severity)
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM alert" + whereClause
	var total int
	err := db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询告警规则总数失败")
		return
	}

	// 查询分页数据
	query := `SELECT id, name, metric_name, condition, threshold, duration, severity, 
		notification_method, created_at, updated_at
		FROM alert` + whereClause + 
		" ORDER BY id DESC LIMIT ? OFFSET ?"
	
	args = append(args, pageSize, offset)
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询告警规则失败")
		return
	}
	defer rows.Close()

	// 构造结果集
	var rules []model.Alert
	for rows.Next() {
		var rule model.Alert
		
		if err := rows.Scan(
			&rule.ID, &rule.Name, &rule.MetricName, &rule.Condition, &rule.Threshold,
			&rule.Duration, &rule.Severity, &rule.NotificationMethod, 
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取告警规则数据失败")
			return
		}
		
		rules = append(rules, rule)
	}

	if err = rows.Err(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "处理告警规则数据失败")
		return
	}

	// 返回分页结果
	ResponsePageSuccess(c, rules, total, page, pageSize)
}

// GetAlertEvents 获取告警事件列表
func GetAlertEvents(c *gin.Context) {
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
	hostID := c.Query("host_id")
	serviceID := c.Query("service_id")
	status := c.Query("status")
	alertID := c.Query("alert_id")

	// 构建查询条件
	whereClause := ""
	args := []interface{}{}
	
	conditions := []string{}
	
	if hostID != "" {
		hostIDInt, err := strconv.Atoi(hostID)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的host_id参数")
			return
		}
		conditions = append(conditions, "host_id = ?")
		args = append(args, hostIDInt)
	}
	
	if serviceID != "" {
		serviceIDInt, err := strconv.Atoi(serviceID)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的service_id参数")
			return
		}
		conditions = append(conditions, "service_id = ?")
		args = append(args, serviceIDInt)
	}
	
	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	
	if alertID != "" {
		alertIDInt, err := strconv.Atoi(alertID)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的alert_id参数")
			return
		}
		conditions = append(conditions, "alert_id = ?")
		args = append(args, alertIDInt)
	}
	
	if len(conditions) > 0 {
		whereClause = " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			whereClause += " AND " + conditions[i]
		}
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM alert_event" + whereClause
	var total int
	err := db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询告警事件总数失败")
		return
	}

	// 查询分页数据
	query := `SELECT ae.id, ae.alert_id, ae.host_id, ae.service_id, ae.status, 
		ae.triggered_at, ae.resolved_at, ae.message, ae.created_at, ae.updated_at,
		a.name as alert_name, a.severity
		FROM alert_event ae
		LEFT JOIN alert a ON ae.alert_id = a.id` + whereClause + 
		" ORDER BY ae.triggered_at DESC LIMIT ? OFFSET ?"
	
	args = append(args, pageSize, offset)
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询告警事件失败")
		return
	}
	defer rows.Close()

	// 构造结果集
	type AlertEventDetail struct {
		model.AlertEvent
		AlertName string `json:"alert_name"`
		Severity  string `json:"severity"`
	}
	
	var events []AlertEventDetail
	for rows.Next() {
		var event AlertEventDetail
		var hostIDNull, serviceIDNull sql.NullInt32
		var resolvedAtNull sql.NullTime
		
		if err := rows.Scan(
			&event.ID, &event.AlertID, &hostIDNull, &serviceIDNull, &event.Status,
			&event.TriggeredAt, &resolvedAtNull, &event.Message, &event.CreatedAt, &event.UpdatedAt,
			&event.AlertName, &event.Severity); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取告警事件数据失败")
			return
		}
		
		if hostIDNull.Valid {
			event.HostID = int(hostIDNull.Int32)
		}
		
		if serviceIDNull.Valid {
			event.ServiceID = int(serviceIDNull.Int32)
		}
		
		if resolvedAtNull.Valid {
			event.ResolvedAt = resolvedAtNull.Time
		}
		
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "处理告警事件数据失败")
		return
	}

	// 返回分页结果
	ResponsePageSuccess(c, events, total, page, pageSize)
}

// HandleAlertEvent 处理告警事件
func HandleAlertEvent(c *gin.Context) {
	eventID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的事件ID")
		return
	}

	var req struct {
		Action string `json:"action" binding:"required"` // ACKNOWLEDGE, RESOLVE
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 检查事件是否存在
	var status string
	statusQuery := "SELECT status FROM alert_event WHERE id = ?"
	err = db.DB.QueryRow(statusQuery, eventID).Scan(&status)
	if err == sql.ErrNoRows {
		ResponseError(c, http.StatusNotFound, "告警事件不存在")
		return
	}
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询事件状态失败")
		return
	}

	// 根据操作类型处理
	var newStatus string
	var updateQuery string
	var args []interface{}

	switch req.Action {
	case "ACKNOWLEDGE":
		if status != "OPEN" {
			ResponseError(c, http.StatusBadRequest, "只有未处理的告警才能标记为已确认")
			return
		}
		newStatus = "ACKNOWLEDGED"
		updateQuery = "UPDATE alert_event SET status = ?, updated_at = ? WHERE id = ?"
		args = []interface{}{newStatus, time.Now(), eventID}
	case "RESOLVE":
		if status == "RESOLVED" {
			ResponseError(c, http.StatusBadRequest, "告警已解决")
			return
		}
		newStatus = "RESOLVED"
		updateQuery = "UPDATE alert_event SET status = ?, resolved_at = ?, updated_at = ? WHERE id = ?"
		args = []interface{}{newStatus, time.Now(), time.Now(), eventID}
	default:
		ResponseError(c, http.StatusBadRequest, "无效的操作类型，必须是ACKNOWLEDGE或RESOLVE")
		return
	}

	// 执行更新
	_, err = db.DB.Exec(updateQuery, args...)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "更新告警事件失败: "+err.Error())
		return
	}

	ResponseSuccessWithMessage(c, "告警事件已处理", gin.H{"status": newStatus})
}

// DeleteAlertRule 删除告警规则
func DeleteAlertRule(c *gin.Context) {
	ruleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的规则ID")
		return
	}

	// 检查规则是否存在
	var exists bool
	existsQuery := "SELECT EXISTS(SELECT 1 FROM alert WHERE id = ?)"
	err = db.DB.QueryRow(existsQuery, ruleID).Scan(&exists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询规则失败")
		return
	}
	if !exists {
		ResponseError(c, http.StatusNotFound, "告警规则不存在")
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "开始事务失败")
		return
	}

	// 删除相关的告警事件
	_, err = tx.Exec("DELETE FROM alert_event WHERE alert_id = ?", ruleID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "删除告警事件失败: "+err.Error())
		return
	}

	// 删除告警规则
	_, err = tx.Exec("DELETE FROM alert WHERE id = ?", ruleID)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "删除告警规则失败: "+err.Error())
		return
	}

	if err = tx.Commit(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	ResponseSuccessWithMessage(c, "告警规则删除成功", nil)
}

// RegisterMonitorRoutes 注册监控相关路由
func RegisterMonitorRoutes(router *gin.RouterGroup) {
	router.Use(JWTAuthMiddleware())
	
	// 需要监控查看权限的接口
	viewRouter := router.Group("/")
	viewRouter.Use(PrivilegeMiddleware("VIEW_METRIC"))
	{
		viewRouter.GET("/metrics", GetMetrics)
		viewRouter.GET("/metric-definitions", GetMetricDefinitions)
		viewRouter.GET("/alerts", GetAlertRules)
		viewRouter.GET("/alert-events", GetAlertEvents)
	}
	
	// 需要告警管理权限的接口
	manageRouter := router.Group("/")
	manageRouter.Use(PrivilegeMiddleware("MANAGE_ALERT"))
	{
		manageRouter.POST("/alerts", CreateAlertRule)
		manageRouter.DELETE("/alerts/:id", DeleteAlertRule)
		manageRouter.PUT("/alert-events/:id", HandleAlertEvent)
	}
} 