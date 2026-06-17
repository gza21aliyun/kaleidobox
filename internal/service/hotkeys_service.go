package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

var (
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
)

// Windows API常量
const (
	KEY_PRESSED = 0x8000
)

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
	robot       *gobot.Robot
	robotMutex6 sync.Mutex
	keyboard    *keyboard.Driver
	joysticks   map[string]*joystick.Driver
	deviceLock  sync.RWMutex

	// robotgo相关（键盘事件监听）
	hookStarted bool
	hookMutex1  sync.RWMutex

	// 按键映射管理
	keyMappings    map[string]*models.Hotkey // actionType是HotkeyActionCustom的映射键
	actionKeys     map[string]*models.Hotkey //映射其他功能键如截图
	mappingLock2   sync.RWMutex
	actionkeyLock3 sync.RWMutex
	// keyLock     sync.RWMutex

	// 按键状态跟踪
	keyStates  map[string]bool // key -> is_pressed
	stateLock4 sync.RWMutex

	// 触摸按钮

	// 当前活动游戏
	activeGameID atomic.Value
	activeLock5  sync.RWMutex

	isMonitoringKeySetting atomic.Bool
	monitoredKey           atomic.Value

	imageService        *ImageService
	startService        *StartService
	touchMappingService *TouchMappingService

	processCheckTicker *time.Ticker
	keyboardTicker     *time.Ticker
	keyboardStopChan   chan struct{}
}

func (s *HotkeyService) SetServices(imageService *ImageService, startService *StartService, touchMappingService *TouchMappingService) {
	s.imageService = imageService
	s.startService = startService
	s.touchMappingService = touchMappingService
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
	// // s.startJoystickListener()
	s.monitoredKey.Store(&models.Hotkey{})
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
		var deviceType string
		var actionType string

		err := rows.Scan(
			&hotkey.ID, &hotkey.GameID, &hotkey.Name, &deviceType, &hotkey.KeyCode,
			&hotkey.Modifiers,
			&actionType, &hotkey.ActionParams,
			&hotkey.IsEnabled, &hotkey.CreatedAt, &hotkey.UpdatedAt,
		)
		hotkey.DeviceType = enums.DeviceType(deviceType)
		hotkey.ActionType = enums.HotkeyActionType(actionType)
		if err != nil {
			applog.LogErrorf(s.ctx, "Failed to scan hotkey: %v", err)
			continue
		}

		rs = append(rs, &hotkey)
	}
	return rs, err
}

