package model

import (
	"time"
)

// 通知类型
type NotificationType string

const (
	NotificationTypeEmail    NotificationType = "EMAIL"
	NotificationTypeSMS      NotificationType = "SMS"
	NotificationTypeWebhook  NotificationType = "WEBHOOK"
)

// 通知配置
type NotificationConfig struct {
	ID           uint             `json:"id" gorm:"primaryKey"`
	Name         string           `json:"name" gorm:"size:100;not null"`
	Description  string           `json:"description" gorm:"size:500"`
	Type         NotificationType `json:"type" gorm:"size:20;not null"`
	Enabled      bool             `json:"enabled" gorm:"default:true"`
	
	// 邮件配置
	EmailRecipients string `json:"email_recipients" gorm:"size:500"` // 逗号分隔的邮件接收者
	
	// SMS配置
	SMSRecipients string `json:"sms_recipients" gorm:"size:500"` // 逗号分隔的手机号码
	
	// Webhook配置
	WebhookURL    string `json:"webhook_url" gorm:"size:255"`
	WebhookMethod string `json:"webhook_method" gorm:"size:10;default:'POST'"`
	WebhookHeaders string `json:"webhook_headers" gorm:"size:1000"` // JSON格式的Headers
	WebhookTemplate string `json:"webhook_template" gorm:"size:2000"` // 自定义webhook内容模板
	
	// 通用配置
	CreatedBy    uint      `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// 通知历史
type NotificationHistory struct {
	ID                uint             `json:"id" gorm:"primaryKey"`
	AlertEventID      uint             `json:"alert_event_id" gorm:"index;not null"`
	NotificationConfigID uint             `json:"notification_config_id" gorm:"index;not null"`
	Type              NotificationType `json:"type" gorm:"size:20;not null"`
	Status            string           `json:"status" gorm:"size:20;not null"` // SUCCESS, FAILED
	Message           string           `json:"message" gorm:"size:500"`
	Recipient         string           `json:"recipient" gorm:"size:255"`
	SentAt            time.Time        `json:"sent_at"`
	CreatedAt         time.Time        `json:"created_at"`
}