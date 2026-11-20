package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// 启动阶段的保护逻辑：
// 1. 致命错误以弹窗形式提示（release 版本没有控制台，静默退出用户只会看到"打不开"）
// 2. 打开数据库时若被残留进程占用，提示用户结束该进程后重试

var (
	startupUser32   = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW = startupUser32.NewProc("MessageBoxW")
)

const (
	mbOK            = 0x00000000
	mbYesNo         = 0x00000004
	mbIconError     = 0x00000010
	mbIconWarning   = 0x00000030
	mbSetForeground = 0x00010000
	mbTopMost       = 0x00040000
	mbIDYes         = 6
)

// showMessageBox 弹出系统消息框，返回用户点击的按钮 ID
func showMessageBox(title, text string, flags uint32) int {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	textPtr, _ := syscall.UTF16PtrFromString(text)
	ret, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(flags),
	)
	return int(ret)
}

// fatalStartup 记录日志、弹窗提示后退出进程
func fatalStartup(title, message string) {
	if appLogger != nil {
		appLogger.Error(message)
	}
	text := message
	if startupLogPath != "" {
		text += "\n\n日志文件：" + startupLogPath
	}
	showMessageBox(title, text, mbOK|mbIconError|mbSetForeground|mbTopMost)
	os.Exit(1)
}

// duckDBLockPattern 匹配 DuckDB 文件被占用时的错误信息：
// File is already open in C:\path\kaleidobox.db (PID 1234)
var duckDBLockPattern = regexp.MustCompile(`File is already open in (.+?) \(PID (\d+)\)`)

type dbLockHolder struct {
	Path string
	PID  int
}

// parseDBLockHolder 从 DuckDB 的报错中解析出占用文件的进程
func parseDBLockHolder(err error) *dbLockHolder {
	if err == nil {
		return nil
	}
	match := duckDBLockPattern.FindStringSubmatch(err.Error())
	if len(match) != 3 {
		return nil
	}
	pid, convErr := strconv.Atoi(match[2])
	if convErr != nil || pid <= 0 {
		return nil
	}
	return &dbLockHolder{
		Path: strings.TrimSpace(match[1]),
		PID:  pid,
	}
}

// isKaleidoBoxExecutable 判断占用数据库的进程是否为本应用的残留进程
func isKaleidoBoxExecutable(path string) bool {
	lowerPath := strings.ToLower(path)
	baseName := strings.ToLower(filepath.Base(lowerPath))
	for _, marker := range []string{"kaleidobox", "kaleidocli", "lunabox", "lunacli"} {
		if strings.Contains(baseName, marker) {
			return true
		}
	}
	// 旧版本被删除到回收站后进程仍在运行的情况
	return strings.Contains(lowerPath, `\$recycle.bin\`)
}

// tryOpenDatabase 打开数据库并建立一次真实连接
func tryOpenDatabase(dbPath string) (*sql.DB, error) {
	database, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}

// openDatabaseWithRecovery 打开数据库；被残留进程占用时提供一键结束并重试
func openDatabaseWithRecovery(dbPath string) (*sql.DB, error) {
	const maxAttempts = 4

	var lastErr error
	killedHolder := false

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		database, err := tryOpenDatabase(dbPath)
		if err == nil {
			return database, nil
		}
		lastErr = err
		if appLogger != nil {
			appLogger.Warning(fmt.Sprintf("打开数据库失败（第 %d/%d 次）：%v", attempt, maxAttempts, err))
		}

		holder := parseDBLockHolder(err)
		if holder == nil {
			// 不是文件占用问题（例如数据库损坏），短暂重试后放弃
			if attempt < maxAttempts {
				time.Sleep(1200 * time.Millisecond)
				continue
			}
			break
		}

		if killedHolder || !isKaleidoBoxExecutable(holder.Path) {
			break
		}

		answer := showMessageBox(
			"KaleidoBox 启动失败",
			fmt.Sprintf(
				"检测到另一个 KaleidoBox 进程正在运行，它占用着数据库文件：\n\n%s\n(PID %d)\n\n可能是该进程还在后台运行（托盘模式），或者上次没有正常退出。\n\n是否结束该进程并重新启动 KaleidoBox？",
				holder.Path, holder.PID,
			),
			mbYesNo|mbIconWarning|mbSetForeground|mbTopMost,
		)
		if answer != mbIDYes {
			break
		}

		proc, findErr := os.FindProcess(holder.PID)
		if findErr != nil {
			if appLogger != nil {
				appLogger.Warning("查找残留进程失败：" + findErr.Error())
			}
			break
		}
		if killErr := proc.Kill(); killErr != nil {
			if appLogger != nil {
				appLogger.Warning("结束残留进程失败：" + killErr.Error())
			}
			break
		}
		if appLogger != nil {
			appLogger.Info(fmt.Sprintf("已结束占用数据库的残留进程 PID %d", holder.PID))
		}
		killedHolder = true
		// 等待操作系统释放文件句柄
		time.Sleep(1500 * time.Millisecond)
	}

	return nil, lastErr
}
