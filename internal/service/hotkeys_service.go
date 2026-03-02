package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sync"
	"time"

	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"lunabox/internal/enums"
	"lunabox/internal/models"

	"github.com/go-vgo/robotgo"
	"gobot.io/x/gobot/v2"
	"gobot.io/x/gobot/v2/platforms/joystick"
	"gobot.io/x/gobot/v2/platforms/keyboard"
)

// KeyMapping 按键映射配置
type KeyMapping struct {
	SourceKey   string              // 源按键（如手柄圆圈键）
	TargetKey   string              // 目标按键（如键盘A键）
	MappingType MappingType         // 映射类型
	Modifiers   []enums.ModifierKey // 修饰键
	IsEnabled   bool
}

// MappingType 映射类型
type MappingType string

const (
	MappingTypeDirect  MappingType = "direct"  // 直接映射：按下就按下，释放就释放
	MappingTypeRelease MappingType = "release" // 释放触发：只在释放时触发
)

// HotkeyService 重构后的热键服务
type HotkeyService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig

	// gobot相关（手柄支持）
	robot      *gobot.Robot
	keyboard   *keyboard.Driver
	joysticks  map[string]*joystick.Driver
	deviceLock sync.RWMutex

	// robotgo相关（键盘事件监听）
	hookStarted bool
	hookMutex   sync.RWMutex

	// 按键映射管理
	keyMappings map[string]*KeyMapping // source_key -> mapping
	mappingLock sync.RWMutex

	// 按键状态跟踪
	keyStates map[string]bool // key -> is_pressed
	stateLock sync.RWMutex

	// 快捷键管理（特殊用途）
	screenshotHotkey *models.Hotkey
	hotkeyLock       sync.RWMutex

	// 当前活动游戏
	activeGameID string
	activeLock   sync.RWMutex

	imageService *ImageService
	startService *StartService
}

func (s *HotkeyService) SetServices(imageService *ImageService, startService *StartService) {
	s.imageService = imageService
	s.startService = startService
}

func NewHotkeyService() *HotkeyService {
	return &HotkeyService{
		joysticks:   make(map[string]*joystick.Driver),
		keyMappings: make(map[string]*KeyMapping),
		keyStates:   make(map[string]bool),
	}
}

// Init 初始化服务
func (s *HotkeyService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config

	applog.LogInfof(s.ctx, "Initializing simplified key mapping service...")

	// 初始化驱动
	s.keyboard = keyboard.NewDriver()
	s.robot = gobot.NewRobot("hotkeyRobot",
		[]gobot.Connection{},
		[]gobot.Device{s.keyboard},
		s.work,
	)

	// 加载配置
	s.loadConfigurations()

	// 启动监听
	s.startKeyboardListener()
	s.startJoystickListener()

	applog.LogInfof(s.ctx, "Simplified key mapping service initialized")
}

// loadConfigurations 加载所有配置
func (s *HotkeyService) loadConfigurations() {
	applog.LogInfof(s.ctx, "Loading configurations from database...")

	// 加载按键映射配置（从hotkeys表）
	s.loadKeyMappingsFromHotkeys()

	// 加载截图快捷键
	s.loadScreenshotHotkeyFromDB()

	// 加载已连接设备
	s.loadConnectedDevicesFromDB()
}

