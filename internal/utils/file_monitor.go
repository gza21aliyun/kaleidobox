package utils

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// FileChangeEvent 表示文件变更事件
type FileChangeEvent struct {
	FilePath   string
	ChangeType uint32
	Timestamp  time.Time
	ProcessID  uint32
	IsWriteOp  bool
}

// FileMonitor 文件监控器
type FileMonitor struct {
	mu      sync.Mutex
	watches map[string]*watchHandle
	events  chan FileChangeEvent
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

type watchHandle struct {
	dir        string
	handle     windows.Handle
	buffer     []byte
	stopEvent  windows.Handle
	overlapped windows.Overlapped
}

const (
	FILE_NOTIFY_CHANGE_FILE_NAME  = 0x00000001
	FILE_NOTIFY_CHANGE_SIZE       = 0x00000008
	FILE_NOTIFY_CHANGE_LAST_WRITE = 0x00000010
	FILE_NOTIFY_CHANGE_CREATION   = 0x00000040

	FILE_ACTION_ADDED            = 0x00000001
	FILE_ACTION_REMOVED          = 0x00000002
	FILE_ACTION_MODIFIED         = 0x00000003
	FILE_ACTION_RENAMED_OLD_NAME = 0x00000004
	FILE_ACTION_RENAMED_NEW_NAME = 0x00000005

	FILE_FLAG_BACKUP_SEMANTICS = 0x02000000
	FILE_SHARE_READ            = 0x00000001
	FILE_SHARE_WRITE           = 0x00000002
	FILE_SHARE_DELETE          = 0x00000004
	OPEN_EXISTING              = 3
)

// NewFileMonitor 创建新的文件监控器
func NewFileMonitor(ctx context.Context) *FileMonitor {
	monitorCtx, cancel := context.WithCancel(ctx)
	return &FileMonitor{
		watches: make(map[string]*watchHandle),
		events:  make(chan FileChangeEvent, 100),
		ctx:     monitorCtx,
		cancel:  cancel,
	}
}

// Start 启动监控
func (fm *FileMonitor) Start() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.running {
		return fmt.Errorf("监控器已在运行中")
	}

	fm.running = true
	go fm.eventLoop()

	return nil
}

// Stop 停止监控
func (fm *FileMonitor) Stop() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if !fm.running {
		return nil
	}

	fm.cancel()
	fm.running = false

	for _, handle := range fm.watches {
		if handle.stopEvent != 0 {
			windows.SetEvent(handle.stopEvent)
		}
		if handle.handle != 0 {
			windows.CancelIoEx(windows.Handle(handle.handle), nil)
			windows.CloseHandle(handle.handle)
		}
		if handle.stopEvent != 0 {
			windows.CloseHandle(handle.stopEvent)
		}
	}
	fm.watches = make(map[string]*watchHandle)

	close(fm.events)
	return nil
}

// AddWatch 添加监控目录
func (fm *FileMonitor) AddWatch(dir string, recursive bool) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, exists := fm.watches[dir]; exists {
		return nil
	}

	handle, err := openDirectory(dir)
	if err != nil {
		return fmt.Errorf("打开目录失败：%w", err)
	}

	stopEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		windows.CloseHandle(handle)
		return fmt.Errorf("创建事件失败：%w", err)
	}

	buffer := make([]byte, 64*1024)

	watch := &watchHandle{
		dir:       dir,
		handle:    handle,
		buffer:    buffer,
		stopEvent: stopEvent,
	}

	fm.watches[dir] = watch

	go fm.watchDirectoryAsync(watch, recursive)

	return nil
}

// RemoveWatch 移除监控目录
func (fm *FileMonitor) RemoveWatch(dir string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	watch, exists := fm.watches[dir]
	if !exists {
		return nil
	}

	windows.SetEvent(watch.stopEvent)
	windows.CancelIoEx(windows.Handle(watch.handle), nil)
	windows.CloseHandle(watch.handle)
	windows.CloseHandle(watch.stopEvent)

	delete(fm.watches, dir)
	return nil
}

// Events 返回事件通道
func (fm *FileMonitor) Events() <-chan FileChangeEvent {
	return fm.events
}

func (fm *FileMonitor) eventLoop() {
	for {
		select {
		case <-fm.ctx.Done():
			return
		case event := <-fm.events:
			fmt.Printf("[FileMonitor] %s - %s\n",
				formatChangeType(event.ChangeType),
				event.FilePath)
		}
	}
}

// ... existing code ...

