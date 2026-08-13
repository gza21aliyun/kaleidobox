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
		time.Sleep(200 * time.Millisecond)
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
		fmt.Printf("🧹 清理: taskkill /F /T /PID %d\n", pid)
		if err := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run(); err != nil {
			fmt.Printf("🧹 清理: taskkill 失败: %v\n", err)
		}
	} else {
		_ = cmd.Process.Kill()
	}
}

// launchBrowser 自己启动浏览器（绕开 chromedp ExecAllocator），返回 (cmd, wsURL)。
// 调用方负责用完后 killProcessTree(cmd)。
func launchBrowser(forJson bool) (cmd *exec.Cmd, wsURL, userDataDir string, err error) {
	// 1. 临时 user-data-dir
	userDataDir, err = os.MkdirTemp(os.TempDir(), "lunabox-chromedp-*")
	if err != nil {
		return nil, "", "", fmt.Errorf("创建临时用户目录失败: %w", err)
	}

	// 2. 找空闲端口
	port, err := findFreePort()
	if err != nil {
		_ = os.RemoveAll(userDataDir)
		return nil, "", "", fmt.Errorf("分配端口失败: %w", err)
	}

	// 3. 找浏览器（先测试 edge，暂时注释掉 chrome）
	browserPath := ""
	if /*chromePath := findChromePath(); chromePath != ""*/ false {
		// browserPath = chromePath
	} else if edgePath := findEdgePath(); edgePath != "" {
		fmt.Println("未检测到 Chrome，尝试使用 Edge:", edgePath)
		browserPath = edgePath
	} else {
		_ = os.RemoveAll(userDataDir)
		return nil, "", "", fmt.Errorf("未找到 Chrome 或 Edge")
	}

	// 4. 构建启动参数
	// 用旧版 --headless（不带 =new）：RemoteAllocator 不依赖 stderr 解析（已绕开
	// Chrome 145 regression），所以旧 headless 可用。旧 headless 的页面加载行为与
	// Chrome 一致（Chrome 用旧版能瞬间完成），而 --headless=new 在 Edge 上可能导致
	// DLsite 页面加载行为不同从而超时。
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
		"about:blank",
	}
	if !forJson {
		args = append(args[:len(args)-1], // 把 about:blank 移到最后面之前插入
			"--disable-web-security",
			"--allow-running-insecure-content",
			"about:blank",
		)
	}

	fmt.Println("🔍 浏览器启动命令行:")
	fmt.Printf("   exe   : %s\n", browserPath)
	fmt.Printf("   port  : %d\n", port)
	fmt.Printf("   args  : %s\n", strings.Join(args, " "))
	fmt.Printf("   user-data-dir: %s\n", userDataDir)

	// 5. 启动浏览器进程
	cmd = exec.Command(browserPath, args...)
	if startErr := cmd.Start(); startErr != nil {
		_ = os.RemoveAll(userDataDir)
		return nil, "", "", fmt.Errorf("启动浏览器失败: %w", startErr)
	}
	fmt.Printf("   PID   : %d\n", cmd.Process.Pid)

	// 6. 轮询等待 DevTools 就绪
	fmt.Println("[5/8] 等待 DevTools 就绪（轮询 /json/version）...")
	wsURL, err = waitForDevTools(port, 30*time.Second)
	if err != nil {
		killProcessTree(cmd)
		_ = os.RemoveAll(userDataDir)
		return nil, "", "", err
	}
	fmt.Printf("[5/8] ✅ DevTools 就绪: %s\n", wsURL)

	return cmd, wsURL, userDataDir, nil
}

