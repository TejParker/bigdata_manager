package model

import (
	"time"
)

// 告警级别
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "CRITICAL"
	SeverityWarning  AlertSeverity = "WARNING"
	SeverityInfo     AlertSeverity = "INFO"
)

// 告警状态
type AlertStatus string

const (
	AlertStatusOpen        AlertStatus = "OPEN"
	AlertStatusAcknowledged AlertStatus = "ACKNOWLEDGED"
	AlertStatusResolved    AlertStatus = "RESOLVED"
)

// 比较操作符
type ComparisonOperator string

const (
	OpGreaterThan        ComparisonOperator = ">"
	OpGreaterThanOrEqual ComparisonOperator = ">="
	OpLessThan           ComparisonOperator = "<"
	OpLessThanOrEqual    ComparisonOperator = "<="
	OpEqual              ComparisonOperator = "=="
	OpNotEqual           ComparisonOperator = "!="
)

// 告警规则
type AlertRule struct {
	ID               uint          `json:"id" gorm:"primaryKey"`
	Name             string        `json:"name" gorm:"size:100;not null"`
	Description      string        `json:"description" gorm:"size:500"`
	MetricName       string        `json:"metric_name" gorm:"size:100;not null"`
	ClusterID        *uint         `json:"cluster_id" gorm:"index"`
	ServiceID        *uint         `json:"service_id" gorm:"index"`
	HostID           *uint         `json:"host_id" gorm:"index"`
	Operator         ComparisonOperator `json:"operator" gorm:"size:10;not null"`
	Threshold        float64       `json:"threshold" gorm:"not null"`
	Duration         int           `json:"duration" gorm:"default:0"` // 持续时间，单位为秒，0表示立即触发
	Severity         AlertSeverity `json:"severity" gorm:"size:20;not null;default:'WARNING'"`
	Enabled          bool          `json:"enabled" gorm:"default:true"`
	NotificationIDs  string        `json:"notification_ids" gorm:"size:255"` // 逗号分隔的通知ID
	CreatedBy        uint          `json:"created_by"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// 告警事件
type AlertEvent struct {
	ID           uint          `json:"id" gorm:"primaryKey"`
	AlertRuleID  uint          `json:"alert_rule_id" gorm:"index;not null"`
	AlertName    string        `json:"alert_name" gorm:"size:100;not null"`
	ClusterID    *uint         `json:"cluster_id" gorm:"index"`
	ServiceID    *uint         `json:"service_id" gorm:"index"`
	HostID       *uint         `json:"host_id" gorm:"index"`
	Hostname     string        `json:"hostname" gorm:"size:100"`
	ServiceName  string        `json:"service_name" gorm:"size:100"`
	MetricName   string        `json:"metric_name" gorm:"size:100;not null"`
	MetricValue  float64       `json:"metric_value"`
	Threshold    float64       `json:"threshold"`
	Operator     string        `json:"operator" gorm:"size:10"`
	Message      string        `json:"message" gorm:"size:500"`
	Severity     AlertSeverity `json:"severity" gorm:"size:20;not null"`
	Status       AlertStatus   `json:"status" gorm:"size:20;not null;default:'OPEN'"`
	TriggeredAt  time.Time     `json:"triggered_at"`
	AcknowledgedAt *time.Time   `json:"acknowledged_at"`
	AcknowledgedBy *uint        `json:"acknowledged_by"`
	ResolvedAt   *time.Time    `json:"resolved_at"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
} 