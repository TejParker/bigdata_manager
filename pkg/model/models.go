package model

import (
	"time"
)

// Cluster 集群模型
type Cluster struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Host 主机模型
type Host struct {
	ID            int       `json:"id"`
	Hostname      string    `json:"hostname"`
	IP            string    `json:"ip"`
	ClusterID     int       `json:"cluster_id"`
	CPUCores      int       `json:"cpu_cores"`
	MemorySize    int64     `json:"memory_size"`
	Status        string    `json:"status"` // ONLINE, OFFLINE, MAINTENANCE
	AgentVersion  string    `json:"agent_version"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Service 服务模型
type Service struct {
	ID          int       `json:"id"`
	ClusterID   int       `json:"cluster_id"`
	ServiceType string    `json:"service_type"`
	ServiceName string    `json:"service_name"`
	Version     string    `json:"version"`
	Status      string    `json:"status"` // INSTALLING, RUNNING, STOPPED, ERROR
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ServiceComponent 服务组件模型
type ServiceComponent struct {
	ID               int       `json:"id"`
	ServiceID        int       `json:"service_id"`
	ComponentType    string    `json:"component_type"`
	DesiredInstances int       `json:"desired_instances"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// HostComponent 主机组件映射模型
type HostComponent struct {
	ID          int       `json:"id"`
	HostID      int       `json:"host_id"`
	ComponentID int       `json:"component_id"`
	Status      string    `json:"status"` // INSTALLING, RUNNING, STOPPED, ERROR
	ProcessID   int       `json:"process_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Config 配置模型
type Config struct {
	ID          int       `json:"id"`
	ScopeType   string    `json:"scope_type"` // CLUSTER, SERVICE, COMPONENT, HOST
	ScopeID     int       `json:"scope_id"`
	ConfigKey   string    `json:"config_key"`
	ConfigValue string    `json:"config_value"`
	Version     int       `json:"version"`
	IsCurrent   bool      `json:"is_current"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PackageRepo 软件包模型
type PackageRepo struct {
	ID            int       `json:"id"`
	ComponentType string    `json:"component_type"`
	Version       string    `json:"version"`
	DownloadURL   string    `json:"download_url"`
	Path          string    `json:"path"`
	Checksum      string    `json:"checksum"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Task 任务模型
type Task struct {
	ID          int       `json:"id"`
	TaskType    string    `json:"task_type"`
	RelatedID   int       `json:"related_id"`
	RelatedType string    `json:"related_type"`
	Status      string    `json:"status"` // PENDING, RUNNING, SUCCESS, FAILED
	Progress    int       `json:"progress"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Metric 指标数据模型
type Metric struct {
	ID         int64     `json:"id"`
	HostID     int       `json:"host_id"`
	ServiceID  int       `json:"service_id"`
	MetricName string    `json:"metric_name"`
	Timestamp  time.Time `json:"timestamp"`
	Value      float64   `json:"value"`
	CreatedAt  time.Time `json:"created_at"`
}

// MetricDefinition 指标定义模型
type MetricDefinition struct {
	ID          int       `json:"id"`
	MetricName  string    `json:"metric_name"`
	DisplayName string    `json:"display_name"`
	Category    string    `json:"category"`
	Unit        string    `json:"unit"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LogRecord 日志记录模型
type LogRecord struct {
	ID          int64     `json:"id"`
	HostID      int       `json:"host_id"`
	ServiceID   int       `json:"service_id"`
	ComponentID int       `json:"component_id"`
	LogLevel    string    `json:"log_level"`
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
}

// Alert 告警规则模型
type Alert struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	MetricName         string    `json:"metric_name"`
	Condition          string    `json:"condition"`
	Threshold          float64   `json:"threshold"`
	Duration           int       `json:"duration"`
	Severity           string    `json:"severity"` // INFO, WARNING, CRITICAL
	NotificationMethod string    `json:"notification_method"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// AlertEvent 告警事件模型
type AlertEvent struct {
	ID          int64     `json:"id"`
	AlertID     int       `json:"alert_id"`
	HostID      int       `json:"host_id"`
	ServiceID   int       `json:"service_id"`
	Status      string    `json:"status"` // OPEN, ACKNOWLEDGED, RESOLVED
	TriggeredAt time.Time `json:"triggered_at"`
	ResolvedAt  time.Time `json:"resolved_at"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// User 用户模型
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // 不输出到JSON
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Status       string    `json:"status"` // ACTIVE, DISABLED
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Role 角色模型
type Role struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserRole 用户角色关联模型
type UserRole struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	RoleID    int       `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Privilege 权限模型
type Privilege struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// RolePrivilege 角色权限关联模型
type RolePrivilege struct {
	ID          int       `json:"id"`
	RoleID      int       `json:"role_id"`
	PrivilegeID int       `json:"privilege_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// 响应模型
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 分页响应模型
type PageResponse struct {
	Success  bool        `json:"success"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data,omitempty"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// 登录请求模型
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 登录响应模型
type LoginResponse struct {
	Token  string `json:"token"`
	UserID int    `json:"user_id"`
}

// Agent心跳请求模型
type HeartbeatRequest struct {
	HostID      int               `json:"host_id"`
	Timestamp   time.Time         `json:"timestamp"`
	CPUUsage    float64           `json:"cpu_usage"`
	MemoryUsage float64           `json:"memory_usage"`
	DiskUsage   float64           `json:"disk_usage"`
	Metrics     []MetricData      `json:"metrics,omitempty"`
	Components  []ComponentStatus `json:"components,omitempty"`
}

// 指标数据模型
type MetricData struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// 组件状态模型
type ComponentStatus struct {
	ComponentID int    `json:"component_id"`
	Status      string `json:"status"`
	ProcessID   int    `json:"process_id"`
	Message     string `json:"message,omitempty"`
}

// Agent命令请求模型
type AgentCommand struct {
	CommandID string         `json:"command_id"`
	Type      string         `json:"type"` // INSTALL, START, STOP, CONFIGURE
	Payload   map[string]any `json:"payload"`
}

// Agent命令响应模型
type AgentCommandResponse struct {
	CommandID string `json:"command_id"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Result    any    `json:"result,omitempty"`
}