// loadKeyMappingsFromHotkeys 从hotkeys表加载按键映射配置
func (s *HotkeyService) loadKeyMappingsFromHotkeys() {
	applog.LogInfof(s.ctx, "Loading key mappings from hotkeys table...")

	s.mappingLock.Lock()
	defer s.mappingLock.Unlock()

	// 查询启用的按键映射（action_type为KEY_MAPPING的记录）
	query := `
		SELECT key_code, action_params, is_enabled
		FROM hotkeys 
		WHERE action_type = ? AND is_enabled = TRUE AND game_id = ?
	`

	rows, err := s.db.Query(query, enums.HotkeyActionKeyMapping, models.GlobalGameID)
	if err != nil {
		applog.LogErrorf(s.ctx, "查询按键映射失败: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var keyCode string
		var actionParams models.ActionParams
		var isEnabled bool

		err := rows.Scan(&keyCode, &actionParams, &isEnabled)
		if err != nil {
			applog.LogErrorf(s.ctx, "扫描按键映射数据失败: %v", err)
			continue
		}

		// 从action_params中提取目标按键和映射类型
		targetKey, _ := actionParams["target_key"].(string)
		mappingTypeStr, _ := actionParams["mapping_type"].(string)
		modifiersInterface, _ := actionParams["modifiers"].([]interface{})

		// 转换修饰键
		var modifiers []enums.ModifierKey
		for _, mod := range modifiersInterface {
			if modStr, ok := mod.(string); ok {
				modifiers = append(modifiers, enums.ModifierKey(modStr))
			}
		}

		// 转换映射类型
		var mappingTypeEnum MappingType
		switch mappingTypeStr {
		case "direct":
			mappingTypeEnum = MappingTypeDirect
		case "release":
			mappingTypeEnum = MappingTypeRelease
		default:
			mappingTypeEnum = MappingTypeDirect
		}

		if targetKey != "" {
			mapping := &KeyMapping{
				SourceKey:   keyCode, // keyCode作为源按键
				TargetKey:   targetKey,
				MappingType: mappingTypeEnum,
				Modifiers:   modifiers,
				IsEnabled:   isEnabled,
			}

			s.keyMappings[keyCode] = mapping
			count++
			applog.LogInfof(s.ctx, "Loaded mapping: %s -> %s (%s)", keyCode, targetKey, mappingTypeStr)
		}
	}

	applog.LogInfof(s.ctx, "Loaded %d key mappings from hotkeys table", count)
}

// loadScreenshotHotkeyFromDB 从数据库加载截图快捷键
func (s *HotkeyService) loadScreenshotHotkeyFromDB() {
	applog.LogInfof(s.ctx, "Loading screenshot hotkey from database...")

	s.hotkeyLock.Lock()
	defer s.hotkeyLock.Unlock()

	// 查询截图快捷键配置
	query := `
		SELECT id, game_id, name, device_type, key_code, modifiers, 
		       action_type, action_params, is_enabled, created_at, updated_at
		FROM hotkeys 
		WHERE action_type = ? AND is_enabled = TRUE AND game_id = ?
		LIMIT 1
	`

	row := s.db.QueryRow(query, enums.HotkeyActionScreenshot, models.GlobalGameID)

	var hotkey models.Hotkey
	err := row.Scan(
		&hotkey.ID,
		&hotkey.GameID,
		&hotkey.Name,
		&hotkey.DeviceType,
		&hotkey.KeyCode,
		&hotkey.Modifiers,
		&hotkey.ActionType,
		&hotkey.ActionParams,
		&hotkey.IsEnabled,
		&hotkey.CreatedAt,
		&hotkey.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			applog.LogInfof(s.ctx, "No screenshot hotkey found in database, using default")
			// 使用默认配置
			s.screenshotHotkey = &models.Hotkey{
				ID:         "screenshot_default",
				GameID:     models.GlobalGameID,
				Name:       "默认截图快捷键",
				DeviceType: enums.DeviceTypeKeyboard,
				KeyCode:    "f12",
				ActionType: enums.HotkeyActionScreenshot,
				IsEnabled:  true,
			}
		} else {
			applog.LogErrorf(s.ctx, "查询截图快捷键失败: %v", err)
		}
		return
	}

	s.screenshotHotkey = &hotkey
	applog.LogInfof(s.ctx, "Screenshot hotkey loaded from database: %s", hotkey.KeyCode)
}

