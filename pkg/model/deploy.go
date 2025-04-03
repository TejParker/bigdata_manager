package model

import "time"

// Component 组件定义
type Component struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	PackageURL  string    `json:"package_url"`
	InstallPath string    `json:"install_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Deployment 部署记录
type Deployment struct {
	ID          string    `json:"id"`
	HostID      int       `json:"host_id"`
	ComponentID int       `json:"component_id"`
	Status      string    `json:"status"` // PENDING, INSTALLING, INSTALLED, FAILED
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
}
