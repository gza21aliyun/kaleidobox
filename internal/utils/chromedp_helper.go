package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// browserOpts 构建浏览器启动参数：优先 Chrome，否则 Edge。
func browserOpts() []chromedp.ExecAllocatorOption {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// 注意：不能用 --headless=new。实测在本机 Edge 上 --headless=new 会导致
		// "chrome failed to start"（new headless 走 GPU 合成，与 --disable-gpu 冲突）。
		// 旧版 --headless 在本机 Edge 上可正常无头启动并加载页面。
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("allow-running-insecure-content", true),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36`),
	)

	if chromePath := findChromePath(); chromePath != "" {
		fmt.Println("✅ 找到 Chrome:", chromePath)
		opts = append(opts, chromedp.ExecPath(chromePath))
	} else if edgePath := findEdgePath(); edgePath != "" {
		fmt.Println("未检测到 Chrome，尝试使用 Edge:", edgePath)
		opts = append(opts, chromedp.ExecPath(edgePath))
	} else {
		fmt.Println("⚠️  未找到 Chrome 或 Edge，将使用系统默认 Chrome 路径")
	}
	return opts
}

// killProcessTree 强制终止浏览器主进程及其全部子进程。
//
// Windows 上 chromedp 的 cancel 只对主进程调用 TerminateProcess，
// 浏览器衍生的子进程（renderer / GPU / network service / utility / crashpad 等）
// 会变成孤儿进程永不被回收，批量搜刮时会堆积大量 msedge.exe。
// 这里用 taskkill /F /T /PID 杀掉整棵进程树来兜底。
func killProcessTree(cmd *exec.Cmd) {
	if runtime.GOOS != "windows" {
		return
	}
	if cmd == nil || cmd.Process == nil {
		return
	}
	// /F 强制终止；/T 终止指定进程及其全部子进程
	_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}

// runInBrowser 启动一个独立浏览器执行 actions，完成后彻底清理整个浏览器进程树。
//
// 采用单次 chromedp.Run 同时启动浏览器并执行动作（与最初能在 Edge 上正常拿到
// HTML 的写法一致）。通过 ModifyCmdFunc 捕获浏览器进程，清理时用 killProcessTree
// 杀掉整棵进程树，避免 Windows 上只杀主进程导致子进程变孤儿（堆积 msedge.exe）。
func runInBrowser(timeout time.Duration, actions ...chromedp.Action) error {
	var browserCmd *exec.Cmd
	opts := append(browserOpts(), chromedp.ModifyCmdFunc(func(cmd *exec.Cmd) {
		browserCmd = cmd
	}))

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)

	// 清理顺序（LIFO 执行）：先杀进程树（此时主进程仍存活，taskkill /T 能命中全部
	// 子进程），再取消超时上下文、chromedp 上下文、allocator。
	defer cancelAlloc()
	defer cancelCtx()
	ctx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()
	defer killProcessTree(browserCmd)

	return chromedp.Run(ctx, actions...)
}

// FetchHtmlWithChromeDP 启动浏览器获取页面 HTML。
func FetchHtmlWithChromeDP(url string) (string, error) {
	fmt.Printf("🚀 使用 chromedp 获取 HTML: %s\n", url)

	var html string
	var statusCode int64

	err := runInBrowser(45*time.Second,
		// 导航到页面
		chromedp.Navigate(url),

		// 等待页面完全加载
		chromedp.WaitReady(":root", chromedp.ByQueryAll),

		// 额外等待 3 秒确保 JavaScript 执行完毕
		chromedp.Sleep(3*time.Second),

		// 获取状态码
		chromedp.Evaluate(`window.performance && window.performance.navigation ? window.performance.navigation.type : 0`, &statusCode),

		// 获取完整的 HTML（包括动态生成的内容）
		chromedp.OuterHTML(":root", &html, chromedp.ByQueryAll),
	)

	if err != nil {
		fmt.Printf("❌ chromedp 获取 HTML 失败：%v\n", err)
		return "", fmt.Errorf("chromedp 执行失败：%v", err)
	}

	fmt.Printf("✅ 成功获取 HTML，长度：%d 字符，状态码：%d\n", len(html), statusCode)
	return html, nil
}

// findChromePath 查找 Google Chrome 的路径
func findChromePath() string {
	// Windows 常见路径
	paths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`%LOCALAPPDATA%\Google\Chrome\Application\chrome.exe`,
	}

	for _, path := range paths {
		expanded := os.ExpandEnv(path)
		if _, err := os.Stat(expanded); err == nil {
			return expanded
		}
	}

	return ""
}

// findEdgePath 查找 Microsoft Edge 的路径
func findEdgePath() string {
	// Windows 常见路径
	paths := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`%LOCALAPPDATA%\Microsoft\Edge\Application\msedge.exe`,
	}

	for _, path := range paths {
		expanded := os.ExpandEnv(path)
		if _, err := os.Stat(expanded); err == nil {
			return expanded
		}
	}

	return ""
}

// FetchWithChromeDPAndDecode 启动浏览器获取 JSON 并解码到 entity。
func FetchWithChromeDPAndDecode(url string, entity any) error {
	fmt.Printf("🌐 使用 chromedp 获取 JSON: %s\n", url)

	var jsonResponse string

	err := runInBrowser(45*time.Second,
		// 导航到 URL（浏览器会处理 TLS 握手）
		chromedp.Navigate(url),

		// 等待页面加载完成
		chromedp.WaitReady(":root", chromedp.ByQueryAll),

		// 额外等待确保响应完全接收
		chromedp.Sleep(2*time.Second),

		// 获取页面的 pre 标签内容（JSON 通常以纯文本显示）
		chromedpText(":root", &jsonResponse),
	)

	if err != nil {
		fmt.Printf("❌ chromedp 获取 JSON 失败：%v\n", err)
		return fmt.Errorf("chromedp 执行失败：%v", err)
	}

	// 清理响应文本（去除可能的空白字符）
	jsonResponse = strings.TrimSpace(jsonResponse)

	if jsonResponse == "" {
		return fmt.Errorf("未能获取到 JSON 响应")
	}

	fmt.Printf("✅ 成功获取 JSON，长度：%d 字符\n", len(jsonResponse))

	// 解码 JSON 到实体
	if err := json.Unmarshal([]byte(jsonResponse), entity); err != nil {
		return fmt.Errorf("JSON 解码失败：%v", err)
	}

	fmt.Printf("✅ JSON 解码成功\n")
	return nil
}

func chromedpText(sel interface{}, text *string, opts ...chromedp.QueryOption) chromedp.Action {
	return chromedp.Text(sel, text, opts...)
}
