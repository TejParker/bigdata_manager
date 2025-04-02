-- 创建数据库
CREATE DATABASE IF NOT EXISTS bigdata_manager DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE bigdata_manager;

-- 集群表
CREATE TABLE IF NOT EXISTS cluster (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 主机表
CREATE TABLE IF NOT EXISTS host (
    id INT AUTO_INCREMENT PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    ip VARCHAR(128) NOT NULL,
    cluster_id INT NOT NULL,
    cpu_cores INT,
    memory_size BIGINT,
    status ENUM('ONLINE', 'OFFLINE', 'MAINTENANCE') DEFAULT 'OFFLINE',
    agent_version VARCHAR(32),
    last_heartbeat TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (cluster_id) REFERENCES cluster(id) ON DELETE CASCADE,
    UNIQUE KEY (hostname, cluster_id)
);

-- 服务表
CREATE TABLE IF NOT EXISTS service (
    id INT AUTO_INCREMENT PRIMARY KEY,
    cluster_id INT NOT NULL,
    service_type VARCHAR(32) NOT NULL,
    service_name VARCHAR(128) NOT NULL,
    version VARCHAR(32),
    status ENUM('INSTALLING', 'RUNNING', 'STOPPED', 'ERROR') DEFAULT 'INSTALLING',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (cluster_id) REFERENCES cluster(id) ON DELETE CASCADE,
    UNIQUE KEY (service_name, cluster_id)
);

-- 服务组件表
CREATE TABLE IF NOT EXISTS service_component (
    id INT AUTO_INCREMENT PRIMARY KEY,
    service_id INT NOT NULL,
    component_type VARCHAR(64) NOT NULL,
    desired_instances INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE,
    UNIQUE KEY (service_id, component_type)
);

-- 主机组件映射表
CREATE TABLE IF NOT EXISTS host_component (
    id INT AUTO_INCREMENT PRIMARY KEY,
    host_id INT NOT NULL,
    component_id INT NOT NULL,
    status ENUM('INSTALLING', 'RUNNING', 'STOPPED', 'ERROR') DEFAULT 'INSTALLING',
    process_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (host_id) REFERENCES host(id) ON DELETE CASCADE,
    FOREIGN KEY (component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    UNIQUE KEY (host_id, component_id)
);

-- 配置表
CREATE TABLE IF NOT EXISTS config (
    id INT AUTO_INCREMENT PRIMARY KEY,
    scope_type ENUM('CLUSTER', 'SERVICE', 'COMPONENT', 'HOST') NOT NULL,
    scope_id INT NOT NULL,
    config_key VARCHAR(255) NOT NULL,
    config_value TEXT,
    version INT DEFAULT 1,
    is_current BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY (scope_type, scope_id, config_key, version)
);

-- 软件仓库表
CREATE TABLE IF NOT EXISTS package_repo (
    id INT AUTO_INCREMENT PRIMARY KEY,
    component_type VARCHAR(64) NOT NULL,
    version VARCHAR(32) NOT NULL,
    download_url VARCHAR(255),
    path VARCHAR(255),
    checksum VARCHAR(128),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY (component_type, version)
);

-- 任务表
CREATE TABLE IF NOT EXISTS task (
    id INT AUTO_INCREMENT PRIMARY KEY,
    task_type VARCHAR(32) NOT NULL,
    related_id INT,
    related_type VARCHAR(32),
    status ENUM('PENDING', 'RUNNING', 'SUCCESS', 'FAILED') DEFAULT 'PENDING',
    progress INT DEFAULT 0,
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 指标数据表
CREATE TABLE IF NOT EXISTS metric (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    host_id INT,
    service_id INT,
    metric_name VARCHAR(128) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    value DOUBLE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_host_metric_time (host_id, metric_name, timestamp),
    INDEX idx_service_metric_time (service_id, metric_name, timestamp)
);

-- 指标定义表
CREATE TABLE IF NOT EXISTS metric_definition (
    id INT AUTO_INCREMENT PRIMARY KEY,
    metric_name VARCHAR(128) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    category VARCHAR(32),
    unit VARCHAR(32),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY (metric_name)
);

-- 日志记录表
CREATE TABLE IF NOT EXISTS log_record (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    host_id INT,
    service_id INT,
    component_id INT,
    log_level VARCHAR(16),
    timestamp TIMESTAMP NOT NULL,
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_host_time (host_id, timestamp),
    INDEX idx_service_time (service_id, timestamp),
    INDEX idx_level_time (log_level, timestamp)
);

-- 告警规则表
CREATE TABLE IF NOT EXISTS alert (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    metric_name VARCHAR(128),
    condition VARCHAR(255) NOT NULL,
    threshold DOUBLE,
    duration INT DEFAULT 0,
    severity ENUM('INFO', 'WARNING', 'CRITICAL') DEFAULT 'WARNING',
    notification_method VARCHAR(32),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 告警事件表
CREATE TABLE IF NOT EXISTS alert_event (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    alert_id INT NOT NULL,
    host_id INT,
    service_id INT,
    status ENUM('OPEN', 'ACKNOWLEDGED', 'RESOLVED') DEFAULT 'OPEN',
    triggered_at TIMESTAMP NOT NULL,
    resolved_at TIMESTAMP,
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (alert_id) REFERENCES alert(id) ON DELETE CASCADE,
    INDEX idx_status_time (status, triggered_at)
);

-- 用户表
CREATE TABLE IF NOT EXISTS user (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(128),
    phone VARCHAR(32),
    status ENUM('ACTIVE', 'DISABLED') DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY (username)
);

-- 角色表
CREATE TABLE IF NOT EXISTS role (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY (name)
);

-- 用户角色映射表
CREATE TABLE IF NOT EXISTS user_role (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    role_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE,
    UNIQUE KEY (user_id, role_id)
);

-- 权限表
CREATE TABLE IF NOT EXISTS privilege (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY (name)
);

-- 角色权限映射表
CREATE TABLE IF NOT EXISTS role_privilege (
    id INT AUTO_INCREMENT PRIMARY KEY,
    role_id INT NOT NULL,
    privilege_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE,
    FOREIGN KEY (privilege_id) REFERENCES privilege(id) ON DELETE CASCADE,
    UNIQUE KEY (role_id, privilege_id)
);

-- 插入基本角色
INSERT INTO role (name, description) VALUES 
('ADMIN', '管理员，拥有所有权限'),
('OPERATOR', '运维人员，有日常操作权限'),
('OBSERVER', '只读用户，只能查看不能修改');

-- 插入基本权限
INSERT INTO privilege (name, description) VALUES
('VIEW_CLUSTER', '查看集群信息'),
('MANAGE_CLUSTER', '管理集群'),
('VIEW_SERVICE', '查看服务信息'),
('MANAGE_SERVICE', '管理服务（启动/停止/配置）'),
('VIEW_HOST', '查看主机信息'),
('MANAGE_HOST', '管理主机'),
('VIEW_LOG', '查看日志'),
('VIEW_METRIC', '查看监控指标'),
('MANAGE_ALERT', '管理告警规则'),
('VIEW_ALERT', '查看告警'),
('MANAGE_USER', '管理用户和权限');

-- 为角色分配权限
-- 管理员拥有所有权限
INSERT INTO role_privilege (role_id, privilege_id)
SELECT 1, id FROM privilege;

-- 运维人员拥有除用户管理外的所有权限
INSERT INTO role_privilege (role_id, privilege_id)
SELECT 2, id FROM privilege WHERE name != 'MANAGE_USER';

-- 只读用户只有查看权限
INSERT INTO role_privilege (role_id, privilege_id)
SELECT 3, id FROM privilege WHERE name LIKE 'VIEW_%';

-- 创建默认管理员用户 (密码为 admin)
INSERT INTO user (username, password_hash, email) 
VALUES ('admin', '$2a$10$IFW5y1BVnT8RrQrWqKn.de6OvE3.YoHbObo75N9UP7nGzkSUsxcBC', 'admin@example.com');

-- 给默认管理员分配管理员角色
INSERT INTO user_role (user_id, role_id) VALUES (1, 1); 