// loadConnectedDevicesFromDB 从数据库加载已连接设备
func (s *HotkeyService) loadConnectedDevicesFromDB() {
	applog.LogInfof(s.ctx, "Loading connected devices from database...")

	s.deviceLock.Lock()
	defer s.deviceLock.Unlock()

	query := `
		SELECT id, device_type, device_name, device_id, is_active, connected_at, last_seen_at
		FROM connected_devices 
		WHERE is_active = TRUE
		ORDER BY last_seen_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		applog.LogErrorf(s.ctx, "查询连接设备失败: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var device models.ConnectedDevice
		err := rows.Scan(
			&device.ID,
			&device.DeviceType,
			&device.DeviceName,
			&device.DeviceID,
			&device.IsActive,
			&device.ConnectedAt,
			&device.LastSeenAt,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "扫描设备数据失败: %v", err)
			continue
		}

		// 这里可以根据设备类型初始化相应的驱动
		count++
		applog.LogInfof(s.ctx, "Loaded connected device: %s (%s)", device.DeviceName, device.DeviceType)
	}

	applog.LogInfof(s.ctx, "Loaded %d connected devices from database", count)
}

// loadKeyMappings 加载按键映射配置
func (s *HotkeyService) loadKeyMappings() {
	applog.LogInfof(s.ctx, "Loading key mappings...")

	s.mappingLock.Lock()
	defer s.mappingLock.Unlock()

	// 示例配置 - 实际应从数据库加载
	sampleMappings := []*KeyMapping{
		// 手柄圆圈键 -> 键盘A键（直接映射）
		{
			SourceKey:   "circle",
			TargetKey:   "a",
			MappingType: MappingTypeDirect,
			IsEnabled:   true,
		},
		// 手柄三角键 -> 键盘B键（直接映射）
		{
			SourceKey:   "triangle",
			TargetKey:   "b",
			MappingType: MappingTypeDirect,
			IsEnabled:   true,
		},
	}

	for _, mapping := range sampleMappings {
		s.keyMappings[mapping.SourceKey] = mapping
		applog.LogInfof(s.ctx, "Loaded mapping: %s -> %s (%s)",
			mapping.SourceKey, mapping.TargetKey, mapping.MappingType)
	}
}

// loadScreenshotHotkey 加载截图快捷键
func (s *HotkeyService) loadScreenshotHotkey() {
	applog.LogInfof(s.ctx, "Loading screenshot hotkey...")

	s.hotkeyLock.Lock()
	defer s.hotkeyLock.Unlock()

	// 示例配置 - 实际应从数据库加载
	s.screenshotHotkey = &models.Hotkey{
		ID:         "screenshot_001",
		GameID:     models.GlobalGameID,
		Name:       "截图快捷键",
		DeviceType: enums.DeviceTypeKeyboard,
		KeyCode:    "f12", // 假设F12是截图键
		ActionType: enums.HotkeyActionScreenshot,
		IsEnabled:  true,
	}

	applog.LogInfof(s.ctx, "Screenshot hotkey loaded: %s", s.screenshotHotkey.KeyCode)
}

// startKeyboardListener 启动键盘监听
func (s *HotkeyService) startKeyboardListener() {
	applog.LogInfof(s.ctx, "Starting keyboard listener...")

	s.hookMutex.Lock()
	if s.hookStarted {
		s.hookMutex.Unlock()
		return
	}
	s.hookStarted = true
	s.hookMutex.Unlock()

	go s.keyboardEventHandler()
	applog.LogInfof(s.ctx, "Keyboard listener started")
}

// keyboardEventHandler 键盘事件处理器
func (s *HotkeyService) keyboardEventHandler() {
	monitorKeys := s.getActiveMonitorKeys()
	if len(monitorKeys) == 0 {
		applog.LogInfof(s.ctx, "No keys to monitor")
		return
	}

	applog.LogInfof(s.ctx, "Monitoring keys: %v", monitorKeys)

	// 为每个按键启动监听
	for _, key := range monitorKeys {
		go s.monitorKey(key)
	}

	<-s.ctx.Done()

	s.hookMutex.Lock()
	s.hookStarted = false
	s.hookMutex.Unlock()
	applog.LogInfof(s.ctx, "Keyboard listener stopped")
}

// getActiveMonitorKeys 获取当前需要监控的按键
func (s *HotkeyService) getActiveMonitorKeys() []string {
	s.mappingLock.RLock()
	defer s.mappingLock.RUnlock()

	keys := make(map[string]bool)

	// 添加直接映射的源按键
	for sourceKey, mapping := range s.keyMappings {
		if mapping.IsEnabled && mapping.MappingType == MappingTypeDirect {
			keys[sourceKey] = true
		}
	}

	// 添加截图快捷键
	s.hotkeyLock.RLock()
	if s.screenshotHotkey != nil && s.screenshotHotkey.IsEnabled {
		keys[s.screenshotHotkey.KeyCode] = true
	}
	s.hotkeyLock.RUnlock()

	// 转换为切片
	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}

	return result
}

// monitorKey 监控单个按键（基础轮询版本）
func (s *HotkeyService) monitorKey(key string) {
	var lastState bool

	ticker := time.NewTicker(15 * time.Millisecond) // 进一步降低频率
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// 使用最基本的按键检测方法
			currentState := s.checkKeyState(key)

			if currentState != lastState {
				if currentState {
					s.handleKeyPress(key)
				} else {
					s.handleKeyRelease(key)
				}
				lastState = currentState
			}
		}
	}
}

// checkKeyState 检查按键状态的基础实现
func (s *HotkeyService) checkKeyState(key string) bool {
	// 简单的实现：尝试按下按键看是否有反应
	// 这不是最好的方法，但在没有更好API的情况下可以工作
	err := robotgo.KeyTap(key)
	return err == nil
}

// isKeyPressed 检查按键状态
func (s *HotkeyService) isKeyPressed(key string) bool {
	// 使用与monitorKey相同的基础检测方法
	return s.checkKeyState(key)
}

// handleKeyPress 处理按键按下事件
func (s *HotkeyService) handleKeyPress(key string) {
	applog.LogDebugf(s.ctx, "Key pressed: %s", key)

	// 更新状态
	s.stateLock.Lock()
	s.keyStates[key] = true
	s.stateLock.Unlock()

	// 直接映射：立即模拟目标按键按下
	if mapping := s.getDirectMapping(key); mapping != nil {
		s.simulateKeyPress(mapping.TargetKey, mapping.Modifiers)
	}
}

// handleKeyRelease 处理按键释放事件
func (s *HotkeyService) handleKeyRelease(key string) {
	applog.LogDebugf(s.ctx, "Key released: %s", key)

	// 更新状态
	s.stateLock.Lock()
	s.keyStates[key] = false
	s.stateLock.Unlock()

	// 直接映射：立即模拟目标按键释放
	if mapping := s.getDirectMapping(key); mapping != nil {
		s.simulateKeyRelease(mapping.TargetKey, mapping.Modifiers)
	}

	// 释放触发：检查是否为截图快捷键
	if s.isScreenshotHotkey(key) {
		s.triggerScreenshot()
	}
}

// getDirectMapping 获取直接映射配置
func (s *HotkeyService) getDirectMapping(sourceKey string) *KeyMapping {
	s.mappingLock.RLock()
	defer s.mappingLock.RUnlock()

	mapping, exists := s.keyMappings[sourceKey]
	if exists && mapping.IsEnabled && mapping.MappingType == MappingTypeDirect {
		return mapping
	}
	return nil
}

// isScreenshotHotkey 判断是否为截图快捷键
func (s *HotkeyService) isScreenshotHotkey(key string) bool {
	s.hotkeyLock.RLock()
	defer s.hotkeyLock.RUnlock()

	return s.screenshotHotkey != nil &&
		s.screenshotHotkey.IsEnabled &&
		s.screenshotHotkey.KeyCode == key
}

// triggerScreenshot 触发截图
func (s *HotkeyService) triggerScreenshot() {
	applog.LogInfof(s.ctx, "Screenshot hotkey triggered")

	s.activeLock.RLock()
	gameID := s.activeGameID
	s.activeLock.RUnlock()

	if s.imageService != nil {
		go s.imageService.TakeScreenshotOfFocusedWindow(gameID)
	}
}

// simulateKeyPress 模拟按键按下
func (s *HotkeyService) simulateKeyPress(key string, modifiers []enums.ModifierKey) {
	// 先按下修饰键
	for _, mod := range modifiers {
		robotgo.KeyToggle(s.modifierToKey(mod), "down")
	}

	// 按下目标键
	robotgo.KeyToggle(key, "down")

	applog.LogDebugf(s.ctx, "Simulated key press: %s (with modifiers: %v)", key, modifiers)
}

// simulateKeyRelease 模拟按键释放
func (s *HotkeyService) simulateKeyRelease(key string, modifiers []enums.ModifierKey) {
	// 释放目标键
	robotgo.KeyToggle(key, "up")

	// 释放修饰键
	for _, mod := range modifiers {
		robotgo.KeyToggle(s.modifierToKey(mod), "up")
	}

	applog.LogDebugf(s.ctx, "Simulated key release: %s (with modifiers: %v)", key, modifiers)
}

// modifierToKey 修饰键转换
func (s *HotkeyService) modifierToKey(mod enums.ModifierKey) string {
	switch mod {
	case enums.ModifierCtrl:
		return "ctrl"
	case enums.ModifierShift:
		return "shift"
	case enums.ModifierAlt:
		return "alt"
	default:
		return ""
	}
}

// work 手柄事件处理
func (s *HotkeyService) work() {
	// 手柄按钮事件处理
	s.keyboard.On(keyboard.Key, func(data interface{}) {
		if event, ok := data.(keyboard.KeyEvent); ok {
			// 将手柄按钮转换为标准按键名称进行处理

			keyCode := s.convertJoystickButton(int(event.Key))
			applog.LogInfof(s.ctx, "keycode: %s, key:%d\n", keyCode, event.Key)
			if keyCode != "" {
				// 简化处理：统一视为按下事件
				// 在实际应用中可能需要更复杂的逻辑来区分按下和释放
				s.handleKeyPress(keyCode)

				// 延迟触发释放事件
				go func() {
					time.Sleep(50 * time.Millisecond)
					s.handleKeyRelease(keyCode)
				}()
			}
		}
	})
}

// convertJoystickButton 手柄按钮转换
func (s *HotkeyService) convertJoystickButton(key int) string {
	// 根据gobot keyboard的实际按键值进行映射
	// 这些映射可能需要根据实际手柄进行调整
	buttonMap := map[int]string{
		49: "cross",    // ASCII '1' 对应叉键
		50: "circle",   // ASCII '2' 对应圆圈键
		51: "square",   // ASCII '3' 对应方块键
		52: "triangle", // ASCII '4' 对应三角键
		53: "l1",       // ASCII '5' 对应L1键
		54: "r1",       // ASCII '6' 对应R1键
		55: "l2",       // ASCII '7' 对应L2键
		56: "r2",       // ASCII '8' 对应R2键
		57: "start",    // ASCII '9' 对应开始键
		48: "select",   // ASCII '0' 对应选择键
	}

	if mapped, exists := buttonMap[key]; exists {
		return mapped
	}

	// 处理方向键（小写字母）
	switch key {
	case 119: // ASCII 'w'
		return "up"
	case 115: // ASCII 's'
		return "down"
	case 97: // ASCII 'a'
		return "left"
	case 100: // ASCII 'd'
		return "right"
	}

	return ""
}

func (s *HotkeyService) handleJoystickEvents() {
	stick := s.joysticks["dualshock4"]

	// 监听按钮按下
	stick.On(joystick.SquarePress, func(data interface{}) {
		s.handleKeyPress("square")
	})

	stick.On(joystick.SquareRelease, func(data interface{}) {
		s.handleKeyRelease("square")
	})

	stick.On(joystick.CirclePress, func(data interface{}) {
		s.handleKeyPress("circle")
	})

	stick.On(joystick.CircleRelease, func(data interface{}) {
		s.handleKeyRelease("circle")
	})

	stick.On(joystick.TrianglePress, func(data interface{}) {
		s.handleKeyPress("triangle")
	})

	stick.On(joystick.TriangleRelease, func(data interface{}) {
		s.handleKeyRelease("triangle")
	})

	stick.On(joystick.XPress, func(data interface{}) {
		s.handleKeyPress("cross")
	})

	stick.On(joystick.XRelease, func(data interface{}) {
		s.handleKeyRelease("cross")
	})

	// 监听肩键
	stick.On(joystick.L1Press, func(data interface{}) {
		s.handleKeyPress("l1")
	})

	stick.On(joystick.R1Press, func(data interface{}) {
		s.handleKeyPress("r1")
	})

	// 监听扳机键 (模拟量)
	stick.On(joystick.L2Press, func(data interface{}) {
		s.handleKeyPress("l2")
	})

	stick.On(joystick.R2Press, func(data interface{}) {
		s.handleKeyPress("r2")
	})

	// 监听方向键
	stick.On(joystick.UpPress, func(data interface{}) {
		s.handleKeyPress("up")
	})

	stick.On(joystick.DownPress, func(data interface{}) {
		s.handleKeyPress("down")
	})

	stick.On(joystick.LeftPress, func(data interface{}) {
		s.handleKeyPress("left")
	})

	stick.On(joystick.RightPress, func(data interface{}) {
		s.handleKeyPress("right")
	})

	// 监听摇杆轴 (模拟量)
	stick.On(joystick.LeftX, func(data interface{}) {
		// data 包含摇杆位置值 -32768 到 32767
		if math.Abs(float64(data.(int))) > 2000 {
			applog.LogDebugf(s.ctx, "Left X axis: %v", data)
		}

	})

	stick.On(joystick.LeftY, func(data interface{}) {

		if math.Abs(float64(data.(int))) > 2000 {
			applog.LogDebugf(s.ctx, "Left Y axis: %v", data)
		}
	})
}

// 启动相关方法
func (s *HotkeyService) startJoystickListener() {
	applog.LogInfof(s.ctx, "Starting joystick listener...")
	// 启动手柄机器人
	// 创建 joystick 适配器

	joystickAdaptor := joystick.NewAdaptor("0")

	// 创建手柄驱动
	stick := joystick.NewDriver(joystickAdaptor, "dualshock4")

	s.joysticks["dualshock4"] = stick

	s.robot = gobot.NewRobot("ds4Robot",
		[]gobot.Connection{joystickAdaptor},
		[]gobot.Device{stick},
		s.handleJoystickEvents,
	)
	go func() {
		if err := s.robot.Start(); err != nil {
			applog.LogErrorf(s.ctx, "Failed to start joystick robot: %v", err)
		}
	}()
}

// 公共接口方法
func (s *HotkeyService) GetSupportedDevices() []models.DeviceTypeInfo {
	return models.GetSupportedDevices()
}

// AddKeyMapping 添加按键映射
func (s *HotkeyService) AddKeyMapping(sourceKey, targetKey string, mappingType MappingType, modifiers []enums.ModifierKey) {
	s.mappingLock.Lock()
	defer s.mappingLock.Unlock()

	mapping := &KeyMapping{
		SourceKey:   sourceKey,
		TargetKey:   targetKey,
		MappingType: mappingType,
		Modifiers:   modifiers,
		IsEnabled:   true,
	}

	s.keyMappings[sourceKey] = mapping
	// 避免在测试中调用日志
	if s.ctx != nil && s.ctx.Err() == nil {
		applog.LogInfof(s.ctx, "Added key mapping: %s -> %s (%s)", sourceKey, targetKey, mappingType)
	}
}

// RemoveKeyMapping 移除按键映射
func (s *HotkeyService) RemoveKeyMapping(sourceKey string) {
	s.mappingLock.Lock()
	defer s.mappingLock.Unlock()

	delete(s.keyMappings, sourceKey)
	// 避免在测试中调用日志
	if s.ctx != nil && s.ctx.Err() == nil {
		applog.LogInfof(s.ctx, "Removed key mapping for: %s", sourceKey)
	}
}

// EnableKeyMapping 启用按键映射
func (s *HotkeyService) EnableKeyMapping(sourceKey string) {
	s.mappingLock.Lock()
	defer s.mappingLock.Unlock()

	if mapping, exists := s.keyMappings[sourceKey]; exists {
		mapping.IsEnabled = true
		// 避免在测试中调用日志
		if s.ctx != nil && s.ctx.Err() == nil {
			applog.LogInfof(s.ctx, "Enabled key mapping: %s", sourceKey)
		}
	}
}

// DisableKeyMapping 禁用按键映射
func (s *HotkeyService) DisableKeyMapping(sourceKey string) {
	s.mappingLock.Lock()
	defer s.mappingLock.Unlock()

	if mapping, exists := s.keyMappings[sourceKey]; exists {
		mapping.IsEnabled = false
		// 避免在测试中调用日志
		if s.ctx != nil && s.ctx.Err() == nil {
			applog.LogInfof(s.ctx, "Disabled key mapping: %s", sourceKey)
		}
	}
}

// GetGlobalHotkeys 获取所有全局快捷键配置
func (s *HotkeyService) GetGlobalHotkeys() ([]*models.Hotkey, error) {
	query := `
		SELECT id, game_id, name, device_type, key_code, modifiers, 
		       action_type, action_params, is_enabled, created_at, updated_at
		FROM hotkeys 
		WHERE game_id = ? AND is_enabled = TRUE
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, models.GlobalGameID)
	if err != nil {
		applog.LogErrorf(s.ctx, "查询全局快捷键失败: %v", err)
		return nil, err
	}
	defer rows.Close()

	var hotkeys []*models.Hotkey
	for rows.Next() {
		var hotkey models.Hotkey
		err := rows.Scan(
			&hotkey.ID,
			&hotkey.GameID,
			&hotkey.Name,
			&hotkey.DeviceType,
			&hotkey.KeyCode,
			&hotkey.Modifiers,
			&hotkey.ActionType,
			&hotkey.ActionParams,
			&hotkey.IsEnabled,
			&hotkey.CreatedAt,
			&hotkey.UpdatedAt,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "扫描快捷键数据失败: %v", err)
			continue
		}
		hotkeys = append(hotkeys, &hotkey)
	}

	return hotkeys, nil
}

