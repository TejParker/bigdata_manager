package repository

import (
	"github.com/TejParker/bigdata-manager/internal/model"
	"time"

	"gorm.io/gorm"
)

// AlertRuleRepository 告警规则仓库
type AlertRuleRepository struct {
	db *gorm.DB
}

// NewAlertRuleRepository 创建告警规则仓库
func NewAlertRuleRepository(db *gorm.DB) *AlertRuleRepository {
	return &AlertRuleRepository{db: db}
}

// Create 创建告警规则
func (r *AlertRuleRepository) Create(rule *model.AlertRule) error {
	return r.db.Create(rule).Error
}

// Update 更新告警规则
func (r *AlertRuleRepository) Update(rule *model.AlertRule) error {
	return r.db.Save(rule).Error
}

// Delete 删除告警规则
func (r *AlertRuleRepository) Delete(id uint) error {
	return r.db.Delete(&model.AlertRule{}, id).Error
}

// GetByID 根据ID获取告警规则
func (r *AlertRuleRepository) GetByID(id uint) (*model.AlertRule, error) {
	var rule model.AlertRule
	err := r.db.First(&rule, id).Error
	return &rule, err
}

// List 列出告警规则
func (r *AlertRuleRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.AlertRule, int64, error) {
	var rules []*model.AlertRule
	var total int64

	query := r.db.Model(&model.AlertRule{})

	// 应用过滤条件
	for key, value := range filters {
		if value != nil && value != "" {
			query = query.Where(key, value)
		}
	}

	// 统计总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	// 排序
	query = query.Order("id DESC")

	// 执行查询
	err = query.Find(&rules).Error
	if err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// FindApplicableRules 查找适用于指定主机、服务和指标的规则
func (r *AlertRuleRepository) FindApplicableRules(hostID uint, serviceID *uint, metricName string) ([]*model.AlertRule, error) {
	var rules []*model.AlertRule

	query := r.db.Model(&model.AlertRule{}).Where("metric_name = ? AND enabled = ?", metricName, true)

	// 策略1: 针对特定主机的规则
	query1 := query.Where("host_id = ?", hostID)

	// 策略2: 针对特定服务的规则（如果有serviceID）
	var query2 *gorm.DB
	if serviceID != nil {
		query2 = query.Where("service_id = ?", *serviceID)
	}

	// 策略3: 全局规则（没有指定主机和服务）
	query3 := query.Where("host_id IS NULL AND service_id IS NULL")

	// 合并查询结果
	if err := query1.Find(&rules).Error; err != nil {
		return nil, err
	}

	if serviceID != nil {
		var serviceRules []*model.AlertRule
		if err := query2.Find(&serviceRules).Error; err != nil {
			return nil, err
		}
		rules = append(rules, serviceRules...)
	}

	var globalRules []*model.AlertRule
	if err := query3.Find(&globalRules).Error; err != nil {
		return nil, err
	}
	rules = append(rules, globalRules...)

	return rules, nil
}

// AlertEventRepository 告警事件仓库
type AlertEventRepository struct {
	db *gorm.DB
}

// NewAlertEventRepository 创建告警事件仓库
func NewAlertEventRepository(db *gorm.DB) *AlertEventRepository {
	return &AlertEventRepository{db: db}
}

// Create 创建告警事件
func (r *AlertEventRepository) Create(event *model.AlertEvent) error {
	return r.db.Create(event).Error
}

// Update 更新告警事件
func (r *AlertEventRepository) Update(event *model.AlertEvent) error {
	return r.db.Save(event).Error
}

// GetByID 根据ID获取告警事件
func (r *AlertEventRepository) GetByID(id uint) (*model.AlertEvent, error) {
	var event model.AlertEvent
	err := r.db.First(&event, id).Error
	return &event, err
}

// List 列出告警事件
func (r *AlertEventRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.AlertEvent, int64, error) {
	var events []*model.AlertEvent
	var total int64

	query := r.db.Model(&model.AlertEvent{})

	// 应用过滤条件
	for key, value := range filters {
		if value != nil && value != "" {
			query = query.Where(key, value)
		}
	}

	// 统计总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	// 排序
	query = query.Order("triggered_at DESC")

	// 执行查询
	err = query.Find(&events).Error
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// FindOpenAlertByRule 查找指定规则的未解决告警
func (r *AlertEventRepository) FindOpenAlertByRule(ruleID, hostID uint, serviceID *uint) (*model.AlertEvent, error) {
	var event model.AlertEvent

	query := r.db.Where(
		"alert_rule_id = ? AND host_id = ? AND status != ?",
		ruleID,
		hostID,
		model.AlertStatusResolved,
	)

	if serviceID != nil {
		query = query.Where("service_id = ?", *serviceID)
	} else {
		query = query.Where("service_id IS NULL")
	}

	err := query.First(&event).Error
	if err != nil {
		return nil, err
	}

	return &event, nil
}

// CountBySeverity 按告警级别统计
func (r *AlertEventRepository) CountBySeverity() (map[string]int64, error) {
	type Result struct {
		Severity string
		Count    int64
	}
	var results []Result

	err := r.db.Model(&model.AlertEvent{}).
		Select("severity as severity, count(*) as count").
		Group("severity").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, result := range results {
		counts[result.Severity] = result.Count
	}

	return counts, nil
}

// CountByStatus 按告警状态统计
func (r *AlertEventRepository) CountByStatus() (map[string]int64, error) {
	type Result struct {
		Status string
		Count  int64
	}
	var results []Result

	err := r.db.Model(&model.AlertEvent{}).
		Select("status as status, count(*) as count").
		Group("status").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, result := range results {
		counts[result.Status] = result.Count
	}

	return counts, nil
}

// CountByTimeRange 按时间范围统计
func (r *AlertEventRepository) CountByTimeRange(start, end time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.AlertEvent{}).
		Where("triggered_at BETWEEN ? AND ?", start, end).
		Count(&count).Error

	return count, err
}
