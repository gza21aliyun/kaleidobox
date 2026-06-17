package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"lunabox/internal/appconf"
)

// TouchMappingService 管理多个始终置顶的半透明按钮窗口。
// 支持两种模式:
//   - "mapping": 点击按钮触发对应虚拟键注入（游戏使用）
//   - "edit":    按钮可拖动改变位置，不注入按键（配置界面使用）
type TouchMappingService struct {
	ctx           context.Context
	config        *appconf.AppConfig
	mu            sync.Mutex
	hwnds         map[int]uintptr // buttonID -> 窗口句柄
	running       bool
	hotkeyService *HotkeyService
	ImageService  *ImageService
}

var (
	touchMappingInstance *TouchMappingService
	touchMappingOnce     sync.Once
)

// NewTouchMappingService 创建触摸映射服务
func NewTouchMappingService() *TouchMappingService {
	return &TouchMappingService{}
}

// Init 初始化服务
func (s *TouchMappingService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.config = config
}

// ============================================================
// Windows API
// ============================================================
var (
	tmUser32                  = syscall.NewLazyDLL("user32.dll")
	tmGdi32                   = syscall.NewLazyDLL("gdi32.dll")
	tmKernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandle       = tmKernel32.NewProc("GetModuleHandleW")
	procLoadCursor            = tmUser32.NewProc("LoadCursorW")
	procRegisterClass         = tmUser32.NewProc("RegisterClassExW")
	procCreateWindow          = tmUser32.NewProc("CreateWindowExW")
	procDefWindowProc         = tmUser32.NewProc("DefWindowProcW")
	procShowWindow            = tmUser32.NewProc("ShowWindow")
	procUpdateWindow          = tmUser32.NewProc("UpdateWindow")
	procGetMessage            = tmUser32.NewProc("GetMessageW")
	procTranslateMessage      = tmUser32.NewProc("TranslateMessage")
	procDispatchMessage       = tmUser32.NewProc("DispatchMessageW")
	procPostQuitMessage       = tmUser32.NewProc("PostQuitMessage")
	procPostMessage           = tmUser32.NewProc("PostMessageW")
	procDestroyWindow         = tmUser32.NewProc("DestroyWindow")
	procSetLayeredWindowAttrs = tmUser32.NewProc("SetLayeredWindowAttributes")
	procBeginPaint            = tmUser32.NewProc("BeginPaint")
	procEndPaint              = tmUser32.NewProc("EndPaint")
	procFillRect              = tmUser32.NewProc("FillRect")
	procSetBkMode             = tmGdi32.NewProc("SetBkMode")
	procSetTextColor          = tmGdi32.NewProc("SetTextColor")
	procDrawText              = tmUser32.NewProc("DrawTextW")
	procCreateSolidBrush      = tmGdi32.NewProc("CreateSolidBrush")
	procDeleteObject          = tmGdi32.NewProc("DeleteObject")
	procGetClientRect         = tmUser32.NewProc("GetClientRect")
	procRoundRect             = tmGdi32.NewProc("RoundRect")
	procCreatePen             = tmGdi32.NewProc("CreatePen")
	procSelectObject          = tmGdi32.NewProc("SelectObject")
	procGetStockObject        = tmGdi32.NewProc("GetStockObject")
	procCreateFontW           = tmGdi32.NewProc("CreateFontW")
	procRedrawWindow          = tmUser32.NewProc("RedrawWindow")
	procSetWindowPos          = tmUser32.NewProc("SetWindowPos")
	procSetCursor             = tmUser32.NewProc("SetCursor")
	procBringWindowToTop      = tmUser32.NewProc("BringWindowToTop")
	procSetWindowRgn          = tmUser32.NewProc("SetWindowRgn")
	procCreateRoundRectRgn    = tmGdi32.NewProc("CreateRoundRectRgn")
	// procGetForegroundWindow / procSetForegroundWindow 由同包中 image_service.go / start_service.go 提供
	procAllowSetForeground = tmUser32.NewProc("AllowSetForegroundWindow")
	procPeekMessage        = tmUser32.NewProc("PeekMessageW")
	procUnregisterClass    = tmUser32.NewProc("UnregisterClassW")
	procGetCursorPos       = tmUser32.NewProc("GetCursorPos")
	// procGetWindowRect / procGetSystemMetrics / procKeybdEvent / SM_CXSCREEN / SM_CYSCREEN 由同包中 image_service.go / start_service.go 提供
)

// Windows 常量
const (
	// Window styles
	WS_EX_TOPMOST    = 0x00000008
	WS_EX_LAYERED    = 0x00080000
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_NOACTIVATE = 0x08000000
	WS_POPUP         = 0x80000000
	WS_VISIBLE       = 0x10000000
	// Window class styles
	CS_HREDRAW = 0x0002
	CS_VREDRAW = 0x0001
	CS_DBLCLKS = 0x0008
	SW_SHOW    = 5
	SW_SHOWNA  = 8

	// Window messages
	WM_DESTROY       = 0x0002
	WM_PAINT         = 0x000F
	WM_CLOSE         = 0x0010
	WM_SETCURSOR     = 0x0020
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONDOWN   = 0x0204
	WM_RBUTTONUP     = 0x0205
	WM_MBUTTONDOWN   = 0x0207
	WM_MBUTTONUP     = 0x0208
	WM_XBUTTONDOWN   = 0x020B
	WM_XBUTTONUP     = 0x020C
	WM_POINTERDOWN   = 0x0246
	WM_POINTERUP     = 0x0247
	WM_USER          = 0x0400

	// Color (RGB) - Win32 COLORREF = 0x00BBGGRR
	RGB_WHITE    = 0x00FFFFFF
	COLOR_WINDOW = 5
	IDC_ARROW    = 32512

	// DrawText format
	DT_CENTER     = 0x00000001
	DT_VCENTER    = 0x00000004
	DT_SINGLELINE = 0x00000020

	// 注意: SM_CXSCREEN / SM_CYSCREEN 由同包内 image_service.go 提供

	// SetWindowPos 标志
	SWP_NOMOVE     = 0x0002
	SWP_NOSIZE     = 0x0001
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040
	SWP_NOZORDER   = 0x0004

	// PeekMessage 标志
	PM_REMOVE = 0x0001

	// Win32 错误码
	ERROR_CLASS_ALREADY_EXISTS = 1410

	// 特别处理 HWND_TOPMOST:
	// Windows 约定其值为 -1，但 Go 的 const 无法直接把 -1 赋给 uintptr。
	// 这里定义为一个可导出的 var（运行时可把 -1 转换为 uintptr 的最大值）。
	// Layered window attributes
	LWA_ALPHA = 0x00000002

	// keybd_event
	VK_UP              = 0x26
	VK_DOWN            = 0x28
	VK_LEFT            = 0x25
	VK_RIGHT           = 0x27
	VK_RETURN          = 0x0D
	VK_CONTROL         = 0x11
	VK_SHIFT           = 0x10
	VK_MENU            = 0x12 // Alt key
	KEYEVENTF_KEYUP_TM = 0x0002

	// 字体粗细 (LOGFONT.lfWeight)
	FW_BOLD = 700

	// 方向键方向常量
	ArrowDirNone  = 0
	ArrowDirUp    = 1
	ArrowDirDown  = 2
	ArrowDirLeft  = 3
	ArrowDirRight = 4

	// 自定义消息: 外部请求关闭窗口
	TM_CLOSE = WM_USER + 1
)

// HWND_TOPMOST 对应 Win32 HWND_TOPMOST = (HWND)-1
var HWND_TOPMOST = ^uintptr(0)

