package applog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RunMode 运行模式
type RunMode int

const (
	ModeGUI RunMode = iota // Wails GUI 模式
	ModeCLI                // 命令行模式
)

// 全局运行模式
var (
	currentMode  RunMode
	modeMu       sync.RWMutex
	colorEnabled = true // CLI 模式下是否启用彩色输出

	logFilePath string
	logFileMu   sync.RWMutex

	logAll   bool
	logAllMu sync.RWMutex
)

// ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// SetMode 设置全局运行模式
func SetMode(mode RunMode) {
	modeMu.Lock()
	defer modeMu.Unlock()
	currentMode = mode
}

func SetLogAll(all bool) {
	logAllMu.Lock()
	defer logAllMu.Unlock()
	logAll = all
}

// GetMode 获取当前运行模式
func GetMode() RunMode {
	modeMu.RLock()
	defer modeMu.RUnlock()
	return currentMode
}

// SetColorEnabled 设置 CLI 模式下是否启用彩色输出
func SetColorEnabled(enabled bool) {
	modeMu.Lock()
	defer modeMu.Unlock()
	colorEnabled = enabled
}

// logToCLI 输出到 CLI 控制台（带颜色）
func logToCLI(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)

	var colorCode string
	var output io.Writer = os.Stdout

	modeMu.RLock()
	color := colorEnabled
	modeMu.RUnlock()

	if color {
		switch level {
		case "TRACE":
			colorCode = colorGray
		case "DEBUG":
			colorCode = colorCyan
		case "INFO":
			colorCode = colorBlue
		case "WARNING":
			colorCode = colorYellow
		case "ERROR", "FATAL":
			colorCode = colorRed
			output = os.Stderr
		default:
			colorCode = colorReset
		}
		fmt.Fprintf(output, "[%s] %s%-8s%s %s\n", timestamp, colorCode, level, colorReset, message)
	} else {
		fmt.Fprintf(output, "[%s] %-8s %s\n", timestamp, level, message)
	}

	// FATAL 级别退出程序
	if level == "FATAL" {
		os.Exit(1)
	}
}

// LogTracef 跟踪级别日志（兼容 runtime.LogTracef）
func LogTracef(ctx context.Context, format string, args ...interface{}) {
	modeMu.RLock()
	mode := currentMode
	modeMu.RUnlock()

	if mode == ModeGUI {
		runtime.LogTracef(ctx, format, args...)
		fmt.Printf(format+"\n", args...)
	} else {
		logToCLI("TRACE", format, args...)
	}
}

// LogDebugf 调试级别日志（兼容 runtime.LogDebugf）
func LogDebugf(ctx context.Context, format string, args ...interface{}) {
	modeMu.RLock()
	mode := currentMode
	modeMu.RUnlock()

	if mode == ModeGUI {
		runtime.LogDebugf(ctx, format, args...)
		fmt.Printf(format+"\n", args...)
	} else {
		logToCLI("DEBUG", format, args...)
	}
}

// LogInfof 信息级别日志（兼容 runtime.LogInfof）
func LogInfof(ctx context.Context, format string, args ...interface{}) {
	modeMu.RLock()
	mode := currentMode
	modeMu.RUnlock()

	if mode == ModeGUI {
		runtime.LogInfof(ctx, format, args...)
		fmt.Printf(format+"\n", args...)
	} else {
		logToCLI("INFO", format, args...)
	}
}

// LogPrintf 打印日志（兼容 runtime.LogPrintf）
func LogPrintf(ctx context.Context, format string, args ...interface{}) {
	modeMu.RLock()
	mode := currentMode
	modeMu.RUnlock()

	if mode == ModeGUI {
		runtime.LogPrintf(ctx, format, args...)
		fmt.Printf(format+"\n", args...)
	} else {
		logToCLI("INFO", format, args...)
	}
}

// LogWarningf 警告级别日志（兼容 runtime.LogWarningf）
func LogWarningf(ctx context.Context, format string, args ...interface{}) {
	modeMu.RLock()
	mode := currentMode
	modeMu.RUnlock()

	if mode == ModeGUI {
		runtime.LogWarningf(ctx, format, args...)
		fmt.Printf(format+"\n", args...)
	} else {
		logToCLI("WARNING", format, args...)
	}
}

// LogErrorf 错误级别日志（兼容 runtime.LogErrorf）
func LogErrorf(ctx context.Context, format string, args ...interface{}) {
	modeMu.RLock()
	mode := currentMode
	modeMu.RUnlock()

	if mode == ModeGUI {
		runtime.LogErrorf(ctx, format, args...)
		fmt.Printf(format+"\n", args...)
	} else {
		logToCLI("ERROR", format, args...)
	}
}

// LogFatalf 致命错误级别日志（兼容 runtime.LogFatalf）
func LogFatalf(ctx context.Context, format string, args ...interface{}) {
	modeMu.RLock()
	mode := currentMode
	modeMu.RUnlock()

	if mode == ModeGUI {
		runtime.LogFatalf(ctx, format, args...)
		fmt.Printf(format+"\n", args...)
	} else {
		logToCLI("FATAL", format, args...)
	}
}

