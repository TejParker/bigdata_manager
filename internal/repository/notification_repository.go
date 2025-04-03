package repository

import (
	"github.com/TejParker/bigdata-manager/internal/model"
	"gorm.io/gorm"
)

// NotificationConfigRepository 通知配置仓库
type NotificationConfigRepository struct {
	db *gorm.DB
}

// NewNotificationConfigRepository 创建通知配置仓库
func NewNotificationConfigRepository(db *gorm.DB) *NotificationConfigRepository {
	return &NotificationConfigRepository{db: db}
}

// Create 创建通知配置
func (r *NotificationConfigRepository) Create(config *model.NotificationConfig) error {
	return r.db.Create(config).Error
}

// Update 更新通知配置
func (r *NotificationConfigRepository) Update(config *model.NotificationConfig) error {
	return r.db.Save(config).Error
}

// Delete 删除通知配置
func (r *NotificationConfigRepository) Delete(id uint) error {
	return r.db.Delete(&model.NotificationConfig{}, id).Error
}

// GetByID 根据ID获取通知配置
func (r *NotificationConfigRepository) GetByID(id uint) (*model.NotificationConfig, error) {
	var config model.NotificationConfig
	err := r.db.First(&config, id).Error
	return &config, err
}

// List 列出通知配置
func (r *NotificationConfigRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.NotificationConfig, int64, error) {
	var configs []*model.NotificationConfig
	var total int64

	query := r.db.Model(&model.NotificationConfig{})

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
	err = query.Find(&configs).Error
	if err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

// NotificationHistoryRepository 通知历史仓库
type NotificationHistoryRepository struct {
	db *gorm.DB
}

// NewNotificationHistoryRepository 创建通知历史仓库
func NewNotificationHistoryRepository(db *gorm.DB) *NotificationHistoryRepository {
	return &NotificationHistoryRepository{db: db}
}

// Create 创建通知历史
func (r *NotificationHistoryRepository) Create(history *model.NotificationHistory) error {
	return r.db.Create(history).Error
}

// GetByID 根据ID获取通知历史
func (r *NotificationHistoryRepository) GetByID(id uint) (*model.NotificationHistory, error) {
	var history model.NotificationHistory
	err := r.db.First(&history, id).Error
	return &history, err
}

// ListByAlertEvent 列出告警事件的通知历史
func (r *NotificationHistoryRepository) ListByAlertEvent(alertEventID uint) ([]*model.NotificationHistory, error) {
	var histories []*model.NotificationHistory
	err := r.db.Where("alert_event_id = ?", alertEventID).
		Order("sent_at DESC").
		Find(&histories).Error
	return histories, err
}

// CountByAlertEvent 统计告警事件的通知次数
func (r *NotificationHistoryRepository) CountByAlertEvent(alertEventID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.NotificationHistory{}).
		Where("alert_event_id = ?", alertEventID).
		Count(&count).Error
	return count, err
}

// CountSuccessByConfig 统计通知配置的成功次数
func (r *NotificationHistoryRepository) CountSuccessByConfig(configID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.NotificationHistory{}).
		Where("notification_config_id = ? AND status = ?", configID, "SUCCESS").
		Count(&count).Error
	return count, err
}

// CountFailureByConfig 统计通知配置的失败次数
func (r *NotificationHistoryRepository) CountFailureByConfig(configID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.NotificationHistory{}).
		Where("notification_config_id = ? AND status = ?", configID, "FAILED").
		Count(&count).Error
	return count, err
}
