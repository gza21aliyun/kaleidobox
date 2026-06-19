package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"lunabox/internal/enums"
)

// HotkeyActionType 快捷键动作类型

// DeviceType 设备类型
// type DeviceType string

// const (
// 	DeviceTypeKeyboard   DeviceType = "keyboard"
// 	DeviceTypeDualSense  DeviceType = "dualsense"  // PlayStation 5 DualSense
// 	DeviceTypeDualShock4 DeviceType = "dualshock4" // PlayStation 4 DualShock 4
// 	DeviceTypeJoyCon     DeviceType = "joycon"     // Nintendo Switch Joy-Con
// 	DeviceTypeXInput     DeviceType = "xinput"     // Xbox controllers and compatible devices
// )

// ModifierKey 修饰键
// type ModifierKey string

// const (
// 	ModifierCtrl  ModifierKey = "ctrl"
// 	ModifierShift ModifierKey = "shift"
// 	ModifierAlt   ModifierKey = "alt"
// 	ModifierWin   ModifierKey = "win"
// )

const (
	// GlobalGameID 表示全局快捷键的特殊游戏ID
	GlobalGameID = "global"
)

// JoystickButton 手柄按钮类型
type JoystickButton string

const (
	JoystickButtonA         JoystickButton = "a"
	JoystickButtonB         JoystickButton = "b"
	JoystickButtonX         JoystickButton = "x"
	JoystickButtonY         JoystickButton = "y"
	JoystickButtonLB        JoystickButton = "lb"    // Left Bumper
	JoystickButtonRB        JoystickButton = "rb"    // Right Bumper
	JoystickButtonLT        JoystickButton = "lt"    // Left Trigger
	JoystickButtonRT        JoystickButton = "rt"    // Right Trigger
	JoystickButtonBack      JoystickButton = "back"  // Back/View button
	JoystickButtonStart     JoystickButton = "start" // Start/Menu button
	JoystickButtonLS        JoystickButton = "ls"    // Left Stick click
	JoystickButtonRS        JoystickButton = "rs"    // Right Stick click
	JoystickButtonDPadUp    JoystickButton = "dpad_up"
	JoystickButtonDPadDown  JoystickButton = "dpad_down"
	JoystickButtonDPadLeft  JoystickButton = "dpad_left"
	JoystickButtonDPadRight JoystickButton = "dpad_right"
	JoystickButtonGuide     JoystickButton = "guide" // Xbox Guide/PS Home button
)

// JoystickMapping 手柄按键映射配置
type JoystickMapping struct {
	ID                string           `json:"id" db:"id"`
	DeviceType        enums.DeviceType `json:"device_type" db:"device_type"`                 // 手柄设备类型
	Button            JoystickButton   `json:"button" db:"button"`                           // 手柄按钮
	TargetKey         string           `json:"target_key" db:"target_key"`                   // 目标键盘按键
	IsLongPress       bool             `json:"is_long_press" db:"is_long_press"`             // 是否为长按映射
	LongPressDuration int              `json:"long_press_duration" db:"long_press_duration"` // 长按持续时间(ms)
	Modifiers         string           `json:"modifiers" db:"modifiers"`                     // 修饰键 格式 "ctrl+shift+alt"
	IsEnabled         bool             `json:"is_enabled" db:"is_enabled"`
	CreatedAt         time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at" db:"updated_at"`
}

// JoystickButtonState 手柄按钮状态跟踪
type JoystickButtonState struct {
	Button               JoystickButton
	IsPressed            bool
	PressStartTime       time.Time
	IsLongPressTriggered bool
}

// Hotkey 快捷键配置
type Hotkey struct {
	ID           string                 `json:"id" db:"id"`
	GameID       string                 `json:"game_id" db:"game_id"` // "global"表示全局配置
	Name         string                 `json:"name" db:"name"`
	DeviceType   enums.DeviceType       `json:"device_type" db:"device_type"`
	KeyCode      string                 `json:"key_code" db:"key_code"`
	Modifiers    string                 `json:"modifiers" db:"modifiers"`
	ActionType   enums.HotkeyActionType `json:"action_type" db:"action_type"`
	ActionParams string                 `json:"action_params" db:"action_params"`
	IsEnabled    bool                   `json:"is_enabled" db:"is_enabled"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// ConnectedDevice 已连接设备
type ConnectedDevice struct {
	ID          string           `json:"id" db:"id"`
	DeviceType  enums.DeviceType `json:"device_type" db:"device_type"`
	DeviceName  string           `json:"device_name" db:"device_name"`
	DeviceID    string           `json:"device_id" db:"device_id"`
	IsActive    bool             `json:"is_active" db:"is_active"`
	ConnectedAt time.Time        `json:"connected_at" db:"connected_at"`
	LastSeenAt  time.Time        `json:"last_seen_at" db:"last_seen_at"`
}

// ModifierKeys 修饰键列表
type ModifierKeys []enums.ModifierKey

// Scan 实现 sql.Scanner 接口
func (mk *ModifierKeys) Scan(value interface{}) error {
	if value == nil {
		*mk = ModifierKeys{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, mk)
}

// Value 实现 driver.Valuer 接口
func (mk ModifierKeys) Value() (driver.Value, error) {
	if mk == nil {
		return nil, nil
	}
	return json.Marshal(mk)
}

// ActionParams 动作参数
type ActionParams map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (ap *ActionParams) Scan(value interface{}) error {
	if value == nil {
		*ap = ActionParams{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, ap)
}

// Value 实现 driver.Valuer 接口
func (ap ActionParams) Value() (driver.Value, error) {
	if ap == nil {
		return nil, nil
	}
	return json.Marshal(ap)
}

// IsGlobal 判断是否为全局快捷键
func (h *Hotkey) IsGlobal() bool {
	return h.GameID == GlobalGameID
}

// HotkeyConfig 快捷键配置（用于API传输）
type HotkeyConfig struct {
	GlobalHotkeys []Hotkey `json:"global_hotkeys"`
	GameHotkeys   []Hotkey `json:"game_hotkeys"`
}

// DeviceTypeInfo 设备类型信息
type DeviceTypeInfo struct {
	Type        enums.DeviceType `json:"type"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
}

// GetSupportedDevices 获取支持的设备类型列表
func GetSupportedDevices() []DeviceTypeInfo {
	return []DeviceTypeInfo{
		{enums.DeviceTypeKeyboard, "Keyboard", "标准键盘"},
		{enums.DeviceTypeDualSense, "DualSense", "PlayStation 5 DualSense手柄"},
		{enums.DeviceTypeDualShock4, "DualShock 4", "PlayStation 4 DualShock 4手柄"},
		{enums.DeviceTypeJoyCon, "Joy-Con", "Nintendo Switch Joy-Con手柄"},
		{enums.DeviceTypeXInput, "XInput", "Xbox手柄及兼容设备"},
		{enums.DeviceTypeTouch, "Touch", "屏幕触摸按钮映射"},
	}
}
