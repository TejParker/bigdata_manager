package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"gorm.io/gorm"

	"bigdata_manager/internal/config"
	"bigdata_manager/internal/model"
	"bigdata_manager/internal/repository"
)

// NotificationService 处理通知发送
type NotificationService struct {
	db                    *gorm.DB
	notificationConfigRepo *repository.NotificationConfigRepository
	notificationHistoryRepo *repository.NotificationHistoryRepository
	cfg                   *config.Config
	emailTemplate         *template.Template
	smsTemplate           *template.Template
}

// NewNotificationService 创建新的通知服务
func NewNotificationService(db *gorm.DB, cfg *config.Config) *NotificationService {
	// 初始化邮件模板
	emailTpl, _ := template.New("email").Parse(`
Subject: 【告警】{{ .AlertEvent.Severity }} - {{ .AlertEvent.AlertName }}

告警信息:
- 级别: {{ .AlertEvent.Severity }}
- 名称: {{ .AlertEvent.AlertName }}
- 时间: {{ .AlertEvent.TriggeredAt.Format "2006-01-02 15:04:05" }}
- 主机: {{ .AlertEvent.Hostname }}
{{ if .AlertEvent.ServiceName }}- 服务: {{ .AlertEvent.ServiceName }}{{ end }}
- 指标: {{ .AlertEvent.MetricName }}
- 当前值: {{ .AlertEvent.MetricValue }}
- 阈值: {{ .AlertEvent.Operator }} {{ .AlertEvent.Threshold }}
- 详情: {{ .AlertEvent.Message }}

请及时处理!
`)

	// 初始化SMS模板
	smsTpl, _ := template.New("sms").Parse(`【告警】{{ .AlertEvent.Severity }}-{{ .AlertEvent.AlertName }}: {{ .AlertEvent.Hostname }} {{ .AlertEvent.MetricName }} {{ .AlertEvent.MetricValue }} {{ .AlertEvent.Operator }} {{ .AlertEvent.Threshold }}`)

	return &NotificationService{
		db:                    db,
		notificationConfigRepo: repository.NewNotificationConfigRepository(db),
		notificationHistoryRepo: repository.NewNotificationHistoryRepository(db),
		cfg:                   cfg,
		emailTemplate:         emailTpl,
		smsTemplate:           smsTpl,
	}
}

// CreateNotificationConfig 创建通知配置
func (s *NotificationService) CreateNotificationConfig(config *model.NotificationConfig) error {
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()
	return s.notificationConfigRepo.Create(config)
}

// UpdateNotificationConfig 更新通知配置
func (s *NotificationService) UpdateNotificationConfig(config *model.NotificationConfig) error {
	config.UpdatedAt = time.Now()
	return s.notificationConfigRepo.Update(config)
}

// DeleteNotificationConfig 删除通知配置
func (s *NotificationService) DeleteNotificationConfig(id uint) error {
	return s.notificationConfigRepo.Delete(id)
}

// GetNotificationConfig 获取通知配置
func (s *NotificationService) GetNotificationConfig(id uint) (*model.NotificationConfig, error) {
	return s.notificationConfigRepo.GetByID(id)
}

// ListNotificationConfigs 列出通知配置
func (s *NotificationService) ListNotificationConfigs(page, pageSize int, filters map[string]interface{}) ([]*model.NotificationConfig, int64, error) {
	return s.notificationConfigRepo.List(page, pageSize, filters)
}

// SendAlertNotification 发送告警通知
func (s *NotificationService) SendAlertNotification(ctx context.Context, notificationID uint, alertEvent *model.AlertEvent) {
	// 异步发送通知
	go func() {
		// 创建一个带取消的上下文，避免长时间阻塞
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config, err := s.GetNotificationConfig(notificationID)
		if err != nil {
			s.logNotificationError(alertEvent.ID, notificationID, "", fmt.Sprintf("Failed to get notification config: %v", err))
			return
		}

		// 检查通知配置是否启用
		if !config.Enabled {
			return
		}

		// 根据通知类型发送相应的通知
		switch config.Type {
		case model.NotificationTypeEmail:
			s.sendEmailNotification(ctx, config, alertEvent)
		case model.NotificationTypeSMS:
			s.sendSMSNotification(ctx, config, alertEvent)
		case model.NotificationTypeWebhook:
			s.sendWebhookNotification(ctx, config, alertEvent)
		}
	}()
}