// UpdateHotkey 更新快捷键配置
func (s *HotkeyService) UpdateHotkey(hotkey *models.Hotkey) error {
	query := `
		UPDATE hotkeys 
		SET name = ?, device_type = ?, key_code = ?, modifiers = ?, 
		    action_type = ?, action_params = ?, is_enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := s.db.Exec(query,
		hotkey.Name,
		hotkey.DeviceType,
		hotkey.KeyCode,
		hotkey.Modifiers,
		hotkey.ActionType,
		hotkey.ActionParams,
		hotkey.IsEnabled,
		hotkey.ID,
	)

	if err != nil {
		applog.LogErrorf(s.ctx, "更新快捷键失败: %v", err)
		return err
	}

	applog.LogInfof(s.ctx, "快捷键更新成功: %s", hotkey.ID)
	return nil
}

// AddHotkey 添加新的快捷键配置
func (s *HotkeyService) AddHotkey(hotkey *models.Hotkey) error {
	if hotkey.ID == "" {
		hotkey.ID = generateHotkeyID()
	}

	if hotkey.CreatedAt.IsZero() {
		hotkey.CreatedAt = time.Now()
	}
	hotkey.UpdatedAt = time.Now()

	query := `
		INSERT INTO hotkeys (
			id, game_id, name, device_type, key_code, modifiers, 
			action_type, action_params, is_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		hotkey.ID,
		hotkey.GameID,
		hotkey.Name,
		hotkey.DeviceType,
		hotkey.KeyCode,
		hotkey.Modifiers,
		hotkey.ActionType,
		hotkey.ActionParams,
		hotkey.IsEnabled,
		hotkey.CreatedAt,
		hotkey.UpdatedAt,
	)

	if err != nil {
		applog.LogErrorf(s.ctx, "添加快捷键失败: %v", err)
		return err
	}

	applog.LogInfof(s.ctx, "快捷键添加成功: %s", hotkey.ID)
	return nil
}