// LogTrace 跟踪级别日志（兼容 runtime.LogTrace）
func LogTrace(ctx context.Context, message string) {
	LogTracef(ctx, "%s", message)
}

// LogDebug 调试级别日志（兼容 runtime.LogDebug）
func LogDebug(ctx context.Context, message string) {
	LogDebugf(ctx, "%s", message)
}

// LogInfo 信息级别日志（兼容 runtime.LogInfo）
func LogInfo(ctx context.Context, message string) {
	LogInfof(ctx, "%s", message)
}

// LogPrint 打印日志（兼容 runtime.LogPrint）
func LogPrint(ctx context.Context, message string) {
	LogPrintf(ctx, "%s", message)
}

// LogWarning 警告级别日志（兼容 runtime.LogWarning）
func LogWarning(ctx context.Context, message string) {
	LogWarningf(ctx, "%s", message)
}

// LogError 错误级别日志（兼容 runtime.LogError）
func LogError(ctx context.Context, message string) {
	LogErrorf(ctx, "%s", message)
}

// LogFatal 致命错误级别日志（兼容 runtime.LogFatal）
func LogFatal(ctx context.Context, message string) {
	LogFatalf(ctx, "%s", message)
}

// ... existing code ...

/**
 * ErrorLogSaveAppLog 无依赖的错误日志保存，能输出到控制台的同时写入 app.log
 * 不依赖 context 和 runtime，可直接在任何地方调用
 */
func ErrorLogSaveAppLog(format string, err error, args ...interface{}) {
	msg := fmt.Sprintf(format, args...) + fmt.Sprintf(", error:\n %v\n", err)

	// 输出到控制台
	fmt.Print(msg)

	// 直接写入 app.log 文件
	saveLogToFile("ERROR", msg)
}

func ErrorLogSaveAppLogs(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	// 输出到控制台
	fmt.Print(msg)

	// 直接写入 app.log 文件
	saveLogToFile("ERROR", msg)
}

func ErrorLogSaveAppOutput(format string, err error, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...) + fmt.Sprintf(", error:\n %v\n", err)

	// 输出到控制台
	fmt.Print(msg)

	// 直接写入 app.log 文件
	saveLogToFile("ERROR", msg)
	return errors.New(msg)
}

func ErrorLogSaveTextOutput(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)

	// 输出到控制台
	fmt.Print(msg)

	// 直接写入 app.log 文件
	saveLogToFile("ERROR", msg)
	return errors.New(msg)
}

/**
 * InfoLogSaveAppLog 无依赖的信息日志保存，能输出到控制台的同时写入 app.log
 * 不依赖 context 和 runtime，可直接在任何地方调用
 */
func InfoLogSaveAppLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	// 输出到控制台
	fmt.Print(msg + "\n")

	// 直接写入 app.log 文件
	logAllMu.RLock()
	defer logAllMu.RUnlock()
	if logAll != false {
		saveLogToFile("Info", msg)
	}
}

/**
 * WarningLogSaveAppLog 无依赖的警告日志保存，能输出到控制台的同时写入 app.log
 * 不依赖 context 和 runtime，可直接在任何地方调用
 */
func WarningLogSaveAppLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	// 输出到控制台
	fmt.Print(msg + "\n")

	// 直接写入 app.log 文件
	logAllMu.RLock()
	defer logAllMu.RUnlock()
	if logAll != false {
		saveLogToFile("WARNING", msg)
	}

}

/**
 * saveLogToFile 直接将日志写入 app.log 文件
 * 使用追加模式打开文件，如果文件不存在则创建
 */
func saveLogToFile(level, message string) {
	msg := strings.ReplaceAll(message, "\n", " \\n ")
	logFile := getLogFilePath()
	if logFile == "" {
		return // 如果未设置日志路径，静默失败
	}

	// 确保日志目录存在
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}
	logFileMu.Lock()
	defer logFileMu.Unlock()

	// 以追加模式打开文件，如果不存在则创建
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return // 如果无法打开文件，静默失败
	}
	defer file.Close()

	// 格式化日志行：LEVEL | timestamp | message
	// timestamp := time.Now().Format("2006-01-02 15:04:05")
	// logLine := fmt.Sprintf("%-8s| %s | %s\n", level, timestamp, msg)
	logLine := fmt.Sprintf("%-6s| %s\n", level, msg)

	file.WriteString(logLine)
}

// SetLogFilePath 设置日志文件路径（在应用启动时调用一次）
func SetLogFilePath(path string) {
	logFileMu.Lock()
	defer logFileMu.Unlock()
	logFilePath = path
}

// getLogFilePath 获取日志文件路径
func getLogFilePath() string {
	logFileMu.RLock()
	defer logFileMu.RUnlock()
	return logFilePath
}
