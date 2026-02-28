package enums

type HotkeyActionType string

const (
	HotkeyActionStartGame   HotkeyActionType = "start_game"
	HotkeyActionStopGame    HotkeyActionType = "stop_game"
	HotkeyActionTogglePause HotkeyActionType = "toggle_pause"
	HotkeyActionScreenshot  HotkeyActionType = "screenshot"
	HotkeyActionCustom      HotkeyActionType = "custom"
)

var AllHotkeyActionTypes = []struct {
	Value  HotkeyActionType
	TSName string
}{

	{HotkeyActionStartGame, "START_GAME"},
	{HotkeyActionStopGame, "STOP_GAME"},
	{HotkeyActionTogglePause, "TOGGLE_PAUSE"},
	{HotkeyActionScreenshot, "SCREENSHOT"},
	{HotkeyActionCustom, "CUSTOM"},
}

type DeviceType string

const (
	DeviceTypeKeyboard   DeviceType = "keyboard"
	DeviceTypeDualSense  DeviceType = "dualsense"  // PlayStation 5 DualSense
	DeviceTypeDualShock4 DeviceType = "dualshock4" // PlayStation 4 DualShock 4
	DeviceTypeJoyCon     DeviceType = "joycon"     // Nintendo Switch Joy-Con
	DeviceTypeXInput     DeviceType = "xinput"     // Xbox controllers and compatible devices
)

var AllDeviceTypes = []struct {
	Value  DeviceType
	TSName string
}{
	{DeviceTypeKeyboard, "KEYBOARD"},
	{DeviceTypeDualSense, "DUALSENSE"},
	{DeviceTypeDualShock4, "DUALSHOCK4"},
	{DeviceTypeJoyCon, "JOYCON"},
	{DeviceTypeXInput, "XINPUT"},
}

type ModifierKey string

const (
	ModifierCtrl  ModifierKey = "ctrl"
	ModifierShift ModifierKey = "shift"
	ModifierAlt   ModifierKey = "alt"
	ModifierWin   ModifierKey = "win"
)

var AllModifierKeys = []struct {
	Value  ModifierKey
	TSName string
}{
	{ModifierCtrl, "CTRL"},
	{ModifierShift, "SHIFT"},
	{ModifierAlt, "ALT"},
	{ModifierWin, "WIN"},
}