// Win32 公共结构体定义（包级别，所有函数共享）
type RECT struct {
	Left, Top, Right, Bottom int32
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type POINT struct {
	X, Y int32
}

// ============================================================
// 按钮配置与状态
// ============================================================

// ButtonConfig 定义按钮的显示和功能配置（从数据库读取）
type ButtonConfig struct {
	ID         int       // 内部按钮 ID
	HotkeyID   string    // 数据库 hotkey.id（用于保存时回写位置）
	Label      string    // 显示文字
	VirtualKey uintptr   // 对应的虚拟键码（映射到哪个键盘按键）
	X          int32     // 屏幕坐标 X
	Y          int32     // 屏幕坐标 Y
	ActionType string    // 动作类型（如 "screenshot"）
	Modifiers  []uintptr // 修饰键数组（如 VK_CONTROL, VK_SHIFT）
}

// ButtonState 跟踪按钮的悬停和按下状态
type ButtonState struct {
	Hovered        int32
	Pressed        int32
	ArrowDirection int32 // 方向键当前按下的方向（ArrowDirNone/Up/Down/Left/Right）
}

// DragState 跟踪按钮的拖动状态（编辑模式下使用）
type DragState struct {
	Dragging    int32 // 1 = 正在拖动
	StartMouseX int32 // 拖动起始点的鼠标屏幕坐标
	StartMouseY int32
	StartWinX   int32 // 拖动起始点的窗口位置
	StartWinY   int32
}

// 运行模式
const (
	ModeMapping = "mapping" // 游戏模式：点击触发按键注入
	ModeEdit    = "edit"    // 编辑模式：可拖动改变位置，不注入按键
)

// 全局状态
var (
	currentButtons []ButtonConfig // 当前活动的按钮列表（动态配置）
	currentMode    string         // 当前运行模式：ModeMapping 或 ModeEdit
	buttonWidthV   = int32(150)   // 窗口宽度
	buttonHeightV  = int32(140)   // 窗口高度

	// 方向键专用尺寸（更大）
	arrowButtonWidth  = int32(250) // 方向键窗口宽度
	arrowButtonHeight = int32(240) // 方向键窗口高度

	// 待注入按键事件
	pendingDown map[int]*int32
	pendingUp   map[int]*int32
	// 方向键待注入方向
	pendingArrowDown map[int]*int32 // buttonID -> 方向（ArrowDirUp/Down/Left/Right）
	pendingArrowUp   map[int]*int32 // buttonID -> 方向（需要松开的方向）
	// 悬停和按下状态
	buttonState     map[int]*ButtonState
	buttonHovered   int32
	clickedButton   int32
	buttonPressed   int32
	savedForeground uintptr

	// 拖动状态（编辑模式）
	dragStates map[int]*DragState

	// 编辑模式下的位置更新记录（key: buttonID）
	updatedPositions map[int]POINT

	// hwnd <-> buttonID 双向映射
	hwndToButtonID map[uintptr]int

	// 防止多次调用 PostQuitMessage
	quitPosted bool
)

// 初始化按钮状态映射（使用 currentButtons，可能为空）
func initButtonStates() {
	pendingDown = make(map[int]*int32)
	pendingUp = make(map[int]*int32)
	pendingArrowDown = make(map[int]*int32)
	pendingArrowUp = make(map[int]*int32)
	buttonState = make(map[int]*ButtonState)
	hwndToButtonID = make(map[uintptr]int)
	dragStates = make(map[int]*DragState)
	updatedPositions = make(map[int]POINT)
	for _, btn := range currentButtons {
		pendingDownVal := int32(0)
		pendingDown[btn.ID] = &pendingDownVal
		pendingUpVal := int32(0)
		pendingUp[btn.ID] = &pendingUpVal
		pendingArrowDownVal := int32(0)
		pendingArrowDown[btn.ID] = &pendingArrowDownVal
		pendingArrowUpVal := int32(0)
		pendingArrowUp[btn.ID] = &pendingArrowUpVal
		buttonState[btn.ID] = &ButtonState{}
		dragStates[btn.ID] = &DragState{}
	}
	atomic.StoreInt32(&buttonHovered, 0)
	atomic.StoreInt32(&clickedButton, -1)
	atomic.StoreInt32(&buttonPressed, 0)
	atomic.StoreUintptr(&savedForeground, 0)
	quitPosted = false
}

// 根据 buttonID 获取 ButtonConfig
func getButtonByID(buttonID int) *ButtonConfig {
	for i := range currentButtons {
		if currentButtons[i].ID == buttonID {
			return &currentButtons[i]
		}
	}
	return nil
}

// GetButtonRect 根据按钮 ID 返回其矩形范围 (x, y, width, height)
func GetButtonRect(buttonID int) (x, y, width, height int32) {
	for _, btn := range currentButtons {
		if btn.ID == buttonID {
			if btn.ActionType == "arrow_keys" {
				return btn.X, btn.Y, arrowButtonWidth, arrowButtonHeight
			}
			return btn.X, btn.Y, buttonWidthV, buttonHeightV
		}
	}
	return 0, 0, 0, 0
}

// HitTestButton 根据鼠标坐标确定命中哪个按钮
func HitTestButton(mouseX, mouseY int32) (buttonID int, hit bool) {
	for _, btn := range currentButtons {
		btnX, btnY, btnW, btnH := GetButtonRect(btn.ID)
		if mouseX >= btnX && mouseX < btnX+btnW && mouseY >= btnY && mouseY < btnY+btnH {
			return btn.ID, true
		}
	}
	return -1, false
}

// 附加消息（鼠标离开时的通知）
const (
	WM_MOUSEMOVE  = 0x0200
	WM_MOUSELEAVE = 0x02A3
)

// HTCLIENT 命中测试常量：鼠标事件落在窗口客户区内
const HTCLIENT = 1

// ============================================================
// TouchMapping 公共 API
// ============================================================

// GetTouchMapping 获取单例实例（保持向后兼容）
// func GetTouchMapping() *TouchMappingService {
// 	touchMappingOnce.Do(func() {
// 		touchMappingInstance = &TouchMappingService{}
// 	})
// 	return touchMappingInstance
// }

// SetHotkeyService 设置热键服务
func (tm *TouchMappingService) SetHotkeyService(service *HotkeyService) {
	tm.hotkeyService = service
}

func (tm *TouchMappingService) SetImageService(service *ImageService) {
	tm.ImageService = service
}

// SetButtons 设置按钮配置。必须在 Start 之前调用。
// 按钮位置来自数据库 key_code 字段（格式 "x:123;y:456"），
// 或由编辑模式下拖动产生。
func (tm *TouchMappingService) SetButtons(buttons []ButtonConfig) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	// 深拷贝
	nb := make([]ButtonConfig, len(buttons))
	copy(nb, buttons)
	currentButtons = nb
}

// StartMapping 以“游戏模式”启动（点击按钮触发按键注入）。
// 若窗口已运行则直接返回。
func (tm *TouchMappingService) StartMapping() error {
	tm.mu.Lock()
	if tm.running {
		tm.mu.Unlock()
		return nil
	}
	tm.running = true
	currentMode = ModeMapping
	tm.mu.Unlock()

	fmt.Println("TouchMapping: 启动映射模式")
	go tm.runWindow()
	return nil
}

// StartEditMode 以“编辑模式”启动（按钮可拖动改变位置，不注入按键）。
// 若窗口已运行则直接返回。
func (tm *TouchMappingService) StartEditMode() error {
	tm.mu.Lock()
	if tm.running {
		tm.mu.Unlock()
		return nil
	}
	tm.running = true
	currentMode = ModeEdit
	tm.mu.Unlock()

	fmt.Println("TouchMapping: 启动编辑模式")
	go tm.runWindow()
	return nil
}

