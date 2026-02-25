package utils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

// CheckIfProcessRunning 检查指定进程是否正在运行
// func CheckIfProcessRunning(processName string) (bool, error) {
// 	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName), "/FO", "CSV", "/NH")
// 	output, err := cmd.Output()
// 	if err != nil {
// 		return false, fmt.Errorf("failed to execute tasklist: %w", err)
// 	}

// 	outputStr := string(output)
// 	// 检查输出中是否包含进程名
// 	return strings.Contains(strings.ToLower(outputStr), strings.ToLower(processName)), nil
// }

// ExecutePowerShellHidden 在Windows环境下无窗口执行PowerShell命令
func ExecutePowerShellHidden(command string) ([]byte, error) {
	if runtime.GOOS != "windows" {
		// 非Windows系统使用普通方式执行
		cmd := exec.Command("powershell", "-Command", command)
		return cmd.Output()
	}

	// Windows环境下隐藏窗口执行
	cmd := exec.Command("powershell", "-Command", command)

	// 设置隐藏窗口属性
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,       // 隐藏窗口
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW 标志
	}

	// 设置UTF-8编码环境变量
	cmd.Env = append([]string{
		"CHCP=65001",         // 设置代码页为UTF-8
		"LANG=zh_CN.UTF-8",   // 设置语言环境
		"LC_ALL=zh_CN.UTF-8", // 设置区域环境
	}, cmd.Environ()...)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute powershell command: %w", err)
	}

	return output, nil
}

// RemoveBOMAndTrim 清理PowerShell输出中的BOM标记并去除空白
func RemoveBOMAndTrim(output []byte) string {
	outputStr := string(output)
	fmt.Println("logutil RemoveBOMAndTrim bofore:", outputStr)
	result := strings.TrimSpace(string(output))

	// 移除可能的BOM标记
	if strings.HasPrefix(result, "\xff\xfe") || strings.HasPrefix(result, "\xfe\xff") {
		// UTF-16 BOM
		result = result[2:]
	} else if strings.HasPrefix(result, "\xef\xbb\xbf") {
		// UTF-8 BOM
		result = result[3:]
	}

	return result
}

func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin": // macOS
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start()
}
