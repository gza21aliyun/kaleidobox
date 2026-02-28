package service

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"sync"
	"time"
	"unsafe"

	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"syscall"

	"gobot.io/x/gobot/v2"
	"gobot.io/x/gobot/v2/platforms/joystick"
	"gobot.io/x/gobot/v2/platforms/keyboard"
)

// HotkeyService 结构体添加当前焦点进程跟踪
type HotkeyService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig

	// gobot相关
	robot      *gobot.Robot
	keyboard   *keyboard.Driver
	joysticks  map[string]*joystick.Driver // device_id -> driver
	deviceLock sync.RWMutex

	// 快捷键管理
	hotkeys    map[string]*models.Hotkey // key_code -> hotkey
	hotkeyLock sync.RWMutex

	// 当前活动游戏
	activeGameID string
	activeLock   sync.RWMutex

	// 已连接设备
	connectedDevices map[string]*models.ConnectedDevice // device_id -> device
	deviceInfoLock   sync.RWMutex

	// 当前焦点进程信息
	focusedProcessName string
	processCheckTicker *time.Ticker
}

// Windows API函数声明
var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowRect              = user32.NewProc("GetWindowRect")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procGetDIBits                  = gdi32.NewProc("GetDIBits")

	procGetDC                  = user32.NewProc("GetDC")
	procReleaseDC              = user32.NewProc("ReleaseDC")
	procGetClientRect          = user32.NewProc("GetClientRect")
	procClientToScreen         = user32.NewProc("ClientToScreen")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	gdi32                      = syscall.NewLazyDLL("gdi32.dll")
)

const (
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	SRCCOPY                           = 0x00CC0020
)

// Rect 结构体定义
type Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// Point 结构体定义
type Point struct {
	X int32
	Y int32
}

func NewHotkeyService() *HotkeyService {
	return &HotkeyService{
		joysticks:        make(map[string]*joystick.Driver),
		hotkeys:          make(map[string]*models.Hotkey),
		connectedDevices: make(map[string]*models.ConnectedDevice),
	}
}

// 修改Init方法以启动进程焦点监控
func (s *HotkeyService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config

	applog.LogInfof(s.ctx, "Initializing hotkey service...")

	// 初始化键盘驱动
	s.keyboard = keyboard.NewDriver()
	applog.LogInfof(s.ctx, "Keyboard driver created")

	// 创建机器人实例
	s.robot = gobot.NewRobot("hotkeyRobot",
		[]gobot.Connection{},
		[]gobot.Device{s.keyboard},
		s.work,
	)
	applog.LogInfof(s.ctx, "Robot instance created")

	// 加载快捷键配置
	s.loadHotkeyConfig()

	// 加载已连接设备信息
	s.loadConnectedDevices()

	// 启动进程焦点监控
	s.startProcessFocusMonitoring()

	// 启动事件监听（带错误处理）
	go func() {
		// 添加一些延迟确保其他服务已初始化
		time.Sleep(100 * time.Millisecond)
		s.startEventListeners()
	}()

	// 启动设备检测
	go s.startDeviceDetection()

	applog.LogInfof(s.ctx, "Hotkey service initialization completed")
}

// work 机器人工作函数
func (s *HotkeyService) work() {
	// 键盘事件监听
	s.keyboard.On(keyboard.Key, func(data interface{}) {
		if event, ok := data.(keyboard.KeyEvent); ok {
			s.handleKeyboardEvent(event)
		}
	})
}

// loadHotkeyConfig 从数据库加载快捷键配置
func (s *HotkeyService) loadHotkeyConfig() {
	s.hotkeyLock.Lock()
	defer s.hotkeyLock.Unlock()

	rows, err := s.db.Query("SELECT id, game_id, name, device_type, key_code, modifiers, action_type, action_params, is_enabled, created_at, updated_at FROM hotkeys WHERE is_enabled = TRUE")
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to load hotkeys: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var hotkey models.Hotkey
		var modifiersBytes, paramsBytes []byte

		err := rows.Scan(
			&hotkey.ID, &hotkey.GameID, &hotkey.Name, &hotkey.DeviceType, &hotkey.KeyCode,
			&modifiersBytes, &hotkey.ActionType, &paramsBytes,
			&hotkey.IsEnabled, &hotkey.CreatedAt, &hotkey.UpdatedAt,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "Failed to scan hotkey: %v", err)
			continue
		}

		// 解析修饰键
		if len(modifiersBytes) > 0 {
			json.Unmarshal(modifiersBytes, &hotkey.Modifiers)
		}

		// 解析动作参数
		if len(paramsBytes) > 0 {
			json.Unmarshal(paramsBytes, &hotkey.ActionParams)
		}

		s.hotkeys[hotkey.KeyCode] = &hotkey
	}

	// 统计各类设备的快捷键数量
	deviceStats := make(map[enums.DeviceType]int)
	for _, hotkey := range s.hotkeys {
		deviceStats[hotkey.DeviceType]++
	}

	statsStr := ""
	for deviceType, count := range deviceStats {
		statsStr += fmt.Sprintf("%s:%d ", deviceType, count)
	}

	applog.LogInfof(s.ctx, "Loaded %d hotkeys (%s)", len(s.hotkeys), statsStr)
}

// loadConnectedDevices 加载已连接设备信息
func (s *HotkeyService) loadConnectedDevices() {
	s.deviceInfoLock.Lock()
	defer s.deviceInfoLock.Unlock()

	rows, err := s.db.Query("SELECT id, device_type, device_name, device_id, is_active, connected_at, last_seen_at FROM connected_devices WHERE is_active = TRUE")
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to load connected devices: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var device models.ConnectedDevice
		err := rows.Scan(
			&device.ID, &device.DeviceType, &device.DeviceName, &device.DeviceID,
			&device.IsActive, &device.ConnectedAt, &device.LastSeenAt,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "Failed to scan connected device: %v", err)
			continue
		}

		s.connectedDevices[device.DeviceID] = &device
	}

	applog.LogInfof(s.ctx, "Loaded %d connected devices", len(s.connectedDevices))
}