// Stop 停止并销毁所有触摸映射窗口
func (tm *TouchMappingService) Stop() {
	tm.mu.Lock()
	hwnds := tm.hwnds
	running := tm.running
	tm.running = false
	tm.mu.Unlock()

	if !running {
		return
	}
	for _, hwnd := range hwnds {
		if hwnd != 0 {
			procPostMessage.Call(hwnd, uintptr(TM_CLOSE), 0, 0)
		}
	}
	fmt.Println("TouchMapping: 请求关闭触摸映射窗口")
}

// GetUpdatedPositions 返回编辑模式下各按钮最终位置（buttonID -> {X, Y}）
// 在停止编辑模式后调用此函数读取用户拖动结果，然后前端可把位置写回数据库。
func (tm *TouchMappingService) GetUpdatedPositions() map[int]POINT {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	result := make(map[int]POINT)
	for id, pt := range updatedPositions {
		result[id] = pt
	}
	// 同时把未拖动过的按钮的初始位置也返回（以便前端可以正确保存）
	for _, btn := range currentButtons {
		if _, ok := result[btn.ID]; !ok {
			result[btn.ID] = POINT{X: btn.X, Y: btn.Y}
		}
	}
	return result
}

// GetMode 返回当前运行模式（"mapping" / "edit" / ""）
func (tm *TouchMappingService) GetMode() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if !tm.running {
		return ""
	}
	return currentMode
}

// 自定义消息: 更新按钮列表
const TM_UPDATE_BUTTONS = WM_USER + 2

// pendingButtons 用于临时存储待更新的按钮列表
var pendingButtons []ButtonConfig

// UpdateEditModeButtons 动态更新编辑模式下的按钮列表。
// 会销毁所有现有窗口并用新按钮列表重新创建窗口。
// 必须在编辑模式下调用。
func (tm *TouchMappingService) UpdateEditModeButtons(buttons []ButtonConfig) error {
	tm.mu.Lock()
	if !tm.running || currentMode != ModeEdit {
		tm.mu.Unlock()
		return fmt.Errorf("not in edit mode")
	}

	// 如果还没有窗口（初始化中），直接更新 currentButtons
	if len(tm.hwnds) == 0 {
		nb := make([]ButtonConfig, len(buttons))
		copy(nb, buttons)
		currentButtons = nb
		tm.mu.Unlock()
		return nil
	}

	// 保存待更新的按钮列表
	pendingButtons = make([]ButtonConfig, len(buttons))
	copy(pendingButtons, buttons)

	// 向消息循环发送更新请求
	for _, hwnd := range tm.hwnds {
		if hwnd != 0 {
			procPostMessage.Call(hwnd, uintptr(TM_UPDATE_BUTTONS), 0, 0)
			break // 只发给第一个窗口即可
		}
	}
	tm.mu.Unlock()

	return nil
}

// doUpdateButtons 在消息循环线程中执行实际的按钮更新操作
func (s *TouchMappingService) doUpdateButtons() {
	// tm := GetTouchMapping()
	s.mu.Lock()

	if !s.running || currentMode != ModeEdit {
		s.mu.Unlock()
		return
	}

	if len(pendingButtons) == 0 {
		s.mu.Unlock()
		return
	}

	// 1. 保存当前窗口列表
	oldHwnds := make(map[int]uintptr)
	for k, v := range s.hwnds {
		oldHwnds[k] = v
	}

	// 2. 更新按钮列表（先复制再清空）
	nb := make([]ButtonConfig, len(pendingButtons))
	copy(nb, pendingButtons)
	pendingButtons = nil
	currentButtons = nb

	// 3. 初始化按钮状态（在创建窗口之前）
	initButtonStatesLocked()

	// 4. 销毁所有现有窗口
	// 设置标志阻止 WM_DESTROY 发送退出消息（因为我们要重建窗口，不是真正退出）
	quitPosted = true // 临时设置为 true，阻止 WM_DESTROY 中的 PostQuitMessage
	for _, hwnd := range oldHwnds {
		if hwnd != 0 {
			procDestroyWindow.Call(hwnd)
		}
	}
	// 清空 hwndToButtonID 映射（窗口已销毁）
	hwndToButtonID = make(map[uintptr]int)
	quitPosted = false // 重置标志，允许后续真正的退出
	s.hwnds = make(map[int]uintptr)

	s.mu.Unlock()

	// 在锁外创建新窗口（避免长时间持有锁）
	exStyle := uintptr(WS_EX_TOPMOST | WS_EX_LAYERED | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE)
	style := uintptr(WS_POPUP | WS_VISIBLE)
	// 编辑模式下使用更高的不透明度，方便用户编辑
	alpha := 100
	if currentMode == ModeEdit {
		alpha = 180
	}
	const cornerRadius = 16

	hInstance, _, _ := procGetModuleHandle.Call(0)
	titleText := syscall.StringToUTF16Ptr("")

	classNameUTF16 := syscall.StringToUTF16(
		fmt.Sprintf("LunaboxTouchMappingClass_%d_%d",
			os.Getpid(), time.Now().UnixNano()))
	className := &classNameUTF16[0]

	type WNDCLASSEX struct {
		CbSize        uint32
		Style         uint32
		LpfnWndProc   uintptr
		CbClsExtra    int32
		CbWndExtra    int32
		HInstance     uintptr
		HIcon         uintptr
		HCursor       uintptr
		HbrBackground uintptr
		LpszMenuName  *uint16
		LpszClassName *uint16
		HIconSm       uintptr
	}
	const NULL_BRUSH_STOCK = 5
	cursor, _, _ := procLoadCursor.Call(0, uintptr(IDC_ARROW))
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		Style:         CS_HREDRAW | CS_VREDRAW | CS_DBLCLKS,
		LpfnWndProc:   syscall.NewCallback(s.touchMappingWndProc),
		HInstance:     hInstance,
		HCursor:       cursor,
		HbrBackground: uintptr(NULL_BRUSH_STOCK),
		LpszClassName: className,
	}
	procRegisterClass.Call(uintptr(unsafe.Pointer(&wc)))

	// 先创建所有窗口并建立映射
	s.mu.Lock()
	for _, btn := range currentButtons {
		// 根据按钮类型选择窗口尺寸
		w := buttonWidthV
		h := buttonHeightV
		if btn.ActionType == "arrow_keys" {
			w = arrowButtonWidth
			h = arrowButtonHeight
		}

		hwnd, _, _ := procCreateWindow.Call(
			exStyle,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(titleText)),
			style,
			uintptr(btn.X),
			uintptr(btn.Y),
			uintptr(w),
			uintptr(h),
			0, 0, hInstance, 0,
		)
		if hwnd == 0 {
			continue
		}

		s.hwnds[btn.ID] = hwnd
		hwndToButtonID[hwnd] = btn.ID
	}
	s.mu.Unlock()

	// 设置窗口属性（在锁外执行，允许消息处理）
	for _, btn := range currentButtons {
		hwnd := s.hwnds[btn.ID]
		if hwnd == 0 {
			continue
		}

		procSetLayeredWindowAttrs.Call(hwnd, 0, uintptr(alpha), uintptr(LWA_ALPHA))

		// 根据按钮类型选择窗口尺寸
		w := buttonWidthV
		h := buttonHeightV
		if btn.ActionType == "arrow_keys" {
			w = arrowButtonWidth
			h = arrowButtonHeight
		}

		hRgn, _, _ := procCreateRoundRectRgn.Call(
			0, 0,
			uintptr(w+1), uintptr(h+1),
			uintptr(cornerRadius), uintptr(cornerRadius),
		)
		procSetWindowRgn.Call(hwnd, hRgn, 1)

		procSetWindowPos.Call(hwnd, HWND_TOPMOST, 0, 0, 0, 0,
			uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE|SWP_SHOWWINDOW|SWP_NOZORDER))

		procShowWindow.Call(hwnd, SW_SHOWNA)
		procUpdateWindow.Call(hwnd)
	}

	// 最后初始化按钮状态（窗口已完全创建并建立映射）
	s.mu.Lock()
	initButtonStatesLocked()
	s.mu.Unlock()
}

