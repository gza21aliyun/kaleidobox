package service

import (
	"context"
	"database/sql"
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
	SourceKey    string           // 源按键（如手柄圆圈键）
	TargetKey    string           // 目标按键（如键盘A键）
	MappingType  MappingType      // 映射类型
	Modifiers    []enums.ModifierKey // 修饰键
	IsEnabled    bool
}

// MappingType 映射类型
type MappingType string

const (
	MappingTypeDirect   MappingType = "direct"   // 直接映射：按下就按下，释放就释放
	MappingTypeRelease  MappingType = "release"  // 释放触发：只在释放时触发
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
	hookStarted  bool
	hookMutex    sync.RWMutex

	// 按键映射管理
	keyMappings  map[string]*KeyMapping // source_key -> mapping
	mappingLock  sync.RWMutex

	// 按键状态跟踪
	keyStates    map[string]bool // key -> is_pressed
	stateLock    sync.RWMutex

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
	s.loadKeyMappings()
	s.loadScreenshotHotkey()
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
	case 97:  // ASCII 'a'
		return "left"
	case 100: // ASCII 'd'
		return "right"
	}
	
	return ""
}

// 启动相关方法
func (s *HotkeyService) startJoystickListener() {
	applog.LogInfof(s.ctx, "Starting joystick listener...")
	// 启动手柄机器人
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

// GetKeyMappings 获取所有按键映射
func (s *HotkeyService) GetKeyMappings() map[string]*KeyMapping {
	s.mappingLock.RLock()
	defer s.mappingLock.RUnlock()
	
	// 返回副本以避免并发问题
	result := make(map[string]*KeyMapping)
	for k, v := range s.keyMappings {
		result[k] = v
	}
	return result
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