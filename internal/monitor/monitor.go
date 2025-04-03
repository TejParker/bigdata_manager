package monitor

import (
	"log"
	"sync"
	"time"

	"github.com/TejParker/bigdata-manager/pkg/model"
)

// MonitorService 监控服务
type MonitorService struct {
	metrics         map[int][]model.MetricData // 按主机ID存储指标数据
	alerts          []model.AlertMonitor       // 告警列表
	alertRules      []model.AlertRule          // 告警规则
	metricsLock     sync.RWMutex               // 指标数据锁
	alertsLock      sync.RWMutex               // 告警锁
	alertRulesLock  sync.RWMutex               // 告警规则锁
	retentionPeriod time.Duration              // 数据保留时间
}

// NewMonitorService 创建新的监控服务
func NewMonitorService(retentionHours int) *MonitorService {
	if retentionHours <= 0 {
		retentionHours = 24 // 默认保留24小时
	}

	service := &MonitorService{
		metrics:         make(map[int][]model.MetricData),
		retentionPeriod: time.Duration(retentionHours) * time.Hour,
	}

	// 启动定期清理任务
	go service.startCleanupTask()

	return service
}

// StoreMetrics 存储指标数据
func (s *MonitorService) StoreMetrics(hostID int, metrics []model.MetricData) {
	s.metricsLock.Lock()
	defer s.metricsLock.Unlock()

	// 添加新的指标数据
	s.metrics[hostID] = append(s.metrics[hostID], metrics...)
}

// GetMetrics 获取指标数据
func (s *MonitorService) GetMetrics(hostID int, metricName string, startTime, endTime time.Time) []model.MetricData {
	s.metricsLock.RLock()
	defer s.metricsLock.RUnlock()

	var result []model.MetricData

	// 如果未指定主机ID，返回所有主机的数据
	if hostID == 0 {
		for _, hostMetrics := range s.metrics {
			for _, metric := range hostMetrics {
				if (metricName == "" || metric.Name == metricName) &&
					(metric.Timestamp.After(startTime) && metric.Timestamp.Before(endTime)) {
					result = append(result, metric)
				}
			}
		}
		return result
	}

	// 指定主机的数据
	hostMetrics, exists := s.metrics[hostID]
	if !exists {
		return nil
	}

	for _, metric := range hostMetrics {
		if (metricName == "" || metric.Name == metricName) &&
			(metric.Timestamp.After(startTime) && metric.Timestamp.Before(endTime)) {
			result = append(result, metric)
		}
	}

	return result
}

// AddAlertRule 添加告警规则
func (s *MonitorService) AddAlertRule(rule model.AlertRule) {
	s.alertRulesLock.Lock()
	defer s.alertRulesLock.Unlock()

	s.alertRules = append(s.alertRules, rule)
}

// GetAlertRules 获取告警规则
func (s *MonitorService) GetAlertRules() []model.AlertRule {
	s.alertRulesLock.RLock()
	defer s.alertRulesLock.RUnlock()

	return s.alertRules
}

// GetAlerts 获取告警列表
func (s *MonitorService) GetAlerts(startTime, endTime time.Time) []model.AlertMonitor {
	s.alertsLock.RLock()
	defer s.alertsLock.RUnlock()

	var result []model.AlertMonitor
	for _, alert := range s.alerts {
		if alert.Timestamp.After(startTime) && alert.Timestamp.Before(endTime) {
			result = append(result, alert)
		}
	}

	return result
}

// CheckAlerts 检查是否需要触发告警
func (s *MonitorService) CheckAlerts() {
	s.metricsLock.RLock()
	metrics := s.metrics
	s.metricsLock.RUnlock()

	s.alertRulesLock.RLock()
	rules := s.alertRules
	s.alertRulesLock.RUnlock()

	now := time.Now()
	var newAlerts []model.AlertMonitor

	// 检查每个告警规则
	for _, rule := range rules {
		for hostID, hostMetrics := range metrics {
			// 只检查最近的指标数据
			recentMetrics := getRecentMetrics(hostMetrics, rule.MetricName, now.Add(-5*time.Minute), now)
			if len(recentMetrics) == 0 {
				continue
			}

			// 比较最新的指标值与告警阈值
			latestMetric := recentMetrics[len(recentMetrics)-1]
			if (rule.Operator == ">" && latestMetric.Value > rule.Threshold) ||
				(rule.Operator == "<" && latestMetric.Value < rule.Threshold) ||
				(rule.Operator == "=" && latestMetric.Value == rule.Threshold) {
				// 触发告警
				alert := model.AlertMonitor{
					RuleID:     rule.ID,
					HostID:     hostID,
					MetricName: rule.MetricName,
					Value:      latestMetric.Value,
					Threshold:  rule.Threshold,
					Message:    rule.Message,
					Severity:   rule.Severity,
					Timestamp:  now,
					Status:     "ACTIVE",
				}
				newAlerts = append(newAlerts, alert)
			}
		}
	}

	// 添加新的告警
	if len(newAlerts) > 0 {
		s.alertsLock.Lock()
		s.alerts = append(s.alerts, newAlerts...)
		s.alertsLock.Unlock()
	}
}

// startCleanupTask 启动定期清理过期数据的任务
func (s *MonitorService) startCleanupTask() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		s.cleanupExpiredData()
	}
}

// cleanupExpiredData 清理过期的指标数据和已解决的告警
func (s *MonitorService) cleanupExpiredData() {
	cutoffTime := time.Now().Add(-s.retentionPeriod)

	// 清理过期的指标数据
	s.metricsLock.Lock()
	for hostID, metrics := range s.metrics {
		var newMetrics []model.MetricData
		for _, metric := range metrics {
			if metric.Timestamp.After(cutoffTime) {
				newMetrics = append(newMetrics, metric)
			}
		}
		if len(newMetrics) > 0 {
			s.metrics[hostID] = newMetrics
		} else {
			delete(s.metrics, hostID)
		}
	}
	s.metricsLock.Unlock()

	// 清理过期的告警
	s.alertsLock.Lock()
	var newAlerts []model.AlertMonitor
	for _, alert := range s.alerts {
		if alert.Status == "RESOLVED" && alert.Timestamp.Before(cutoffTime) {
			continue
		}
		newAlerts = append(newAlerts, alert)
	}
	s.alerts = newAlerts
	s.alertsLock.Unlock()

	log.Printf("清理了过期的监控数据，当前存储: %d个主机的指标, %d个告警",
		len(s.metrics), len(s.alerts))
}

// 获取指定时间范围内的指标数据
func getRecentMetrics(metrics []model.MetricData, metricName string, startTime, endTime time.Time) []model.MetricData {
	var result []model.MetricData
	for _, metric := range metrics {
		if metric.Name == metricName && metric.Timestamp.After(startTime) && metric.Timestamp.Before(endTime) {
			result = append(result, metric)
		}
	}
	return result
}

// ServiceInstance 监控服务的单例实例
var ServiceInstance *MonitorService
var once sync.Once

// GetMonitorService 获取监控服务实例
func GetMonitorService() *MonitorService {
	once.Do(func() {
		ServiceInstance = NewMonitorService(24)
	})
	return ServiceInstance
}