// initButtonStatesLocked 在持有锁的情况下初始化按钮状态（供 UpdateEditModeButtons 使用）
func initButtonStatesLocked() {
	pendingDown = make(map[int]*int32)
	pendingUp = make(map[int]*int32)
	pendingArrowDown = make(map[int]*int32)
	pendingArrowUp = make(map[int]*int32)
	buttonState = make(map[int]*ButtonState)
	// 注意：这里不再重新初始化 hwndToButtonID，保留现有的窗口映射
	dragStates = make(map[int]*DragState)
	updatedPositions = make(map[int]POINT)
	for _, btn := range currentButtons {
		pendingDownVal := int32(0)
		pendingDown[btn.ID] = &pendingDownVal
		pendingUpVal := int32(0)
		pendingUp[btn.ID] = &pendingUpVal
		pendingArrowDownVal := int32(0)
		pendingArrowDown[btn.ID] = &pendingArrowDownVal
		pendingArrowUpVal := int32(0)
		pendingArrowUp[btn.ID] = &pendingArrowUpVal
		buttonState[btn.ID] = &ButtonState{}
		dragStates[btn.ID] = &DragState{}
	}
	atomic.StoreInt32(&buttonHovered, 0)
	atomic.StoreInt32(&clickedButton, -1)
	atomic.StoreInt32(&buttonPressed, 0)
	atomic.StoreUintptr(&savedForeground, 0)
	quitPosted = false
}

// keepOnTopLoop 定期检查并保持所有按钮窗口在最顶层
func (tm *TouchMappingService) keepOnTopLoop() {
	ticker := time.NewTicker(2000 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		tm.mu.Lock()
		running := tm.running
		hwnds := tm.hwnds
		tm.mu.Unlock()

		if !running {
			break
		}

		for _, hwnd := range hwnds {
			if hwnd != 0 {
				procBringWindowToTop.Call(hwnd)
				procSetWindowPos.Call(hwnd, HWND_TOPMOST, 0, 0, 0, 0,
					uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE|SWP_SHOWWINDOW))
			}
		}
	}
}

// IsRunning 返回窗口是否运行中
func (tm *TouchMappingService) IsRunning() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.running
}

// ============================================================
// 窗口运行与消息循环
// ============================================================

func (tm *TouchMappingService) runWindow() {
	// Windows 要求：创建窗口的线程 = 消息循环线程 = 接收 WndProc 回调的线程。
	// Go 默认会把 goroutine 迁移到不同 OS 线程，必须 LockOSThread 防止线程迁移导致消息派发失败。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 生成一个带唯一后缀的窗口类名（多次 Start/Stop 时不会发生类名冲突）
	classNameUTF16 := syscall.StringToUTF16(
		fmt.Sprintf("LunaboxTouchMappingClass_%d_%d",
			os.Getpid(), time.Now().UnixNano()))
	className := &classNameUTF16[0]

	titleText := syscall.StringToUTF16Ptr("")

	// 1. 获取实例句柄
	hInstance, _, _ := procGetModuleHandle.Call(0)

	// 2. 加载鼠标箭头光标
	cursor, _, _ := procLoadCursor.Call(0, uintptr(IDC_ARROW))

	// 3. 注册窗口类（忽略“类已存在”错误，可能是上一轮未注销干净）
	type WNDCLASSEX struct {
		CbSize        uint32
		Style         uint32
		LpfnWndProc   uintptr
		CbClsExtra    int32
		CbWndExtra    int32
		HInstance     uintptr
		HIcon         uintptr
		HCursor       uintptr
		HbrBackground uintptr
		LpszMenuName  *uint16
		LpszClassName *uint16
		HIconSm       uintptr
	}

	// NULL_BRUSH = 5（Win32 GetStockObject(NULL_BRUSH)），避免系统先用白色擦除窗口背景
	const NULL_BRUSH_STOCK = 5
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		Style:         CS_HREDRAW | CS_VREDRAW | CS_DBLCLKS,
		LpfnWndProc:   syscall.NewCallback(tm.touchMappingWndProc),
		HInstance:     hInstance,
		HCursor:       cursor,
		HbrBackground: uintptr(NULL_BRUSH_STOCK),
		LpszClassName: className,
	}

	atom, _, err := procRegisterClass.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		if errNo, ok := err.(syscall.Errno); !ok || uint(errNo) != ERROR_CLASS_ALREADY_EXISTS {
			fmt.Printf("TouchMapping: 注册窗口类失败: %v\n", err)
			tm.mu.Lock()
			tm.running = false
			tm.mu.Unlock()
			return
		}
	}

	// 4. 为每个按钮创建独立窗口，位置由 currentButtons 的 X/Y 指定
	exStyle := uintptr(WS_EX_TOPMOST | WS_EX_LAYERED | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE)
	style := uintptr(WS_POPUP | WS_VISIBLE)

	// 编辑模式下使用更高的不透明度，方便用户编辑
	alpha := 100
	if currentMode == ModeEdit {
		alpha = 180
	}
	const cornerRadius = 16

	// 加锁保护：初始化状态和创建窗口在同一锁范围内，避免竞态条件
	tm.mu.Lock()
	// 初始化按钮状态（在锁内，确保与 tm.hwnds 写入原子一致）
	initButtonStates()
	tm.hwnds = make(map[int]uintptr)

	for _, btn := range currentButtons {
		// 根据按钮类型选择窗口尺寸
		w := buttonWidthV
		h := buttonHeightV
		if btn.ActionType == "arrow_keys" {
			w = arrowButtonWidth
			h = arrowButtonHeight
		}

		hwnd, _, err := procCreateWindow.Call(
			exStyle,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(titleText)),
			style,
			uintptr(btn.X),
			uintptr(btn.Y),
			uintptr(w),
			uintptr(h),
			0, 0, hInstance, 0,
		)
		if hwnd == 0 {
			fmt.Printf("TouchMapping: 创建按钮窗口失败 (ID=%d): %v\n", btn.ID, err)
			tm.running = false
			tm.mu.Unlock()
			return
		}

		// 建立 hwnd <-> buttonID 映射（必须在 ShowWindow/UpdateWindow 之前）
		tm.hwnds[btn.ID] = hwnd
		hwndToButtonID[hwnd] = btn.ID

		procSetLayeredWindowAttrs.Call(hwnd, 0, uintptr(alpha), uintptr(LWA_ALPHA))

		hRgn, _, _ := procCreateRoundRectRgn.Call(
			0, 0,
			uintptr(w+1), uintptr(h+1),
			uintptr(cornerRadius), uintptr(cornerRadius),
		)
		procSetWindowRgn.Call(hwnd, hRgn, 1)

		procSetWindowPos.Call(hwnd, HWND_TOPMOST, 0, 0, 0, 0,
			uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE|SWP_SHOWWINDOW|SWP_NOZORDER))

		procShowWindow.Call(hwnd, SW_SHOWNA)
		procUpdateWindow.Call(hwnd)

		fmt.Printf("TouchMapping: 窗口已创建 (ID=%d, HWND=%d, X=%d, Y=%d)\n", btn.ID, hwnd, btn.X, btn.Y)
	}
	tm.mu.Unlock()

	// 如果有待更新的按钮列表，立即处理
	if len(pendingButtons) > 0 {
		tm.doUpdateButtons()
	}

	// 启动定期置顶检查
	go tm.keepOnTopLoop()

	// 9. 消息循环：使用 PeekMessage 让循环永不阻塞；无消息时 Sleep。
	//    DispatchMessage 之后再处理挂起的按键注入（始终在 WndProc 之外执行 keybd_event）。
	type MSG struct {
		Hwnd    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		PtX     int32
		PtY     int32
	}

	var msg MSG
	for {
		ret, _, _ := procPeekMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0,
			uintptr(PM_REMOVE),
		)
		if ret != 0 {
			if msg.Message == 0x0012 { // WM_QUIT
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))

			// 延迟执行按键注入（仅在 WndProc 外）。
			tm.tmProcessPendingKeys()
			continue
		}

		// 无消息：仍要检查挂起的按键注入
		tm.tmProcessPendingKeys()

		// 让出 CPU
		time.Sleep(5 * time.Millisecond)
	}

	tm.mu.Lock()
	tm.hwnds = nil
	tm.running = false
	tm.mu.Unlock()

	// 10. 销毁我们注册的窗口类（失败忽略）
	procUnregisterClass.Call(uintptr(unsafe.Pointer(className)), hInstance)

	fmt.Println("TouchMapping: 窗口消息循环结束")
}

