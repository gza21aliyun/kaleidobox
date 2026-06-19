package enums

type HotkeyActionType string

const (
	HotkeyActionStartGame   HotkeyActionType = "start_game"
	HotkeyActionStopGame    HotkeyActionType = "stop_game"
	HotkeyActionTogglePause HotkeyActionType = "toggle_pause"
	HotkeyActionScreenshot  HotkeyActionType = "screenshot"
	HotkeyActionKeyMapping  HotkeyActionType = "key_mapping" // 按键映射
	HotkeyActionArrowKeys   HotkeyActionType = "arrow_keys"  // 方向键（上下左右）
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
	{HotkeyActionArrowKeys, "ARROW_KEYS"},
	{HotkeyActionCustom, "CUSTOM"},
}

type DeviceType string

const (
	DeviceTypeKeyboard   DeviceType = "keyboard"
	DeviceTypeDualSense  DeviceType = "dualsense"  // PlayStation 5 DualSense
	DeviceTypeDualShock4 DeviceType = "dualshock4" // PlayStation 4 DualShock 4
	DeviceTypeJoyCon     DeviceType = "joyconPair" // Nintendo Switch Joy-Con
	DeviceTypeXInput     DeviceType = "xbox360"    // Xbox controllers and compatible devices
	DeviceTypeTouch      DeviceType = "touch"      // Touch mapping buttons (屏幕触摸按钮)
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
	{DeviceTypeTouch, "TOUCH"},
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

// JoystickButton 手柄按钮枚举
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

var AllJoystickButtons = []struct {
	Value  JoystickButton
	TSName string
	Name   string
}{
	{JoystickButtonA, "A", "A键"},
	{JoystickButtonB, "B", "B键"},
	{JoystickButtonX, "X", "X键"},
	{JoystickButtonY, "Y", "Y键"},
	{JoystickButtonLB, "LB", "左肩键"},
	{JoystickButtonRB, "RB", "右肩键"},
	{JoystickButtonLT, "LT", "左扳机"},
	{JoystickButtonRT, "RT", "右扳机"},
	{JoystickButtonBack, "BACK", "返回键"},
	{JoystickButtonStart, "START", "开始键"},
	{JoystickButtonLS, "LS", "左摇杆按下"},
	{JoystickButtonRS, "RS", "右摇杆按下"},
	{JoystickButtonDPadUp, "DPAD_UP", "方向键上"},
	{JoystickButtonDPadDown, "DPAD_DOWN", "方向键下"},
	{JoystickButtonDPadLeft, "DPAD_LEFT", "方向键左"},
	{JoystickButtonDPadRight, "DPAD_RIGHT", "方向键右"},
	{JoystickButtonGuide, "GUIDE", "主页键"},
}
