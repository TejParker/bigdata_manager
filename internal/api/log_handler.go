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

// GetLogs 查询日志记录
func GetLogs(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	// 过滤参数
	hostID := c.Query("host_id")
	serviceID := c.Query("service_id")
	componentID := c.Query("component_id")
	logLevel := c.Query("log_level")
	startTimeStr := c.DefaultQuery("start_time", "")
	endTimeStr := c.DefaultQuery("end_time", "")
	keyword := c.Query("keyword")

	// 构建查询条件
	whereClause := ""
	args := []interface{}{}
	
	conditions := []string{}

	// 解析时间范围
	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的开始时间格式，请使用RFC3339格式")
			return
		}
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, startTime)
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的结束时间格式，请使用RFC3339格式")
			return
		}
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, endTime)
	}
	
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
	
	if componentID != "" {
		componentIDInt, err := strconv.Atoi(componentID)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的component_id参数")
			return
		}
		conditions = append(conditions, "component_id = ?")
		args = append(args, componentIDInt)
	}
	
	if logLevel != "" {
		conditions = append(conditions, "log_level = ?")
		args = append(args, logLevel)
	}
	
	if keyword != "" {
		conditions = append(conditions, "message LIKE ?")
		args = append(args, "%"+keyword+"%")
	}
	
	if len(conditions) > 0 {
		whereClause = " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			whereClause += " AND " + conditions[i]
		}
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM log_record" + whereClause
	var total int
	err = db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询日志总数失败")
		return
	}

	// 查询分页数据
	query := `SELECT lr.id, lr.host_id, lr.service_id, lr.component_id, lr.log_level, 
		lr.timestamp, lr.message, lr.created_at,
		h.hostname, s.service_name, sc.component_type
		FROM log_record lr
		LEFT JOIN host h ON lr.host_id = h.id
		LEFT JOIN service s ON lr.service_id = s.id
		LEFT JOIN service_component sc ON lr.component_id = sc.id` + 
		whereClause + 
		" ORDER BY lr.timestamp DESC LIMIT ? OFFSET ?"
	
	args = append(args, pageSize, offset)
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询日志失败")
		return
	}
	defer rows.Close()

	// 构造结果集
	type LogDetail struct {
		ID            int64     `json:"id"`
		HostID        int       `json:"host_id,omitempty"`
		ServiceID     int       `json:"service_id,omitempty"`
		ComponentID   int       `json:"component_id,omitempty"`
		LogLevel      string    `json:"log_level"`
		Timestamp     time.Time `json:"timestamp"`
		Message       string    `json:"message"`
		CreatedAt     time.Time `json:"created_at"`
		Hostname      string    `json:"hostname,omitempty"`
		ServiceName   string    `json:"service_name,omitempty"`
		ComponentType string    `json:"component_type,omitempty"`
	}
	
	var logs []LogDetail
	for rows.Next() {
		var log LogDetail
		var hostIDNull, serviceIDNull, componentIDNull sql.NullInt32
		var hostnameNull, serviceNameNull, componentTypeNull sql.NullString
		
		if err := rows.Scan(
			&log.ID, &hostIDNull, &serviceIDNull, &componentIDNull, &log.LogLevel,
			&log.Timestamp, &log.Message, &log.CreatedAt,
			&hostnameNull, &serviceNameNull, &componentTypeNull); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取日志数据失败")
			return
		}
		
		if hostIDNull.Valid {
			log.HostID = int(hostIDNull.Int32)
		}
		
		if serviceIDNull.Valid {
			log.ServiceID = int(serviceIDNull.Int32)
		}
		
		if componentIDNull.Valid {
			log.ComponentID = int(componentIDNull.Int32)
		}
		
		if hostnameNull.Valid {
			log.Hostname = hostnameNull.String
		}
		
		if serviceNameNull.Valid {
			log.ServiceName = serviceNameNull.String
		}
		
		if componentTypeNull.Valid {
			log.ComponentType = componentTypeNull.String
		}
		
		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "处理日志数据失败")
		return
	}

	// 返回分页结果
	ResponsePageSuccess(c, logs, total, page, pageSize)
}

// UploadLogs 上传日志记录
func UploadLogs(c *gin.Context) {
	var req struct {
		HostID      int             `json:"host_id" binding:"required"`
		ServiceID   int             `json:"service_id,omitempty"`
		ComponentID int             `json:"component_id,omitempty"`
		Logs        []model.LogRecord `json:"logs" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证主机是否存在
	var hostExists bool
	hostQuery := "SELECT EXISTS(SELECT 1 FROM host WHERE id = ?)"
	err := db.DB.QueryRow(hostQuery, req.HostID).Scan(&hostExists)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "验证主机失败: "+err.Error())
		return
	}
	if !hostExists {
		ResponseError(c, http.StatusBadRequest, "指定的主机不存在")
		return
	}

	// 验证日志数量
	if len(req.Logs) == 0 {
		ResponseError(c, http.StatusBadRequest, "未提供日志数据")
		return
	}
	if len(req.Logs) > 1000 {
		ResponseError(c, http.StatusBadRequest, "日志数量超过限制，最多一次上传1000条")
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "开始事务失败")
		return
	}

	// 批量插入日志
	stmt, err := tx.Prepare(`
		INSERT INTO log_record (host_id, service_id, component_id, log_level, timestamp, message)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		ResponseError(c, http.StatusInternalServerError, "准备SQL语句失败")
		return
	}
	defer stmt.Close()

	for _, log := range req.Logs {
		var serviceIDParam, componentIDParam interface{}
		
		// 使用请求中的ID或日志记录中的ID
		serviceIDParam = sql.NullInt32{Int32: int32(req.ServiceID), Valid: req.ServiceID > 0}
		if log.ServiceID > 0 {
			serviceIDParam = log.ServiceID
		}
		
		componentIDParam = sql.NullInt32{Int32: int32(req.ComponentID), Valid: req.ComponentID > 0}
		if log.ComponentID > 0 {
			componentIDParam = log.ComponentID
		}
		
		_, err := stmt.Exec(
			req.HostID,
			serviceIDParam,
			componentIDParam,
			log.LogLevel,
			log.Timestamp,
			log.Message,
		)
		if err != nil {
			tx.Rollback()
			ResponseError(c, http.StatusInternalServerError, "插入日志失败: "+err.Error())
			return
		}
	}

	if err = tx.Commit(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "提交事务失败")
		return
	}

	ResponseSuccessWithMessage(c, "日志上传成功", gin.H{"count": len(req.Logs)})
}