// startEventListeners 启动事件监听器
func (s *HotkeyService) startEventListeners() {
	// 尝试启动机器人
	if err := s.robot.Start(); err != nil {
		applog.LogErrorf(s.ctx, "Failed to start gobot robot: %v", err)

		// 如果gobot启动失败，尝试使用备用方案
		applog.LogInfof(s.ctx, "Falling back to alternative keyboard monitoring...")
		go s.startAlternativeKeyListener()
		return
	}

	applog.LogInfof(s.ctx, "Gobot robot started successfully")
}

// Windows API常量
const (
	KEY_PRESSED = 0x8000
)

// 虚拟键码常量
const (
	VK_ESCAPE  = 0x1B
	VK_F1      = 0x70
	VK_F2      = 0x71
	VK_F3      = 0x72
	VK_F4      = 0x73
	VK_F5      = 0x74
	VK_F6      = 0x75
	VK_F7      = 0x76
	VK_F8      = 0x77
	VK_F9      = 0x78
	VK_F10     = 0x79
	VK_F11     = 0x7A
	VK_F12     = 0x7B
	VK_SPACE   = 0x20
	VK_RETURN  = 0x0D
	VK_SHIFT   = 0x10
	VK_CONTROL = 0x11
	VK_MENU    = 0x12 // Alt键
)

// isFocusedProcessMatchesActiveGame 检查当前焦点进程是否匹配活动游戏
func (s *HotkeyService) isFocusedProcessMatchesActiveGame() bool {
	return true
	// s.activeLock.RLock()
	// activeGameID := s.activeGameID
	// s.activeLock.RUnlock()

	// if activeGameID == "" {
	// 	return false
	// }

	// // 获取活动游戏信息
	// var game models.Game
	// err := s.db.QueryRow("SELECT process_name FROM games WHERE id = ?", activeGameID).Scan(&game.ProcessName)
	// if err != nil {
	// 	applog.LogErrorf(s.ctx, "Failed to get active game process name: %v", err)
	// 	return false
	// }

	// if game.ProcessName == "" {
	// 	return false
	// }

	// // 比较当前焦点进程和游戏进程名（不区分大小写）
	// currentProcess := s.getCurrentForegroundProcessName()
	// return strings.EqualFold(currentProcess, game.ProcessName)
}

// 修改checkKeyboardState方法以包含进程焦点检查
func (s *HotkeyService) checkKeyboardState(lastKeyState map[int]bool) {
	s.hotkeyLock.RLock()
	defer s.hotkeyLock.RUnlock()

	// 首先检查是否有活动游戏且当前焦点进程匹配
	if !s.isFocusedProcessMatchesActiveGame() {
		// 如果没有匹配的活动游戏进程，不处理键盘事件
		return
	}

	// 原有的键码映射
	keyMap := map[int]int{
		// 字母键
		65: 0x41, // A
		66: 0x42, // B
		67: 0x43, // C
		68: 0x44, // D
		69: 0x45, // E
		70: 0x46, // F
		71: 0x47, // G
		72: 0x48, // H
		73: 0x49, // I
		74: 0x4A, // J
		75: 0x4B, // K
		76: 0x4C, // L
		77: 0x4D, // M
		78: 0x4E, // N
		79: 0x4F, // O
		80: 0x50, // P
		81: 0x51, // Q
		82: 0x52, // R
		83: 0x53, // S
		84: 0x54, // T
		85: 0x55, // U
		86: 0x56, // V
		87: 0x57, // W
		88: 0x58, // X
		89: 0x59, // Y
		90: 0x5A, // Z

		// 数字键
		48: 0x30, // 0
		49: 0x31, // 1
		50: 0x32, // 2
		51: 0x33, // 3
		52: 0x34, // 4
		53: 0x35, // 5
		54: 0x36, // 6
		55: 0x37, // 7
		56: 0x38, // 8
		57: 0x39, // 9

		// 功能键
		112: VK_F1,
		113: VK_F2,
		114: VK_F3,
		115: VK_F4,
		116: VK_F5,
		117: VK_F6,
		118: VK_F7,
		119: VK_F8,
		120: VK_F9,
		121: VK_F10,
		122: VK_F11,
		123: VK_F12,

		// 控制键
		16: VK_SHIFT,
		17: VK_CONTROL,
		18: VK_MENU,
		32: VK_SPACE,
		13: VK_RETURN,
		27: VK_ESCAPE,
	}

	// 获取当前修饰键
	modifiers := s.getModifierKeys()

	// 检查每个监控的键
	for keyCode, vkCode := range keyMap {
		currentState := s.isKeyPressed(vkCode)
		lastState, exists := lastKeyState[keyCode]

		// 如果按键状态发生变化且当前是按下状态，则触发事件
		if (!exists || !lastState) && currentState {
			// applog.LogInfof(s.ctx, "按下01 %s", keyCode)
			// 创建键盘事件
			event := keyboard.KeyEvent{
				Key: keyCode,
			}

			// 将ModifierKey切片转换为整数切片用于比较
			modifierInts := make([]int, len(modifiers))
			for i, mod := range modifiers {
				switch mod {
				case enums.ModifierCtrl:
					modifierInts[i] = 17
				case enums.ModifierShift:
					modifierInts[i] = 16
				case enums.ModifierAlt:
					modifierInts[i] = 18
				}
			}

			// 检查是否有匹配的快捷键（包括修饰键）
			if s.hasMatchingHotkey(keyCode, modifierInts) {
				go s.handleKeyboardEvent(event)
			}
		}

		// 更新状态
		lastKeyState[keyCode] = currentState
	}
}

// isKeyPressed 检查指定虚拟键是否被按下
func (s *HotkeyService) isKeyPressed(vkCode int) bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vkCode))
	return (ret & KEY_PRESSED) != 0
}

// startAlternativeKeyListener 备用键盘监听方案
func (s *HotkeyService) startAlternativeKeyListener() {
	applog.LogInfof(s.ctx, "Starting alternative keyboard listener...")

	// 初始化键状态映射
	lastKeyState := make(map[int]bool)

	// 使用较短的时间间隔以获得更好的响应性
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkKeyboardState(lastKeyState)
		case <-s.ctx.Done():
			applog.LogInfof(s.ctx, "Alternative keyboard listener stopped")
			return
		}
	}
}