// ============================================================
// 窗口过程
// ============================================================

func (s *TouchMappingService) touchMappingWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	// 从 hwnd 查找对应的按钮 ID
	buttonID, exists := hwndToButtonID[hwnd]

	// 如果找不到按钮映射或按钮不存在，直接返回默认处理
	if !exists || getButtonByID(buttonID) == nil {
		// 对于窗口消息，确保显示箭头光标
		if msg == WM_SETCURSOR {
			cursor, _, _ := procLoadCursor.Call(0, uintptr(IDC_ARROW))
			procSetCursor.Call(cursor)
			return 1
		}
		result, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
		return result
	}

	switch msg {
	case WM_PAINT:
		tmPaintWindow(hwnd)
		return 0

	// 鼠标悬停：强制显示箭头光标
	case WM_SETCURSOR:
		hitTest := int32(lParam & 0xFFFF)
		if hitTest == HTCLIENT || hitTest == 0 {
			cursor, _, _ := procLoadCursor.Call(0, uintptr(IDC_ARROW))
			procSetCursor.Call(cursor)
			return 1
		}

	// 鼠标移动
	case WM_MOUSEMOVE:
		// 编辑模式：如果正在拖动，则移动窗口
		if currentMode == ModeEdit {
			ds := dragStates[buttonID]
			if ds != nil && atomic.LoadInt32(&ds.Dragging) == 1 {
				// 获取当前鼠标屏幕坐标
				pt := POINT{}
				procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

				// 计算新的窗口位置
				newX := ds.StartWinX + (pt.X - ds.StartMouseX)
				newY := ds.StartWinY + (pt.Y - ds.StartMouseY)

				// 移动窗口
				procSetWindowPos.Call(hwnd, HWND_TOPMOST,
					uintptr(newX), uintptr(newY), 0, 0,
					uintptr(SWP_NOSIZE|SWP_NOACTIVATE|SWP_NOZORDER))
			}
		}

		// 通用：悬停状态（带 nil 检查，避免 panic）
		if bs := buttonState[buttonID]; bs != nil {
			atomic.StoreInt32(&bs.Hovered, 1)
		}
		atomic.StoreInt32(&buttonHovered, 1)
		tmInvalidateRect(hwnd)
		return 0

	case WM_MOUSELEAVE:
		if bs := buttonState[buttonID]; bs != nil {
			atomic.StoreInt32(&bs.Hovered, 0)
		}
		atomic.StoreInt32(&buttonHovered, 0)
		tmInvalidateRect(hwnd)
		return 0

	// 鼠标按下
	case WM_LBUTTONDOWN, WM_LBUTTONDBLCLK,
		WM_RBUTTONDOWN,
		WM_MBUTTONDOWN,
		WM_XBUTTONDOWN,
		WM_POINTERDOWN:

		if currentMode == ModeEdit {
			// 编辑模式：记录拖动起始点，不注入按键
			pt := POINT{}
			procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			wr := RECT{}
			procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))

			ds := dragStates[buttonID]
			if ds != nil {
				atomic.StoreInt32(&ds.Dragging, 1)
				ds.StartMouseX = pt.X
				ds.StartMouseY = pt.Y
				ds.StartWinX = wr.Left
				ds.StartWinY = wr.Top
			}
			if bs := buttonState[buttonID]; bs != nil {
				atomic.StoreInt32(&bs.Pressed, 1)
			}
			atomic.StoreInt32(&buttonPressed, 1)
			tmInvalidateRect(hwnd)
			return 0
		}

		// 映射模式：记录前台窗口，延迟注入按键
		fg, _, _ := procGetForegroundWindow.Call()
		atomic.StoreUintptr(&savedForeground, fg)

		btn := getButtonByID(buttonID)
		if btn != nil {
			fmt.Printf(
				"[WndProc] button press detected: id=%d name=%s action=%s fgHwnd=0x%x modifiers=%v vk=%d\n",
				btn.ID, btn.Label, btn.ActionType, fg, btn.Modifiers, btn.VirtualKey)
		}

		if btn != nil && btn.ActionType == "arrow_keys" {
			// 方向键：计算触摸位置对应的方向
			// 获取坐标（从 lParam 提取）
			clientX := int32(lParam & 0xFFFF)
			clientY := int32((lParam >> 16) & 0xFFFF)

			// WM_LBUTTONDOWN 的 lParam 已经是客户区坐标，不需要转换
			// WM_POINTERDOWN 的 lParam 是屏幕坐标，需要转换为客户区坐标
			if msg == WM_POINTERDOWN {
				// 获取窗口在屏幕上的位置
				var wr RECT
				procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
				// 将屏幕坐标转换为窗口客户区坐标
				clientX = clientX - wr.Left
				clientY = clientY - wr.Top
			}

			arrowDir := tmCalcArrowDirection(hwnd, clientX, clientY)
			if bs := buttonState[buttonID]; bs != nil {
				atomic.StoreInt32(&bs.Pressed, 1)
				atomic.StoreInt32(&bs.ArrowDirection, arrowDir)
			}
			if pad := pendingArrowDown[buttonID]; pad != nil {
				atomic.StoreInt32(pad, arrowDir)
			}
		} else {
			// 普通按键
			if bs := buttonState[buttonID]; bs != nil {
				atomic.StoreInt32(&bs.Pressed, 1)
			}
			if pd := pendingDown[buttonID]; pd != nil {
				atomic.StoreInt32(pd, 1)
			}
		}
		atomic.StoreInt32(&buttonPressed, 1)
		atomic.StoreInt32(&clickedButton, int32(buttonID))
		tmInvalidateRect(hwnd)
		return 0

	// 鼠标松开
	case WM_LBUTTONUP,
		WM_RBUTTONUP,
		WM_MBUTTONUP,
		WM_XBUTTONUP,
		WM_POINTERUP:

		if currentMode == ModeEdit {
			// 编辑模式：结束拖动，记录最终位置
			ds := dragStates[buttonID]
			if ds != nil && atomic.LoadInt32(&ds.Dragging) == 1 {
				atomic.StoreInt32(&ds.Dragging, 0)

				wr := RECT{}
				procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))

				// 保存位置更新
				updatedPositions[buttonID] = POINT{X: wr.Left, Y: wr.Top}

				// 同时更新 currentButtons 中对应的按钮位置（以便后续查询到的是最新位置）
				btn := getButtonByID(buttonID)
				if btn != nil {
					btn.X = wr.Left
					btn.Y = wr.Top
				}
			}
			if bs := buttonState[buttonID]; bs != nil {
				atomic.StoreInt32(&bs.Pressed, 0)
				atomic.StoreInt32(&bs.ArrowDirection, ArrowDirNone)
			}
			atomic.StoreInt32(&buttonPressed, 0)
			tmInvalidateRect(hwnd)
			return 0
		}

		// 映射模式：延迟注入按键松开
		btn := getButtonByID(buttonID)
		if btn != nil && btn.ActionType == "arrow_keys" {
			// 方向键：记录当前方向用于松开
			if bs := buttonState[buttonID]; bs != nil {
				currentDir := atomic.LoadInt32(&bs.ArrowDirection)
				if pau := pendingArrowUp[buttonID]; pau != nil && currentDir != ArrowDirNone {
					atomic.StoreInt32(pau, currentDir)
				}
				atomic.StoreInt32(&bs.Pressed, 0)
				atomic.StoreInt32(&bs.ArrowDirection, ArrowDirNone)
			}
		} else {
			// 普通按键
			if pu := pendingUp[buttonID]; pu != nil {
				atomic.StoreInt32(pu, 1)
			}
			if bs := buttonState[buttonID]; bs != nil {
				atomic.StoreInt32(&bs.Pressed, 0)
			}
		}
		atomic.StoreInt32(&buttonPressed, 0)
		atomic.StoreInt32(&clickedButton, -1)
		tmInvalidateRect(hwnd)
		return 0

	case TM_UPDATE_BUTTONS:
		// 在消息循环线程中更新按钮列表
		s.doUpdateButtons()
		return 0

	case TM_CLOSE, WM_CLOSE:
		// 销毁所有按钮窗口
		for winHwnd := range hwndToButtonID {
			procDestroyWindow.Call(winHwnd)
		}
		return 0

	case WM_DESTROY:
		// 只在第一个窗口销毁时发起 Quit，防止多次调用
		if !quitPosted {
			quitPosted = true
			procPostQuitMessage.Call(0)
		}
		return 0
	}

	// 默认处理
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

