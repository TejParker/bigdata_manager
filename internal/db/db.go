package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

var (
	// DB 全局数据库连接池
	DB *sql.DB
)

// InitDB 初始化数据库连接
func InitDB() error {
	host := viper.GetString("database.host")
	port := viper.GetInt("database.port")
	user := viper.GetString("database.user")
	password := viper.GetString("database.password")
	dbname := viper.GetString("database.name")
	maxConns := viper.GetInt("database.max_connections")
	timeout := viper.GetInt("database.timeout")

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		user, password, host, port, dbname)

	// 连接数据库
	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("无法连接数据库: %v", err)
	}

	// 设置连接池参数
	DB.SetMaxOpenConns(maxConns)
	DB.SetMaxIdleConns(maxConns / 2)
	DB.SetConnMaxLifetime(time.Duration(timeout) * time.Minute)

	// 测试连接
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("数据库Ping失败: %v", err)
	}

	log.Println("数据库连接成功")
	return nil
}

// CloseDB 关闭数据库连接
func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("数据库连接已关闭")
	}
}

// Transaction 事务处理封装
func Transaction(fn func(*sql.Tx) error) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // 重新抛出panic
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
} 