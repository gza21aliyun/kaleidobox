package main

import (
	"fmt"
	"os"

	"lunabox/internal/cli"
	"lunabox/internal/cli/ipc"
)

func main() {
	args := os.Args[1:]

	if err := validateArgs(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Println("Usage: kaleidocli <command>")
		os.Exit(1)
	}

	// Special case for interactive easter egg (must run locally for stdin/stdout)
	if args[0] == "luna-sama" {
		runLocalCommand(args)
		return
	}

	// 1. 尝试通过 IPC 在 GUI 进程中运行命令
	if ipc.IsServerRunning() {
		if err := runRemoteCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 2. 如果 GUI 未运行，提示用户启动
	printGuiNotRunningMessage()
	os.Exit(1)
}

// validateArgs 验证命令行参数
func validateArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Error: command required")
	}
	return nil
}

// runLocalCommand 运行本地命令
func runLocalCommand(args []string) {
	cli.RunCommand(os.Stdout, &cli.CoreApp{}, args)
}

// runRemoteCommand 通过 IPC 运行远程命令
func runRemoteCommand(args []string) error {
	return ipc.RemoteRun(args)
}

// printGuiNotRunningMessage 打印 GUI 未运行的消息
func printGuiNotRunningMessage() {
	fmt.Println("Error: KaleidoBox application is not running.")
	fmt.Println("Please start KaleidoBox first to use CLI commands.")
}