// tmInvalidateRect 让窗口立即重绘（用于悬停/按下状态切换后刷新 UI）
func tmInvalidateRect(hwnd uintptr) {
	const (
		RDW_INVALIDATE = 0x0001
		RDW_UPDATENOW  = 0x0100
	)
	procRedrawWindow.Call(hwnd, 0, 0, RDW_INVALIDATE|RDW_UPDATENOW)
}

// ============================================================
// 绘制: 每个窗口只绘制自己对应的按钮
// ============================================================

func tmPaintWindow(hwnd uintptr) {
	var ps PAINTSTRUCT
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}

	// 根据 hwnd 找到对应的按钮 ID
	buttonID := hwndToButtonID[hwnd]
	btn := getButtonByID(buttonID)
	if btn == nil {
		// 按钮不存在，直接结束绘制
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return
	}

	var rc RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))

	// 检查是否是方向键类型
	if btn.ActionType == "arrow_keys" {
		tmPaintArrowKeysWindow(hwnd, hdc, btn, rc)
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return
	}

	// 绘制深色底色
	bgColor := uintptr(0x00141414)
	hBgBrush, _, _ := procCreateSolidBrush.Call(bgColor)
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), hBgBrush)
	procDeleteObject.Call(hBgBrush)

	// 带 nil 检查，避免 panic
	pressed := false
	hovered := false
	if bs := buttonState[btn.ID]; bs != nil {
		pressed = atomic.LoadInt32(&bs.Pressed) == 1
		hovered = atomic.LoadInt32(&bs.Hovered) == 1
	}

	// 颜色定义 (COLORREF = 0x00BBGGRR)
	var (
		gradTop    uintptr
		gradBottom uintptr
		borderCol  uintptr
		textCol    uintptr
	)

	switch {
	case pressed:
		gradTop = 0x004A3232
		gradBottom = 0x00261818
		borderCol = 0x00E0C090
		textCol = RGB_WHITE
	case hovered:
		gradTop = 0x005A3A3A
		gradBottom = 0x002A1A1A
		borderCol = 0x00FFD700
		textCol = RGB_WHITE
	default:
		gradTop = 0x0045302C
		gradBottom = 0x001E1414
		borderCol = 0x00807060
		textCol = RGB_WHITE
	}

	// 绘制渐变背景
	height := int(rc.Bottom - rc.Top)
	for i := 0; i < height; i++ {
		t := float64(i) / float64(height)
		col := tmLerpColor(gradTop, gradBottom, t)

		bandRect := RECT{
			Left:   rc.Left,
			Top:    rc.Top + int32(i),
			Right:  rc.Right,
			Bottom: rc.Top + int32(i+1),
		}
		if bandRect.Bottom > rc.Bottom {
			bandRect.Bottom = rc.Bottom
		}
		hBrush, _, _ := procCreateSolidBrush.Call(col)
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&bandRect)), hBrush)
		procDeleteObject.Call(hBrush)
	}

	// 绘制圆角边框
	const corner = 12
	pen, _, _ := procCreatePen.Call(0, 3, borderCol)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)

	NULL_BRUSH := uintptr(5)
	oldBrush, _, _ := procGetStockObject.Call(NULL_BRUSH)
	oldBrush2, _, _ := procSelectObject.Call(hdc, oldBrush)

	procRoundRect.Call(
		hdc,
		uintptr(rc.Left),
		uintptr(rc.Top),
		uintptr(rc.Right),
		uintptr(rc.Bottom),
		uintptr(corner),
		uintptr(corner),
	)

	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush2)
	procDeleteObject.Call(pen)

	// 绘制文字（粗体）
	const TRANSPARENT = 1
	procSetBkMode.Call(hdc, TRANSPARENT)
	procSetTextColor.Call(hdc, textCol)

	// 创建粗体字体: CreateFontW(nHeight, nWidth, nEscapement, nOrientation, fnWeight, ...)
	// 高度 28, 粗细 FW_BOLD, 其他用默认值 (0)
	hFont, _, _ := procCreateFontW.Call(
		28,               // nHeight: 字体高度
		0,                // nWidth: 0 = 使用默认比例
		0,                // nEscapement: 水平书写
		0,                // nOrientation: 字形角度
		uintptr(FW_BOLD), // fnWeight: 粗体
		0,                // fdwItalic: 不斜体
		0,                // fdwUnderline: 不下划线
		0,                // fdwStrikeOut: 不删除线
		0,                // fdwCharSet: DEFAULT_CHARSET
		0,                // fdwOutputPrecision: 默认
		0,                // fdwClipPrecision: 默认
		0,                // fdwQuality: 默认
		0,                // fdwPitchAndFamily: 默认
		0,                // lpszFace: 默认字体
	)
	oldFont, _, _ := procSelectObject.Call(hdc, hFont)

	textUTF16, _ := syscall.UTF16PtrFromString(btn.Label)
	procDrawText.Call(
		hdc,
		uintptr(unsafe.Pointer(textUTF16)),
		^uintptr(0),
		uintptr(unsafe.Pointer(&rc)),
		uintptr(DT_CENTER|DT_VCENTER|DT_SINGLELINE),
	)

	// 恢复旧字体并销毁临时字体
	procSelectObject.Call(hdc, oldFont)
	procDeleteObject.Call(hFont)

	procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

