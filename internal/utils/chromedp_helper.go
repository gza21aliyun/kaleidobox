package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func FetchHtmlWithChromeDP(url string) (string, error) {
	fmt.Printf("🚀 使用 chromedp 获取 HTML: %s\n", url)

	// 创建 chromedp 上下文
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true), // 无头模式（不显示界面）
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("allow-running-insecure-content", true),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36`),
	)

	// 如果检测到系统没有 Chrome，尝试使用 Edge
	if true {
		fmt.Println("未检测到 Chrome，尝试使用 Edge...")
		opts = append(opts,
			chromedp.ExecPath(findEdgePath()),
		)
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// 设置超时（DLsite 可能需要较长时间加载）
	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var html string
	var statusCode int64

	// 执行任务
	err := chromedp.Run(ctx,
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
		return "", fmt.Errorf("chromedp 执行失败：%v", err)
	}

	fmt.Printf("✅ 成功获取 HTML，长度：%d 字符，状态码：%d\n", len(html), statusCode)
	return html, nil
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

// func FetchWithChromeDPAndDecode(url string, entity any) error {
// 	jsonStr, err := FetchHtmlWithChromeDP(url)
// 	if err != nil {
// 		return err
// 	}
// 	json.NewDecoder(resp.Body).Decode(&entity)
// 	return nil

// }

func FetchWithChromeDPAndDecode(url string, entity any) error {
	fmt.Printf("🌐 使用 chromedp 获取 JSON: %s\n", url)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36`),
	)

	// 尝试使用 Edge
	if edgePath := findEdgePath(); edgePath != "" {
		fmt.Println("✅ 找到 Edge:", edgePath)
		opts = append(opts, chromedp.ExecPath(edgePath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// 设置超时
	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var jsonResponse string

	// 执行任务
	err := chromedp.Run(ctx,
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
		return fmt.Errorf("chromedp 执行失败：%v", err)
	}

	// 清理响应文本（去除可能的空白字符）
	jsonResponse = strings.TrimSpace(jsonResponse)

	if jsonResponse == "" {
		return fmt.Errorf("未能获取到 JSON 响应")
	}

	fmt.Printf("✅ 成功获取 JSON，长度：%d 字符\n", len(jsonResponse))
	// fmt.Println(jsonResponse)

	// 调试：保存 JSON 到文件
	// debugFile := fmt.Sprintf("chromedp_debug_%s.json", time.Now().Format("20060102_150405"))
	// if err := os.WriteFile(debugFile, []byte(jsonResponse), 0644); err == nil {
	// 	fmt.Printf("   调试文件已保存：%s\n", debugFile)
	// }

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