// sendEmailNotification 发送邮件通知
func (s *NotificationService) sendEmailNotification(ctx context.Context, config *model.NotificationConfig, alertEvent *model.AlertEvent) {
	// 检查邮件配置
	if s.cfg.Email.SMTPServer == "" || s.cfg.Email.SMTPPort == 0 || s.cfg.Email.From == "" {
		s.logNotificationError(alertEvent.ID, config.ID, "", "Email configuration is incomplete")
		return
	}

	// 解析收件人列表
	recipients := strings.Split(config.EmailRecipients, ",")
	if len(recipients) == 0 {
		s.logNotificationError(alertEvent.ID, config.ID, "", "No email recipients configured")
		return
	}

	// 渲染邮件内容
	data := struct {
		AlertEvent *model.AlertEvent
		Config     *model.NotificationConfig
	}{
		AlertEvent: alertEvent,
		Config:     config,
	}

	var bodyBuf bytes.Buffer
	if err := s.emailTemplate.Execute(&bodyBuf, data); err != nil {
		s.logNotificationError(alertEvent.ID, config.ID, "", fmt.Sprintf("Failed to render email template: %v", err))
		return
	}

	// 构建邮件头
	body := bodyBuf.String()
	parts := strings.SplitN(body, "\n\n", 2)
	subject := strings.TrimPrefix(parts[0], "Subject: ")
	
	headers := make(map[string]string)
	headers["From"] = s.cfg.Email.From
	headers["To"] = strings.Join(recipients, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"utf-8\""
	
	var message bytes.Buffer
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(parts[1])

	// 连接到SMTP服务器
	auth := smtp.PlainAuth("", s.cfg.Email.Username, s.cfg.Email.Password, s.cfg.Email.SMTPServer)
	
	// 发送邮件
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", s.cfg.Email.SMTPServer, s.cfg.Email.SMTPPort),
		auth,
		s.cfg.Email.From,
		recipients,
		message.Bytes(),
	)
	
	if err != nil {
		s.logNotificationError(alertEvent.ID, config.ID, strings.Join(recipients, ","), fmt.Sprintf("Failed to send email: %v", err))
		return
	}
	
	// 记录发送成功
	for _, recipient := range recipients {
		s.logNotificationSuccess(alertEvent.ID, config.ID, model.NotificationTypeEmail, recipient)
	}
}

// sendSMSNotification 发送短信通知
func (s *NotificationService) sendSMSNotification(ctx context.Context, config *model.NotificationConfig, alertEvent *model.AlertEvent) {
	// 检查SMS配置
	if s.cfg.SMS.Provider == "" || s.cfg.SMS.APIKey == "" {
		s.logNotificationError(alertEvent.ID, config.ID, "", "SMS configuration is incomplete")
		return
	}

	// 解析收件人列表
	recipients := strings.Split(config.SMSRecipients, ",")
	if len(recipients) == 0 {
		s.logNotificationError(alertEvent.ID, config.ID, "", "No SMS recipients configured")
		return
	}

	// 渲染短信内容
	data := struct {
		AlertEvent *model.AlertEvent
		Config     *model.NotificationConfig
	}{
		AlertEvent: alertEvent,
		Config:     config,
	}

	var contentBuf bytes.Buffer
	if err := s.smsTemplate.Execute(&contentBuf, data); err != nil {
		s.logNotificationError(alertEvent.ID, config.ID, "", fmt.Sprintf("Failed to render SMS template: %v", err))
		return
	}
	content := contentBuf.String()

	// 由于短信服务提供商不同，这里只是模拟发送短信的逻辑
	// 实际项目中需要替换为真正的短信发送API
	for _, recipient := range recipients {
		// 模拟发送短信
		// 实际项目中需要调用短信服务商的API
		fmt.Printf("Sending SMS to %s: %s\n", recipient, content)
		
		// 记录发送成功
		s.logNotificationSuccess(alertEvent.ID, config.ID, model.NotificationTypeSMS, recipient)
	}
}