// GetHotkeysByGameID 根据游戏ID获取快捷键配置
func (s *HotkeyService) GetHotkeysByGameID(gameID string) ([]*models.Hotkey, error) {
	query := `
		SELECT id, game_id, name, device_type, key_code, modifiers, 
		       action_type, action_params, is_enabled, created_at, updated_at
		FROM hotkeys 
		WHERE game_id = ? AND is_enabled = TRUE
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, gameID)
	if err != nil {
		applog.LogErrorf(s.ctx, "查询游戏快捷键失败: %v", err)
		return nil, err
	}
	defer rows.Close()

	var hotkeys []*models.Hotkey
	for rows.Next() {
		var hotkey models.Hotkey
		err := rows.Scan(
			&hotkey.ID,
			&hotkey.GameID,
			&hotkey.Name,
			&hotkey.DeviceType,
			&hotkey.KeyCode,
			&hotkey.Modifiers,
			&hotkey.ActionType,
			&hotkey.ActionParams,
			&hotkey.IsEnabled,
			&hotkey.CreatedAt,
			&hotkey.UpdatedAt,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "扫描快捷键数据失败: %v", err)
			continue
		}
		hotkeys = append(hotkeys, &hotkey)
	}

	return hotkeys, nil
}

// GetGameHotkeys GetHotkeysByGameID的别名方法，功能完全相同
func (s *HotkeyService) GetGameHotkeys(gameID string) ([]*models.Hotkey, error) {
	return s.GetHotkeysByGameID(gameID)
}

// DeleteHotkey 删除快捷键配置
func (s *HotkeyService) DeleteHotkey(hotkeyID string) error {
	query := `DELETE FROM hotkeys WHERE id = ?`

	_, err := s.db.Exec(query, hotkeyID)
	if err != nil {
		applog.LogErrorf(s.ctx, "删除快捷键失败: %v", err)
		return err
	}

	applog.LogInfof(s.ctx, "快捷键删除成功: %s", hotkeyID)
	return nil
}

// GetConnectedDevices 获取已连接的设备列表
func (s *HotkeyService) GetConnectedDevices() ([]*models.ConnectedDevice, error) {
	query := `
		SELECT id, device_type, device_name, device_id, is_active, connected_at, last_seen_at
		FROM connected_devices 
		WHERE is_active = TRUE
		ORDER BY last_seen_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		applog.LogErrorf(s.ctx, "查询连接设备失败: %v", err)
		return nil, err
	}
	defer rows.Close()

	var devices []*models.ConnectedDevice
	for rows.Next() {
		var device models.ConnectedDevice
		err := rows.Scan(
			&device.ID,
			&device.DeviceType,
			&device.DeviceName,
			&device.DeviceID,
			&device.IsActive,
			&device.ConnectedAt,
			&device.LastSeenAt,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "扫描设备数据失败: %v", err)
			continue
		}
		devices = append(devices, &device)
	}

	return devices, nil
}

