package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// 日志级别定义
const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var levelNames = map[int]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

// LogConfig 日志配置
type LogConfig struct {
	Level      int    // 日志级别
	Filename   string // 日志文件名
	MaxSize    int64  // 每个日志文件最大尺寸（字节）
	MaxBackups int    // 最大历史文件数
	Console    bool   // 是否输出到控制台
}

// Logger 日志记录器
type Logger struct {
	config LogConfig
	mu     sync.Mutex
	file   *os.File
	logger *log.Logger
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Init 初始化默认日志记录器
func Init(config LogConfig) error {
	var err error
	once.Do(func() {
		defaultLogger, err = NewLogger(config)
	})
	return err
}

// NewLogger 创建新的日志记录器
func NewLogger(config LogConfig) (*Logger, error) {
	logger := &Logger{
		config: config,
	}

	// 设置默认值
	if config.Level < LevelDebug || config.Level > LevelFatal {
		logger.config.Level = LevelInfo
	}

	if config.MaxSize <= 0 {
		logger.config.MaxSize = 100 * 1024 * 1024 // 默认100MB
	}

	if config.MaxBackups <= 0 {
		logger.config.MaxBackups = 10 // 默认保留10个备份
	}

	// 创建或打开日志文件
	if config.Filename != "" {
		if err := logger.openFile(); err != nil {
			return nil, err
		}
	}

	// 设置日志输出
	var writers []io.Writer
	if logger.file != nil {
		writers = append(writers, logger.file)
	}
	if config.Console {
		writers = append(writers, os.Stdout)
	}

	if len(writers) == 0 {
		// 默认输出到标准输出
		writers = append(writers, os.Stdout)
	}

	// 创建多输出目标
	multiWriter := io.MultiWriter(writers...)
	logger.logger = log.New(multiWriter, "", 0) // 不使用前缀和标志，我们自己格式化

	return logger, nil
}

// openFile 打开或创建日志文件
func (l *Logger) openFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 关闭之前的文件
	if l.file != nil {
		l.file.Close()
	}

	// 创建目录
	dir := filepath.Dir(l.config.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 打开文件
	file, err := os.OpenFile(l.config.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	l.file = file
	return nil
}

// rotateFile 检查文件大小并进行轮转
func (l *Logger) rotateFile() error {
	if l.file == nil {
		return nil
	}

	// 获取文件信息
	info, err := l.file.Stat()
	if err != nil {
		return fmt.Errorf("获取日志文件信息失败: %v", err)
	}

	// 检查文件大小
	if info.Size() < l.config.MaxSize {
		return nil
	}

	// 关闭当前文件
	l.file.Close()

	// 备份文件名
	timestamp := time.Now().Format("20060102150405")
	backupName := fmt.Sprintf("%s.%s", l.config.Filename, timestamp)

	// 重命名当前日志文件为备份文件
	if err := os.Rename(l.config.Filename, backupName); err != nil {
		return fmt.Errorf("重命名日志文件失败: %v", err)
	}

	// 清理过多的历史文件
	l.cleanBackups()

	// 创建新的日志文件
	return l.openFile()
}

// cleanBackups 清理过多的历史日志文件
func (l *Logger) cleanBackups() {
	pattern := l.config.Filename + ".*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		l.log(LevelError, "查找历史日志文件失败: %v", err)
		return
	}

	if len(matches) <= l.config.MaxBackups {
		return
	}

	// 按修改时间排序
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	files := make([]fileInfo, 0, len(matches))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path, info.ModTime()})
	}

	// 排序（最早的在前面）
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].modTime.After(files[j].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// 删除多余的文件
	toDelete := len(files) - l.config.MaxBackups
	for i := 0; i < toDelete; i++ {
		os.Remove(files[i].path)
	}
}

// log 写入日志
func (l *Logger) log(level int, format string, args ...interface{}) {
	if level < l.config.Level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 轮转文件
	if err := l.rotateFile(); err != nil {
		fmt.Fprintf(os.Stderr, "轮转日志文件失败: %v\n", err)
	}

	// 获取调用信息
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}

	// 格式化日志
	timeStr := time.Now().Format("2006-01-02 15:04:05.000")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("%s [%s] %s:%d: %s", timeStr, levelName, file, line, message)

	// 写入日志
	l.logger.Println(logLine)

	// 如果是致命错误，终止程序
	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug 输出调试级别日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info 输出信息级别日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn 输出警告级别日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error 输出错误级别日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal 输出致命级别日志并终止程序
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
}

// 默认日志方法

// Debug 输出调试级别日志
func Debug(format string, args ...interface{}) {
	if defaultLogger == nil {
		fmt.Printf("WARNING: 日志系统未初始化，使用默认配置\n")
		if err := Init(LogConfig{Level: LevelDebug, Console: true}); err != nil {
			fmt.Printf("ERROR: 初始化日志系统失败: %v\n", err)
			return
		}
	}
	defaultLogger.Debug(format, args...)
}

// Info 输出信息级别日志
func Info(format string, args ...interface{}) {
	if defaultLogger == nil {
		fmt.Printf("WARNING: 日志系统未初始化，使用默认配置\n")
		if err := Init(LogConfig{Level: LevelDebug, Console: true}); err != nil {
			fmt.Printf("ERROR: 初始化日志系统失败: %v\n", err)
			return
		}
	}
	defaultLogger.Info(format, args...)
}

// Warn 输出警告级别日志
func Warn(format string, args ...interface{}) {
	if defaultLogger == nil {
		fmt.Printf("WARNING: 日志系统未初始化，使用默认配置\n")
		if err := Init(LogConfig{Level: LevelDebug, Console: true}); err != nil {
			fmt.Printf("ERROR: 初始化日志系统失败: %v\n", err)
			return
		}
	}
	defaultLogger.Warn(format, args...)
}

// Error 输出错误级别日志
func Error(format string, args ...interface{}) {
	if defaultLogger == nil {
		fmt.Printf("WARNING: 日志系统未初始化，使用默认配置\n")
		if err := Init(LogConfig{Level: LevelDebug, Console: true}); err != nil {
			fmt.Printf("ERROR: 初始化日志系统失败: %v\n", err)
			return
		}
	}
	defaultLogger.Error(format, args...)
}

// Fatal 输出致命级别日志并终止程序
func Fatal(format string, args ...interface{}) {
	if defaultLogger == nil {
		fmt.Printf("WARNING: 日志系统未初始化，使用默认配置\n")
		if err := Init(LogConfig{Level: LevelDebug, Console: true}); err != nil {
			fmt.Printf("ERROR: 初始化日志系统失败: %v\n", err)
			return
		}
	}
	defaultLogger.Fatal(format, args...)
}
