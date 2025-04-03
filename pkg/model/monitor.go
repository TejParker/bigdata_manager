package model

import "time"

// MetricData 指标数据
type AlertMetricData struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// Alert 告警信息
type AlertMonitor struct {
	ID         int64     `json:"id"`
	RuleID     int64     `json:"rule_id"`
	HostID     int       `json:"host_id"`
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	Message    string    `json:"message"`
	Severity   string    `json:"severity"` // INFO, WARNING, CRITICAL
	Timestamp  time.Time `json:"timestamp"`
	Status     string    `json:"status"` // ACTIVE, RESOLVED
}

// AlertRule 告警规则
type AlertRule struct {
	ID         int64     `json:"id"`
	MetricName string    `json:"metric_name"`
	Operator   string    `json:"operator"` // >, <, =
	Threshold  float64   `json:"threshold"`
	Message    string    `json:"message"`
	Severity   string    `json:"severity"` // INFO, WARNING, CRITICAL
	CreatedAt  time.Time `json:"created_at"`
}
