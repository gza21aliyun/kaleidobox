package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

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
// type KeyMapping struct {
// 	SourceKey   string              // 源按键（如手柄圆圈键）
// 	TargetKey   string              // 目标按键（如键盘A键）
// 	MappingType MappingType         // 映射类型
// 	Modifiers   []enums.ModifierKey // 修饰键
// 	IsEnabled   bool
// }

// MappingType 映射类型
type MappingType string

const (
	MappingTypeDirect  MappingType = "direct"  // 直接映射：按下就按下，释放就释放
	MappingTypeRelease MappingType = "release" // 释放触发：只在释放时触发

	KeyDs4L3Left   string = "l3left"
	KeyDs4L3Right  string = "l3right"
	KeyDs4L3Up     string = "l3up"
	KeyDs4L3Down   string = "l3down"
	KeyDs4R3Left   string = "r3left"
	KeyDs4R3Right  string = "r3right"
	KeyDs4R3Up     string = "r3up"
	KeyDs4R3Down   string = "r3down"
	KeyDs4L2       string = "l2"
	KeyDs4R2       string = "r2"
	KeyDs4L1       string = "l1"
	KeyDs4R1       string = "r1"
	KeyDs4Share    string = "share"
	KeyDs4Options  string = "options"
	KeyDs4PS       string = "ps"
	KeyDs4Touchpad string = "touchpad"
	KeyDs4Triangle string = "triangle"
	KeyDs4Circle   string = "circle"
	KeyDs4Cross    string = "cross"
	KeyDs4Square   string = "square"
)

// HotkeyService 重构后的热键服务
type HotkeyService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig

	// gobot相关（手柄支持）
	robot      *gobot.Robot
	robotMutex sync.Mutex
	keyboard   *keyboard.Driver
	joysticks  map[string]*joystick.Driver
	deviceLock sync.RWMutex

	// robotgo相关（键盘事件监听）
	hookStarted bool
	hookMutex   sync.RWMutex

	// 按键映射管理
	keyMappings   map[string]*models.Hotkey // source_key -> mapping
	actionKeys    map[string]*models.Hotkey
	mappingLock   sync.RWMutex
	actionkeyLock sync.RWMutex
	// keyLock     sync.RWMutex

	// 按键状态跟踪
	keyStates map[string]bool // key -> is_pressed
	stateLock sync.RWMutex

	// 快捷键管理（特殊用途）
	// screenshotHotkey *models.Hotkey

	// 当前活动游戏
	activeGameID atomic.Value
	activeLock   sync.RWMutex

	isMonitoringKeySetting atomic.Bool
	monitoredKey           atomic.Value

	imageService *ImageService
	startService *StartService

	processCheckTicker *time.Ticker
}

func (s *HotkeyService) SetServices(imageService *ImageService, startService *StartService) {
	s.imageService = imageService
	s.startService = startService
}

func NewHotkeyService() *HotkeyService {
	return &HotkeyService{
		joysticks:   make(map[string]*joystick.Driver),
		keyMappings: make(map[string]*models.Hotkey),
		actionKeys:  make(map[string]*models.Hotkey),
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
	// s.keyboard = keyboard.NewDriver()
	// s.robot = gobot.NewRobot("hotkeyRobot",
	// 	[]gobot.Connection{},
	// 	[]gobot.Device{s.keyboard},
	// 	s.work,
	// )

	// 加载配置
	s.loadConfigurations()

	// 启动监听
	// s.startKeyboardListener()
	s.startJoystickListener()
	s.monitoredKey.Store("")
	s.activeGameID.Store("")

	applog.LogInfof(s.ctx, "Simplified key mapping service initialized")
}

// loadConfigurations 加载所有配置
func (s *HotkeyService) loadConfigurations() {
	applog.LogInfof(s.ctx, "Loading configurations from database...")

	// 加载按键映射配置（从hotkeys表）
	// s.loadKeyMappingsFromHotkeys()
	s.loadHotkeyConfig("")

	// 加载截图快捷键
	// s.loadScreenshotHotkeyFromDB()

	// 加载已连接设备
	s.loadConnectedDevicesFromDB()
}

func (s *HotkeyService) fetchHotkeys(query string) ([]*models.Hotkey, error) {
	var rs []*models.Hotkey = []*models.Hotkey{}
	rows, err := s.db.Query(query)
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to load hotkeys: %v", err)
		return rs, err
	}
	defer rows.Close()

	for rows.Next() {
		var hotkey models.Hotkey
		// var modifiersBytes []byte

		err := rows.Scan(
			&hotkey.ID, &hotkey.GameID, &hotkey.Name, &hotkey.DeviceType, &hotkey.KeyCode,
			// &modifiersBytes,
			&hotkey.ActionType, &hotkey.ActionParams,
			&hotkey.IsEnabled, &hotkey.CreatedAt, &hotkey.UpdatedAt,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "Failed to scan hotkey: %v", err)
			continue
		}

		// 解析修饰键
		// if len(modifiersBytes) > 0 {
		// 	json.Unmarshal(modifiersBytes, &hotkey.Modifiers)
		// }

		// 解析动作参数
		// if len(paramsBytes) > 0 {
		// 	json.Unmarshal(paramsBytes, &hotkey.ActionParams)
		// }

		rs = append(rs, &hotkey)
		applog.LogInfof(s.ctx, "Loaded hotkey: %s, \n %v\n", hotkey.Name, hotkey)
	}
	return rs, err
}