// runInBrowser 是核心：自己启动浏览器 → RemoteAllocator 连接 → 执行 actions → 清理。
func runInBrowser(timeout time.Duration, forJson bool, actions ...chromedp.Action) error {
	fmt.Println("[1/8] 启动浏览器（RemoteAllocator 方案）...")
	cmd, wsURL, userDataDir, err := launchBrowser(forJson)
	if err != nil {
		fmt.Printf("[1/8] ❌ 失败: %v\n", err)
		return err
	}
	fmt.Println("[1/8] ✅ 浏览器已启动")

	// 清理：先杀进程树，再删临时目录
	defer func() {
		killProcessTree(cmd)
		time.Sleep(300 * time.Millisecond)
		if err := os.RemoveAll(userDataDir); err != nil {
			fmt.Printf("🧹 清理: 删除临时目录失败: %v\n", err)
		} else {
			fmt.Printf("🧹 清理: 已删除临时目录 %s\n", userDataDir)
		}
	}()

	fmt.Println("[2/8] 创建 RemoteAllocator...")
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), wsURL)
	defer cancelAlloc()
	fmt.Println("[2/8] ✅ 完成")

	fmt.Println("[3/8] 创建 chromedp Context（新标签页）...")
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()
	fmt.Println("[3/8] ✅ 完成")

	fmt.Println("[4/8] 设置超时...")
	ctx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()
	fmt.Println("[4/8] ✅ 完成")

	fmt.Println("[6/8] 执行 chromedp.Run...")
	err = chromedp.Run(ctx, actions...)
	fmt.Println("[7/8] ✅ chromedp.Run 返回")
	if err != nil {
		return err
	}

	fmt.Println("[8/8] ✅ 完成")
	return nil
}

// logStep 包装一个 chromedp.Action，在执行前后打印日志和耗时，用于定位卡在哪步。
func logStep(name string, action chromedp.Action) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		fmt.Printf("   ▸ %s 开始...\n", name)
		start := time.Now()
		err := action.Do(ctx)
		if err != nil {
			fmt.Printf("   ▸ %s ❌ 失败 (%v): %v\n", name, time.Since(start), err)
		} else {
			fmt.Printf("   ▸ %s ✅ 完成 (%v)\n", name, time.Since(start))
		}
		return err
	})
}

// FetchHtmlWithChromeDP 启动浏览器获取页面 HTML。
func FetchHtmlWithChromeDP(url string) (string, error) {
	fmt.Printf("🚀 使用 chromedp 获取 HTML: %s\n", url)

	var html string
	var statusCode int64

	err := runInBrowser(45*time.Second, false,
		logStep("Navigate", chromedp.Navigate(url)),
		logStep("WaitReady(:root)", chromedp.WaitReady(":root", chromedp.ByQueryAll)),
		logStep("Sleep 3s", chromedp.Sleep(3*time.Second)),
		logStep("Evaluate(statusCode)", chromedp.Evaluate(`window.performance && window.performance.navigation ? window.performance.navigation.type : 0`, &statusCode)),
		logStep("OuterHTML(:root)", chromedp.OuterHTML(":root", &html, chromedp.ByQueryAll)),
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
	fmt.Printf("🌐 使用 chromedp 获取 JSON: %s\n", url)

	var jsonResponse string

	err := runInBrowser(45*time.Second, true,
		logStep("Navigate", chromedp.Navigate(url)),
		logStep("WaitReady(:root)", chromedp.WaitReady(":root", chromedp.ByQueryAll)),
		logStep("Sleep 2s", chromedp.Sleep(2*time.Second)),
		logStep("Text(:root)", chromedpText(":root", &jsonResponse)),
	)

	if err != nil {
		fmt.Printf("❌ chromedp 获取 JSON 失败：%v\n", err)
		return fmt.Errorf("chromedp 执行失败：%v", err)
	}

	jsonResponse = strings.TrimSpace(jsonResponse)

	if jsonResponse == "" {
		fmt.Println("⚠️  获取到的 JSON 响应为空")
		return fmt.Errorf("未能获取到 JSON 响应")
	}

	fmt.Printf("✅ 成功获取 JSON，长度：%d 字符\n", len(jsonResponse))

	if err := json.Unmarshal([]byte(jsonResponse), entity); err != nil {
		return fmt.Errorf("JSON 解码失败：%v", err)
	}

	fmt.Printf("✅ JSON 解码成功\n")
	return nil
}

func chromedpText(sel interface{}, text *string, opts ...chromedp.QueryOption) chromedp.Action {
	return chromedp.Text(sel, text, opts...)
}