func (s *HotkeyService) loadHotkeyConfig(gameId string) map[enums.DeviceType]enums.DeviceType {
	applog.LogInfof(s.ctx, "Loading hotkey configuration")
	s.actionkeyLock3.Lock()
	s.mappingLock2.Lock()
	defer s.mappingLock2.Unlock()
	defer s.actionkeyLock3.Unlock()
	s.keyMappings = make(map[string]*models.Hotkey)
	s.actionKeys = make(map[string]*models.Hotkey)
	var devicetypes map[enums.DeviceType]enums.DeviceType = make(map[enums.DeviceType]enums.DeviceType)
	var devicetype enums.DeviceType
	if s.config.JoystickType != "" {
		devicetype = enums.DeviceType(s.config.JoystickType)
		// devicetypes[devicetype] = devicetype
	} else {
		devicetype = enums.DeviceTypeKeyboard
	}
	applog.InfoLogSaveAppLog("deviceType:%v\n", devicetype)

	query := `SELECT id, game_id, name, device_type, key_code, modifiers, action_type, action_params, is_enabled, created_at, updated_at 
	FROM hotkeys`
	if gameId != "" {
		query += fmt.Sprintf(" WHERE (game_id = '%s' OR game_id = '%s')", gameId, "global")
	}

	rows, _ := s.fetchHotkeys(query)
	// globalHotkeys := []models.Hotkey{}
	localCount := 0

	for _, hotkey := range rows {
		if !(hotkey.DeviceType == devicetype || hotkey.DeviceType == enums.DeviceTypeKeyboard) || hotkey.IsGlobal() {
			continue
		}
		if hotkey.ActionType != enums.HotkeyActionCustom {
			if !hotkey.IsGlobal() {
				s.actionKeys[hotkey.KeyCode] = hotkey
				if hotkey.DeviceType == enums.DeviceTypeTouch {
					x, y := parseTouchPosition(hotkey.KeyCode)
					vk := parseVirtualKey(hotkey.ActionParams)
					fmt.Printf("触摸按钮加载: name=%s x=%d y=%d vk=%d\n", hotkey.Name, x, y, vk)
				}
				devicetypes[hotkey.DeviceType] = hotkey.DeviceType
				if devicetype == hotkey.DeviceType {
					localCount++
				}

				// fmt.Printf("快捷键设备变为%s,keycode:%s\n", string(devicetype), hotkey.KeyCode)
			}
		} else {
			if !hotkey.IsGlobal() {
				s.keyMappings[hotkey.KeyCode] = hotkey
				if hotkey.DeviceType == enums.DeviceTypeTouch {
					x, y := parseTouchPosition(hotkey.KeyCode)
					vk := parseVirtualKey(hotkey.ActionParams)
					fmt.Printf("触摸按钮加载: name=%s x=%d y=%d vk=%d\n", hotkey.Name, x, y, vk)
				}

				devicetypes[hotkey.DeviceType] = hotkey.DeviceType
				if devicetype == hotkey.DeviceType {
					localCount++
				}
				// fmt.Printf("快捷键设备变为%s,keycode:%s\n", string(devicetype), hotkey.KeyCode)
			}
		}

		applog.LogInfof(s.ctx, "Loaded game hotkey: %s, \n %v\ngameKeyCount:%d\n", hotkey.Name, hotkey, localCount)
	}
	if localCount == 0 {
		for _, hotkey := range rows {
			if !(hotkey.DeviceType == devicetype || hotkey.DeviceType == enums.DeviceTypeKeyboard) {
				continue
			}
			if hotkey.ActionType != enums.HotkeyActionCustom {
				if hotkey.IsGlobal() {
					s.actionKeys[hotkey.KeyCode] = hotkey
					if hotkey.DeviceType == enums.DeviceTypeTouch {
						x, y := parseTouchPosition(hotkey.KeyCode)
						vk := parseVirtualKey(hotkey.ActionParams)
						fmt.Printf("触摸按钮加载: name=%s x=%d y=%d vk=%d\n", hotkey.Name, x, y, vk)
					}
					devicetypes[hotkey.DeviceType] = hotkey.DeviceType
				}
			} else {
				if hotkey.IsGlobal() {
					s.keyMappings[hotkey.KeyCode] = hotkey
					if hotkey.DeviceType == enums.DeviceTypeTouch {
						x, y := parseTouchPosition(hotkey.KeyCode)
						vk := parseVirtualKey(hotkey.ActionParams)
						fmt.Printf("触摸按钮加载: name=%s x=%d y=%d vk=%d\n", hotkey.Name, x, y, vk)
					}

					devicetypes[hotkey.DeviceType] = hotkey.DeviceType
				}
			}

			applog.LogInfof(s.ctx, "Loaded global hotkey: %s, \n %v\n", hotkey.Name, hotkey)
		}
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

	// 统计触摸按钮数量
	touchButtonCount := 0
	for _, hotkey := range s.keyMappings {
		if hotkey.DeviceType == enums.DeviceTypeTouch {
			touchButtonCount++
		}
	}
	for _, hotkey := range s.actionKeys {
		if hotkey.DeviceType == enums.DeviceTypeTouch {
			touchButtonCount++
		}
	}
	applog.LogInfof(s.ctx, "Loaded %d hotkeys (%s), touch buttons=%d", len(s.keyMappings), statsStr, touchButtonCount)
	// keys := make([]enums.DeviceType, 0, len(devicetypes))
	// for dt := range devicetypes {
	// 	keys = append(keys, dt)
	// }
	return devicetypes
}

// parseTouchPosition 从 "x:123;y:456" 格式中解析坐标
func parseTouchPosition(keyCode string) (int32, int32) {
	var x, y int32 = 0, 0
	parts := strings.Split(keyCode, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "x:") {
			if val, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(p, "x:"))); err == nil {
				x = int32(val)
			}
		} else if strings.HasPrefix(p, "y:") {
			if val, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(p, "y:"))); err == nil {
				y = int32(val)
			}
		}
	}
	// 默认位置（避免未设置时在屏幕左上角）
	if x == 0 && y == 0 {
		x = 1700
		y = 340
	}
	return x, y
}

