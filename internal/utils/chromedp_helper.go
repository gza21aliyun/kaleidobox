package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// ──────────────────────────────────────────────────────────────────────────────
// 背景：chromedp 的 ExecAllocator 依赖从浏览器 stderr 解析
//   "DevTools listening on ws://127.0.0.1:PORT/devtools/browser/..."
// 来发现浏览器随机选的端口（--remote-debugging-port=0）。
// 但 Chrome 145 / 新版 Edge（Chromium 145+）在 Windows 上不再把这行打印到 stderr，
// 导致 chromedp 超时报 "chrome failed to start:"（冒号后空白），但浏览器实际已启动。
//
// 解决方案（参考 pinchtab#108）：完全绕开 ExecAllocator，自己启动浏览器 +
// 固定端口 + 轮询 /json/version 拿 wsURL + 用 NewRemoteAllocator 连接。
// ──────────────────────────────────────────────────────────────────────────────

// findFreePort 让 OS 分配一个空闲 TCP 端口（立即关闭 listener，把端口号给浏览器用）。
func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForDevTools 轮询 http://127.0.0.1:port/json/version，返回 webSocketDebuggerUrl。
// 首次立即尝试（不 sleep），后续每 100ms 重试。
func waitForDevTools(port int, timeout time.Duration) (string, error) {
	u := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(u)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var v struct {
				WSURL string `json:"webSocketDebuggerUrl"`
			}
			if json.Unmarshal(body, &v) == nil && v.WSURL != "" {
				return v.WSURL, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", fmt.Errorf("等待 DevTools 就绪超时（端口 %d，%v）", port, timeout)
}

// killProcessTree 在 Windows 上用 taskkill /F /T 杀整棵进程树。
func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
	} else {
		_ = cmd.Process.Kill()
	}
}

