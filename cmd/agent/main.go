package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/TejParker/bigdata-manager/pkg/model"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

var (
	serverAddr    string
	hostID        int
	heartbeatSec  int
	collectionSec int
	apiEndpoint   string
	version       = "0.1.0"
)

// ComponentProcess 本地进程信息
type ComponentProcess struct {
	ComponentID int
	ProcessID   int
	Status      string
}

// 本地组件进程映射
var componentProcesses = make(map[int]*ComponentProcess)

func init() {
	// 解析命令行参数
	flag.StringVar(&serverAddr, "server", "http://localhost:8080", "管理服务器地址")
	flag.IntVar(&hostID, "id", 0, "主机ID")
	flag.IntVar(&heartbeatSec, "heartbeat", 10, "心跳间隔(秒)")
	flag.IntVar(&collectionSec, "collection", 15, "指标收集间隔(秒)")
	flag.Parse()

	if hostID == 0 {
		log.Fatal("必须提供主机ID参数")
	}

	// 构建API端点
	apiEndpoint = fmt.Sprintf("%s/api/v1/agent/heartbeat", serverAddr)
}

func main() {
	log.Printf("Agent 启动 (版本: %s), 连接服务器: %s\n", version, serverAddr)
	log.Printf("主机ID: %d, 心跳间隔: %d秒, 收集间隔: %d秒\n", hostID, heartbeatSec, collectionSec)

	// 初始化心跳定时器
	heartbeatTicker := time.NewTicker(time.Duration(heartbeatSec) * time.Second)
	defer heartbeatTicker.Stop()

	// 初始化采集定时器
	collectionTicker := time.NewTicker(time.Duration(collectionSec) * time.Second)
	defer collectionTicker.Stop()

	// 退出信号处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 发送首次心跳
	sendHeartbeat()

	// 主循环
	for {
		select {
		case <-heartbeatTicker.C:
			// 发送心跳
			sendHeartbeat()
		case <-collectionTicker.C:
			// 收集指标（不触发心跳发送）
			collectMetrics()
		case <-quit:
			log.Println("接收到退出信号，正在关闭...")
			return
		}
	}
}

