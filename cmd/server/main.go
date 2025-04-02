package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"github.com/TejParker/bigdata-manager/internal/api"
	"github.com/TejParker/bigdata-manager/internal/db"
)

func init() {
	// 设置配置文件
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	
	// 尝试读取配置
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}
	
	log.Println("配置文件加载成功")
}

func main() {
	// 初始化数据库连接
	if err := db.InitDB(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.CloseDB()
	
	// 设置API路由
	router := api.SetupRouter()
	
	// 获取端口配置
	port := viper.GetInt("server.port")
	
	// 启动HTTP服务器
	go func() {
		log.Printf("服务器启动，监听端口: %d", port)
		if err := router.Run(fmt.Sprintf(":%d", port)); err != nil {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()
	
	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")
} 