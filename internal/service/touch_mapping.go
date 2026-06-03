package service

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// TouchMapping 管理多个始终置顶的半透明按钮窗口。
// 每个按钮有独立的窗口，位置由 ButtonConfig 的 X/Y 指定。
// 当用户用鼠标或触摸按下窗口时，触发对应按键按下；松开时触发按键松开。
type TouchMapping struct {
	mu      sync.Mutex
	hwnds   map[int]uintptr // buttonID -> 窗口句柄
	running bool
}

var (
	touchMappingInstance *TouchMapping
	touchMappingOnce     sync.Once
)

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
	procBringWindowToTop     = tmUser32.NewProc("BringWindowToTop")
	procSetWindowRgn          = tmUser32.NewProc("SetWindowRgn")
	procCreateRoundRectRgn    = tmGdi32.NewProc("CreateRoundRectRgn")
	// procGetForegroundWindow / procSetForegroundWindow 由同包中 image_service.go / start_service.go 提供
	procAllowSetForeground = tmUser32.NewProc("AllowSetForegroundWindow")
	procPeekMessage        = tmUser32.NewProc("PeekMessageW")
	procUnregisterClass    = tmUser32.NewProc("UnregisterClassW")
	// procGetSystemMetrics / procKeybdEvent / SM_CXSCREEN / SM_CYSCREEN 由同包中 image_service.go / start_service.go 提供
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
	VK_RETURN          = 0x0D
	VK_CONTROL         = 0x11
	KEYEVENTF_KEYUP_TM = 0x0002

	// 字体粗细 (LOGFONT.lfWeight)
	FW_BOLD = 700

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

// ============================================================
// 按钮配置与状态
// ============================================================

// ButtonConfig 定义按钮的显示和功能配置
type ButtonConfig struct {
	ID         int    // 按钮唯一标识
	Label      string // 显示文字
	VirtualKey uintptr // 对应的虚拟键码
	X          int32  // 按钮左上角 X 坐标（相对于窗口客户区）
	Y          int32  // 按钮左上角 Y 坐标（相对于窗口客户区）
}

// ButtonState 跟踪按钮的悬停和按下状态
type ButtonState struct {
	Hovered int32 // 0 = 未悬停，1 = 悬停
	Pressed int32 // 0 = 未按下，1 = 按下
}

// 定义按钮列表（每个按钮有独立窗口，X/Y 为屏幕绝对坐标）
var buttonConfigs = []ButtonConfig{
	{ID: 0, Label: "Enter", VirtualKey: VK_RETURN, X: 1750, Y: 340},
	{ID: 1, Label: "Ctrl", VirtualKey: VK_CONTROL, X: 1750, Y: 450},
}

// 全局状态
var (
	// 待注入按键事件: buttonID -> pendingDown/pendingUp
	// 使用 *int32 而不是 int32，因为 Go map 的 value 本身不可寻址，
	// 无法直接对 map[key] 调用 atomic（需要 &val）
	pendingDown   map[int]*int32 // buttonID -> 1 表示需要按下
	pendingUp     map[int]*int32 // buttonID -> 1 表示需要松开
	buttonState   map[int]*ButtonState // buttonID -> 悬停/按下状态
	buttonHovered int32 // 当前有按钮被悬停（用于鼠标追踪）
	clickedButton int32 // 当前被点击的按钮ID（-1表示无）
	savedForeground uintptr // 按下按钮时保存的前台窗口句柄
	buttonPressed int32 // 是否有按钮处于按下状态（用于UI重绘）

	// hwnd <-> buttonID 双向映射
	hwndToButtonID map[uintptr]int // hwnd -> buttonID

	// 防止多次调用 PostQuitMessage
	quitPosted bool

	// 窗口尺寸配置
	buttonWidth  = 120
	buttonHeight = 100
)

// 初始化按钮状态映射
func initButtonStates() {
	pendingDown = make(map[int]*int32)
	pendingUp = make(map[int]*int32)
	buttonState = make(map[int]*ButtonState)
	hwndToButtonID = make(map[uintptr]int)
	for _, btn := range buttonConfigs {
		pendingDownVal := int32(0)
		pendingDown[btn.ID] = &pendingDownVal
		pendingUpVal := int32(0)
		pendingUp[btn.ID] = &pendingUpVal
		buttonState[btn.ID] = &ButtonState{}
	}
	atomic.StoreInt32(&buttonHovered, 0)
	atomic.StoreInt32(&clickedButton, -1)
	atomic.StoreInt32(&buttonPressed, 0)
	atomic.StoreUintptr(&savedForeground, 0)
	quitPosted = false
}

// GetButtonRect 根据按钮 ID 返回其矩形范围 (x, y, width, height)
func GetButtonRect(buttonID int) (x, y, width, height int32) {
	for _, btn := range buttonConfigs {
		if btn.ID == buttonID {
			return btn.X, btn.Y, int32(buttonWidth), int32(buttonHeight)
		}
	}
	return 0, 0, 0, 0
}

