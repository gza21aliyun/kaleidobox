//go:build !dev

package main

import "github.com/wailsapp/wails/v2/pkg/options"

// buildSingleInstanceLock 返回单实例锁配置。
// 重复启动时 Wails 会把启动请求转给已运行的实例，这里负责把主窗口唤到前台，
// 避免第二个进程去抢数据库文件和端口而静默退出。
func buildSingleInstanceLock() *options.SingleInstanceLock {
	return &options.SingleInstanceLock{
		UniqueId: "kaleidobox",
		OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
			if appLogger != nil {
				appLogger.Info("检测到重复启动，激活已运行的主窗口")
			}
			appState.ShowMainWindow()
		},
	}
}
