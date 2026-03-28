package utils

import (
	"errors"
	"fmt"
	"lunabox/internal/models"
	"sync"
	"syscall"
	"unsafe"
)

// ... existing code ...

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumDisplayMonitors      = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
)

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

const (
	MONITORINFOF_PRIMARY = 0x00000001
	SW_SHOW              = 5
	SWP_NOSIZE           = 0x0001
	SWP_NOMOVE           = 0x0002
	SWP_NOZORDER         = 0x0004
	SWP_SHOWWINDOW       = 0x0040
)

type windowEnumData struct {
	pid   uint32
	hwnds []uintptr
}
type MONITORINFOEXW struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
	SzDevice  [32]uint16
}

type monitorEnumData struct {
	monitors []models.MonitorInfo
	index    int
	mu       sync.Mutex
}

// 全局变量保存回调函数引用，防止被 GC 回收

var (
	displayMonitorCallback uintptr
	windowEnumCallbackRef  uintptr
)

// EnumDisplayMonitors 枚举所有显示器（使用 golang.org/x/sys/windows）
func EnumDisplayMonitors() ([]models.MonitorInfo, error) {
	fmt.Printf("monitors 01: start enumeration\n")

	result := &monitorEnumData{
		monitors: make([]models.MonitorInfo, 0),
		index:    0,
	}

	var enumErr error

	// 创建回调函数
	displayMonitorCallback = syscall.NewCallback(func(hMonitor uintptr, hdcMonitor uintptr, lprcMonitor *RECT, dwData uintptr) uintptr {
		info := MONITORINFOEXW{}
		info.CbSize = uint32(unsafe.Sizeof(info))

		ret, _, _ := procGetMonitorInfoW.Call(
			hMonitor,
			uintptr(unsafe.Pointer(&info)),
		)

		fmt.Printf(">>> Callback executed, ret=%d, device=%s\n", ret, syscall.UTF16ToString(info.SzDevice[:]))

		if ret != 0 {
			result.mu.Lock()
			monitorInfo := models.MonitorInfo{
				Index:      result.index,
				DeviceName: syscall.UTF16ToString(info.SzDevice[:]),
				Left:       info.RcMonitor.Left,
				Top:        info.RcMonitor.Top,
				Right:      info.RcMonitor.Right,
				Bottom:     info.RcMonitor.Bottom,
				IsPrimary:  (info.DwFlags & MONITORINFOF_PRIMARY) != 0,
			}
			result.monitors = append(result.monitors, monitorInfo)
			result.index++
			result.mu.Unlock()
		}

		return 1 // continue enumeration
	})

	fmt.Printf("About to call EnumDisplayMonitors, callback=%d\n", displayMonitorCallback)

	// 调用 API
	ret, _, lastErr := procEnumDisplayMonitors.Call(
		0,
		0,
		displayMonitorCallback,
		0,
	)

	fmt.Printf("EnumDisplayMonitors returned: ret=%d, lastErr=%v\n", ret, lastErr)

	if ret == 0 {
		enumErr = lastErr
	}

	fmt.Printf("Total monitors found: %d\n", len(result.monitors))

	if enumErr != nil {
		return nil, enumErr
	}

	// 打印所有找到的显示器
	for i, m := range result.monitors {
		fmt.Printf("  [%d] %s (%dx%d) primary=%v\n",
			i, m.DeviceName, m.Right-m.Left, m.Bottom-m.Top, m.IsPrimary)
	}

	return result.monitors, nil
}

func GetDisplayByName(name string) (models.MonitorInfo, error) {
	monitors, err := EnumDisplayMonitors()
	if err != nil || len(monitors) == 0 {
		return models.MonitorInfo{}, err
	}
	monitor := Find(monitors, func(t1 models.MonitorInfo) bool { return t1.DeviceName == name })
	if monitor == nil {
		return models.MonitorInfo{}, errors.New("未找到指定显示器")
	}
	return *monitor, nil
}