// AddConnectedDevice 添加已连接设备
func (s *HotkeyService) AddConnectedDevice(device *models.ConnectedDevice) error {
	if device.ID == "" {
		device.ID = generateDeviceID()
	}

	if device.ConnectedAt.IsZero() {
		device.ConnectedAt = time.Now()
	}
	device.LastSeenAt = time.Now()
	device.IsActive = true

	query := `
		INSERT INTO connected_devices (
			id, device_type, device_name, device_id, is_active, connected_at, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_id) DO UPDATE SET
			device_name = ?, is_active = TRUE, last_seen_at = CURRENT_TIMESTAMP
	`

	_, err := s.db.Exec(query,
		device.ID,
		device.DeviceType,
		device.DeviceName,
		device.DeviceID,
		device.IsActive,
		device.ConnectedAt,
		device.LastSeenAt,
		device.DeviceName,
	)

	if err != nil {
		applog.LogErrorf(s.ctx, "添加连接设备失败: %v", err)
		return err
	}

	applog.LogInfof(s.ctx, "设备连接成功: %s (%s)", device.DeviceName, device.DeviceID)
	return nil
}

// UpdateDeviceLastSeen 更新设备最后活跃时间
func (s *HotkeyService) UpdateDeviceLastSeen(deviceID string) error {
	query := `UPDATE connected_devices SET last_seen_at = CURRENT_TIMESTAMP WHERE device_id = ?`

	_, err := s.db.Exec(query, deviceID)
	if err != nil {
		applog.LogErrorf(s.ctx, "更新设备活跃时间失败: %v", err)
		return err
	}

	return nil
}