// killBrowserByUserDataDir 通过 --user-data-dir 查找并杀掉所有相关浏览器进程。
//
// 问题：RemoteAllocator 的 cancel 会关闭 WebSocket，导致 Edge 主进程自动退出。
// 随后 taskkill /F /T /PID 找不到主进程（exit 128），子进程（renderer / GPU /
// network service 等）全部变孤儿，批量搜刮时堆积大量 msedge.exe。
//
// 方案：用 wmic 按命令行匹配 user-data-dir，找到所有相关进程（含子进程）逐个
// taskkill /F /T /PID。不依赖主进程是否还活着。
func killBrowserByUserDataDir(userDataDir string) {
	if runtime.GOOS != "windows" || userDataDir == "" {
		return
	}
	// wmic like 子句中反斜杠需要转义为 \\
	escaped := strings.ReplaceAll(userDataDir, `\`, `\\`)
	out, err := exec.Command("wmic", "process", "where",
		fmt.Sprintf("commandline like '%%%s%%'", escaped),
		"get", "processid", "/format:list").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ProcessId=") {
			continue
		}
		pid := strings.TrimSpace(strings.TrimPrefix(line, "ProcessId="))
		if pid == "" {
			continue
		}
		_ = exec.Command("taskkill", "/F", "/T", "/PID", pid).Run()
	}
}

// launchBrowser 自己启动浏览器（绕开 chromedp ExecAllocator），返回 (cmd, wsURL)。
func launchBrowser(forJson bool) (cmd *exec.Cmd, wsURL, userDataDir string, err error) {
	userDataDir, err = os.MkdirTemp(os.TempDir(), "lunabox-chromedp-*")
	if err != nil {
		return nil, "", "", fmt.Errorf("创建临时用户目录失败: %w", err)
	}

	port, err := findFreePort()
	if err != nil {
		_ = os.RemoveAll(userDataDir)
		return nil, "", "", fmt.Errorf("分配端口失败: %w", err)
	}

	// 先测试 edge，暂时注释掉 chrome。
	browserPath := ""
	if /*chromePath := findChromePath(); chromePath != ""*/ false {
		// browserPath = chromePath
	} else if edgePath := findEdgePath(); edgePath != "" {
		browserPath = edgePath
	} else {
		_ = os.RemoveAll(userDataDir)
		return nil, "", "", fmt.Errorf("未找到 Chrome 或 Edge")
	}

	// 用旧版 --headless（不带 =new）：RemoteAllocator 不依赖 stderr 解析（已绕开
	// Chrome 145 regression），所以旧 headless 可用。旧 headless 的页面加载行为与
	// Chrome 一致，而 --headless=new 在 Edge 上可能导致页面加载超时。
	args := []string{
		"--headless",
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
	if !forJson {
		args = append(args,
			"--disable-web-security",
			"--allow-running-insecure-content",
		)
	}
	args = append(args, "about:blank")

	cmd = exec.Command(browserPath, args...)
	if startErr := cmd.Start(); startErr != nil {
		_ = os.RemoveAll(userDataDir)
		return nil, "", "", fmt.Errorf("启动浏览器失败: %w", startErr)
	}

	wsURL, err = waitForDevTools(port, 30*time.Second)
	if err != nil {
		killProcessTree(cmd)
		killBrowserByUserDataDir(userDataDir)
		_ = os.RemoveAll(userDataDir)
		return nil, "", "", err
	}

	return cmd, wsURL, userDataDir, nil
}

// runInBrowser 是核心：自己启动浏览器 → RemoteAllocator 连接 → 执行 actions → 清理。
func runInBrowser(timeout time.Duration, forJson bool, actions ...chromedp.Action) error {
	startTime := time.Now()

	cmd, wsURL, userDataDir, err := launchBrowser(forJson)
	if err != nil {
		return err
	}

	// 清理：先杀进程树（主进程），再用 user-data-dir 兜底杀孤儿子进程，最后删临时目录。
	// 注意：cancelAlloc/cancelCtx 在此 defer 之前执行（LIFO），会关闭 WebSocket 导致
	// Edge 主进程自动退出，所以 killProcessTree 可能找不到主进程（exit 128）。
	// killBrowserByUserDataDir 不依赖主进程 PID，通过 wmic 按命令行匹配杀所有子进程。
	defer func() {
		killProcessTree(cmd)                  // 快速杀主进程（如果还活着）
		killBrowserByUserDataDir(userDataDir) // 兜底杀孤儿子进程
		time.Sleep(100 * time.Millisecond)    // 等 OS 回收文件句柄
		_ = os.RemoveAll(userDataDir)
	}()

	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), wsURL)
	defer cancelAlloc()
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()
	ctx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()

	if err := chromedp.Run(ctx, actions...); err != nil {
		return err
	}

	fmt.Printf("⏱ chromedp 总耗时: %v\n", time.Since(startTime))
	return nil
}

// setDlsiteCookie 设置 DLsite 的 app_key cookie（在 Navigate 之前执行）。
func setDlsiteCookie() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		return network.SetCookie("adultchecked", "1").
			WithDomain(".dlsite.com").
			WithPath("/").
			Do(ctx)
	})
}

// logStep 包装一个 chromedp.Action，在执行前后打印耗时，用于定位瓶颈。
func logStep(name string, action chromedp.Action) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		start := time.Now()
		err := action.Do(ctx)
		if err != nil {
			fmt.Printf("   ▸ %s ❌ (%v): %v\n", name, time.Since(start), err)
		} else {
			fmt.Printf("   ▸ %s (%v)\n", name, time.Since(start))
		}
		return err
	})
}

// FetchHtmlWithChromeDP 启动浏览器获取页面 HTML。
func FetchHtmlWithChromeDP(url string) (string, error) {
	fmt.Printf("🚀 chromedp 获取 HTML: %s\n", url)

	var html string
	var statusCode int64

	// Sleep 从 3s 减到 500ms：WaitReady 已确保 DOM 就绪，500ms 足够 JS 渲染。
	err := runInBrowser(45*time.Second, false,
		logStep("SetCookie", setDlsiteCookie()),
		logStep("Navigate", chromedp.Navigate(url)),
		logStep("WaitReady", chromedp.WaitReady(":root", chromedp.ByQueryAll)),
		logStep("Sleep", chromedp.Sleep(500*time.Millisecond)),
		logStep("Evaluate", chromedp.Evaluate(`window.performance && window.performance.navigation ? window.performance.navigation.type : 0`, &statusCode)),
		logStep("OuterHTML", chromedp.OuterHTML(":root", &html, chromedp.ByQueryAll)),
	)

	if err != nil {
		fmt.Printf("❌ 获取 HTML 失败：%v\n", err)
		return "", fmt.Errorf("chromedp 执行失败：%v", err)
	}

	fmt.Printf("✅ HTML 长度: %d, 状态码: %d\n", len(html), statusCode)
	return html, nil
}

// findChromePath 查找 Google Chrome 的路径
func findChromePath() string {
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
	fmt.Printf("🌐 chromedp 获取 JSON: %s\n", url)

	var jsonResponse string

	// Sleep 从 2s 减到 500ms：WaitReady 已确保 DOM 就绪，500ms 足够 JS 渲染。
	err := runInBrowser(45*time.Second, true,
		logStep("SetCookie", setDlsiteCookie()),
		logStep("Navigate", chromedp.Navigate(url)),
		logStep("WaitReady", chromedp.WaitReady(":root", chromedp.ByQueryAll)),
		logStep("Sleep", chromedp.Sleep(500*time.Millisecond)),
		logStep("Text", chromedpText(":root", &jsonResponse)),
	)

	if err != nil {
		fmt.Printf("❌ 获取 JSON 失败：%v\n", err)
		return fmt.Errorf("chromedp 执行失败：%v", err)
	}

	jsonResponse = strings.TrimSpace(jsonResponse)

	if jsonResponse == "" {
		return fmt.Errorf("未能获取到 JSON 响应")
	}

	fmt.Printf("✅ JSON 长度: %d\n", len(jsonResponse))
	fmt.Printf("✅ JSON 内容: %s\n", jsonResponse)

	if err := json.Unmarshal([]byte(jsonResponse), entity); err != nil {
		return fmt.Errorf("JSON 解码失败：%v", err)
	}

	return nil
}

func chromedpText(sel interface{}, text *string, opts ...chromedp.QueryOption) chromedp.Action {
	return chromedp.Text(sel, text, opts...)
}
