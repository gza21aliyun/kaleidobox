//go:build dev

package main

import "github.com/wailsapp/wails/v2/pkg/options"

// buildSingleInstanceLock dev 模式下不启用单实例锁，
// 避免 wails dev 热重载重启时被尚未退出的旧进程挡住。
func buildSingleInstanceLock() *options.SingleInstanceLock {
	return nil
}
