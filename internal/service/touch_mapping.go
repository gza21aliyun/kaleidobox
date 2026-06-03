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

// TouchMapping 管理一个始终置顶的半透明小窗口。
// 当用户用鼠标或触摸按下窗口时，触发一次 Enter 按键按下；
// 当用户松开时，触发一次 Enter 按键松开。
type TouchMapping struct {
	mu      sync.Mutex
	hwnd    uintptr
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
	KEYEVENTF_KEYUP_TM = 0x0002

	// 自定义消息: 外部请求关闭窗口
	TM_CLOSE = WM_USER + 1
)

// HWND_TOPMOST 对应 Win32 HWND_TOPMOST = (HWND)-1
var HWND_TOPMOST = ^uintptr(0)

// 延迟按键事件队列：WndProc 仅在此“记一笔”，真正调用 keybd_event 放到消息循环之外，
// 避免在 WndProc 内部注入输入导致系统消息处理重入或窗口被标记为未响应。
// 0 = 无事件，1 = Enter 按下，2 = Enter 松开。
// 用两个独立 int32 而不是一个，避免并发下读取与写入混淆。
var (
	pendingEnterDown int32
	pendingEnterUp   int32
	savedForeground  uintptr // 按下按钮时的前台窗口（通常就是游戏窗口），keybd_event 前恢复

	// UI 状态：悬停/按下，用于 tmPaintWindow 改变颜色
	buttonHovered int32 // 0 = 未悬停，1 = 悬停
	buttonPressed int32 // 0 = 未按下，1 = 按下
)

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

// Stop 停止并销毁触摸映射窗口
func (tm *TouchMapping) Stop() {
	tm.mu.Lock()
	hwnd := tm.hwnd
	running := tm.running
	tm.running = false
	tm.mu.Unlock()

	if !running {
		return
	}
	if hwnd != 0 {
		// 向窗口线程发送关闭消息，避免跨线程销毁窗口
		procPostMessage.Call(hwnd, uintptr(TM_CLOSE), 0, 0)
	}
	fmt.Println("TouchMapping: 请求关闭触摸映射窗口")
}