func (s *HotkeyService) loadHotkeyConfig(gameId string) {
	applog.LogInfof(s.ctx, "Loading hotkey configuration")
	s.actionkeyLock.Lock()
	s.mappingLock.Lock()
	defer s.actionkeyLock.Unlock()
	defer s.mappingLock.Unlock()
	s.keyMappings = make(map[string]*models.Hotkey)
	s.actionKeys = make(map[string]*models.Hotkey)

	query := `SELECT id, game_id, name, device_type, key_code, action_type, action_params, is_enabled, created_at, updated_at 
	FROM hotkeys WHERE is_enabled = TRUE`
	if gameId != "" {
		query += fmt.Sprintf(" AND game_id = '%s'", gameId)
	}

	rows, _ := s.fetchHotkeys(query)

	for _, hotkey := range rows {
		if hotkey.ActionType != enums.HotkeyActionKeyMapping {
			s.actionKeys[hotkey.KeyCode] = hotkey
		} else {
			s.keyMappings[hotkey.KeyCode] = hotkey
		}

		applog.LogInfof(s.ctx, "Loaded hotkey: %s, \n %v\n", hotkey.Name, hotkey)
	}

	// 统计各类设备的快捷键数量
	deviceStats := make(map[enums.DeviceType]int)
	for _, hotkey := range s.keyMappings {
		deviceStats[hotkey.DeviceType]++
	}

	statsStr := ""
	for deviceType, count := range deviceStats {
		statsStr += fmt.Sprintf("%s:%d ", deviceType, count)
	}

	applog.LogInfof(s.ctx, "Loaded %d hotkeys (%s)", len(s.keyMappings), statsStr)
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

// startKeyboardListener 启动键盘监听
func (s *HotkeyService) startKeyboardListener() {
	applog.LogInfof(s.ctx, "Starting keyboard listener...")

	// go s.keyboardEventHandler()
	applog.LogInfof(s.ctx, "Keyboard listener started")
	s.robotMutex.Lock()
	defer s.robotMutex.Unlock()
	s.keyboard = keyboard.NewDriver()
	s.robot = gobot.NewRobot("keyboardbot",
		[]gobot.Connection{},
		[]gobot.Device{s.keyboard},
		s.handleKeboardEvents,
	)
	go func() {
		if err := s.robot.Start(); err != nil {
			applog.LogErrorf(s.ctx, "Failed to start keyboard robot: %v", err)
		}
	}()
}

func (s *HotkeyService) GetActiveGameID() string {
	if v := s.activeGameID.Load(); v != nil {
		return v.(string)
	}
	return ""
}

func (s *HotkeyService) SetActiveGameID(gameID string) {
	s.activeGameID.Store(gameID)
}

// handleKeyPress 处理按键按下事件
func (s *HotkeyService) handleKeyPress(key, name string, device enums.DeviceType) {
	applog.LogDebugf(s.ctx, "Key pressed: %s", key)
	hk := models.Hotkey{
		KeyCode:    key,
		Name:       name,
		DeviceType: device,
	}
	if s.GetActiveGameID() == "" {
		s.processCheck()
		if s.GetActiveGameID() == "" {
			s.monitoredKey.Store(hk)
			return
		}
	}

	// 更新状态
	s.stateLock.Lock()
	s.keyStates[key] = true
	s.stateLock.Unlock()

	if s.actionKeys[key] != nil { // 映射的按键
		s.monitoredKey.Store(hk)
		return
	}

	// 直接映射：立即模拟目标按键按下
	hotkey := s.keyMappings[key]
	if hotkey != nil {
		s.simulateKeyPress(hotkey.ActionParams, []enums.ModifierKey{})
	}
	s.monitoredKey.Store(hk)
}

func (s *HotkeyService) handleActionKey(key *models.Hotkey) {
	if key.ActionType == enums.HotkeyActionScreenshot {
		s.imageService.TakeScreenshotOfFocusedWindow(s.GetActiveGameID())
	}
}

// handleKeyRelease 处理按键释放事件
func (s *HotkeyService) handleKeyRelease(key, name string, device enums.DeviceType) {
	applog.LogDebugf(s.ctx, "Key released: %s", key)
	hk := models.Hotkey{
		KeyCode:    key,
		Name:       name,
		DeviceType: device,
	}
	// 更新状态
	s.stateLock.Lock()
	s.keyStates[key] = false
	s.stateLock.Unlock()

	hotkey := s.keyMappings[key]
	if hotkey != nil {
		s.simulateKeyRelease(hotkey.ActionParams, []enums.ModifierKey{})
		s.monitoredKey.Store(hk)
		return
	}
	hotkey = s.actionKeys[key]
	if hotkey != nil {
		s.handleActionKey(hotkey)
		s.monitoredKey.Store(hk)
		return
	}

}

// simulateKeyPress 模拟按键按下
func (s *HotkeyService) simulateKeyPress(key string, modifiers []enums.ModifierKey) {
	// 先按下修饰键
	// for _, mod := range modifiers {
	// 	robotgo.KeyToggle(s.modifierToKey(mod), "down")
	// }

	// 按下目标键
	robotgo.KeyToggle(key, "down")

	applog.LogDebugf(s.ctx, "Simulated key press: %s (with modifiers: %v)", key, modifiers)
}

// simulateKeyRelease 模拟按键释放
func (s *HotkeyService) simulateKeyRelease(key string, modifiers []enums.ModifierKey) {
	// 释放目标键
	robotgo.KeyToggle(key, "up")

	// 释放修饰键
	// for _, mod := range modifiers {
	// 	robotgo.KeyToggle(s.modifierToKey(mod), "up")
	// }

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
func (s *HotkeyService) handleKeboardEvents() {
	// 手柄按钮事件处理
	s.keyboard.On(keyboard.Key, func(data interface{}) {
		fmt.Println("handleKeboardEvents 01")
		if event, ok := data.(keyboard.KeyEvent); ok {
			// 将手柄按钮转换为标准按键名称进行处理

			// keyCode := s.convertJoystickButton(int(event.Key))
			// keyCode := fmt.Sprintf("%c", event.Key)
			keyCode := event.Char
			applog.LogInfof(s.ctx, "keycode: %s, key:%d\n", keyCode, event.Key)
			if keyCode != "" {
				// 简化处理：统一视为按下事件
				// 在实际应用中可能需要更复杂的逻辑来区分按下和释放
				s.actionkeyLock.Lock()
				defer s.actionkeyLock.Unlock()
				hotkey := s.actionKeys[keyCode]
				if hotkey != nil {
					s.handleActionKey(hotkey)
				}

				// s.handleKeyPress(keyCode)

				// // 延迟触发释放事件
				// go func() {
				// 	time.Sleep(50 * time.Millisecond)
				// 	s.handleKeyRelease(keyCode)
				// }()
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
	device := enums.DeviceTypeDualShock4
	stick := s.joysticks[string(device)]

	// 监听按钮按下
	stick.On(joystick.SquarePress, func(data interface{}) {
		s.handleKeyPress("square", "□", device)
	})

	stick.On(joystick.SquareRelease, func(data interface{}) {
		s.handleKeyRelease("square", "□", device)
	})

	stick.On(joystick.CirclePress, func(data interface{}) {
		s.handleKeyPress("circle", "○", device)
	})

	stick.On(joystick.CircleRelease, func(data interface{}) {
		s.handleKeyRelease("circle", "○", device)
	})

	stick.On(joystick.TrianglePress, func(data interface{}) {
		s.handleKeyPress("triangle", "△", device)
	})

	stick.On(joystick.TriangleRelease, func(data interface{}) {
		s.handleKeyRelease("triangle", "△", device)
	})

	stick.On(joystick.XPress, func(data interface{}) {
		s.handleKeyPress("cross", "X", device)
	})

	stick.On(joystick.XRelease, func(data interface{}) {
		s.handleKeyRelease("cross", "X", device)
	})

	// 监听肩键
	stick.On(joystick.L1Press, func(data interface{}) {
		s.handleKeyPress("l1", "L1", device)
	})

	stick.On(joystick.R1Press, func(data interface{}) {
		s.handleKeyPress("r1", "R1", device)
	})

	// 监听扳机键 (模拟量)
	stick.On(joystick.L2Press, func(data interface{}) {
		s.handleKeyPress("l2", "L2", device)
	})

	stick.On(joystick.R2Press, func(data interface{}) {
		s.handleKeyPress("r2", "R2", device)
	})

	stick.On(joystick.L3Press, func(data interface{}) {
		s.handleKeyPress("l3", "L3", device)
	})

	stick.On(joystick.R3Press, func(data interface{}) {
		s.handleKeyPress("r3", "R3", device)
	})

	// 监听方向键
	stick.On(joystick.UpPress, func(data interface{}) {
		s.handleKeyPress("up", "↑", device)
	})

	stick.On(joystick.DownPress, func(data interface{}) {
		s.handleKeyPress("down", "↓", device)
	})

	stick.On(joystick.LeftPress, func(data interface{}) {
		s.handleKeyPress("left", "←", device)
	})

	stick.On(joystick.RightPress, func(data interface{}) {
		s.handleKeyPress("right", "→", device)
	})

	// 监听摇杆轴 (模拟量)
	stick.On(joystick.LeftX, func(data interface{}) {
		// data 包含摇杆位置值 -32768 到 32767
		// var l3LeftIsRelease = true
		// var l3RightIsRelease = true
		applog.LogDebugf(s.ctx, "Left X axis: %v", data)
		if data.(int) > 5000 {

			s.toggleKey(KeyDs4L3Right, true, KeyDs4L3Right, device)
		} else if data.(int) < 5000 && data.(int) > -5000 {
			s.toggleKey(KeyDs4L3Right, false, KeyDs4L3Right, device)
			s.toggleKey(KeyDs4L3Left, false, KeyDs4L3Left, device)
		} else {
			s.toggleKey(KeyDs4L3Left, true, KeyDs4L3Left, device)
		}

	})

	stick.On(joystick.RightX, func(data interface{}) {
		// data 包含摇杆位置值 -32768 到 32767
		// var l3LeftIsRelease = true
		// var l3RightIsRelease = true
		applog.LogDebugf(s.ctx, "Left X axis: %v", data)
		if data.(int) > 5000 {

			s.toggleKey(KeyDs4R3Right, true, KeyDs4R3Right, device)
		} else if data.(int) < 5000 && data.(int) > -5000 {
			s.toggleKey(KeyDs4R3Right, false, KeyDs4R3Right, device)
			s.toggleKey(KeyDs4R3Left, false, KeyDs4R3Left, device)
		} else {
			s.toggleKey(KeyDs4R3Left, true, KeyDs4R3Left, device)
		}

	})

	stick.On(joystick.LeftY, func(data interface{}) {

		if data.(int) > 5000 {

			s.toggleKey(KeyDs4L3Up, true, KeyDs4L3Up, device)
		} else if data.(int) < 5000 && data.(int) > -5000 {
			s.toggleKey(KeyDs4L3Up, false, KeyDs4L3Up, device)
			s.toggleKey(KeyDs4L3Down, false, KeyDs4L3Down, device)
		} else {
			s.toggleKey(KeyDs4L3Down, true, KeyDs4L3Down, device)
		}
	})

	stick.On(joystick.RightY, func(data interface{}) {

		if data.(int) > 5000 {

			s.toggleKey(KeyDs4R3Up, true, KeyDs4R3Up, device)
		} else if data.(int) < 5000 && data.(int) > -5000 {
			s.toggleKey(KeyDs4R3Up, false, KeyDs4R3Up, device)
			s.toggleKey(KeyDs4R3Down, false, KeyDs4R3Down, device)
		} else {
			s.toggleKey(KeyDs4R3Down, true, KeyDs4R3Down, device)
		}
	})
}

func (s *HotkeyService) toggleKey(key string, isPress bool, name string, device enums.DeviceType) {
	s.stateLock.Lock()

	defer s.stateLock.Unlock()
	if isPress {
		if !s.keyStates[key] {
			s.handleKeyPress(key, name, device)
			s.keyStates[key] = true
		}
	} else {
		if s.keyStates[key] {
			s.handleKeyRelease(key, name, device)
			s.keyStates[key] = false
		}
	}

}

// 启动相关方法
func (s *HotkeyService) startJoystickListener() {
	applog.LogInfof(s.ctx, "Starting joystick listener...")
	devicetype := enums.DeviceTypeDualShock4
	s.robotMutex.Lock()
	defer s.robotMutex.Unlock()
	// 启动手柄机器人
	// 创建 joystick 适配器

	joystickAdaptor := joystick.NewAdaptor("0")

	// 创建手柄驱动
	stick := joystick.NewDriver(joystickAdaptor, string(devicetype))

	s.joysticks[string(devicetype)] = stick

	s.robot = gobot.NewRobot(string(devicetype)+"Robot",
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

// // EnableKeyMapping 启用按键映射
// func (s *HotkeyService) EnableKeyMapping(sourceKey string) {
// 	s.mappingLock.Lock()
// 	defer s.mappingLock.Unlock()

// 	if mapping, exists := s.keyMappings[sourceKey]; exists {
// 		mapping.IsEnabled = true
// 		// 避免在测试中调用日志
// 		if s.ctx != nil && s.ctx.Err() == nil {
// 			applog.LogInfof(s.ctx, "Enabled key mapping: %s", sourceKey)
// 		}
// 	}
// }

// DisableKeyMapping 禁用按键映射
// func (s *HotkeyService) DisableKeyMapping(sourceKey string) {
// 	s.mappingLock.Lock()
// 	defer s.mappingLock.Unlock()

// 	if mapping, exists := s.keyMappings[sourceKey]; exists {
// 		mapping.IsEnabled = false
// 		// 避免在测试中调用日志
// 		if s.ctx != nil && s.ctx.Err() == nil {
// 			applog.LogInfof(s.ctx, "Disabled key mapping: %s", sourceKey)
// 		}
// 	}
// }

// GetGlobalHotkeys 获取所有全局快捷键配置
func (s *HotkeyService) GetGlobalHotkeys() ([]models.Hotkey, error) {
	query := `
		SELECT id, game_id, name, device_type, key_code, 
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

	var hotkeys []models.Hotkey = []models.Hotkey{}
	for rows.Next() {
		var hotkey models.Hotkey
		err := rows.Scan(
			&hotkey.ID,
			&hotkey.GameID,
			&hotkey.Name,
			&hotkey.DeviceType,
			&hotkey.KeyCode,
			// &hotkey.Modifiers,
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
		hotkeys = append(hotkeys, hotkey)
	}

	return hotkeys, nil
}

// UpdateHotkey 更新快捷键配置
func (s *HotkeyService) UpdateHotkey(hotkey models.Hotkey) error {
	applog.LogInfo(s.ctx, "start to UpdateHotkey")
	query := `
		UPDATE hotkeys 
		SET name = ?, device_type = ?, key_code = ?, 
		    action_type = ?, action_params = ?, is_enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := s.db.Exec(query,
		hotkey.Name,
		hotkey.DeviceType,
		hotkey.KeyCode,
		// hotkey.Modifiers,
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
func (s *HotkeyService) AddHotkey(hotkey models.Hotkey) error {
	applog.LogInfo(s.ctx, "start to AddHotkey")
	if hotkey.ID == "" {
		hotkey.ID = generateHotkeyID()
	}

	if hotkey.CreatedAt.IsZero() {
		hotkey.CreatedAt = time.Now()
	}
	hotkey.UpdatedAt = time.Now()

	query := `
		INSERT INTO hotkeys (
			id, game_id, name, device_type, key_code, 
			action_type, action_params, is_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		hotkey.ID,
		hotkey.GameID,
		hotkey.Name,
		hotkey.DeviceType,
		hotkey.KeyCode,
		// hotkey.Modifiers,
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
func (s *HotkeyService) GetHotkeysByGameID(gameID string) ([]models.Hotkey, error) {
	query := `
		SELECT id, game_id, name, device_type, key_code, 
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

	var hotkeys []models.Hotkey = []models.Hotkey{}
	for rows.Next() {
		var hotkey models.Hotkey
		err := rows.Scan(
			&hotkey.ID,
			&hotkey.GameID,
			&hotkey.Name,
			&hotkey.DeviceType,
			&hotkey.KeyCode,
			// &hotkey.Modifiers,
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
		hotkeys = append(hotkeys, hotkey)
	}

	return hotkeys, nil
}

// GetGameHotkeys GetHotkeysByGameID的别名方法，功能完全相同
func (s *HotkeyService) GetGameHotkeys(gameID string) ([]models.Hotkey, error) {
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
// func (s *HotkeyService) GetKeyMappingsInternal() map[string]*KeyMapping {
// 	return s.keyMappings
// }

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

// startProcessFocusMonitoring 启动进程焦点监控
func (s *HotkeyService) startProcessFocusMonitoring() {
	s.processCheckTicker = time.NewTicker(50000 * time.Millisecond) // 每500ms检查一次
	applog.LogInfof(s.ctx, "startProcessFocusMonitoring 01")

	go func() {
		for {
			select {
			case <-s.processCheckTicker.C:
				// sessions := s.activeTimeTracker.GetAllActiveSessions()
				s.processCheck()

			case <-s.ctx.Done():
				if s.processCheckTicker != nil {
					s.processCheckTicker.Stop()
				}
				return
			}
		}
	}()
}

func (s *HotkeyService) processCheck() {
	games := s.startService.getSessionGames()

	newProcessId := getCurrentForegroundProcessId()
	// applog.LogInfof(s.ctx, "startProcessFocusMonitoring 02, num of sessions:%s, pid:%d\n",
	// 	utils.JoinString(games, ",", func(t1 GameProcess) string { return strconv.FormatUint(uint64(t1.ProcessID), 10) }), newProcessId)
	checked := false
	activeGameId := s.GetActiveGameID()
	for _, game := range games {
		if game.ProcessID == newProcessId {
			if activeGameId != game.GameId {
				s.SetActiveGameID(game.GameId)
				applog.LogDebugf(s.ctx, "Focus changed to game: %s", game.GameId)
			}
			checked = true
			break
		}
	}
	if !checked && activeGameId != "" {
		s.SetActiveGameID("")
		applog.LogDebugf(s.ctx, "Focus changed to unknown process: %d", newProcessId)
	}
}

// getCurrentForegroundProcessId 获取当前前台窗口的进程名称
func getCurrentForegroundProcessId() uint32 {
	// 获取前台窗口句柄
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0
	}

	// 获取窗口对应的进程ID
	var processID uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processID)))
	if processID == 0 {
		return 0
	}
	return processID
}

func (s *HotkeyService) MonitorKeySetting() (models.Hotkey, error) {
	s.monitoredKey.Store(nil)
	s.isMonitoringKeySetting.Store(true)
	var tiker *time.Ticker = time.NewTicker(time.Millisecond * 500)
	defer s.isMonitoringKeySetting.Store(false)
	defer s.monitoredKey.Store(nil)

	s.startJoystickListener()
	for {
		select {
		case <-tiker.C:
			key := s.monitoredKey.Load().(*models.Hotkey)
			if key != nil {
				s.robot.Stop()
				return *key, nil
			}
			if s.isMonitoringKeySetting.Load() == false {
				s.robot.Stop()
				return models.Hotkey{}, nil
			}
		case <-s.ctx.Done():
			s.robot.Stop()
			return models.Hotkey{}, errors.New("key setting monitoring cancelled")
		}

	}

}

func (s *HotkeyService) CancelMonitorKeySetting() {
	s.isMonitoringKeySetting.Store(false)
}

func (s *HotkeyService) readyHotkeysForGame(gameId string) {
	s.SetActiveGameID(gameId)
	s.startJoystickListener()
	s.loadHotkeyConfig(gameId)
}

func (s *HotkeyService) clearkeysForGame() {
	s.isMonitoringKeySetting.Store(false)
	s.SetActiveGameID("")
	s.mappingLock.Lock()
	s.actionkeyLock.Lock()
	s.robotMutex.Lock()
	defer s.robotMutex.Unlock()
	defer s.actionkeyLock.Unlock()
	defer s.mappingLock.Unlock()
	s.keyMappings = make(map[string]*models.Hotkey)
	s.actionKeys = make(map[string]*models.Hotkey)
	if s.robot != nil {
		s.robot.Stop()
		s.robot = nil
	}
}