// sendWebhookNotification 发送Webhook通知
func (s *NotificationService) sendWebhookNotification(ctx context.Context, config *model.NotificationConfig, alertEvent *model.AlertEvent) {
	// 检查Webhook配置
	if config.WebhookURL == "" {
		s.logNotificationError(alertEvent.ID, config.ID, "", "Webhook URL is not configured")
		return
	}

	// 准备要发送的数据
	var payload interface{}
	
	// 如果有自定义模板，使用模板渲染
	if config.WebhookTemplate != "" {
		tpl, err := template.New("webhook").Parse(config.WebhookTemplate)
		if err != nil {
			s.logNotificationError(alertEvent.ID, config.ID, config.WebhookURL, fmt.Sprintf("Failed to parse webhook template: %v", err))
			return
		}
		
		data := struct {
			AlertEvent *model.AlertEvent
			Config     *model.NotificationConfig
		}{
			AlertEvent: alertEvent,
			Config:     config,
		}
		
		var contentBuf bytes.Buffer
		if err := tpl.Execute(&contentBuf, data); err != nil {
			s.logNotificationError(alertEvent.ID, config.ID, config.WebhookURL, fmt.Sprintf("Failed to render webhook template: %v", err))
			return
		}
		
		// 尝试解析JSON
		var jsonPayload interface{}
		if err := json.Unmarshal(contentBuf.Bytes(), &jsonPayload); err == nil {
			payload = jsonPayload
		} else {
			// 如果不是有效的JSON，使用纯文本
			payload = map[string]string{
				"text": contentBuf.String(),
			}
		}
	} else {
		// 使用默认格式
		payload = map[string]interface{}{
			"alert": map[string]interface{}{
				"id":          alertEvent.ID,
				"name":        alertEvent.AlertName,
				"severity":    alertEvent.Severity,
				"status":      alertEvent.Status,
				"hostname":    alertEvent.Hostname,
				"service":     alertEvent.ServiceName,
				"metric_name": alertEvent.MetricName,
				"value":       alertEvent.MetricValue,
				"threshold":   alertEvent.Threshold,
				"operator":    alertEvent.Operator,
				"message":     alertEvent.Message,
				"triggered_at": alertEvent.TriggeredAt,
			},
		}
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		s.logNotificationError(alertEvent.ID, config.ID, config.WebhookURL, fmt.Sprintf("Failed to marshal JSON: %v", err))
		return
	}

	// 设置HTTP请求方法
	method := config.WebhookMethod
	if method == "" {
		method = "POST"
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		s.logNotificationError(alertEvent.ID, config.ID, config.WebhookURL, fmt.Sprintf("Failed to create HTTP request: %v", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	
	// 添加自定义HTTP头
	if config.WebhookHeaders != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(config.WebhookHeaders), &headers); err == nil {
			for key, value := range headers {
				req.Header.Set(key, value)
			}
		}
	}

	// 发送HTTP请求
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		s.logNotificationError(alertEvent.ID, config.ID, config.WebhookURL, fmt.Sprintf("Failed to send webhook: %v", err))
		return
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logNotificationError(alertEvent.ID, config.ID, config.WebhookURL, fmt.Sprintf("Webhook returned non-success status: %d", resp.StatusCode))
		return
	}

	// 记录发送成功
	s.logNotificationSuccess(alertEvent.ID, config.ID, model.NotificationTypeWebhook, config.WebhookURL)
}

// logNotificationSuccess 记录通知发送成功
func (s *NotificationService) logNotificationSuccess(alertEventID, notificationConfigID uint, notificationType model.NotificationType, recipient string) {
	history := &model.NotificationHistory{
		AlertEventID:        alertEventID,
		NotificationConfigID: notificationConfigID,
		Type:                notificationType,
		Status:              "SUCCESS",
		Recipient:           recipient,
		SentAt:              time.Now(),
		CreatedAt:           time.Now(),
	}
	
	if err := s.notificationHistoryRepo.Create(history); err != nil {
		fmt.Printf("Failed to log notification success: %v\n", err)
	}
}

// logNotificationError 记录通知发送失败
func (s *NotificationService) logNotificationError(alertEventID, notificationConfigID uint, recipient, message string) {
	var notificationType model.NotificationType
	
	config, err := s.GetNotificationConfig(notificationConfigID)
	if err == nil {
		notificationType = config.Type
	}
	
	history := &model.NotificationHistory{
		AlertEventID:        alertEventID,
		NotificationConfigID: notificationConfigID,
		Type:                notificationType,
		Status:              "FAILED",
		Message:             message,
		Recipient:           recipient,
		SentAt:              time.Now(),
		CreatedAt:           time.Now(),
	}
	
	if err := s.notificationHistoryRepo.Create(history); err != nil {
		fmt.Printf("Failed to log notification error: %v\n", err)
	}
} 