// handleKeyboardEvent 处理键盘事件
func (s *HotkeyService) handleKeyboardEvent(event keyboard.KeyEvent) {
	fmt.Printf("key press 01:%d\n", event.Key)
	s.hotkeyLock.RLock()
	defer s.hotkeyLock.RUnlock()

	keyCode := fmt.Sprintf("%d", event.Key)
	hotkey, exists := s.hotkeys[keyCode]
	if keyCode == "74" {
		s.takeScreenshot()
		return
	}
	if !exists {
		return
	}

	// 只处理键盘类型的快捷键
	if hotkey.DeviceType != enums.DeviceTypeKeyboard {
		return
	}

	// 检查是否应该执行此快捷键
	if s.shouldExecuteHotkey(hotkey) {
		fmt.Printf("key press:%s\n", hotkey.KeyCode)
		s.executeHotkeyAction(hotkey.ActionType, hotkey.ActionParams)
	}
}

// handleJoystickEvent 处理手柄事件
func (s *HotkeyService) handleJoystickEvent(deviceID string, event interface{}) {
	s.hotkeyLock.RLock()
	defer s.hotkeyLock.RUnlock()

	// 根据设备ID找到对应的设备类型
	s.deviceInfoLock.RLock()
	device, deviceExists := s.connectedDevices[deviceID]
	applog.LogDebugf(s.ctx, "device: %v", device)
	s.deviceInfoLock.RUnlock()

	if !deviceExists {
		return
	}

	// 根据手柄事件类型处理
	// 这里需要根据具体的gobot joystick事件格式来实现
	// 暂时留空，后续根据实际的手柄事件结构来完善
}

// shouldExecuteHotkey 判断是否应该执行快捷键
func (s *HotkeyService) shouldExecuteHotkey(hotkey *models.Hotkey) bool {
	// 全局快捷键总是可以执行
	if hotkey.IsGlobal() {
		return true
	}

	// 游戏特定快捷键只有在对应游戏运行时才能执行
	s.activeLock.RLock()
	activeGameID := s.activeGameID
	s.activeLock.RUnlock()

	return hotkey.GameID == activeGameID
}

// executeHotkeyAction 执行快捷键动作
func (s *HotkeyService) executeHotkeyAction(actionType enums.HotkeyActionType, params models.ActionParams) {
	switch actionType {
	case enums.HotkeyActionStartGame:
		gameID, ok := params["game_id"].(string)
		if ok && gameID != "" {
			s.startGame(gameID)
		}
	case enums.HotkeyActionStopGame:
		s.stopActiveGame()
	case enums.HotkeyActionTogglePause:
		s.toggleGamePause()
	case enums.HotkeyActionScreenshot:
		s.takeScreenshot()
	case enums.HotkeyActionCustom:
		s.executeCustomAction(params)
	default:
		applog.LogWarningf(s.ctx, "Unknown hotkey action type: %s", actionType)
	}
}

// startDeviceDetection 启动设备检测
func (s *HotkeyService) startDeviceDetection() {
	// 定期检测连接的设备
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.detectConnectedDevices()
		case <-s.ctx.Done():
			return
		}
	}
}

// detectConnectedDevices 检测连接的设备
func (s *HotkeyService) detectConnectedDevices() {
	s.deviceInfoLock.Lock()
	defer s.deviceInfoLock.Unlock()

	// 检测键盘（始终认为已连接）
	s.updateDeviceStatus(string(enums.DeviceTypeKeyboard), "Standard Keyboard", "keyboard_001", true)

	// TODO: 检测其他手柄设备
	// 这里需要根据实际的设备检测逻辑来实现

	// 清理长时间未活动的设备
	now := time.Now()
	for deviceID, device := range s.connectedDevices {
		applog.LogDebugf(s.ctx, "Checking device %s: %+v", deviceID, device)
		if now.Sub(device.LastSeenAt) > 30*time.Second {
			device.IsActive = false
			// 更新数据库
			s.updateDeviceInDB(device)
		}
	}
}

// updateDeviceStatus 更新设备状态
func (s *HotkeyService) updateDeviceStatus(deviceType, deviceName, deviceID string, isConnected bool) {
	device, exists := s.connectedDevices[deviceID]
	now := time.Now()

	if !exists {
		// 新设备
		device = &models.ConnectedDevice{
			ID:          fmt.Sprintf("device_%s_%d", deviceID, now.Unix()),
			DeviceType:  enums.DeviceType(deviceType),
			DeviceName:  deviceName,
			DeviceID:    deviceID,
			IsActive:    isConnected,
			ConnectedAt: now,
			LastSeenAt:  now,
		}
		s.connectedDevices[deviceID] = device
		s.insertDeviceToDB(device)
	} else {
		// 更新现有设备
		device.IsActive = isConnected
		device.LastSeenAt = now
		s.updateDeviceInDB(device)
	}
}