func (fm *FileMonitor) watchDirectoryAsync(watch *watchHandle, recursive bool) {
	var bytesRet uint32

	err := windows.ReadDirectoryChanges(
		windows.Handle(watch.handle),
		&watch.buffer[0],
		uint32(len(watch.buffer)),
		recursive,
		FILE_NOTIFY_CHANGE_FILE_NAME|
			FILE_NOTIFY_CHANGE_SIZE|
			FILE_NOTIFY_CHANGE_LAST_WRITE|
			FILE_NOTIFY_CHANGE_CREATION,
		&bytesRet,
		&watch.overlapped,
		0,
	)

	if err != nil && err != syscall.ERROR_IO_PENDING {
		return
	}

	events := []windows.Handle{watch.stopEvent}
	if watch.overlapped.HEvent != 0 {
		events = append(events, watch.overlapped.HEvent)
	}

	for {
		ret, err := windows.WaitForMultipleObjects(events, false, INFINITE)
		if err != nil {
			return
		}

		if ret == 0 {
			windows.GetOverlappedResult(windows.Handle(watch.handle), &watch.overlapped, &bytesRet, false)
			return
		}

		if watch.overlapped.HEvent != 0 {
			windows.ResetEvent(watch.overlapped.HEvent)
		}

		offset := 0
		for {
			if offset >= int(bytesRet) {
				break
			}

			nextOffset := *(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(&watch.buffer[offset])) + 0))
			action := *(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(&watch.buffer[offset])) + 4))
			fileNameLength := *(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(&watch.buffer[offset])) + 8))

			fileNamePtr := unsafe.Pointer(uintptr(unsafe.Pointer(&watch.buffer[offset])) + 12)
			fileName := windows.UTF16ToString((*[syscall.MAX_PATH]uint16)(fileNamePtr)[:fileNameLength/2])
			fullPath := filepath.Join(watch.dir, fileName)

			event := FileChangeEvent{
				FilePath:   fullPath,
				ChangeType: action,
				Timestamp:  time.Now(),
				IsWriteOp:  action == FILE_ACTION_MODIFIED || action == FILE_ACTION_ADDED,
			}

			select {
			case fm.events <- event:
			case <-fm.ctx.Done():
				return
			}

			if nextOffset == 0 {
				break
			}
			offset += int(nextOffset)
		}

		err = windows.ReadDirectoryChanges(
			windows.Handle(watch.handle),
			&watch.buffer[0],
			uint32(len(watch.buffer)),
			recursive,
			FILE_NOTIFY_CHANGE_FILE_NAME|
				FILE_NOTIFY_CHANGE_SIZE|
				FILE_NOTIFY_CHANGE_LAST_WRITE|
				FILE_NOTIFY_CHANGE_CREATION,
			&bytesRet,
			&watch.overlapped,
			0,
		)

		if err != nil && err != syscall.ERROR_IO_PENDING {
			return
		}
	}
}

// ... existing code ...

func openDirectory(dir string) (windows.Handle, error) {
	dirPtr, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}

	handle, err := windows.CreateFile(
		dirPtr,
		windows.FILE_LIST_DIRECTORY,
		FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE,
		nil,
		OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)

	if err != nil {
		return 0, err
	}

	return handle, nil
}

func formatChangeType(changeType uint32) string {
	switch changeType {
	case FILE_ACTION_ADDED:
		return "CREATED"
	case FILE_ACTION_REMOVED:
		return "DELETED"
	case FILE_ACTION_MODIFIED:
		return "MODIFIED"
	case FILE_ACTION_RENAMED_OLD_NAME:
		return "RENAMED_OLD"
	case FILE_ACTION_RENAMED_NEW_NAME:
		return "RENAMED_NEW"
	default:
		return "UNKNOWN"
	}
}

// IsLikelySaveFile 判断文件是否可能是存档文件
func IsLikelySaveFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	saveExtensions := []string{
		".sav", ".save", ".dat", ".data", ".bin",
		".cfg", ".ini", ".json", ".xml", ".yaml",
		".profile", ".progress", ".game", ".gms",
		".rpgsave", ".rpgssv", ".lsd", ".rgssad",
		".mpk", ".zip", ".7z", ".rar",
	}

	for _, saveExt := range saveExtensions {
		if ext == saveExt {
			return true
		}
	}

	filename := strings.ToLower(filepath.Base(filePath))
	if strings.Contains(filename, "save") ||
		strings.Contains(filename, "存档") ||
		strings.Contains(filename, "backup") ||
		strings.Contains(filename, "profile") {
		return true
	}

	return false
}
