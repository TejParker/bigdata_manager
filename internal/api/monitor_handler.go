package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/TejParker/bigdata-manager/internal/db"
	"github.com/TejParker/bigdata-manager/internal/monitor"
	"github.com/TejParker/bigdata-manager/pkg/model"
	"github.com/gin-gonic/gin"
)

// GetMetrics 获取指标数据
func GetMetrics(c *gin.Context) {
	// 解析请求参数
	hostIDStr := c.DefaultQuery("host_id", "0")
	hostID, err := strconv.Atoi(hostIDStr)
	if err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	metricName := c.DefaultQuery("metric_name", "")

	// 解析时间范围
	startTimeStr := c.DefaultQuery("start_time", "")
	endTimeStr := c.DefaultQuery("end_time", "")

	var startTime, endTime time.Time

	if startTimeStr == "" {
		// 默认查询过去1小时的数据
		startTime = time.Now().Add(-1 * time.Hour)
	} else {
		parsedTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的开始时间格式")
			return
		}
		startTime = parsedTime
	}

	if endTimeStr == "" {
		// 默认到当前时间
		endTime = time.Now()
	} else {
		parsedTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的结束时间格式")
			return
		}
		endTime = parsedTime
	}

	// 获取监控服务
	monitorService := monitor.GetMonitorService()

	// 查询指标数据
	metrics := monitorService.GetMetrics(hostID, metricName, startTime, endTime)

	// 返回结果
	ResponseSuccess(c, metrics)
}

// GetAlerts 获取告警数据
func GetAlerts(c *gin.Context) {
	// 解析时间范围
	startTimeStr := c.DefaultQuery("start_time", "")
	endTimeStr := c.DefaultQuery("end_time", "")

	var startTime, endTime time.Time

	if startTimeStr == "" {
		// 默认查询过去24小时的数据
		startTime = time.Now().Add(-24 * time.Hour)
	} else {
		parsedTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的开始时间格式")
			return
		}
		startTime = parsedTime
	}

	if endTimeStr == "" {
		// 默认到当前时间
		endTime = time.Now()
	} else {
		parsedTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			ResponseError(c, http.StatusBadRequest, "无效的结束时间格式")
			return
		}
		endTime = parsedTime
	}

	// 获取监控服务
	monitorService := monitor.GetMonitorService()

	// 查询告警数据
	alerts := monitorService.GetAlerts(startTime, endTime)

	// 返回结果
	ResponseSuccess(c, alerts)
}

// CreateAlertRule 创建告警规则
func CreateAlertRule(c *gin.Context) {
	var rule model.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		ResponseError(c, http.StatusBadRequest, "无效的请求参数")
		return
	}

	// 验证规则参数
	if rule.MetricName == "" {
		ResponseError(c, http.StatusBadRequest, "指标名称不能为空")
		return
	}

	if rule.Operator != ">" && rule.Operator != "<" && rule.Operator != "=" {
		ResponseError(c, http.StatusBadRequest, "无效的运算符，支持 >, <, =")
		return
	}

	// 设置规则ID和创建时间
	rule.ID = time.Now().UnixNano()
	rule.CreatedAt = time.Now()

	// 获取监控服务
	monitorService := monitor.GetMonitorService()

	// 添加告警规则
	monitorService.AddAlertRule(rule)

	// 返回结果
	ResponseSuccess(c, rule)
}

// GetAlertRules 获取告警规则
func GetAlertRules(c *gin.Context) {
	// 获取监控服务
	monitorService := monitor.GetMonitorService()

	// 获取所有告警规则
	rules := monitorService.GetAlertRules()

	// 返回结果
	ResponseSuccess(c, rules)
}

// RegisterMonitorRoutes 注册监控相关路由
func RegisterMonitorRoutes(router *gin.RouterGroup) {
	router.GET("/metrics", GetMetrics)
	router.GET("/alerts", GetAlerts)
	router.GET("/alert-rules", GetAlertRules)
	router.POST("/alert-rules", CreateAlertRule)
}