// keepOnTopLoop 定期检查并保持窗口在最顶层，防止被全屏应用覆盖
func (tm *TouchMapping) keepOnTopLoop(hwnd uintptr) {
	// 间隔：2000ms，在保证置顶效果的同时降低资源消耗
	ticker := time.NewTicker(2000 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		tm.mu.Lock()
		running := tm.running
		tm.mu.Unlock()

		if !running {
			break
		}

		// 使用 BringWindowToTop + SetWindowPos 组合，更激进地置顶
		procBringWindowToTop.Call(hwnd)

		// 使用 SWP_NOACTIVATE 避免激活窗口，但强制置顶
		procSetWindowPos.Call(hwnd, HWND_TOPMOST, 0, 0, 0, 0,
			uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE|SWP_SHOWWINDOW))
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

	// 4. 计算窗口位置: 屏幕右侧居中, 120x120 大小
	const wndWidth = 120
	const wndHeight = 120
	cx, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	cy, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	x := int(cx) - wndWidth - 40
	y := (int(cy) - wndHeight) / 2

	// 5. 创建窗口
	// WS_EX_TOOLWINDOW: 不出现在任务栏 / Alt+Tab
	// WS_EX_LAYERED:    支持透明度
	// WS_EX_TOPMOST:    始终置顶
	// WS_EX_NOACTIVATE: 点击我们窗口不会抢焦点（注入按键会落到原本的前台窗口上）
	// WS_POPUP:         无标题栏无边框
	exStyle := uintptr(WS_EX_TOPMOST | WS_EX_LAYERED | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE)
	style := uintptr(WS_POPUP | WS_VISIBLE)

	hwnd, _, err := procCreateWindow.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(titleText)),
		style,
		uintptr(x),
		uintptr(y),
		uintptr(wndWidth),
		uintptr(wndHeight),
		0, // parent
		0, // menu
		hInstance,
		0,
	)
	if hwnd == 0 {
		fmt.Printf("TouchMapping: 创建窗口失败: %v\n", err)
		tm.mu.Lock()
		tm.running = false
		tm.mu.Unlock()
		return
	}

	// 6. 设置透明度 (alpha = 180, 约 70% 不透明)
	const alpha = 100
	procSetLayeredWindowAttrs.Call(hwnd, 0, uintptr(alpha), uintptr(LWA_ALPHA))

	// 6.5 设置窗口区域为圆角矩形，裁剪掉圆角外的区域（避免显示深色直角边框）
	const cornerRadius = 16
	hRgn, _, _ := procCreateRoundRectRgn.Call(
		0, 0,
		uintptr(wndWidth+1), uintptr(wndHeight+1),
		uintptr(cornerRadius), uintptr(cornerRadius),
	)
	procSetWindowRgn.Call(hwnd, hRgn, 1) // 1 = redraw immediately

	// 7. 始终置顶（不抢焦点）
	procSetWindowPos.Call(hwnd, HWND_TOPMOST, 0, 0, 0, 0,
		uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE|SWP_SHOWWINDOW|SWP_NOZORDER))

	// 8. 显示但不激活（前台仍为游戏窗口）
	procShowWindow.Call(hwnd, SW_SHOWNA)
	procUpdateWindow.Call(hwnd)

	tm.mu.Lock()
	tm.hwnd = hwnd
	tm.mu.Unlock()

	fmt.Printf("TouchMapping: 窗口已创建 (HWND=%d)\n", hwnd)

	// 启动定期置顶检查：每隔500ms检查一次，确保窗口始终在最顶层
	go tm.keepOnTopLoop(hwnd)

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
			if atomic.CompareAndSwapInt32(&pendingEnterDown, 1, 0) {
				tmPressEnterDirect(true)
			}
			if atomic.CompareAndSwapInt32(&pendingEnterUp, 1, 0) {
				tmPressEnterDirect(false)
			}
			continue
		}

		// 无消息：仍要检查挂起的按键注入（若用户按下后没有系统消息，也要及时发出去）。
		downConsumed := false
		if atomic.CompareAndSwapInt32(&pendingEnterDown, 1, 0) {
			tmPressEnterDirect(true)
			downConsumed = true
		}
		if atomic.CompareAndSwapInt32(&pendingEnterUp, 1, 0) {
			tmPressEnterDirect(false)
		}
		if downConsumed {
			// 按下后让调度器跑一下，避免按键被忽略。
			time.Sleep(time.Millisecond)
		}

		// 让出 CPU
		time.Sleep(5 * time.Millisecond)
	}

	tm.mu.Lock()
	tm.hwnd = 0
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

	// 鼠标移动 → 进入悬停状态；首次进入时打开 WM_MOUSELEAVE 追踪
	case WM_MOUSEMOVE:
		if atomic.LoadInt32(&buttonHovered) == 0 {
			atomic.StoreInt32(&buttonHovered, 1)
			tmInvalidateRect(hwnd)
		}
		return 0

	case WM_MOUSELEAVE:
		if atomic.LoadInt32(&buttonHovered) != 0 {
			atomic.StoreInt32(&buttonHovered, 0)
			tmInvalidateRect(hwnd)
		}
		return 0

	// 按下（鼠标左/右/中/X 按钮、双击、通用触摸/笔）→ 延迟触发 Enter 按下
	case WM_LBUTTONDOWN, WM_LBUTTONDBLCLK,
		WM_RBUTTONDOWN,
		WM_MBUTTONDOWN,
		WM_XBUTTONDOWN,
		WM_POINTERDOWN:
		// 记录按下时的前台窗口（通常是游戏窗口）
		fg, _, _ := procGetForegroundWindow.Call()
		atomic.StoreUintptr(&savedForeground, fg)
		atomic.StoreInt32(&pendingEnterDown, 1)
		// UI: 标记按下状态并重绘
		atomic.StoreInt32(&buttonPressed, 1)
		tmInvalidateRect(hwnd)
		return 0

	// 松开 → 延迟触发 Enter 松开
	case WM_LBUTTONUP,
		WM_RBUTTONUP,
		WM_MBUTTONUP,
		WM_XBUTTONUP,
		WM_POINTERUP:
		atomic.StoreInt32(&pendingEnterUp, 1)
		// UI: 清除按下状态并重绘
		if atomic.CompareAndSwapInt32(&buttonPressed, 1, 0) {
			tmInvalidateRect(hwnd)
		}
		return 0

	case TM_CLOSE, WM_CLOSE:
		atomic.StoreInt32(&buttonHovered, 0)
		atomic.StoreInt32(&buttonPressed, 0)
		procDestroyWindow.Call(hwnd)
		return 0

	case WM_DESTROY:
		procPostQuitMessage.Call(0)
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
// 绘制: 深色背景 + 白色 "Enter" 文字
// ============================================================