// tmPaintArrowKeysWindow 绘制方向键窗口（四个区域：上、下、左、右）
func tmPaintArrowKeysWindow(hwnd uintptr, hdc uintptr, btn *ButtonConfig, rc RECT) {
	width := int(rc.Right - rc.Left)
	height := int(rc.Bottom - rc.Top)
	centerX := width / 2
	centerY := height / 2

	// 获取当前按下的方向
	var arrowDir int32 = ArrowDirNone
	if bs := buttonState[btn.ID]; bs != nil {
		arrowDir = atomic.LoadInt32(&bs.ArrowDirection)
	}

	// 颜色定义 (COLORREF = 0x00BBGGRR)
	// 默认颜色：深色背景
	defaultBgColor := uintptr(0x001E1414)
	defaultBorderCol := uintptr(0x00807060)
	// 悬停/按下颜色：更亮的颜色
	activeBgColor := uintptr(0x004A3232)
	activeBorderCol := uintptr(0x00E0C090)

	// 绘制整体背景
	hBgBrush, _, _ := procCreateSolidBrush.Call(defaultBgColor)
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), hBgBrush)
	procDeleteObject.Call(hBgBrush)

	// 定义四个区域的矩形（类似键盘方向键的布局）
	// 上区域：顶部中间，宽度较小
	// 下区域：底部中间，宽度较小
	// 左区域：左侧中间
	// 右区域：右侧中间
	// 中间有一个空白区域（类似方向键的中心）

	// 区域尺寸计算
	arrowWidth := width / 3   // 每个箭头区域的宽度
	arrowHeight := height / 3 // 每个箭头区域的高度
	gap := 4                  // 区域之间的间隙

	// 上区域 (↑)
	upRect := RECT{
		Left:   rc.Left + int32(centerX-arrowWidth/2),
		Top:    rc.Top + int32(gap),
		Right:  rc.Left + int32(centerX+arrowWidth/2),
		Bottom: rc.Top + int32(arrowHeight),
	}
	// 下区域 (↓)
	downRect := RECT{
		Left:   rc.Left + int32(centerX-arrowWidth/2),
		Top:    rc.Bottom - int32(arrowHeight+gap),
		Right:  rc.Left + int32(centerX+arrowWidth/2),
		Bottom: rc.Bottom - int32(gap),
	}
	// 左区域 (←)
	leftRect := RECT{
		Left:   rc.Left + int32(gap),
		Top:    rc.Top + int32(centerY-arrowHeight/2),
		Right:  rc.Left + int32(arrowWidth),
		Bottom: rc.Top + int32(centerY+arrowHeight/2),
	}
	// 右区域 (→)
	rightRect := RECT{
		Left:   rc.Right - int32(arrowWidth+gap),
		Top:    rc.Top + int32(centerY-arrowHeight/2),
		Right:  rc.Right - int32(gap),
		Bottom: rc.Top + int32(centerY+arrowHeight/2),
	}

	// 绘制四个区域
	const corner = 8 // 圆角半径

	// 绘制上区域
	tmDrawArrowRegion(hdc, upRect, arrowDir == ArrowDirUp, "↑", corner, defaultBgColor, defaultBorderCol, activeBgColor, activeBorderCol)
	// 绘制下区域
	tmDrawArrowRegion(hdc, downRect, arrowDir == ArrowDirDown, "↓", corner, defaultBgColor, defaultBorderCol, activeBgColor, activeBorderCol)
	// 绘制左区域
	tmDrawArrowRegion(hdc, leftRect, arrowDir == ArrowDirLeft, "←", corner, defaultBgColor, defaultBorderCol, activeBgColor, activeBorderCol)
	// 绘制右区域
	tmDrawArrowRegion(hdc, rightRect, arrowDir == ArrowDirRight, "→", corner, defaultBgColor, defaultBorderCol, activeBgColor, activeBorderCol)

	// 绘制整体圆角边框
	pen, _, _ := procCreatePen.Call(0, 3, defaultBorderCol)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)

	NULL_BRUSH := uintptr(5)
	oldBrush, _, _ := procGetStockObject.Call(NULL_BRUSH)
	oldBrush2, _, _ := procSelectObject.Call(hdc, oldBrush)

	procRoundRect.Call(
		hdc,
		uintptr(rc.Left),
		uintptr(rc.Top),
		uintptr(rc.Right),
		uintptr(rc.Bottom),
		uintptr(12),
		uintptr(12),
	)

	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush2)
	procDeleteObject.Call(pen)
}

// tmDrawArrowRegion 绘制单个方向键区域
func tmDrawArrowRegion(hdc uintptr, rect RECT, active bool, symbol string, corner int,
	defaultBgColor, defaultBorderCol, activeBgColor, activeBorderCol uintptr) {
	// 选择颜色
	bgColor := defaultBgColor
	borderCol := defaultBorderCol
	if active {
		bgColor = activeBgColor
		borderCol = activeBorderCol
	}

	// 绘制区域背景
	hBrush, _, _ := procCreateSolidBrush.Call(bgColor)
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rect)), hBrush)
	procDeleteObject.Call(hBrush)

	// 绘制区域边框
	pen, _, _ := procCreatePen.Call(0, 2, borderCol)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)

	NULL_BRUSH := uintptr(5)
	oldBrush, _, _ := procGetStockObject.Call(NULL_BRUSH)
	oldBrush2, _, _ := procSelectObject.Call(hdc, oldBrush)

	procRoundRect.Call(
		hdc,
		uintptr(rect.Left),
		uintptr(rect.Top),
		uintptr(rect.Right),
		uintptr(rect.Bottom),
		uintptr(corner),
		uintptr(corner),
	)

	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush2)
	procDeleteObject.Call(pen)

	// 绘制箭头符号
	const TRANSPARENT = 1
	procSetBkMode.Call(hdc, TRANSPARENT)
	procSetTextColor.Call(hdc, RGB_WHITE)

	// 创建字体
	hFont, _, _ := procCreateFontW.Call(
		24,               // nHeight: 字体高度
		0,                // nWidth: 0 = 使用默认比例
		0,                // nEscapement: 水平书写
		0,                // nOrientation: 字形角度
		uintptr(FW_BOLD), // fnWeight: 粗体
		0,                // fdwItalic: 不斜体
		0,                // fdwUnderline: 不下划线
		0,                // fdwStrikeOut: 不删除线
		0,                // fdwCharSet: DEFAULT_CHARSET
		0,                // fdwOutputPrecision: 默认
		0,                // fdwClipPrecision: 默认
		0,                // fdwQuality: 默认
		0,                // fdwPitchAndFamily: 默认
		0,                // lpszFace: 默认字体
	)
	oldFont, _, _ := procSelectObject.Call(hdc, hFont)

	textUTF16, _ := syscall.UTF16PtrFromString(symbol)
	procDrawText.Call(
		hdc,
		uintptr(unsafe.Pointer(textUTF16)),
		^uintptr(0),
		uintptr(unsafe.Pointer(&rect)),
		uintptr(DT_CENTER|DT_VCENTER|DT_SINGLELINE),
	)

	procSelectObject.Call(hdc, oldFont)
	procDeleteObject.Call(hFont)
}

// tmLerpColor 在两个 COLORREF 之间做线性插值。t=0 取 color1, t=1 取 color2
func tmLerpColor(color1, color2 uintptr, t float64) uintptr {
	r1 := byte(color1 & 0xFF)
	g1 := byte((color1 >> 8) & 0xFF)
	b1 := byte((color1 >> 16) & 0xFF)

	r2 := byte(color2 & 0xFF)
	g2 := byte((color2 >> 8) & 0xFF)
	b2 := byte((color2 >> 16) & 0xFF)

	r := byte(float64(r1) + t*(float64(r2)-float64(r1)))
	g := byte(float64(g1) + t*(float64(g2)-float64(g1)))
	b := byte(float64(b1) + t*(float64(b2)-float64(b1)))

	return uintptr(r) | (uintptr(g) << 8) | (uintptr(b) << 16)
}