// GetLogLevels 获取日志级别列表
func GetLogLevels(c *gin.Context) {
	// 查询数据库中存在的日志级别
	query := `SELECT DISTINCT log_level FROM log_record ORDER BY 
		CASE 
			WHEN log_level = 'ERROR' THEN 1
			WHEN log_level = 'WARN' THEN 2
			WHEN log_level = 'INFO' THEN 3
			WHEN log_level = 'DEBUG' THEN 4
			WHEN log_level = 'TRACE' THEN 5
			ELSE 6
		END`
	
	rows, err := db.DB.Query(query)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询日志级别失败")
		return
	}
	defer rows.Close()

	var levels []string
	for rows.Next() {
		var level string
		if err := rows.Scan(&level); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取日志级别失败")
			return
		}
		levels = append(levels, level)
	}

	if err = rows.Err(); err != nil {
		ResponseError(c, http.StatusInternalServerError, "处理日志级别数据失败")
		return
	}

	ResponseSuccess(c, levels)
}

// GetLogStats 获取日志统计信息
func GetLogStats(c *gin.Context) {
	// 解析查询参数
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days < 1 || days > 30 {
		days = 7 // 默认过去7天
	}

	// 计算起始时间
	startTime := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	// 查询各个级别的日志数量
	levelQuery := `
		SELECT log_level, COUNT(*) as count 
		FROM log_record 
		WHERE timestamp >= ? 
		GROUP BY log_level 
		ORDER BY count DESC
	`
	
	levelRows, err := db.DB.Query(levelQuery, startTime)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询日志级别统计失败")
		return
	}
	defer levelRows.Close()

	levelStats := make(map[string]int)
	for levelRows.Next() {
		var level string
		var count int
		if err := levelRows.Scan(&level, &count); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取日志级别统计失败")
			return
		}
		levelStats[level] = count
	}

	// 查询每天的日志数量
	dailyQuery := `
		SELECT DATE(timestamp) as date, COUNT(*) as count 
		FROM log_record 
		WHERE timestamp >= ? 
		GROUP BY DATE(timestamp) 
		ORDER BY date
	`
	
	dailyRows, err := db.DB.Query(dailyQuery, startTime)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询日志日期统计失败")
		return
	}
	defer dailyRows.Close()

	type DailyCount struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	
	var dailyStats []DailyCount
	for dailyRows.Next() {
		var stat DailyCount
		if err := dailyRows.Scan(&stat.Date, &stat.Count); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取日志日期统计失败")
			return
		}
		dailyStats = append(dailyStats, stat)
	}

	// 查询主机的日志数量
	hostQuery := `
		SELECT h.hostname, COUNT(l.id) as count 
		FROM log_record l
		JOIN host h ON l.host_id = h.id
		WHERE l.timestamp >= ? 
		GROUP BY l.host_id
		ORDER BY count DESC
		LIMIT 10
	`
	
	hostRows, err := db.DB.Query(hostQuery, startTime)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, "查询日志主机统计失败")
		return
	}
	defer hostRows.Close()

	type HostCount struct {
		Hostname string `json:"hostname"`
		Count    int    `json:"count"`
	}
	
	var hostStats []HostCount
	for hostRows.Next() {
		var stat HostCount
		if err := hostRows.Scan(&stat.Hostname, &stat.Count); err != nil {
			ResponseError(c, http.StatusInternalServerError, "读取日志主机统计失败")
			return
		}
		hostStats = append(hostStats, stat)
	}

	ResponseSuccess(c, gin.H{
		"level_stats": levelStats,
		"daily_stats": dailyStats,
		"host_stats":  hostStats,
		"days":        days,
	})
}

// RegisterLogRoutes 注册日志相关路由
func RegisterLogRoutes(router *gin.RouterGroup) {
	// 代理上传日志接口不需要认证
	router.POST("/agent/logs", UploadLogs)
	
	// 以下路由需要认证
	authRouter := router.Group("/")
	authRouter.Use(JWTAuthMiddleware())
	
	// 需要日志查看权限的接口
	viewRouter := authRouter.Group("/")
	viewRouter.Use(PrivilegeMiddleware("VIEW_LOG"))
	{
		viewRouter.GET("/logs", GetLogs)
		viewRouter.GET("/log-levels", GetLogLevels)
		viewRouter.GET("/log-stats", GetLogStats)
	}
} 