// HitTestButton 根据鼠标坐标确定命中哪个按钮，返回按钮 ID 和是否命中
func HitTestButton(mouseX, mouseY int32) (buttonID int, hit bool) {
	for _, btn := range buttonConfigs {
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

// GetTouchMapping 获取单例实例
func GetTouchMapping() *TouchMapping {
	touchMappingOnce.Do(func() {
		touchMappingInstance = &TouchMapping{}
	})
	return touchMappingInstance
}

// Start 启动触摸映射窗口。若窗口已运行则直接返回。
func (tm *TouchMapping) Start() error {
	tm.mu.Lock()
	if tm.running {
		tm.mu.Unlock()
		fmt.Println("TouchMapping: 窗口已在运行")
		return nil
	}
	tm.running = true
	tm.mu.Unlock()

	fmt.Println("TouchMapping: 启动触摸映射窗口")
	go tm.runWindow()
	return nil
}

// Stop 停止并销毁所有触摸映射窗口
func (tm *TouchMapping) Stop() {
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

// keepOnTopLoop 定期检查并保持所有按钮窗口在最顶层，防止被全屏应用覆盖
func (tm *TouchMapping) keepOnTopLoop() {
	// 间隔：2000ms，在保证置顶效果的同时降低资源消耗
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
func (tm *TouchMapping) IsRunning() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.running
}

// ============================================================
// 窗口运行与消息循环
// ============================================================

func (tm *TouchMapping) runWindow() {
	// 初始化按钮状态
	initButtonStates()

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
		LpfnWndProc:   syscall.NewCallback(touchMappingWndProc),
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

	// 4. 为每个按钮创建独立窗口，位置由 buttonConfigs 的 X/Y 指定
	exStyle := uintptr(WS_EX_TOPMOST | WS_EX_LAYERED | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE)
	style := uintptr(WS_POPUP | WS_VISIBLE)

	const alpha = 100
	const cornerRadius = 16

	tm.hwnds = make(map[int]uintptr)
	for _, btn := range buttonConfigs {
		hwnd, _, err := procCreateWindow.Call(
			exStyle,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(titleText)),
			style,
			uintptr(btn.X),
			uintptr(btn.Y),
			uintptr(buttonWidth),
			uintptr(buttonHeight),
			0, 0, hInstance, 0,
		)
		if hwnd == 0 {
			fmt.Printf("TouchMapping: 创建按钮窗口失败 (ID=%d): %v\n", btn.ID, err)
			tm.mu.Lock()
			tm.running = false
			tm.mu.Unlock()
			return
		}

		// 建立 hwnd <-> buttonID 映射（必须在 ShowWindow/UpdateWindow 之前，否则
		// UpdateWindow 会立即触发 WM_PAINT，此时映射还不存在）
		tm.hwnds[btn.ID] = hwnd
		hwndToButtonID[hwnd] = btn.ID

		procSetLayeredWindowAttrs.Call(hwnd, 0, uintptr(alpha), uintptr(LWA_ALPHA))

		hRgn, _, _ := procCreateRoundRectRgn.Call(
			0, 0,
			uintptr(buttonWidth+1), uintptr(buttonHeight+1),
			uintptr(cornerRadius), uintptr(cornerRadius),
		)
		procSetWindowRgn.Call(hwnd, hRgn, 1)

		procSetWindowPos.Call(hwnd, HWND_TOPMOST, 0, 0, 0, 0,
			uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE|SWP_SHOWWINDOW|SWP_NOZORDER))

		procShowWindow.Call(hwnd, SW_SHOWNA)
		procUpdateWindow.Call(hwnd)

		fmt.Printf("TouchMapping: 窗口已创建 (ID=%d, HWND=%d, X=%d, Y=%d)\n", btn.ID, hwnd, btn.X, btn.Y)
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
			tmProcessPendingKeys()
			continue
		}

		// 无消息：仍要检查挂起的按键注入
		tmProcessPendingKeys()

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

func touchMappingWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	// 从 hwnd 查找对应的按钮 ID
	buttonID := hwndToButtonID[hwnd]

	switch msg {
	case WM_PAINT:
		tmPaintWindow(hwnd)
		return 0

	// 鼠标悬停：强制显示箭头光标，避免系统默认的“忙/转圈”光标
	case WM_SETCURSOR:
		hitTest := int32(lParam & 0xFFFF)
		if hitTest == HTCLIENT || hitTest == 0 {
			cursor, _, _ := procLoadCursor.Call(0, uintptr(IDC_ARROW))
			procSetCursor.Call(cursor)
			return 1
		}

	// 鼠标移动 → 标记当前按钮为悬停
	case WM_MOUSEMOVE:
		atomic.StoreInt32(&buttonState[buttonID].Hovered, 1)
		atomic.StoreInt32(&buttonHovered, 1)
		tmInvalidateRect(hwnd)
		return 0

	case WM_MOUSELEAVE:
		atomic.StoreInt32(&buttonState[buttonID].Hovered, 0)
		atomic.StoreInt32(&buttonHovered, 0)
		tmInvalidateRect(hwnd)
		return 0

	// 按下 → 延迟触发按键按下
	case WM_LBUTTONDOWN, WM_LBUTTONDBLCLK,
		WM_RBUTTONDOWN,
		WM_MBUTTONDOWN,
		WM_XBUTTONDOWN,
		WM_POINTERDOWN:
		fg, _, _ := procGetForegroundWindow.Call()
		atomic.StoreUintptr(&savedForeground, fg)

		atomic.StoreInt32(&buttonState[buttonID].Pressed, 1)
		atomic.StoreInt32(pendingDown[buttonID], 1)
		atomic.StoreInt32(&buttonPressed, 1)
		atomic.StoreInt32(&clickedButton, int32(buttonID))
		tmInvalidateRect(hwnd)
		return 0

	// 松开 → 延迟触发按键松开
	case WM_LBUTTONUP,
		WM_RBUTTONUP,
		WM_MBUTTONUP,
		WM_XBUTTONUP,
		WM_POINTERUP:
		atomic.StoreInt32(pendingUp[buttonID], 1)
		atomic.StoreInt32(&buttonState[buttonID].Pressed, 0)
		atomic.StoreInt32(&buttonPressed, 0)
		atomic.StoreInt32(&clickedButton, -1)
		tmInvalidateRect(hwnd)
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
	btn := buttonConfigs[buttonID]

	var rc RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))

	// 绘制深色底色
	bgColor := uintptr(0x00141414)
	hBgBrush, _, _ := procCreateSolidBrush.Call(bgColor)
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), hBgBrush)
	procDeleteObject.Call(hBgBrush)

	pressed := atomic.LoadInt32(&buttonState[btn.ID].Pressed) == 1
	hovered := atomic.LoadInt32(&buttonState[btn.ID].Hovered) == 1

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
		28,              // nHeight: 字体高度
		0,               // nWidth: 0 = 使用默认比例
		0,               // nEscapement: 水平书写
		0,               // nOrientation: 字形角度
		uintptr(FW_BOLD), // fnWeight: 粗体
		0,               // fdwItalic: 不斜体
		0,               // fdwUnderline: 不下划线
		0,               // fdwStrikeOut: 不删除线
		0,               // fdwCharSet: DEFAULT_CHARSET
		0,               // fdwOutputPrecision: 默认
		0,               // fdwClipPrecision: 默认
		0,               // fdwQuality: 默认
		0,               // fdwPitchAndFamily: 默认
		0,               // lpszFace: 默认字体
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

