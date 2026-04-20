package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// 日志级别定义
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// 级别字符串映射
var levelStrings = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
}

// 日志配置
type Config struct {
	Level      Level  // 日志级别
	Directory  string // 日志文件目录
	MaxDays    int    // 日志保留天数，默认7天
	MaxSize    int64  // 单个日志文件最大大小(MB)，默认100MB
	CallerDept int    // 调用栈深度，默认2
	JsonFormat bool   // 是否输出JSON格式
}

// Logger 日志器
type Logger struct {
	config    Config
	file      *os.File
	fileMu    sync.Mutex
	date      string
	startTime time.Time
}

// 全局日志器实例
var defaultLogger *Logger
var once sync.Once

// 获取函数名
func getCallerFuncName(depth int) string {
	pc, _, line, ok := runtime.Caller(depth)
	if !ok {
		return "unknown"
	}
	funcName := runtime.FuncForPC(pc).Name()
	// 提取函数名（去除包路径）
	if idx := strings.LastIndex(funcName, "."); idx != -1 {
		funcName = funcName[idx+1:]
	}
	return fmt.Sprintf("%s:%d", funcName, line)
}

// 格式化时间
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

// 初始化日志器
func Init(config Config) {
	once.Do(func() {
		// 设置默认值
		if config.Directory == "" {
			config.Directory = "./logs"
		}
		if config.MaxDays == 0 {
			config.MaxDays = 7
		}
		if config.MaxSize == 0 {
			config.MaxSize = 100
		}
		if config.CallerDept == 0 {
			config.CallerDept = 2
		}

		defaultLogger = &Logger{
			config:    config,
			startTime: time.Now(),
			file:      nil,
		}
		defaultLogger.initFile()
		// 启动日志清理协程
		go defaultLogger.cleanOldLogs()
	})
}

// 初始化文件
func (l *Logger) initFile() error {
	l.fileMu.Lock()
	defer l.fileMu.Unlock()

	// 创建日志目录
	if err := os.MkdirAll(l.config.Directory, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return err
	}

	// 获取当前日期
	currentDate := time.Now().Format("2006-01-02")

	// 如果日期变化或文件未打开，重新打开文件
	if l.date != currentDate || l.file == nil {
		if l.file != nil {
			l.file.Close()
		}

		filename := filepath.Join(l.config.Directory, fmt.Sprintf("app_%s.log", currentDate))
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("打开日志文件失败: %v\n", err)
			return err
		}
		l.file = file
		l.date = currentDate
	}

	return nil
}

// 清理过期日志
func (l *Logger) cleanOldLogs() {
	ticker := time.NewTicker(time.Hour * 6)
	defer ticker.Stop()

	for range ticker.C {
		l.fileMu.Lock()
		files, err := filepath.Glob(filepath.Join(l.config.Directory, "app_*.log"))
		if err != nil {
			l.fileMu.Unlock()
			continue
		}

		cutoff := time.Now().AddDate(0, 0, -l.config.MaxDays)
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(f)
			}
		}
		l.fileMu.Unlock()
	}
}

// 格式化日志消息
func (l *Logger) formatMessage(level Level, msg string) string {
	funcName := getCallerFuncName(l.config.CallerDept)
	levelStr := levelStrings[level]
	timestamp := formatTime(time.Now())

	return fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, levelStr, funcName, msg)
}

// 输出到控制台和文件
func (l *Logger) output(level Level, msg string) {
	// 检查日志级别
	if level < l.config.Level {
		return
	}

	message := l.formatMessage(level, msg)

	// 输出到控制台（带颜色）
	color := ""
	switch level {
	case DebugLevel:
		color = "\033[36m" // 青色
	case InfoLevel:
		color = "\033[32m" // 绿色
	case WarnLevel:
		color = "\033[33m" // 黄色
	case ErrorLevel:
		color = "\033[31m" // 红色
	}
	reset := "\033[0m"
	fmt.Printf("%s%s%s", color, message, reset)

	// 确保文件已初始化且日期正确
	l.initFile()

	// 输出到文件
	l.fileMu.Lock()
	defer l.fileMu.Unlock()
	if l.file != nil {
		l.file.WriteString(message)
		// 检查文件大小
		info, _ := l.file.Stat()
		if info.Size() > l.config.MaxSize*1024*1024 {
			l.rotateFile()
		}
	}
}

// 切割日志文件
func (l *Logger) rotateFile() {
	if l.file != nil {
		l.file.Close()
	}
	backupName := filepath.Join(l.config.Directory, fmt.Sprintf("app_%s_%d.log", l.date, time.Now().Unix()))
	os.Rename(filepath.Join(l.config.Directory, fmt.Sprintf("app_%s.log", l.date)), backupName)
	l.initFile()
}

// Debug 输出调试日志
func Debug(format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(Config{Level: DebugLevel})
	}
	defaultLogger.output(DebugLevel, fmt.Sprintf(format, args...))
}

// Info 输出信息日志
func Info(format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(Config{Level: DebugLevel})
	}
	defaultLogger.output(InfoLevel, fmt.Sprintf(format, args...))
}

// Warn 输出警告日志
func Warn(format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(Config{Level: DebugLevel})
	}
	defaultLogger.output(WarnLevel, fmt.Sprintf(format, args...))
}

// Error 输出错误日志
func Error(format string, args ...interface{}) {
	if defaultLogger == nil {
		Init(Config{Level: DebugLevel})
	}
	defaultLogger.output(ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatal 输出致命错误日志并退出
func Fatal(format string, args ...interface{}) {
	Error(format, args...)
	os.Exit(1)
}

// 获取默认日志器
func GetLogger() *Logger {
	if defaultLogger == nil {
		Init(Config{Level: DebugLevel})
	}
	return defaultLogger
}

// 关闭日志器
func Close() {
	if defaultLogger != nil && defaultLogger.file != nil {
		defaultLogger.fileMu.Lock()
		defer defaultLogger.fileMu.Unlock()
		defaultLogger.file.Close()
	}
}