// insertDeviceToDB 插入设备到数据库
func (s *HotkeyService) insertDeviceToDB(device *models.ConnectedDevice) {
	query := `INSERT INTO connected_devices (id, device_type, device_name, device_id, is_active, connected_at, last_seen_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query,
		device.ID, device.DeviceType, device.DeviceName, device.DeviceID,
		device.IsActive, device.ConnectedAt, device.LastSeenAt)

	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to insert device: %v", err)
	}
}

// updateDeviceInDB 更新设备信息到数据库
func (s *HotkeyService) updateDeviceInDB(device *models.ConnectedDevice) {
	query := `UPDATE connected_devices SET is_active=?, last_seen_at=? WHERE device_id=?`
	_, err := s.db.Exec(query, device.IsActive, device.LastSeenAt, device.DeviceID)

	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to update device: %v", err)
	}
}

// setActiveGame 设置当前活动游戏
func (s *HotkeyService) setActiveGame(gameID string) {
	s.activeLock.Lock()
	defer s.activeLock.Unlock()
	s.activeGameID = gameID
}

// startGame 启动游戏
func (s *HotkeyService) startGame(gameID string) {
	// 这里需要调用游戏服务来启动游戏
	// 暂时只是记录日志
	applog.LogInfof(s.ctx, "Starting game: %s", gameID)
	s.setActiveGame(gameID)
}

// stopActiveGame 停止当前活动游戏
func (s *HotkeyService) stopActiveGame() {
	s.activeLock.RLock()
	gameID := s.activeGameID
	s.activeLock.RUnlock()

	if gameID != "" {
		applog.LogInfof(s.ctx, "Stopping game: %s", gameID)
		s.setActiveGame("")
	}
}

// toggleGamePause 切换游戏暂停状态
func (s *HotkeyService) toggleGamePause() {
	s.activeLock.RLock()
	gameID := s.activeGameID
	s.activeLock.RUnlock()

	if gameID != "" {
		applog.LogInfof(s.ctx, "Toggling pause for game: %s", gameID)
	}
}

// 修正后的纯Windows API截图实现
func (s *HotkeyService) takeScreenshot() {
	applog.LogInfof(s.ctx, "Taking screenshot using Windows API")

	// 获取前台窗口句柄
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		applog.LogErrorf(s.ctx, "Failed to get foreground window")
		return
	}

	// 获取窗口位置和大小
	var rect Rect
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		applog.LogErrorf(s.ctx, "Failed to get window rectangle")
		return
	}

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)

	if width <= 0 || height <= 0 {
		applog.LogErrorf(s.ctx, "Invalid window dimensions: %dx%d", width, height)
		return
	}

	// 获取屏幕DC
	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		applog.LogErrorf(s.ctx, "Failed to get screen DC")
		return
	}
	defer procReleaseDC.Call(0, screenDC)

	// 创建内存DC
	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		applog.LogErrorf(s.ctx, "Failed to create memory DC")
		return
	}
	defer procDeleteDC.Call(memDC)

	// 创建位图
	hBitmap, _, _ := procCreateCompatibleBitmap.Call(screenDC, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		applog.LogErrorf(s.ctx, "Failed to create bitmap")
		return
	}
	defer procDeleteObject.Call(hBitmap)

	// 选择位图到内存DC
	oldObj, _, _ := procSelectObject.Call(memDC, hBitmap)
	defer procSelectObject.Call(memDC, oldObj)

	// 复制屏幕内容到位图
	success, _, _ := procBitBlt.Call(
		memDC, 0, 0, uintptr(width), uintptr(height),
		screenDC, uintptr(rect.Left), uintptr(rect.Top), SRCCOPY,
	)

	if success == 0 {
		applog.LogErrorf(s.ctx, "Failed to copy screen to bitmap")
		return
	}

	// 将位图数据提取出来并保存为文件
	s.saveBitmapToFile(hBitmap, width, height, "1.jpg")
	applog.LogInfof(s.ctx, "Screenshot saved as 1.jpg (%dx%d)", width, height)
}

// saveBitmapToFile 将位图保存为JPEG文件
// func (s *HotkeyService) saveBitmapToFile(hBitmap uintptr, width, height int, filename string) {
// 	// 获取位图信息
// 	type BITMAPINFOHEADER struct {
// 		BiSize          uint32
// 		BiWidth         int32
// 		BiHeight        int32
// 		BiPlanes        uint16
// 		BiBitCount      uint16
// 		BiCompression   uint32
// 		BiSizeImage     uint32
// 		BiXPelsPerMeter int32
// 		BiYPelsPerMeter int32
// 		BiClrUsed       uint32
// 		BiClrImportant  uint32
// 	}

// 	var bmi struct {
// 		BmiHeader BITMAPINFOHEADER
// 	}

// 	bmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bmi.BmiHeader))
// 	bmi.BmiHeader.BiWidth = int32(width)
// 	bmi.BmiHeader.BiHeight = int32(-height) // 负值表示自上而下
// 	bmi.BmiHeader.BiPlanes = 1
// 	bmi.BmiHeader.BiBitCount = 32
// 	bmi.BmiHeader.BiCompression = 0 // BI_RGB

// 	// 分配像素缓冲区
// 	bufferSize := width * height * 4
// 	pixels := make([]byte, bufferSize)

// 	// 获取设备上下文
// 	screenDC, _, _ := procGetDC.Call(0)
// 	defer procReleaseDC.Call(0, screenDC)

// 	// 获取位图数据
// 	ret, _, _ := procGetDIBits.Call(
// 		screenDC,
// 		hBitmap,
// 		0,
// 		uintptr(height),
// 		uintptr(unsafe.Pointer(&pixels[0])),
// 		uintptr(unsafe.Pointer(&bmi)),
// 		0, // DIB_RGB_COLORS
// 	)

// 	if ret == 0 {
// 		applog.LogErrorf(s.ctx, "Failed to get DIB bits")
// 		return
// 	}

// 	// 将BGRA数据转换为image.RGBA格式
// 	img := image.NewRGBA(image.Rect(0, 0, width, height))

// 	for y := 0; y < height; y++ {
// 		for x := 0; x < width; x++ {
// 			srcIdx := (y*width + x) * 4
// 			dstIdx := y*img.Stride + x*4

// 			// BGRA to RGBA转换
// 			img.Pix[dstIdx+0] = pixels[srcIdx+2] // R
// 			img.Pix[dstIdx+1] = pixels[srcIdx+1] // G
// 			img.Pix[dstIdx+2] = pixels[srcIdx+0] // B
// 			img.Pix[dstIdx+3] = pixels[srcIdx+3] // A
// 		}
// 	}

// 	// 保存为JPEG文件
// 	file, err := os.Create(filename)
// 	if err != nil {
// 		applog.LogErrorf(s.ctx, "Failed to create JPEG file %s: %v", filename, err)
// 		return
// 	}
// 	defer file.Close()

// 	// JPEG编码选项
// 	options := jpeg.Options{
// 		Quality: 90, // 高质量
// 	}

// 	err = jpeg.Encode(file, img, &options)
// 	if err != nil {
// 		applog.LogErrorf(s.ctx, "Failed to encode JPEG: %v", err)
// 		return
// 	}

// 	applog.LogInfof(s.ctx, "Screenshot saved as %s (%dx%d)", filename, width, height)
// }

// saveBitmapToFile 将位图保存为JPEG文件 - 修正版本
func (s *HotkeyService) saveBitmapToFile(hBitmap uintptr, width, height int, filename string) {
	// 使用更简单直接的方法获取位图数据
	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		applog.LogErrorf(s.ctx, "Failed to get screen DC")
		return
	}
	defer procReleaseDC.Call(0, screenDC)

	// 创建内存DC
	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		applog.LogErrorf(s.ctx, "Failed to create memory DC")
		return
	}
	defer procDeleteDC.Call(memDC)

	// 创建新的位图用于数据提取
	newBitmap, _, _ := procCreateCompatibleBitmap.Call(screenDC, uintptr(width), uintptr(height))
	if newBitmap == 0 {
		applog.LogErrorf(s.ctx, "Failed to create compatible bitmap")
		return
	}
	defer procDeleteObject.Call(newBitmap)

	// 选择新位图到内存DC
	oldBitmap, _, _ := procSelectObject.Call(memDC, newBitmap)
	defer procSelectObject.Call(memDC, oldBitmap)

	// 从原始位图复制数据到新位图
	success, _, _ := procBitBlt.Call(
		memDC, 0, 0, uintptr(width), uintptr(height),
		screenDC, 0, 0, SRCCOPY,
	)

	if success == 0 {
		applog.LogErrorf(s.ctx, "Failed to copy bitmap data")
		return
	}

	// 使用更可靠的GetDIBits调用方式
	type BITMAPINFOHEADER struct {
		BiSize          uint32
		BiWidth         int32
		BiHeight        int32
		BiPlanes        uint16
		BiBitCount      uint16
		BiCompression   uint32
		BiSizeImage     uint32
		BiXPelsPerMeter int32
		BiYPelsPerMeter int32
		BiClrUsed       uint32
		BiClrImportant  uint32
	}

	// 创建BITMAPINFO结构
	bmi := struct {
		Header BITMAPINFOHEADER
		Colors [256]uint32 // 调色板颜色（32位位图不需要）
	}{
		Header: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      int32(-height), // 负值表示自上而下的DIB
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: 0, // BI_RGB
		},
	}

	// 分配像素缓冲区 (BGRA格式)
	bufferSize := width * height * 4
	pixels := make([]byte, bufferSize)

	// 获取位图数据 - 使用正确的参数顺序
	ret, _, err := procGetDIBits.Call(
		memDC,                               // hdc
		newBitmap,                           // hbmp
		0,                                   // uStartScan
		uintptr(height),                     // cScanLines
		uintptr(unsafe.Pointer(&pixels[0])), // lpvBits
		uintptr(unsafe.Pointer(&bmi)),       // lpbi
		0,                                   // uUsage (DIB_RGB_COLORS = 0)
	)

	if ret == 0 {
		applog.LogErrorf(s.ctx, "Failed to get DIB bits, error: %v", err)
		// 尝试另一种方法
		s.saveAsBMPDirectly(newBitmap, width, height, "1.bmp")
		return
	}

	// 转换BGRA到RGBA
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcIdx := (y*width + x) * 4
			dstIdx := y*img.Stride + x*4

			// BGRA to RGBA
			img.Pix[dstIdx+0] = pixels[srcIdx+2] // R
			img.Pix[dstIdx+1] = pixels[srcIdx+1] // G
			img.Pix[dstIdx+2] = pixels[srcIdx+0] // B
			img.Pix[dstIdx+3] = pixels[srcIdx+3] // A
		}
	}

	// 保存为JPEG
	jpegFilename := "1.jpg"
	file, err := os.Create(jpegFilename)
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to create JPEG file: %v", err)
		return
	}
	defer file.Close()

	options := jpeg.Options{Quality: 90}
	err = jpeg.Encode(file, img, &options)
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to encode JPEG: %v", err)
		return
	}

	applog.LogInfof(s.ctx, "Screenshot saved as %s (%dx%d)", jpegFilename, width, height)
}

// saveAsBMPDirectly 实现BMP格式保存
func (s *HotkeyService) saveAsBMPDirectly(hBitmap uintptr, width, height int, filename string) {
	applog.LogInfof(s.ctx, "Saving bitmap as BMP format: %s", filename)

	// 获取设备上下文
	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		applog.LogErrorf(s.ctx, "Failed to get screen DC for BMP saving")
		return
	}
	defer procReleaseDC.Call(0, screenDC)

	// 创建BITMAPINFO结构用于获取位图数据
	type BITMAPINFOHEADER struct {
		BiSize          uint32
		BiWidth         int32
		BiHeight        int32
		BiPlanes        uint16
		BiBitCount      uint16
		BiCompression   uint32
		BiSizeImage     uint32
		BiXPelsPerMeter int32
		BiYPelsPerMeter int32
		BiClrUsed       uint32
		BiClrImportant  uint32
	}

	bmi := struct {
		Header BITMAPINFOHEADER
	}{
		Header: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      int32(-height), // 负值表示自上而下
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: 0, // BI_RGB
		},
	}

	// 分配像素缓冲区
	bufferSize := width * height * 4
	pixels := make([]byte, bufferSize)

	// 获取位图数据
	ret, _, _ := procGetDIBits.Call(
		screenDC,
		hBitmap,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bmi)),
		0, // DIB_RGB_COLORS
	)

	if ret == 0 {
		applog.LogErrorf(s.ctx, "Failed to get DIB bits for BMP saving")
		return
	}

	// 创建BMP文件
	file, err := os.Create(filename)
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to create BMP file %s: %v", filename, err)
		return
	}
	defer file.Close()

	// 写入BMP文件头 (14 bytes)
	fileHeader := make([]byte, 14)
	fileHeader[0] = 'B'
	fileHeader[1] = 'M'

	// 计算文件总大小
	fileSize := uint32(14 + 40 + bufferSize) // 文件头 + 信息头 + 像素数据
	binary.LittleEndian.PutUint32(fileHeader[2:6], fileSize)

	// 保留字段设为0
	binary.LittleEndian.PutUint32(fileHeader[6:10], 0)

	// 数据偏移量 (文件头 + 信息头)
	binary.LittleEndian.PutUint32(fileHeader[10:14], 54)

	file.Write(fileHeader)

	// 写入BMP信息头 (40 bytes)
	infoHeader := make([]byte, 40)

	// 信息头大小
	binary.LittleEndian.PutUint32(infoHeader[0:4], 40)

	// 图像宽度和高度
	binary.LittleEndian.PutUint32(infoHeader[4:8], uint32(width))
	binary.LittleEndian.PutUint32(infoHeader[8:12], uint32(height))

	// 平面数
	binary.LittleEndian.PutUint16(infoHeader[12:14], 1)

	// 位深度
	binary.LittleEndian.PutUint16(infoHeader[14:16], 32)

	// 压缩方式 (BI_RGB = 0)
	binary.LittleEndian.PutUint32(infoHeader[16:20], 0)

	// 图像大小 (0表示未压缩)
	binary.LittleEndian.PutUint32(infoHeader[20:24], 0)

	// 分辨率 (设为默认值)
	binary.LittleEndian.PutUint32(infoHeader[24:28], 0)
	binary.LittleEndian.PutUint32(infoHeader[28:32], 0)

	// 颜色数和重要颜色数
	binary.LittleEndian.PutUint32(infoHeader[32:36], 0)
	binary.LittleEndian.PutUint32(infoHeader[36:40], 0)

	file.Write(infoHeader)

	// 写入像素数据（BGRA格式）
	file.Write(pixels)

	applog.LogInfof(s.ctx, "BMP screenshot saved as %s (%dx%d)", filename, width, height)
}

// createBMPFileHeader 创建BMP文件头
func (s *HotkeyService) createBMPFileHeader(width, height, dataSize int) []byte {
	fileSize := 14 + 40 + dataSize // 文件头(14) + 信息头(40) + 数据大小
	header := make([]byte, 14)

	header[0] = 'B'
	header[1] = 'M'
	binary.LittleEndian.PutUint32(header[2:6], uint32(fileSize))
	binary.LittleEndian.PutUint32(header[10:14], 54) // 数据偏移量

	return header
}

// createBMPInfoHeader 创建BMP信息头
func (s *HotkeyService) createBMPInfoHeader(width, height int) []byte {
	header := make([]byte, 40)

	binary.LittleEndian.PutUint32(header[0:4], 40) // 头大小
	binary.LittleEndian.PutUint32(header[4:8], uint32(width))
	binary.LittleEndian.PutUint32(header[8:12], uint32(height))
	binary.LittleEndian.PutUint16(header[12:14], 1)  // 平面数
	binary.LittleEndian.PutUint16(header[14:16], 32) // 位深度
	// 其他字段保持为0

	return header
}

// executeCustomAction 执行自定义动作
func (s *HotkeyService) executeCustomAction(params models.ActionParams) {
	command, ok := params["command"].(string)
	if !ok {
		applog.LogWarning(s.ctx, "Custom action missing command parameter")
		return
	}

	applog.LogInfof(s.ctx, "Executing custom command: %s", command)
	// 执行自定义命令逻辑
}

// GetSupportedDevices 获取支持的设备类型
func (s *HotkeyService) GetSupportedDevices() []models.DeviceTypeInfo {
	return models.GetSupportedDevices()
}

// GetConnectedDevices 获取已连接的设备
func (s *HotkeyService) GetConnectedDevices() ([]models.ConnectedDevice, error) {
	s.deviceInfoLock.RLock()
	defer s.deviceInfoLock.RUnlock()

	devices := make([]models.ConnectedDevice, 0, len(s.connectedDevices))
	for _, device := range s.connectedDevices {
		if device.IsActive {
			devices = append(devices, *device)
		}
	}

	return devices, nil
}

// GetAllHotkeys 获取所有快捷键配置
func (s *HotkeyService) GetAllHotkeys() (*models.HotkeyConfig, error) {
	s.hotkeyLock.RLock()
	defer s.hotkeyLock.RUnlock()

	config := &models.HotkeyConfig{
		GlobalHotkeys: make([]models.Hotkey, 0),
		GameHotkeys:   make([]models.Hotkey, 0),
	}

	for _, hotkey := range s.hotkeys {
		if hotkey.IsGlobal() {
			config.GlobalHotkeys = append(config.GlobalHotkeys, *hotkey)
		} else {
			config.GameHotkeys = append(config.GameHotkeys, *hotkey)
		}
	}

	return config, nil
}

// GetHotkeysByDeviceType 根据设备类型获取快捷键
func (s *HotkeyService) GetHotkeysByDeviceType(deviceType enums.DeviceType) ([]models.Hotkey, error) {
	s.hotkeyLock.RLock()
	defer s.hotkeyLock.RUnlock()

	var hotkeys []models.Hotkey
	for _, hotkey := range s.hotkeys {
		if hotkey.DeviceType == deviceType {
			hotkeys = append(hotkeys, *hotkey)
		}
	}

	return hotkeys, nil
}

// AddHotkey 添加快捷键
func (s *HotkeyService) AddHotkey(hotkey *models.Hotkey) error {
	s.hotkeyLock.Lock()
	defer s.hotkeyLock.Unlock()

	// 插入数据库
	query := `INSERT INTO hotkeys (id, game_id, name, device_type, key_code, modifiers, action_type, 
	action_params, 
	is_enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	paras, err := json.Marshal(hotkey.ActionParams)
	_, err = s.db.Exec(query,
		hotkey.ID, hotkey.GameID, hotkey.Name, string(hotkey.DeviceType), hotkey.KeyCode,
		utils.JoinString(hotkey.Modifiers, ",", func(t1 enums.ModifierKey) string { return string(t1) }), string(hotkey.ActionType), string(paras),
		hotkey.IsEnabled, hotkey.CreatedAt, hotkey.UpdatedAt)

	if err != nil {
		fmt.Printf("failed to insert hotkey: %v\n", err)
		return fmt.Errorf("failed to insert hotkey: %w", err)
	}

	// 更新内存缓存
	s.hotkeys[hotkey.KeyCode] = hotkey

	return nil
}

// UpdateHotkey 更新快捷键
func (s *HotkeyService) UpdateHotkey(hotkey *models.Hotkey) error {
	s.hotkeyLock.Lock()
	defer s.hotkeyLock.Unlock()

	query := `UPDATE hotkeys SET game_id=?, name=?, device_type=?, key_code=?, modifiers=?, action_type=?, action_params=?, is_enabled=?, updated_at=? WHERE id=?`
	paras, err := json.Marshal(hotkey.ActionParams)
	_, err = s.db.Exec(query,
		hotkey.GameID, hotkey.Name, string(hotkey.DeviceType), hotkey.KeyCode,
		utils.JoinString(hotkey.Modifiers, ",", func(t1 enums.ModifierKey) string { return string(t1) }), string(hotkey.ActionType), string(paras),
		hotkey.IsEnabled, hotkey.UpdatedAt, hotkey.ID)

	if err != nil {
		fmt.Printf("failed to update hotkey: %v\n", err)
		return fmt.Errorf("failed to update hotkey: %w", err)
	}

	// 更新内存缓存
	s.hotkeys[hotkey.KeyCode] = hotkey

	return nil
}

// DeleteHotkey 删除快捷键
func (s *HotkeyService) DeleteHotkey(hotkeyID string) error {
	s.hotkeyLock.Lock()
	defer s.hotkeyLock.Unlock()

	// 找到要删除的快捷键
	var keyCodeToDelete string
	for keyCode, hotkey := range s.hotkeys {
		if hotkey.ID == hotkeyID {
			keyCodeToDelete = keyCode
			break
		}
	}

	if keyCodeToDelete == "" {
		return fmt.Errorf("hotkey not found: %s", hotkeyID)
	}

	// 从数据库删除
	query := `DELETE FROM hotkeys WHERE id=?`
	_, err := s.db.Exec(query, hotkeyID)
	if err != nil {
		return fmt.Errorf("failed to delete hotkey: %w", err)
	}

	// 从内存缓存删除
	delete(s.hotkeys, keyCodeToDelete)

	return nil
}

// EnableHotkey 启用快捷键
func (s *HotkeyService) EnableHotkey(hotkeyID string) error {
	return s.updateHotkeyStatus(hotkeyID, true)
}

// DisableHotkey 禁用快捷键
func (s *HotkeyService) DisableHotkey(hotkeyID string) error {
	return s.updateHotkeyStatus(hotkeyID, false)
}

// updateHotkeyStatus 更新快捷键启用状态
func (s *HotkeyService) updateHotkeyStatus(hotkeyID string, enabled bool) error {
	s.hotkeyLock.Lock()
	defer s.hotkeyLock.Unlock()

	// 更新数据库
	query := `UPDATE hotkeys SET is_enabled=?, updated_at=? WHERE id=?`
	_, err := s.db.Exec(query, enabled, time.Now(), hotkeyID)
	if err != nil {
		return fmt.Errorf("failed to update hotkey status: %w", err)
	}

	// 更新内存缓存
	for _, hotkey := range s.hotkeys {
		if hotkey.ID == hotkeyID {
			hotkey.IsEnabled = enabled
			hotkey.UpdatedAt = time.Now()
			break
		}
	}

	return nil
}

// ... existing code ...

// GetGlobalHotkeys 获取全局快捷键
func (s *HotkeyService) GetGlobalHotkeys() ([]models.Hotkey, error) {
	s.hotkeyLock.RLock()
	defer s.hotkeyLock.RUnlock()

	var globalHotkeys []models.Hotkey
	for _, hotkey := range s.hotkeys {
		if hotkey.IsGlobal() {
			globalHotkeys = append(globalHotkeys, *hotkey)
		}
	}

	return globalHotkeys, nil
}

// GetScreenshotHotkey 获取截图快捷键
func (s *HotkeyService) GetScreenshotHotkey() (*models.Hotkey, error) {
	s.hotkeyLock.RLock()
	defer s.hotkeyLock.RUnlock()

	for _, hotkey := range s.hotkeys {
		if hotkey.IsGlobal() && hotkey.ActionType == enums.HotkeyActionScreenshot {
			return hotkey, nil
		}
	}

	return nil, nil // 没有设置截图快捷键
}

// SetScreenshotHotkey 设置截图快捷键
func (s *HotkeyService) SetScreenshotHotkey(hotkey *models.Hotkey) error {
	// 确保是全局截图快捷键
	hotkey.GameID = models.GlobalGameID
	hotkey.ActionType = enums.HotkeyActionScreenshot
	hotkey.Name = "截图快捷键"
	hotkey.IsEnabled = true
	hotkey.UpdatedAt = time.Now()

	// 如果已存在截图快捷键，先删除旧的
	existing, err := s.GetScreenshotHotkey()
	if err != nil {
		return err
	}

	if existing != nil {
		// 删除旧的截图快捷键
		err = s.DeleteHotkey(existing.ID)
		if err != nil {
			return fmt.Errorf("failed to delete existing screenshot hotkey: %w", err)
		}
	}

	// 添加新的截图快捷键
	return s.AddHotkey(hotkey)
}

// getModifierKeys 获取当前按下的修饰键
func (s *HotkeyService) getModifierKeys() []enums.ModifierKey {
	var modifiers []enums.ModifierKey

	if s.isKeyPressed(VK_CONTROL) {
		modifiers = append(modifiers, enums.ModifierCtrl)
	}
	if s.isKeyPressed(VK_SHIFT) {
		modifiers = append(modifiers, enums.ModifierShift)
	}
	if s.isKeyPressed(VK_MENU) {
		modifiers = append(modifiers, enums.ModifierAlt)
	}

	return modifiers
}

func (s *HotkeyService) checkKeyboardStateWithModifiers(lastKeyState map[int]bool) {
	s.hotkeyLock.RLock()
	defer s.hotkeyLock.RUnlock()

	// 原有的键码映射保持不变
	keyMap := map[int]int{
		65: 0x41, // A
		66: 0x42, // B
		67: 0x43, // C
		68: 0x44, // D
		69: 0x45, // E
		70: 0x46, // F
		71: 0x47, // G
		72: 0x48, // H
		73: 0x49, // I
		74: 0x4A, // J
		75: 0x4B, // K
		76: 0x4C, // L
		77: 0x4D, // M
		78: 0x4E, // N
		79: 0x4F, // O
		80: 0x50, // P
		81: 0x51, // Q
		82: 0x52, // R
		83: 0x53, // S
		84: 0x54, // T
		85: 0x55, // U
		86: 0x56, // V
		87: 0x57, // W
		88: 0x58, // X
		89: 0x59, // Y
		90: 0x5A, // Z

		// 数字键
		48: 0x30, // 0
		49: 0x31, // 1
		50: 0x32, // 2
		51: 0x33, // 3
		52: 0x34, // 4
		53: 0x35, // 5
		54: 0x36, // 6
		55: 0x37, // 7
		56: 0x38, // 8
		57: 0x39, // 9

		// 功能键
		112: VK_F1,
		113: VK_F2,
		114: VK_F3,
		115: VK_F4,
		116: VK_F5,
		117: VK_F6,
		118: VK_F7,
		119: VK_F8,
		120: VK_F9,
		121: VK_F10,
		122: VK_F11,
		123: VK_F12,

		// 控制键
		16: VK_SHIFT,
		17: VK_CONTROL,
		18: VK_MENU,
		32: VK_SPACE,
		13: VK_RETURN,
		27: VK_ESCAPE,
	}

	modifiers := s.getModifierKeys()

	// 检查每个监控的键
	for keyCode, vkCode := range keyMap {
		currentState := s.isKeyPressed(vkCode)
		lastState, exists := lastKeyState[keyCode]

		// 如果按键状态发生变化且当前是按下状态，则触发事件
		if (!exists || !lastState) && currentState {
			// 创建包含修饰键信息的键盘事件
			event := keyboard.KeyEvent{
				Key: keyCode,
			}

			// 将ModifierKey切片转换为整数切片用于比较
			modifierInts := make([]int, len(modifiers))
			for i, mod := range modifiers {
				switch mod {
				case enums.ModifierCtrl:
					modifierInts[i] = 17
				case enums.ModifierShift:
					modifierInts[i] = 16
				case enums.ModifierAlt:
					modifierInts[i] = 18
				}
			}

			// 检查是否有匹配的快捷键（包括修饰键）
			if s.hasMatchingHotkey(keyCode, modifierInts) {
				go s.handleKeyboardEvent(event)
			}
		}

		// 更新状态
		lastKeyState[keyCode] = currentState
	}
}

// hasMatchingHotkey 检查是否存在匹配的快捷键（包括修饰键）
// hasMatchingHotkey 检查是否存在匹配的快捷键（包括修饰键）
func (s *HotkeyService) hasMatchingHotkey(keyCode int, modifiers []int) bool {
	return true
	keyCodeStr := fmt.Sprintf("%d", keyCode)
	hotkey, exists := s.hotkeys[keyCodeStr]
	if !exists {
		return false
	}

	// 将整数修饰键转换为字符串修饰键进行比较
	modifierStrings := make([]enums.ModifierKey, len(modifiers))
	for i, mod := range modifiers {
		switch mod {
		case 17: // Ctrl
			modifierStrings[i] = enums.ModifierCtrl
		case 16: // Shift
			modifierStrings[i] = enums.ModifierShift
		case 18: // Alt
			modifierStrings[i] = enums.ModifierAlt
		default:
			continue // 跳过不支持的修饰键
		}
	}

	// 检查修饰键是否匹配
	if len(modifierStrings) == 0 && len(hotkey.Modifiers) == 0 {
		return true
	}

	// 比较修饰键数组长度
	if len(modifierStrings) != len(hotkey.Modifiers) {
		return false
	}

	// 创建映射便于查找
	requiredMods := make(map[enums.ModifierKey]bool)
	for _, requiredMod := range hotkey.Modifiers {
		requiredMods[requiredMod] = true
	}

	// 检查每个修饰键是否存在
	for _, modStr := range modifierStrings {
		if !requiredMods[modStr] {
			return false
		}
	}

	return true
}

// getCurrentForegroundProcessName 获取当前前台窗口的进程名称
func (s *HotkeyService) getCurrentForegroundProcessName() string {
	// 获取前台窗口句柄
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}

	// 获取窗口对应的进程ID
	var processID uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processID)))
	if processID == 0 {
		return ""
	}

	// 打开进程句柄
	handle, _, _ := procOpenProcess.Call(
		PROCESS_QUERY_LIMITED_INFORMATION,
		0,
		uintptr(processID),
	)
	if handle == 0 {
		return ""
	}
	defer procCloseHandle.Call(handle)

	// 获取进程名称
	buffer := make([]uint16, 260)
	bufferSize := uint32(len(buffer))

	ret, _, _ := procQueryFullProcessImageNameW.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&bufferSize)),
	)

	if ret == 0 {
		return ""
	}

	// 转换为Go字符串并提取文件名
	fullPath := syscall.UTF16ToString(buffer[:bufferSize])

	// 从完整路径中提取可执行文件名
	lastSlash := -1
	for i := len(fullPath) - 1; i >= 0; i-- {
		if fullPath[i] == '\\' || fullPath[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash >= 0 && lastSlash < len(fullPath)-1 {
		return fullPath[lastSlash+1:]
	}

	return fullPath
}

// startProcessFocusMonitoring 启动进程焦点监控
func (s *HotkeyService) startProcessFocusMonitoring() {
	s.processCheckTicker = time.NewTicker(500 * time.Millisecond) // 每500ms检查一次

	go func() {
		for {
			select {
			case <-s.processCheckTicker.C:
				newProcessName := s.getCurrentForegroundProcessName()
				if newProcessName != s.focusedProcessName {
					s.focusedProcessName = newProcessName
					applog.LogDebugf(s.ctx, "Focus changed to process: %s", newProcessName)
				}
			case <-s.ctx.Done():
				if s.processCheckTicker != nil {
					s.processCheckTicker.Stop()
				}
				return
			}
		}
	}()
}