// ============================================================
// 合成按键: 多按钮支持
// 注意: 仅在消息循环层（WndProc 外）调用，避免在窗口过程内注入输入。
// ============================================================

// tmProcessPendingKeys 处理所有待注入的按键事件
func tmProcessPendingKeys() {
	for _, btn := range buttonConfigs {
		// 处理按下
		if atomic.LoadInt32(pendingDown[btn.ID]) == 1 {
			if atomic.CompareAndSwapInt32(pendingDown[btn.ID], 1, 0) {
				tmPressKeyDirect(btn.VirtualKey, true)
			}
		}
		// 处理松开
		if atomic.LoadInt32(pendingUp[btn.ID]) == 1 {
			if atomic.CompareAndSwapInt32(pendingUp[btn.ID], 1, 0) {
				tmPressKeyDirect(btn.VirtualKey, false)
			}
		}
	}
}

// tmPressKeyDirect 直接调用 keybd_event（仅在消息循环内部、WndProc 外调用）
// 在注入前先确认按键前把焦点恢复到按下我们按钮之前的前台窗口（通常就是游戏窗口）。
func tmPressKeyDirect(vk uintptr, down bool) {
	// 1. 如果记录的目标 HWND（在按下时的 WndProc 里已保存）
	target := atomic.LoadUintptr(&savedForeground)

	// 2. 若有目标存在且不是我们自己的窗口，先把焦点还回去
	//    （我们窗口有 WS_EX_NOACTIVATE，理论上不会抢焦点，但在某些系统配置下仍可能被设为前景。
	if target != 0 {
		current, _, _ := procGetForegroundWindow.Call()
		if current != target {
			// 允许目标进程接收前台（避免 SetForegroundWindow 被 UIPI 限制
			procAllowSetForeground.Call(0xFFFFFFFF) // ASFW_ANY = -1
			procSetForegroundWindow.Call(target)
			// 给系统一点时间处理焦点切换
			time.Sleep(2 * time.Millisecond)
		}
	}

	// 3. 注入按键
	var dwFlags uintptr
	if down {
		dwFlags = 0
	} else {
		dwFlags = KEYEVENTF_KEYUP_TM
	}
	procKeybdEvent.Call(vk, 0, dwFlags, 0)
}

// tmPressEnter 兼容旧接口，内部调用 tmProcessPendingKeys
func tmPressEnter(down bool) {
	// 旧接口已废弃，按键注入由 tmProcessPendingKeys 统一处理
}
