package config

// Config 应用程序配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Email    EmailConfig    `mapstructure:"email"`    // 新增邮件配置
	SMS      SMSConfig      `mapstructure:"sms"`      // 新增短信配置
	Alert    AlertConfig    `mapstructure:"alert"`    // 新增告警配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port      int    `mapstructure:"port"`
	JWTSecret string `mapstructure:"jwt_secret"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	TokenExpiration int `mapstructure:"token_expiration"` // 单位分钟
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Output string `mapstructure:"output"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	SMTPServer string `mapstructure:"smtp_server"`
	SMTPPort   int    `mapstructure:"smtp_port"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	From       string `mapstructure:"from"`
}

// SMSConfig 短信配置
type SMSConfig struct {
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	Secret   string `mapstructure:"secret"`
}

// AlertConfig 告警配置
type AlertConfig struct {
	ProcessInterval  int `mapstructure:"process_interval"`  // 告警处理间隔，单位秒
	RetentionDays    int `mapstructure:"retention_days"`    // 告警保留天数
	MaxNotifications int `mapstructure:"max_notifications"` // 每个告警最大通知次数
} 