// sendHeartbeat 发送心跳请求
func sendHeartbeat() {
	metrics := collectMetrics()
	components := collectComponentStatus()

	// 构造心跳请求
	req := model.HeartbeatRequest{
		HostID:      hostID,
		Timestamp:   time.Now(),
		CPUUsage:    metrics["cpu_usage"],
		MemoryUsage: metrics["memory_usage"],
		DiskUsage:   metrics["disk_usage"],
		Metrics:     buildMetricsData(metrics),
		Components:  components,
	}

	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		log.Printf("序列化心跳请求失败: %v", err)
		return
	}

	// 发送请求
	resp, err := http.Post(apiEndpoint, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("发送心跳失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("心跳返回错误状态码: %d", resp.StatusCode)
		return
	}

	// 解析响应
	var heartbeatResp struct {
		Success  bool                 `json:"success"`
		Message  string               `json:"message"`
		Data     map[string]any       `json:"data"`
		Commands []model.AgentCommand `json:"commands,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		log.Printf("解析心跳响应失败: %v", err)
		return
	}

	// 处理服务器返回的命令
	if heartbeatResp.Data != nil && heartbeatResp.Data["commands"] != nil {
		log.Printf("收到 %d 个命令", len(heartbeatResp.Commands))
		for _, cmd := range heartbeatResp.Commands {
			handleCommand(cmd)
		}
	}
}

// collectMetrics 收集系统指标
func collectMetrics() map[string]float64 {
	metrics := make(map[string]float64)

	// 采集CPU使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Printf("获取CPU使用率失败: %v", err)
	} else if len(cpuPercent) > 0 {
		metrics["cpu_usage"] = cpuPercent[0]
	}

	// 采集内存使用率
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("获取内存信息失败: %v", err)
	} else {
		metrics["memory_usage"] = memInfo.UsedPercent
		metrics["memory_total"] = float64(memInfo.Total)
		metrics["memory_used"] = float64(memInfo.Used)
	}

	// 采集磁盘使用率
	diskInfo, err := disk.Usage("/")
	if err != nil {
		log.Printf("获取磁盘信息失败: %v", err)
	} else {
		metrics["disk_usage"] = diskInfo.UsedPercent
		metrics["disk_total"] = float64(diskInfo.Total)
		metrics["disk_used"] = float64(diskInfo.Used)
	}

	// 系统负载
	if runtime.GOOS != "windows" {
		loadInfo, err := load.Avg()
		if err != nil {
			log.Printf("获取系统负载失败: %v", err)
		} else {
			metrics["load1"] = loadInfo.Load1
			metrics["load5"] = loadInfo.Load5
			metrics["load15"] = loadInfo.Load15
		}
	}

	return metrics
}

// buildMetricsData 构造指标数据数组
func buildMetricsData(metrics map[string]float64) []model.MetricData {
	var result []model.MetricData
	now := time.Now()

	for name, value := range metrics {
		// 只发送自定义指标，基础指标已在心跳结构体中
		if name != "cpu_usage" && name != "memory_usage" && name != "disk_usage" {
			result = append(result, model.MetricData{
				Name:      name,
				Value:     value,
				Timestamp: now,
			})
		}
	}

	return result
}

// collectComponentStatus 收集组件进程状态
func collectComponentStatus() []model.ComponentStatus {
	var result []model.ComponentStatus

	for _, cp := range componentProcesses {
		// 检查进程是否存在
		proc, err := process.NewProcess(int32(cp.ProcessID))
		if err != nil {
			// 进程不存在，标记为STOPPED
			cp.Status = "STOPPED"
		} else {
			// 检查进程状态
			status, err := proc.Status()
			if err != nil {
				log.Printf("获取进程状态失败: %v", err)
				cp.Status = "UNKNOWN"
			} else {
				// 根据进程状态设置组件状态
				if len(status) > 0 && (status[0] == "R" || status[0] == "S") {
					cp.Status = "RUNNING"
				} else {
					cp.Status = "STOPPED"
				}
			}
		}

		// 添加到结果
		result = append(result, model.ComponentStatus{
			ComponentID: cp.ComponentID,
			Status:      cp.Status,
			ProcessID:   cp.ProcessID,
		})
	}

	return result
}

// handleCommand 处理服务器发送的命令
func handleCommand(cmd model.AgentCommand) {
	log.Printf("处理命令: %s (ID: %s)", cmd.Type, cmd.CommandID)

	var result any
	var success bool
	var message string

	// 根据命令类型执行不同操作
	switch cmd.Type {
	case "INSTALL":
		// 安装组件
		success, message, result = handleInstall(cmd)
	case "START":
		// 启动组件
		success, message, result = handleStart(cmd)
	case "STOP":
		// 停止组件
		success, message, result = handleStop(cmd)
	case "CONFIGURE":
		// 配置组件
		success, message, result = handleConfigure(cmd)
	default:
		success = false
		message = "未知命令类型"
	}

	// 发送命令执行结果
	sendCommandResponse(cmd.CommandID, success, message, result)
}

// handleInstall 处理安装命令
func handleInstall(cmd model.AgentCommand) (bool, string, any) {
	// TODO: 实现安装逻辑
	componentID, _ := cmd.Payload["component_id"].(float64)
	packageURL, _ := cmd.Payload["package_url"].(string)

	log.Printf("正在安装组件 %d, 包地址: %s", int(componentID), packageURL)

	// 在实际实现中，这里会下载软件包并执行安装脚本
	// 这里仅模拟安装成功
	time.Sleep(2 * time.Second)

	// 记录组件关联的进程
	componentProcesses[int(componentID)] = &ComponentProcess{
		ComponentID: int(componentID),
		ProcessID:   0, // 安装后未启动
		Status:      "STOPPED",
	}

	return true, "组件安装成功", nil
}

// handleStart 处理启动命令
func handleStart(cmd model.AgentCommand) (bool, string, any) {
	// TODO: 实现启动逻辑
	componentID, _ := cmd.Payload["component_id"].(float64)
	log.Printf("正在启动组件 %d", int(componentID))

	// 在实际实现中，这里会调用启动脚本
	// 这里仅模拟启动成功，生成一个随机进程ID
	time.Sleep(1 * time.Second)
	fakeProcessID := 10000 + int(componentID)

	// 更新组件进程状态
	cp, exists := componentProcesses[int(componentID)]
	if !exists {
		cp = &ComponentProcess{
			ComponentID: int(componentID),
		}
		componentProcesses[int(componentID)] = cp
	}
	cp.ProcessID = fakeProcessID
	cp.Status = "RUNNING"

	return true, "组件启动成功", map[string]any{
		"process_id": fakeProcessID,
	}
}

// handleStop 处理停止命令
func handleStop(cmd model.AgentCommand) (bool, string, any) {
	// TODO: 实现停止逻辑
	componentID, _ := cmd.Payload["component_id"].(float64)
	log.Printf("正在停止组件 %d", int(componentID))

	// 在实际实现中，这里会调用停止脚本
	// 这里仅模拟停止成功
	time.Sleep(1 * time.Second)

	// 更新组件进程状态
	cp, exists := componentProcesses[int(componentID)]
	if exists {
		cp.ProcessID = 0
		cp.Status = "STOPPED"
	}

	return true, "组件停止成功", nil
}

// handleConfigure 处理配置命令
func handleConfigure(cmd model.AgentCommand) (bool, string, any) {
	// TODO: 实现配置逻辑
	componentID, _ := cmd.Payload["component_id"].(float64)
	configMap, _ := cmd.Payload["config"].(map[string]any)

	log.Printf("正在配置组件 %d, 配置项数量: %d", int(componentID), len(configMap))

	// 在实际实现中，这里会修改配置文件
	// 这里仅模拟配置成功
	time.Sleep(1 * time.Second)

	return true, "组件配置成功", nil
}

// sendCommandResponse 发送命令执行结果
func sendCommandResponse(commandID string, success bool, message string, result any) {
	// 构造响应
	resp := model.AgentCommandResponse{
		CommandID: commandID,
		Success:   success,
		Message:   message,
		Result:    result,
	}

	// 序列化响应
	respBody, err := json.Marshal(resp)
	if err != nil {
		log.Printf("序列化命令响应失败: %v", err)
		return
	}

	// 发送响应
	url := fmt.Sprintf("%s/api/v1/agent/command-result", serverAddr)
	httpResp, err := http.Post(url, "application/json", bytes.NewBuffer(respBody))
	if err != nil {
		log.Printf("发送命令响应失败: %v", err)
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		log.Printf("命令响应返回错误状态码: %d", httpResp.StatusCode)
	}
}