// parseVirtualKey 从 action_params 解析虚拟键码
func parseVirtualKey(params string) uint {
	if params == "" {
		return 0x0D // 默认为 ENTER
	}
	// 尝试解析为数字（虚拟键码）
	if v, err := strconv.ParseUint(params, 10, 32); err == nil {
		return uint(v)
	}
	// 常见按键名映射
	switch strings.ToUpper(params) {
	case "ENTER", "RETURN":
		return 0x0D
	case "CTRL", "CONTROL":
		return 0x11
	case "SPACE":
		return 0x20
	case "ESC":
		return 0x1B
	case "TAB":
		return 0x09
	case "F1":
		return 0x70
	case "F2":
		return 0x71
	case "F3":
		return 0x72
	case "F4":
		return 0x73
	case "F5":
		return 0x74
	case "F6":
		return 0x75
	case "F7":
		return 0x76
	case "F8":
		return 0x77
	case "F9":
		return 0x78
	case "F10":
		return 0x79
	case "F11":
		return 0x7A
	case "F12":
		return 0x7B
	case "A":
		return 0x41
	case "B":
		return 0x42
	case "C":
		return 0x43
	case "D":
		return 0x44
	case "E":
		return 0x45
	case "F":
		return 0x46
	case "G":
		return 0x47
	case "H":
		return 0x48
	case "I":
		return 0x49
	case "J":
		return 0x4A
	case "K":
		return 0x4B
	case "L":
		return 0x4C
	case "M":
		return 0x4D
	case "N":
		return 0x4E
	case "O":
		return 0x4F
	case "P":
		return 0x50
	case "Q":
		return 0x51
	case "R":
		return 0x52
	case "S":
		return 0x53
	case "T":
		return 0x54
	case "U":
		return 0x55
	case "V":
		return 0x56
	case "W":
		return 0x57
	case "X":
		return 0x58
	case "Y":
		return 0x59
	case "Z":
		return 0x5A
	case "0":
		return 0x30
	case "1":
		return 0x31
	case "2":
		return 0x32
	case "3":
		return 0x33
	case "4":
		return 0x34
	case "5":
		return 0x35
	case "6":
		return 0x36
	case "7":
		return 0x37
	case "8":
		return 0x38
	case "9":
		return 0x39
	case "SHIFT":
		return 0x10
	case "ALT":
		return 0x12
	case "WIN", "LWIN":
		return 0x5B
	}
	return 0x0D // 默认 ENTER
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

	s.robotMutex6.Lock()
	defer s.robotMutex6.Unlock()
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
	//这里true的话每次按键都会去找前台游戏，但可能会增加延迟。注意如果改为true，一开始的loadhotkeys应该改为读取所有hotkeys而不是为单个游戏
	if /*true ||*/ s.GetActiveGameID() == "" {
		s.processCheck()
		if s.GetActiveGameID() == "" {
			s.monitoredKey.Store(&hk)
			return
		}
	}

	// 更新状态
	s.stateLock4.Lock()
	s.keyStates[key] = true
	s.stateLock4.Unlock()

	s.actionkeyLock3.RLock()
	actionkey := s.actionKeys[key]
	s.actionkeyLock3.RUnlock()
	if actionkey != nil { // 映射的按键
		s.monitoredKey.Store(&hk)
		return
	}

	// 直接映射：立即模拟目标按键按下

	s.mappingLock2.RLock()
	hotkey := s.keyMappings[key]
	s.mappingLock2.RUnlock()
	if hotkey != nil {
		s.simulateKeyPress(hotkey.ActionParams, []enums.ModifierKey{})
	}
	s.monitoredKey.Store(&hk)
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
	s.stateLock4.Lock()
	s.keyStates[key] = false
	s.stateLock4.Unlock()

	s.mappingLock2.RLock()

	hotkey := s.keyMappings[key]

	s.mappingLock2.RUnlock()

	fmt.Printf("Key: %v , mappings:%v\n", hotkey, s.keyMappings)
	if hotkey != nil {
		s.simulateKeyRelease(hotkey.ActionParams, []enums.ModifierKey{})
		s.monitoredKey.Store(&hk)
		return
	}
	s.actionkeyLock3.RLock()
	hotkey = s.actionKeys[key]
	s.actionkeyLock3.RUnlock()
	if hotkey != nil {
		s.handleActionKey(hotkey)
		s.monitoredKey.Store(&hk)
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
				s.actionkeyLock3.RLock()
				hotkey := s.actionKeys[keyCode]
				s.actionkeyLock3.RUnlock()
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

func (s *HotkeyService) handleDS4Events() {
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
		// applog.LogDebugf(s.ctx, "Left X axis: %v", data)
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
		// applog.LogDebugf(s.ctx, "Left X axis: %v", data)
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
	s.stateLock4.Lock()

	defer s.stateLock4.Unlock()

	if isPress {
		if !s.keyStates[key] {
			fmt.Printf("toggle key press %s\n", key)
			s.handleKeyPress(key, name, device)
			s.keyStates[key] = true
		}
	} else {
		if s.keyStates[key] {
			fmt.Printf("toggle key up %s\n", key)
			s.handleKeyRelease(key, name, device)
			s.keyStates[key] = false
		}
	}

}

// 启动相关方法
func (s *HotkeyService) startJoystickListener(devicetype enums.DeviceType) {
	applog.LogInfof(s.ctx, "Starting joystick listener...")
	// devicetype := enums.DeviceTypeDualShock4
	s.robotMutex6.Lock()
	defer s.robotMutex6.Unlock()
	// 启动手柄机器人
	// 创建 joystick 适配器

	joystickAdaptor := joystick.NewAdaptor("0")

	// 创建手柄驱动
	stick := joystick.NewDriver(joystickAdaptor, string(devicetype))

	s.joysticks[string(devicetype)] = stick
	if devicetype == enums.DeviceTypeDualShock4 {
		s.robot = gobot.NewRobot(string(devicetype)+"Robot",
			[]gobot.Connection{joystickAdaptor},
			[]gobot.Device{stick},
			s.handleDS4Events,
		)
		go func() {
			if err := s.robot.Start(); err != nil {
				applog.LogErrorf(s.ctx, "Failed to start joystick robot: %v", err)
			}
		}()
	}

}

// 公共接口方法
func (s *HotkeyService) GetSupportedDevices() []models.DeviceTypeInfo {
	return models.GetSupportedDevices()
}

// RemoveKeyMapping 移除按键映射
func (s *HotkeyService) RemoveKeyMapping(sourceKey string) {
	s.mappingLock2.Lock()
	delete(s.keyMappings, sourceKey)
	s.mappingLock2.Unlock()

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

	var hotkeys []models.Hotkey = []models.Hotkey{}
	for rows.Next() {
		var hotkey models.Hotkey
		var deviceType string
		var actionType string
		err := rows.Scan(
			&hotkey.ID,
			&hotkey.GameID,
			&hotkey.Name,
			&deviceType,
			&hotkey.KeyCode,
			&hotkey.Modifiers,
			&actionType,
			&hotkey.ActionParams,
			&hotkey.IsEnabled,
			&hotkey.CreatedAt,
			&hotkey.UpdatedAt,
		)
		hotkey.DeviceType = enums.DeviceType(deviceType)
		hotkey.ActionType = enums.HotkeyActionType(actionType)
		if err != nil {
			applog.LogErrorf(s.ctx, "扫描快捷键数据失败: %v", err)
			continue
		}
		hotkeys = append(hotkeys, hotkey)
	}
	// 打印每个快捷键的详细信息
	names := make([]string, 0, len(hotkeys))
	for _, h := range hotkeys {
		names = append(names, fmt.Sprintf("%s(%s)", h.Name, h.DeviceType))
	}
	fmt.Printf("已加载快捷键:%d - [%s]\n", len(hotkeys), strings.Join(names, ", "))

	return hotkeys, nil
}

// UpdateHotkey 更新快捷键配置
func (s *HotkeyService) UpdateHotkey(hotkey models.Hotkey) error {
	applog.LogInfof(s.ctx, "start to UpdateHotkey - name: %s, id: %s", hotkey.Name, hotkey.ID)
	query := `
		UPDATE hotkeys 
		SET name = ?, device_type = ?, key_code = ?, modifiers = ?,
		    action_type = ?, action_params = ?, is_enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := s.db.Exec(query,
		hotkey.Name,
		string(hotkey.DeviceType),
		hotkey.KeyCode,
		hotkey.Modifiers,
		string(hotkey.ActionType),
		hotkey.ActionParams,
		hotkey.IsEnabled,
		hotkey.ID,
	)

	if err != nil {
		applog.LogErrorf(s.ctx, "更新快捷键失败: %v", err)
		return err
	}

	applog.LogInfof(s.ctx, "快捷键更新成功: name=%s, id=%s", hotkey.Name, hotkey.ID)
	return nil
}

// AddHotkey 添加新的快捷键配置
func (s *HotkeyService) AddHotkey(hotkey models.Hotkey) error {
	applog.LogInfof(s.ctx, "start to AddHotkey - name: %s, device: %s, game_id: %s", hotkey.Name, hotkey.DeviceType, hotkey.GameID)
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
		string(hotkey.DeviceType),
		hotkey.KeyCode,
		hotkey.Modifiers,
		string(hotkey.ActionType),
		hotkey.ActionParams,
		hotkey.IsEnabled,
		hotkey.CreatedAt,
		hotkey.UpdatedAt,
	)

	if err != nil {
		applog.LogErrorf(s.ctx, "添加快捷键失败: %v", err)
		return err
	}

	applog.LogInfof(s.ctx, "快捷键添加成功: name=%s, id=%s", hotkey.Name, hotkey.ID)
	return nil
}

// GetHotkeysByGameID 根据游戏ID获取快捷键配置
func (s *HotkeyService) GetHotkeysByGameID(gameID string) ([]models.Hotkey, error) {
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

	var hotkeys []models.Hotkey = []models.Hotkey{}
	for rows.Next() {
		var hotkey models.Hotkey
		var deviceType string
		var actionType string
		err := rows.Scan(
			&hotkey.ID,
			&hotkey.GameID,
			&hotkey.Name,
			&deviceType,
			&hotkey.KeyCode,
			&hotkey.Modifiers,
			&actionType,
			&hotkey.ActionParams,
			&hotkey.IsEnabled,
			&hotkey.CreatedAt,
			&hotkey.UpdatedAt,
		)
		hotkey.DeviceType = enums.DeviceType(deviceType)
		hotkey.ActionType = enums.HotkeyActionType(actionType)
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
	applog.LogInfof(s.ctx, "start to DeleteHotkey - id: %s", hotkeyID)
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
	return &s.mappingLock2
}

// IsKeyPressed 查询按键是否被按下
func (s *HotkeyService) IsKeyPressed(key string) bool {
	s.stateLock4.RLock()
	defer s.stateLock4.RUnlock()
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
				applog.LogDebugf(s.ctx, "Focus changed to game: %s(%s)", game.GameId, game.ProcessName)
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

func (s *HotkeyService) MonitorKeySetting(devicetype enums.DeviceType) (models.Hotkey, error) {
	// if devicetype == enums.DeviceTypeKeyboard {
	// 	return
	// }
	s.monitoredKey.Store(&models.Hotkey{})
	s.isMonitoringKeySetting.Store(true)
	var tiker *time.Ticker = time.NewTicker(time.Millisecond * 500)
	defer s.isMonitoringKeySetting.Store(false)
	defer s.monitoredKey.Store(&models.Hotkey{})

	s.startJoystickListener(devicetype)
	for {
		select {
		case <-tiker.C:
			key := s.monitoredKey.Load().(*models.Hotkey)
			if key.KeyCode != "" {
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
	devicetypes := s.loadHotkeyConfig(gameId)
	applog.InfoLogSaveAppLog("readyHotkeysForGame: %s, %v\n", gameId, devicetypes)
	for _, devicetype := range devicetypes {
		if devicetype == enums.DeviceTypeKeyboard {
			go s.startAlternativeKeyListener()
		} else if devicetype == enums.DeviceTypeTouch {
			go s.startTouchMapping()
		} else {
			go s.startJoystickListener(devicetype)
		}
	}

}

func (s *HotkeyService) clearkeysForGame(isEmptyGames bool) {
	s.isMonitoringKeySetting.Store(false)
	s.SetActiveGameID("")
	s.actionkeyLock3.Lock()
	s.mappingLock2.Lock()
	s.keyMappings = make(map[string]*models.Hotkey)
	s.actionKeys = make(map[string]*models.Hotkey)
	s.mappingLock2.Unlock()
	s.actionkeyLock3.Unlock()

	// 停止触摸按钮
	s.stopTouchMapping()

	s.robotMutex6.Lock()
	defer s.robotMutex6.Unlock()
	s.stopKeyboardListener()

	if s.robot != nil {
		s.robot.Stop()
		s.robot = nil
	}
}

// hotkeyToButtonConfig 将 Hotkey 转换为 ButtonConfig
func hotkeyToButtonConfig(hotkey *models.Hotkey, id int) ButtonConfig {
	x, y := parseTouchPosition(hotkey.KeyCode)
	vk := parseVirtualKey(hotkey.ActionParams)
	// 解析修饰键为虚拟键码数组
	modifierVKs := parseModifiers(hotkey.Modifiers)
	cfg := ButtonConfig{
		ID:         id,
		HotkeyID:   hotkey.ID,
		Label:      hotkey.Name,
		VirtualKey: uintptr(vk),
		X:          x,
		Y:          y,
		ActionType: string(hotkey.ActionType),
		Modifiers:  modifierVKs,
	}
	fmt.Printf("[TouchButton] id=%d name=%s action=%s vk=%d modifiers=%v modifierVKs=%v pos=(%d,%d)\n",
		id, hotkey.Name, hotkey.ActionType, vk, hotkey.Modifiers, modifierVKs, x, y)
	return cfg
}

// parseModifiers 将修饰键字符串（如 "ctrl+shift+alt"）解析为虚拟键码数组
func parseModifiers(modifiersStr string) []uintptr {
	var vks []uintptr
	if modifiersStr == "" {
		return vks
	}
	parts := strings.Split(modifiersStr, "+")
	for _, m := range parts {
		m = strings.TrimSpace(strings.ToLower(m))
		switch m {
		case "ctrl":
			vks = append(vks, VK_CONTROL)
		case "shift":
			vks = append(vks, VK_SHIFT)
		case "alt":
			vks = append(vks, VK_MENU)
		case "win":
			vks = append(vks, VK_LWIN)
		}
	}
	return vks
}

// getTouchButtons 从 keyMappings 和 actionKeys 中获取所有触摸按钮
func (s *HotkeyService) getTouchButtons() []ButtonConfig {
	s.mappingLock2.RLock()
	s.actionkeyLock3.RLock()
	defer s.mappingLock2.RUnlock()
	defer s.actionkeyLock3.RUnlock()

	var buttons []ButtonConfig
	id := 0

	// 从 keyMappings 获取普通按键映射的触摸按钮
	for _, hotkey := range s.keyMappings {
		if hotkey.DeviceType == enums.DeviceTypeTouch {
			buttons = append(buttons, hotkeyToButtonConfig(hotkey, id))
			id++
		}
	}

	// 从 actionKeys 获取功能键类型的触摸按钮（如截图）
	for _, hotkey := range s.actionKeys {
		if hotkey.DeviceType == enums.DeviceTypeTouch {
			buttons = append(buttons, hotkeyToButtonConfig(hotkey, id))
			id++
		}
	}

	return buttons
}

// startTouchMapping 启动触摸按钮（映射模式）
func (s *HotkeyService) startTouchMapping() {
	fmt.Println("TouchMapping: 启动触摸映射窗口")
	buttons := s.getTouchButtons()

	if len(buttons) == 0 {
		fmt.Println("TouchMapping: 没有配置触摸按钮，跳过启动")
		return
	}
	fmt.Printf("TouchMapping: 启动触摸映射窗口(%d 个按钮)\n", len(buttons))

	// tm := GetTouchMapping()
	s.touchMappingService.SetButtons(buttons)
	err := s.touchMappingService.StartMapping()
	if err != nil {
		fmt.Printf("TouchMapping: 启动失败: %v\n", err)
	} else {
		fmt.Printf("TouchMapping: 已启动映射模式，共 %d 个按钮\n", len(buttons))
	}
}

// stopTouchMapping 停止触摸按钮窗口
func (s *HotkeyService) stopTouchMapping() {
	s.touchMappingService.Stop()
}

// TouchButtonInfo 前端与后端之间传递的触摸按钮信息结构
type TouchButtonInfo struct {
	Index      int    // 在列表中的索引（0-based），用于关联编辑后的位置
	Name       string // 按钮显示的文字
	VirtualKey uint32 // 对应的虚拟键码（如 0x0D = Enter），使用 uint32 确保 Wails 序列化兼容
	X          int32  // 屏幕坐标 X
	Y          int32  // 屏幕坐标 Y
	ActionType string // 动作类型（如 "screenshot"）
	Modifiers  string // 修饰键组合字符串（如 "ctrl+shift+alt"）
}

// TouchButtonPosition 单个按钮的位置更新
type TouchButtonPosition struct {
	Index int32
	X     int32
	Y     int32
}

// StartTouchEditMode 启动触摸按钮的编辑模式。
// 前端调用此方法，把当前按钮列表传进来，后端会在屏幕上显示按钮，用户可用鼠标拖动改变位置。
func (s *HotkeyService) StartTouchEditMode(buttons []TouchButtonInfo) error {
	// 转换为内部 ButtonConfig
	configs := make([]ButtonConfig, len(buttons))
	for i, b := range buttons {
		modifierVKs := parseModifiers(b.Modifiers)
		configs[i] = ButtonConfig{
			ID:         b.Index,
			HotkeyID:   b.Name,
			Label:      b.Name,
			VirtualKey: uintptr(b.VirtualKey),
			X:          b.X,
			Y:          b.Y,
			ActionType: b.ActionType,
			Modifiers:  modifierVKs,
		}
	}

	// tm := GetTouchMapping()
	s.touchMappingService.SetButtons(configs)
	return s.touchMappingService.StartEditMode()
}

// StopTouchEditMode 停止编辑模式。
// 返回所有按钮的更新后的位置，前端可据此更新列表的 X/Y。
func (s *HotkeyService) StopTouchEditMode() []TouchButtonPosition {
	// tm := GetTouchMapping()
	positions := s.touchMappingService.GetUpdatedPositions()
	s.touchMappingService.Stop()

	result := make([]TouchButtonPosition, 0, len(positions))
	for idx, pt := range positions {
		result = append(result, TouchButtonPosition{
			Index: int32(idx),
			X:     pt.X,
			Y:     pt.Y,
		})
	}
	return result
}

// UpdateTouchEditModeButtons 动态更新编辑模式下的按钮列表。
// 在编辑模式下，新增/删除/载入按钮时调用此方法同步后端 overlay。
func (s *HotkeyService) UpdateTouchEditModeButtons(buttons []TouchButtonInfo) error {
	// 转换为内部 ButtonConfig
	configs := make([]ButtonConfig, len(buttons))
	for i, b := range buttons {
		configs[i] = ButtonConfig{
			ID:         b.Index,
			HotkeyID:   b.Name,
			Label:      b.Name,
			VirtualKey: uintptr(b.VirtualKey),
			X:          b.X,
			Y:          b.Y,
			ActionType: b.ActionType,
		}
	}

	// tm := GetTouchMapping()
	return s.touchMappingService.UpdateEditModeButtons(configs)
}

// startAlternativeKeyListener 备用键盘监听方案
func (s *HotkeyService) startAlternativeKeyListener() {
	applog.LogInfof(s.ctx, "Starting alternative keyboard listener...")

	// 初始化键状态映射
	lastKeyState := make(map[int]bool)
	count := 0

	// 初始化停止 channel（带缓冲，防止阻塞）
	s.keyboardStopChan = make(chan struct{}, 1)
	// 使用较短的时间间隔以获得更好的响应性
	s.keyboardTicker = time.NewTicker(50 * time.Millisecond)
	fmt.Println("startAlternativeKeyListener 10")
	s.actionkeyLock3.RLock()
	keys := s.actionKeys
	s.actionkeyLock3.RUnlock()
	fmt.Println("startAlternativeKeyListener 11")
	defer func() {
		if s.keyboardTicker != nil {
			s.keyboardTicker.Stop()
			s.keyboardTicker = nil
		}
	}()

	for {
		select {
		case <-s.keyboardTicker.C:
			s.checkKeyboardState(lastKeyState, count, keys)
			count++

		case <-s.keyboardStopChan: // ✅ 监听专用的停止 channel
			applog.LogInfof(s.ctx, "Alternative keyboard listener stopped")
			return

		case <-s.ctx.Done():
			applog.LogInfof(s.ctx, "Alternative keyboard listener stopped")
			return
		}
	}
}

func (s *HotkeyService) stopKeyboardListener() {
	if s.keyboardStopChan == nil && s.keyboardTicker == nil {
		return
	}

	applog.LogInfo(s.ctx, "Stopping keyboard listener...")

	// 发送停止信号（非阻塞，因为有缓冲）
	select {
	case s.keyboardStopChan <- struct{}{}:
		applog.LogInfo(s.ctx, "Stop signal sent to keyboard listener")
	default:
		// channel 已经有信号了，不需要重复发送
	}

	// 停止 ticker
	if s.keyboardTicker != nil {
		s.keyboardTicker.Stop()
		s.keyboardTicker = nil
	}

	// 等待一小段时间让 goroutine 退出
	time.Sleep(60 * time.Millisecond)

	applog.LogInfo(s.ctx, "Keyboard listener stopped")
}

func (s *HotkeyService) checkKeyboardState(lastKeyState map[int]bool, count int, keys map[string]*models.Hotkey) {
	// s.stateLock.Lock()
	// defer s.stateLock.Unlock()
	// s.stateLock

	// 首先检查是否有活动游戏且当前焦点进程匹配
	// if s.GetActiveGameID() == "" {

	// }
	s.processCheck()
	if s.GetActiveGameID() == "" {
		return
	}

	// 获取当前修饰键
	// modifiers := s.getModifierKeys()

	// 检查每个监控的键
	for keyCode, hotkey := range keys {
		// js, _ := json.MarshalIndent(hotkey, "", "  ")
		// fmt.Printf("checkKeyboardState 20 keyCode:%s,key:\n%s\n", keyCode, string(js))

		if keyCode == "" {
			continue
		}
		vkCode := 0
		if len(keyCode) > 1 {
			switch keyCode {
			case "space":
				vkCode = 32
			case "enter":
				vkCode = 13
			case "backspace":
				vkCode = 8
			case "tab":
				vkCode = 9
			case "esc":
				vkCode = 27
			case "CONTEXT_MENU":
				vkCode = 93
			case "CONTEXTMENU":
				vkCode = 93
			case "BROWSER_HOME":
				vkCode = 172
			case "BROWSERHOME":
				vkCode = 172
			case "BROWSER_BACK":
				vkCode = 166
			case "BROWSER_FORWARD":
				vkCode = 167
			case "BROWSER_REFRESH":
				vkCode = 168
			case "BROWSER_STOP":
				vkCode = 169
			case "BROWSER_SEARCH":
				vkCode = 170
			case "BROWSER_FAVORITES":
				vkCode = 171
			case "VOLUME_MUTE":
				vkCode = 173
			case "VOLUME_DOWN":
				vkCode = 174
			case "VOLUME_UP":
				vkCode = 175
			}
		} else if len(keyCode) == 1 {
			vkCode = int(keyCode[0])
		}
		if vkCode == 0 {
			continue
		}

		currentState := s.isKeyPressed(vkCode)
		lastState, exists := lastKeyState[vkCode]
		// 如果按键状态发生变化且当前是按下状态，则触发事件
		if (!exists || !lastState) && currentState {
			// applog.LogInfof(s.ctx, "按下01 %s", keyCode)
			// 创建键盘事件
			// event := keyboard.KeyEvent{
			// 	Key: vkCode,
			// }

			// 检查是否有匹配的快捷键（包括修饰键）
			// go s.handleKeyboardEvent(event)
			fmt.Printf("checkKeyboardState keyCode: %s\n", hotkey.KeyCode)
			go s.handleActionKey(hotkey)
		}

		// 更新状态
		lastKeyState[vkCode] = currentState
	}

}

// isKeyPressed 检查指定虚拟键是否被按下
func (s *HotkeyService) isKeyPressed(vkCode int) bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vkCode))
	return (ret & KEY_PRESSED) != 0
}
