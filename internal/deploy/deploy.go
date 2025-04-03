package deploy

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/TejParker/bigdata-manager/pkg/model"
)

var (
	// ErrComponentNotFound 组件未找到
	ErrComponentNotFound = errors.New("组件未找到")
	// ErrInvalidStatus 状态无效
	ErrInvalidStatus = errors.New("组件状态无效")
	// ErrInvalidOperation 操作无效
	ErrInvalidOperation = errors.New("对当前组件状态，该操作无效")
)

// DeployService 部署服务
type DeployService struct {
	components        map[int]*model.Component        // 按ID存储组件
	deployments       map[int][]model.Deployment      // 按主机ID存储部署记录
	componentLock     sync.RWMutex                    // 组件锁
	deploymentLock    sync.RWMutex                    // 部署记录锁
	commandChan       chan model.AgentCommand         // 命令通道
	commandResultChan chan model.AgentCommandResponse // 命令结果通道
}

// NewDeployService 创建部署服务
func NewDeployService() *DeployService {
	return &DeployService{
		components:        make(map[int]*model.Component),
		deployments:       make(map[int][]model.Deployment),
		commandChan:       make(chan model.AgentCommand, 100),
		commandResultChan: make(chan model.AgentCommandResponse, 100),
	}
}

// RegisterComponent 注册组件
func (s *DeployService) RegisterComponent(component model.Component) {
	s.componentLock.Lock()
	defer s.componentLock.Unlock()

	s.components[component.ID] = &component
}

// GetComponent 获取组件
func (s *DeployService) GetComponent(componentID int) (*model.Component, error) {
	s.componentLock.RLock()
	defer s.componentLock.RUnlock()

	component, exists := s.components[componentID]
	if !exists {
		return nil, ErrComponentNotFound
	}
	return component, nil
}

// GetAllComponents 获取所有组件
func (s *DeployService) GetAllComponents() []model.Component {
	s.componentLock.RLock()
	defer s.componentLock.RUnlock()

	result := make([]model.Component, 0, len(s.components))
	for _, component := range s.components {
		result = append(result, *component)
	}
	return result
}

// Deploy 部署组件到主机
func (s *DeployService) Deploy(hostID, componentID int) (string, error) {
	component, err := s.GetComponent(componentID)
	if err != nil {
		return "", err
	}

	// 创建部署记录
	deploymentID := fmt.Sprintf("deploy_%d_%d_%d", hostID, componentID, time.Now().Unix())
	deployment := model.Deployment{
		ID:          deploymentID,
		HostID:      hostID,
		ComponentID: componentID,
		Status:      "PENDING",
		StartTime:   time.Now(),
	}

	// 保存部署记录
	s.deploymentLock.Lock()
	s.deployments[hostID] = append(s.deployments[hostID], deployment)
	s.deploymentLock.Unlock()

	// 发送安装命令
	cmd := model.AgentCommand{
		CommandID: deploymentID,
		Type:      "INSTALL",
		Payload: map[string]interface{}{
			"component_id": float64(componentID),
			"package_url":  component.PackageURL,
		},
	}

	go func() {
		// 发送命令到Agent
		s.commandChan <- cmd

		// 处理安装结果
		go s.handleInstallResult(deploymentID, hostID)
	}()

	return deploymentID, nil
}

// 处理安装结果
func (s *DeployService) handleInstallResult(deploymentID string, hostID int) {
	// 模拟等待安装结果
	// 在实际实现中，这里应该监听来自Agent的响应
	time.Sleep(5 * time.Second)

	// 更新部署状态为成功
	s.updateDeploymentStatus(deploymentID, hostID, "INSTALLED")
}

// 更新部署状态
func (s *DeployService) updateDeploymentStatus(deploymentID string, hostID int, status string) {
	s.deploymentLock.Lock()
	defer s.deploymentLock.Unlock()

	for i, deployment := range s.deployments[hostID] {
		if deployment.ID == deploymentID {
			s.deployments[hostID][i].Status = status
			if status == "INSTALLED" || status == "FAILED" {
				s.deployments[hostID][i].EndTime = time.Now()
			}
			break
		}
	}
}

// GetDeployments 获取主机的部署记录
func (s *DeployService) GetDeployments(hostID int) []model.Deployment {
	s.deploymentLock.RLock()
	defer s.deploymentLock.RUnlock()

	return s.deployments[hostID]
}

// StartComponent 启动组件
func (s *DeployService) StartComponent(hostID, componentID int) (string, error) {
	commandID := fmt.Sprintf("start_%d_%d_%d", hostID, componentID, time.Now().Unix())

	// 发送启动命令
	cmd := model.AgentCommand{
		CommandID: commandID,
		Type:      "START",
		Payload: map[string]interface{}{
			"component_id": float64(componentID),
		},
	}

	// 发送命令到Agent
	s.commandChan <- cmd

	return commandID, nil
}

// StopComponent 停止组件
func (s *DeployService) StopComponent(hostID, componentID int) (string, error) {
	commandID := fmt.Sprintf("stop_%d_%d_%d", hostID, componentID, time.Now().Unix())

	// 发送停止命令
	cmd := model.AgentCommand{
		CommandID: commandID,
		Type:      "STOP",
		Payload: map[string]interface{}{
			"component_id": float64(componentID),
		},
	}

	// 发送命令到Agent
	s.commandChan <- cmd

	return commandID, nil
}

// ConfigureComponent 配置组件
func (s *DeployService) ConfigureComponent(hostID, componentID int, config map[string]interface{}) (string, error) {
	commandID := fmt.Sprintf("config_%d_%d_%d", hostID, componentID, time.Now().Unix())

	// 发送配置命令
	cmd := model.AgentCommand{
		CommandID: commandID,
		Type:      "CONFIGURE",
		Payload: map[string]interface{}{
			"component_id": float64(componentID),
			"config":       config,
		},
	}

	// 发送命令到Agent
	s.commandChan <- cmd

	return commandID, nil
}

// ProcessCommandResult 处理Agent返回的命令结果
func (s *DeployService) ProcessCommandResult(result model.AgentCommandResponse) {
	// 记录命令结果
	log.Printf("收到命令结果: ID=%s, 成功=%v, 消息=%s",
		result.CommandID, result.Success, result.Message)

	// 根据命令ID的前缀判断命令类型
	if len(result.CommandID) > 7 {
		prefix := result.CommandID[:6]
		switch prefix {
		case "deploy":
			// 处理部署结果
			if result.Success {
				// 更新部署状态
				// 从commandID中提取hostID (在实际实现中应该有更好的方法)
				// 这里简化处理
				// 在实际实现中，部署状态应该存储在数据库中
			}
		case "start_":
			// 处理启动结果
		case "stop_":
			// 处理停止结果
		case "config":
			// 处理配置结果
		}
	}

	// 将结果发送到结果通道，供API层使用
	s.commandResultChan <- result
}

// GetCommandChannel 获取命令通道
func (s *DeployService) GetCommandChannel() <-chan model.AgentCommand {
	return s.commandChan
}

// GetCommandResultChannel 获取命令结果通道
func (s *DeployService) GetCommandResultChannel() <-chan model.AgentCommandResponse {
	return s.commandResultChan
}

// ServiceInstance 部署服务的单例实例
var ServiceInstance *DeployService
var once sync.Once

// GetDeployService 获取部署服务实例
func GetDeployService() *DeployService {
	once.Do(func() {
		ServiceInstance = NewDeployService()
	})
	return ServiceInstance
}
