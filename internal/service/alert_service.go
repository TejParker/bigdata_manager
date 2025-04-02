package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"bigdata_manager/internal/model"
	"bigdata_manager/internal/repository"
)

// AlertService 处理告警规则和事件
type AlertService struct {
	db              *gorm.DB
	alertRuleRepo   *repository.AlertRuleRepository
	alertEventRepo  *repository.AlertEventRepository
	notificationSvc *NotificationService
}

// NewAlertService 创建新的告警服务
func NewAlertService(db *gorm.DB, notificationSvc *NotificationService) *AlertService {
	return &AlertService{
		db:              db,
		alertRuleRepo:   repository.NewAlertRuleRepository(db),
		alertEventRepo:  repository.NewAlertEventRepository(db),
		notificationSvc: notificationSvc,
	}
}

// CreateAlertRule 创建告警规则
func (s *AlertService) CreateAlertRule(rule *model.AlertRule) error {
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	return s.alertRuleRepo.Create(rule)
}

// UpdateAlertRule 更新告警规则
func (s *AlertService) UpdateAlertRule(rule *model.AlertRule) error {
	rule.UpdatedAt = time.Now()
	return s.alertRuleRepo.Update(rule)
}

// DeleteAlertRule 删除告警规则
func (s *AlertService) DeleteAlertRule(id uint) error {
	return s.alertRuleRepo.Delete(id)
}

// GetAlertRule 获取告警规则
func (s *AlertService) GetAlertRule(id uint) (*model.AlertRule, error) {
	return s.alertRuleRepo.GetByID(id)
}

// ListAlertRules 列出告警规则
func (s *AlertService) ListAlertRules(page, pageSize int, filters map[string]interface{}) ([]*model.AlertRule, int64, error) {
	return s.alertRuleRepo.List(page, pageSize, filters)
}

// ProcessMetric 处理指标数据，检查是否触发告警
func (s *AlertService) ProcessMetric(ctx context.Context, hostID uint, hostname string, serviceID *uint, serviceName string, metricName string, value float64) error {
	// 查找适用的规则
	rules, err := s.alertRuleRepo.FindApplicableRules(hostID, serviceID, metricName)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		// 检查规则是否启用
		if !rule.Enabled {
			continue
		}

		// 检查是否满足告警条件
		triggered := s.evaluateRule(rule, value)
		if triggered {
			// 创建告警事件
			err = s.createAlertEvent(ctx, rule, hostID, hostname, serviceID, serviceName, metricName, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// evaluateRule 评估告警规则是否触发
func (s *AlertService) evaluateRule(rule *model.AlertRule, value float64) bool {
	switch rule.Operator {
	case model.OpGreaterThan:
		return value > rule.Threshold
	case model.OpGreaterThanOrEqual:
		return value >= rule.Threshold
	case model.OpLessThan:
		return value < rule.Threshold
	case model.OpLessThanOrEqual:
		return value <= rule.Threshold
	case model.OpEqual:
		return value == rule.Threshold
	case model.OpNotEqual:
		return value != rule.Threshold
	default:
		return false
	}
}

// createAlertEvent 创建告警事件并发送通知
func (s *AlertService) createAlertEvent(ctx context.Context, rule *model.AlertRule, hostID uint, hostname string, serviceID *uint, serviceName string, metricName string, value float64) error {
	// 检查是否已存在未解决的相同告警
	existingAlert, err := s.alertEventRepo.FindOpenAlertByRule(rule.ID, hostID, serviceID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 如果已存在未解决的告警，不重复创建
	if existingAlert != nil {
		return nil
	}

	// 创建新的告警事件
	alertEvent := &model.AlertEvent{
		AlertRuleID: rule.ID,
		AlertName:   rule.Name,
		HostID:      &hostID,
		Hostname:    hostname,
		ServiceID:   serviceID,
		ServiceName: serviceName,
		MetricName:  metricName,
		MetricValue: value,
		Threshold:   rule.Threshold,
		Operator:    string(rule.Operator),
		Message:     fmt.Sprintf("%s: %s %.2f %s %.2f", rule.Name, metricName, value, rule.Operator, rule.Threshold),
		Severity:    rule.Severity,
		Status:      model.AlertStatusOpen,
		TriggeredAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if rule.ClusterID != nil {
		alertEvent.ClusterID = rule.ClusterID
	}

	// 保存告警事件
	err = s.alertEventRepo.Create(alertEvent)
	if err != nil {
		return err
	}

	// 发送通知
	if rule.NotificationIDs != "" {
		notificationIDs := strings.Split(rule.NotificationIDs, ",")
		for _, idStr := range notificationIDs {
			id, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				continue
			}
			s.notificationSvc.SendAlertNotification(ctx, uint(id), alertEvent)
		}
	}

	return nil
}

// GetAlertEvent 获取告警事件
func (s *AlertService) GetAlertEvent(id uint) (*model.AlertEvent, error) {
	return s.alertEventRepo.GetByID(id)
}

// ListAlertEvents 列出告警事件
func (s *AlertService) ListAlertEvents(page, pageSize int, filters map[string]interface{}) ([]*model.AlertEvent, int64, error) {
	return s.alertEventRepo.List(page, pageSize, filters)
}

// AcknowledgeAlertEvent 确认告警事件
func (s *AlertService) AcknowledgeAlertEvent(id uint, userID uint) error {
	event, err := s.GetAlertEvent(id)
	if err != nil {
		return err
	}

	if event.Status != model.AlertStatusOpen {
		return errors.New("only open alerts can be acknowledged")
	}

	now := time.Now()
	event.Status = model.AlertStatusAcknowledged
	event.AcknowledgedAt = &now
	event.AcknowledgedBy = &userID
	event.UpdatedAt = now

	return s.alertEventRepo.Update(event)
}

// ResolveAlertEvent 解决告警事件
func (s *AlertService) ResolveAlertEvent(id uint) error {
	event, err := s.GetAlertEvent(id)
	if err != nil {
		return err
	}

	if event.Status == model.AlertStatusResolved {
		return errors.New("alert is already resolved")
	}

	now := time.Now()
	event.Status = model.AlertStatusResolved
	event.ResolvedAt = &now
	event.UpdatedAt = now

	return s.alertEventRepo.Update(event)
}

// GetAlertStatistics 获取告警统计信息
func (s *AlertService) GetAlertStatistics() (map[string]interface{}, error) {
	// 获取各个级别的告警数量
	severityCounts, err := s.alertEventRepo.CountBySeverity()
	if err != nil {
		return nil, err
	}

	// 获取各个状态的告警数量
	statusCounts, err := s.alertEventRepo.CountByStatus()
	if err != nil {
		return nil, err
	}

	// 获取今日的告警数量
	todayStart := time.Now().Truncate(24 * time.Hour)
	todayCount, err := s.alertEventRepo.CountByTimeRange(todayStart, time.Now())
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"by_severity": severityCounts,
		"by_status":   statusCounts,
		"today":       todayCount,
		"total":       statusCounts["OPEN"] + statusCounts["ACKNOWLEDGED"] + statusCounts["RESOLVED"],
	}, nil
} 