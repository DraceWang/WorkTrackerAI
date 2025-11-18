package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	logFile     *os.File
	debugMode   bool
)

// Init 初始化日志系统
// debug: 是否为调试模式(同时输出到控制台和文件)
func Init(logsDir string, debug bool) error {
	debugMode = debug

	// 确保日志目录存在
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// 创建日志文件(按日期)
	logFileName := fmt.Sprintf("worktracker_%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logsDir, logFileName)

	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// 根据模式选择输出目标
	var writer io.Writer
	if debugMode {
		// 调试模式: 同时输出到文件和控制台
		writer = io.MultiWriter(os.Stdout, logFile)
		fmt.Printf("🐛 调试模式已启用,日志输出到控制台和文件: %s\n", logPath)
	} else {
		// 普通模式: 仅输出到文件
		writer = logFile
	}

	infoLogger = log.New(writer, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	warnLogger = log.New(writer, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(writer, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
	debugLogger = log.New(writer, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)

	Info("日志系统初始化完成,日志文件: %s, 调试模式: %v", logPath, debugMode)
	return nil
}

// Close 关闭日志文件
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// Info 信息日志
func Info(format string, v ...interface{}) {
	if infoLogger != nil {
		infoLogger.Output(2, fmt.Sprintf(format, v...))
	} else {
		// 如果日志系统未初始化,输出到控制台
		fmt.Printf("[INFO] "+format+"\n", v...)
	}
}

// Warn 警告日志
func Warn(format string, v ...interface{}) {
	if warnLogger != nil {
		warnLogger.Output(2, fmt.Sprintf(format, v...))
	} else {
		fmt.Printf("[WARN] "+format+"\n", v...)
	}
}

// Error 错误日志
func Error(format string, v ...interface{}) {
	if errorLogger != nil {
		errorLogger.Output(2, fmt.Sprintf(format, v...))
	} else {
		fmt.Printf("[ERROR] "+format+"\n", v...)
	}
}

// Debug 调试日志
func Debug(format string, v ...interface{}) {
	if debugLogger != nil {
		debugLogger.Output(2, fmt.Sprintf(format, v...))
	} else {
		fmt.Printf("[DEBUG] "+format+"\n", v...)
	}
}