func tmPaintWindow(hwnd uintptr) {
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

	var ps PAINTSTRUCT
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}

	var rc RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))

	height := int(rc.Bottom - rc.Top)

	// 根据当前状态（悬停/按下）选择配色
	pressed := atomic.LoadInt32(&buttonPressed) == 1
	hovered := atomic.LoadInt32(&buttonHovered) == 1

	// 颜色定义 (COLORREF = 0x00BBGGRR)
	var (
		// 渐变颜色: 顶部亮 / 底部稍暗
		gradTop    uintptr
		gradBottom uintptr
		borderCol  uintptr
		textCol    uintptr
	)

	switch {
	case pressed:
		// 按下: 更深色（凹陷感
		gradTop = 0x004A3232    // 稍深
		gradBottom = 0x00261818 // 更深
		borderCol = 0x00E0C090  // 金色边框
		textCol = RGB_WHITE
	case hovered:
		// 悬停: 更亮 + 金色边框
		gradTop = 0x005A3A3A
		gradBottom = 0x002A1A1A
		borderCol = 0x00FFD700 // 金色
		textCol = RGB_WHITE
	default:
		// 默认: 深色渐变
		gradTop = 0x0045302C
		gradBottom = 0x001E1414
		borderCol = 0x00807060 // 暗金
		textCol = RGB_WHITE
	}

	// 1) 先在整个窗口绘制深色底色，彻底覆盖系统可能残留的白色背景
	bgColor := gradBottom
	hBgBrush, _, _ := procCreateSolidBrush.Call(bgColor)
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), hBgBrush)
	procDeleteObject.Call(hBgBrush)

	// 2) 用水平色带模拟渐变（从上到下），覆盖整个窗口
	const bandHeight = 1
	if height > 0 {
		for i := 0; i < height; i += bandHeight {
			t := float64(i) / float64(height)
			col := tmLerpColor(gradTop, gradBottom, t)

			bandRect := RECT{
				Left:   rc.Left,
				Top:    rc.Top + int32(i),
				Right:  rc.Right,
				Bottom: rc.Top + int32(i+bandHeight),
			}
			if bandRect.Bottom > rc.Bottom {
				bandRect.Bottom = rc.Bottom
			}
			hBrush, _, _ := procCreateSolidBrush.Call(col)
			procFillRect.Call(hdc, uintptr(unsafe.Pointer(&bandRect)), hBrush)
			procDeleteObject.Call(hBrush)
		}
	}

	// 3) 圆角边框（覆盖整个窗口范围，不留边距）
	const corner = 16
	pen, _, _ := procCreatePen.Call(0, 4, borderCol) // PS_SOLID, 4px
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

	// 4) 透明文字背景
	const TRANSPARENT = 1
	procSetBkMode.Call(hdc, TRANSPARENT)

	// 5) 文字颜色
	procSetTextColor.Call(hdc, textCol)

	// 6) 居中绘制 "Enter"
	textUTF16, _ := syscall.UTF16PtrFromString("Enter")
	textRect := rc
	procDrawText.Call(
		hdc,
		uintptr(unsafe.Pointer(textUTF16)),
		^uintptr(0),
		uintptr(unsafe.Pointer(&textRect)),
		uintptr(DT_CENTER|DT_VCENTER|DT_SINGLELINE),
	)

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
// 合成按键: Enter 按下 / 松开
// 注意: 仅在消息循环层（WndProc 外）调用，避免在窗口过程内注入输入。
// ============================================================

func tmPressEnter(down bool) {
	// 保留旧签名以方便未来扩展，当前只做一次写入原子位。
	if down {
		atomic.StoreInt32(&pendingEnterDown, 1)
	} else {
		atomic.StoreInt32(&pendingEnterUp, 1)
	}
}

// tmPressEnterDirect 直接调用 keybd_event（仅在消息循环内部、WndProc 外调用）
// 在注入前先确认按键前把焦点恢复到按下我们按钮之前的前台窗口（通常就是游戏窗口）。
func tmPressEnterDirect(down bool) {
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
	procKeybdEvent.Call(uintptr(VK_RETURN), 0, dwFlags, 0)
}