// ... existing code ...

// EnumWindowsByProcessID 根据进程 ID 枚举窗口
func EnumWindowsByProcessID(pid uint32) ([]uintptr, error) {
	data := &windowEnumData{
		pid:   pid,
		hwnds: make([]uintptr, 0),
	}

	// 保存回调函数引用 - 注意返回值必须是 uintptr
	windowEnumCallbackRef = syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
		data := (*windowEnumData)(unsafe.Pointer(lParam))

		var processID uint32
		ret, _, _ := procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processID)))

		if ret != 0 && processID == data.pid {
			// 检查窗口是否可见
			visible, _, _ := procIsWindowVisible.Call(hwnd)
			if visible != 0 {
				data.hwnds = append(data.hwnds, hwnd)
				fmt.Printf(">>> Window found: hwnd=%d, pid=%d\n", hwnd, processID)
			}
		}

		return 1 // 返回 uintptr 类型，继续枚举
	})

	ret, _, err := procEnumWindows.Call(
		windowEnumCallbackRef,
		uintptr(unsafe.Pointer(data)),
	)

	if ret == 0 {
		return nil, err
	}

	fmt.Printf("Total windows found for PID %d: %d\n", pid, len(data.hwnds))
	return data.hwnds, nil
}

// ... existing code ...

func enumWindowsCallback(hwnd uintptr, lParam uintptr) bool {
	data := (*windowEnumData)(unsafe.Pointer(lParam))

	var processID uint32
	ret, _, _ := procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processID)))

	if ret != 0 && processID == data.pid {
		// 检查窗口是否可见
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible != 0 {
			data.hwnds = append(data.hwnds, hwnd)
		}
	}

	return true
}

// enumDisplayMonitorsCallback 显示器枚举回调
func enumDisplayMonitorsCallback(hMonitor uintptr, hdcMonitor uintptr, lprcMonitor *RECT, dwData uintptr) bool {
	type monitorEnumData struct {
		monitors *[]models.MonitorInfo
		index    *int
	}

	data := (*monitorEnumData)(unsafe.Pointer(dwData))

	info := MONITORINFOEXW{}
	info.CbSize = uint32(unsafe.Sizeof(info))

	ret, _, _ := procGetMonitorInfoW.Call(
		hMonitor,
		uintptr(unsafe.Pointer(&info)),
	)

	if ret != 0 {
		monitorInfo := models.MonitorInfo{
			Index:      *data.index,
			DeviceName: syscall.UTF16ToString(info.SzDevice[:]),
			Left:       info.RcMonitor.Left,
			Top:        info.RcMonitor.Top,
			Right:      info.RcMonitor.Right,
			Bottom:     info.RcMonitor.Bottom,
			IsPrimary:  (info.DwFlags & MONITORINFOF_PRIMARY) != 0,
		}
		*data.monitors = append(*data.monitors, monitorInfo)
		*data.index++
	}

	return true
}

// MoveWindowToMonitor 将窗口移动到指定显示器
func MoveWindowToMonitor(hwnd uintptr, monitor models.MonitorInfo) error {
	// 获取当前窗口位置
	var rect RECT
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return fmt.Errorf("failed to get window rectangle")
	}

	windowWidth := rect.Right - rect.Left
	windowHeight := rect.Bottom - rect.Top

	// 计算新位置（居中显示）
	newX := monitor.Left + (monitor.Right-monitor.Left-windowWidth)/2
	newY := monitor.Top + (monitor.Bottom-monitor.Top-windowHeight)/2

	// 移动窗口
	ret, _, err := procSetWindowPos.Call(
		hwnd,
		0,
		uintptr(newX),
		uintptr(newY),
		0,
		0,
		SWP_NOZORDER|SWP_NOSIZE|SWP_SHOWWINDOW,
	)

	if ret == 0 {
		return fmt.Errorf("failed to move window: %v", err)
	}

	return nil
}