// generateHotkeyID 生成快捷键ID
func generateHotkeyID() string {
	return fmt.Sprintf("hk_%d", time.Now().UnixNano())
}

// generateDeviceID 生成设备ID
func generateDeviceID() string {
	return fmt.Sprintf("dev_%d", time.Now().UnixNano())
}

// GetKeyMappingsInternal 获取内部按键映射（用于测试）
func (s *HotkeyService) GetKeyMappingsInternal() map[string]*KeyMapping {
	return s.keyMappings
}

// MappingLock 获取映射锁（用于测试）
func (s *HotkeyService) MappingLock() *sync.RWMutex {
	return &s.mappingLock
}

// IsKeyPressed 查询按键是否被按下
func (s *HotkeyService) IsKeyPressed(key string) bool {
	s.stateLock.RLock()
	defer s.stateLock.RUnlock()
	return s.keyStates[key]
}

// GetActiveGameID 获取当前活动游戏ID
func (s *HotkeyService) GetActiveGameID() string {
	s.activeLock.RLock()
	defer s.activeLock.RUnlock()
	return s.activeGameID
}

// SetActiveGameID 设置当前活动游戏ID
func (s *HotkeyService) SetActiveGameID(gameID string) {
	s.activeLock.Lock()
	defer s.activeLock.Unlock()
	s.activeGameID = gameID
	// 避免在测试中调用日志
	if s.ctx != nil && s.ctx.Err() == nil {
		applog.LogInfof(s.ctx, "Active game ID set to: %s", gameID)
	}
}