// tmCalcArrowDirection 根据触摸点在窗口中的相对位置计算方向键方向
func tmCalcArrowDirection(hwnd uintptr, clientX, clientY int32) int32 {
	var rc RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))

	width := int(rc.Right - rc.Left)
	height := int(rc.Bottom - rc.Top)
	centerX := width / 2
	centerY := height / 2

	// 计算相对于中心的位置
	relX := int(clientX) - centerX
	relY := int(clientY) - centerY

	// 计算角度来确定方向
	// 使用 atan2 来计算角度，然后根据角度范围确定方向
	// 上：-45° 到 45°（以 -90° 为中心）
	// 下：135° 到 225°（以 180° 为中心）
	// 左：45° 到 135°（以 90° 为中心）
	// 右：-135° 到 -45°（以 -90° 为中心）

	// 简化：比较绝对值来确定主方向
	absX := abs(relX)
	absY := abs(relY)

	// 如果在中心区域（距离中心太近），返回无方向
	threshold := min(width, height) / 6
	if absX < threshold && absY < threshold {
		return ArrowDirNone
	}

	// 根据相对位置确定方向
	if absX > absY {
		// 水平方向为主
		if relX > 0 {
			return ArrowDirRight
		} else {
			return ArrowDirLeft
		}
	} else {
		// 垂直方向为主
		if relY > 0 {
			return ArrowDirDown
		} else {
			return ArrowDirUp
		}
	}
}

// abs 返回整数的绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// min 返回两个整数的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================
// 合成按键: 多按钮支持
// 注意: 仅在消息循环层（WndProc 外）调用，避免在窗口过程内注入输入。
// ============================================================

// tmProcessPendingKeys 处理所有待注入的按键事件
func (s *TouchMappingService) tmProcessPendingKeys() {
	// 在处理任何触摸按钮事件之前，先检查进程状态确保游戏ID是最新的
	// tm := s.GetTouchMapping()
	if s.hotkeyService != nil && s.hotkeyService.GetActiveGameID() == "" {
		s.hotkeyService.processCheck()
	}

	for _, btn := range currentButtons {
		// 处理方向键按下
		if btn.ActionType == "arrow_keys" {
			if pad := pendingArrowDown[btn.ID]; pad != nil {
				arrowDir := atomic.LoadInt32(pad)
				if arrowDir != ArrowDirNone {
					if atomic.CompareAndSwapInt32(pad, arrowDir, ArrowDirNone) {
						// 根据方向注入对应的按键
						var vk uintptr
						switch arrowDir {
						case ArrowDirUp:
							vk = VK_UP
						case ArrowDirDown:
							vk = VK_DOWN
						case ArrowDirLeft:
							vk = VK_LEFT
						case ArrowDirRight:
							vk = VK_RIGHT
						}
						if vk != 0 {
							fmt.Printf(
								"[TouchPress] id=%d name=%s action=arrow_keys dir=%d vk=%d\n",
								btn.ID, btn.Label, arrowDir, vk)
							tmPressKeyDirect(btn.ID, vk, true)
						}
					}
				}
			}
			// 处理方向键松开
			if pau := pendingArrowUp[btn.ID]; pau != nil {
				arrowDir := atomic.LoadInt32(pau)
				if arrowDir != ArrowDirNone {
					if atomic.CompareAndSwapInt32(pau, arrowDir, ArrowDirNone) {
						// 根据方向松开对应的按键
						var vk uintptr
						switch arrowDir {
						case ArrowDirUp:
							vk = VK_UP
						case ArrowDirDown:
							vk = VK_DOWN
						case ArrowDirLeft:
							vk = VK_LEFT
						case ArrowDirRight:
							vk = VK_RIGHT
						}
						if vk != 0 {
							fmt.Printf(
								"[TouchRelease] id=%d name=%s action=arrow_keys dir=%d vk=%d\n",
								btn.ID, btn.Label, arrowDir, vk)
							tmPressKeyDirect(btn.ID, vk, false)
						}
					}
				}
			}
			continue
		}

		// 处理普通按键按下
		if atomic.LoadInt32(pendingDown[btn.ID]) == 1 {
			if atomic.CompareAndSwapInt32(pendingDown[btn.ID], 1, 0) {
				if btn.ActionType == "screenshot" {
					// 功能键：截图
					fmt.Printf(
						"[TouchPress] id=%d name=%s action=screenshot (no key injection)",
						btn.ID, btn.Label)
					s.tmTakeScreenshot()
				} else {
					// 先按下修饰键，再按主键
					modifierNames := make([]string, 0, len(btn.Modifiers))
					for _, modVK := range btn.Modifiers {
						modifierNames = append(modifierNames, fmt.Sprintf("vk=%d", modVK))
					}
					fmt.Printf(
						"[TouchPress] id=%d name=%s action=%s vk=%d modifiers=%v (pressing modifiers then main key)\n",
						btn.ID, btn.Label, btn.ActionType, btn.VirtualKey, modifierNames)

					for _, modVK := range btn.Modifiers {
						tmPressKeyDirect(btn.ID, modVK, true)
					}
					tmPressKeyDirect(btn.ID, btn.VirtualKey, true)
				}
			}
		}
		// 处理普通按键松开
		if atomic.LoadInt32(pendingUp[btn.ID]) == 1 {
			if atomic.CompareAndSwapInt32(pendingUp[btn.ID], 1, 0) {
				if btn.ActionType == "custom" {
					fmt.Printf(
						"[TouchRelease] id=%d name=%s vk=%d modifiers=%v (releasing main key then modifiers in reverse)\n",
						btn.ID, btn.Label, btn.VirtualKey, btn.Modifiers)
					// 先松主键，再按相反顺序松开修饰键
					tmPressKeyDirect(btn.ID, btn.VirtualKey, false)
					for i := len(btn.Modifiers) - 1; i >= 0; i-- {
						tmPressKeyDirect(btn.ID, btn.Modifiers[i], false)
					}
				}
			}
		}
	}
}

// tmTakeScreenshot 触发截图功能
func (s *TouchMappingService) tmTakeScreenshot() {
	// 获取当前活动游戏的 ID（游戏ID检查已在 tmProcessPendingKeys 中完成）
	var gameID string
	if s.hotkeyService != nil {
		gameID = s.hotkeyService.GetActiveGameID()
	}

	s.ImageService.TakeScreenshotOfFocusedWindow(gameID)
}

// tmPressKeyDirect 直接调用 keybd_event（仅在消息循环内部、WndProc 外调用）
// 在注入前先确认按键前把焦点恢复到按下我们按钮之前的前台窗口（通常就是游戏窗口）。
func tmPressKeyDirect(btnID int, vk uintptr, down bool) {
	action := "DOWN"
	if !down {
		action = "UP"
	}
	// 1. 如果记录的目标 HWND（在按下时的 WndProc 里已保存）
	target := atomic.LoadUintptr(&savedForeground)

	// 2. 若有目标存在且不是我们自己的窗口，先把焦点还回去
	//    （我们窗口有 WS_EX_NOACTIVATE，理论上不会抢焦点，但在某些系统配置下仍可能被设为前景。
	var fgLog string
	if target != 0 {
		current, _, _ := procGetForegroundWindow.Call()
		if current != target {
			// 允许目标进程接收前台（避免 SetForegroundWindow 被 UIPI 限制
			procAllowSetForeground.Call(0xFFFFFFFF) // ASFW_ANY = -1
			procSetForegroundWindow.Call(target)
			// 给系统一点时间处理焦点切换
			time.Sleep(2 * time.Millisecond)
			fgLog = fmt.Sprintf(" fg-restored-to-hwnd=0x%x", target)
		} else {
			fgLog = " fg-already-correct"
		}
	} else {
		fgLog = " fg-not-set"
	}

	// 3. 注入按键
	var dwFlags uintptr
	if down {
		dwFlags = 0
	} else {
		dwFlags = KEYEVENTF_KEYUP_TM
	}
	procKeybdEvent.Call(vk, 0, dwFlags, 0)

	fmt.Printf(
		"[KeyInject] btn=%d vk=%d action=%s dwFlags=%d%s\n",
		btnID, vk, action, dwFlags, fgLog)
}

// tmPressEnter 兼容旧接口，内部调用 tmProcessPendingKeys
func tmPressEnter(down bool) {
	// 旧接口已废弃，按键注入由 tmProcessPendingKeys 统一处理
}
