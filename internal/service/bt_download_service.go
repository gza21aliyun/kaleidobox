package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"lunabox/internal/applog"
)

// BTDownloadService BT下载服务
type BTDownloadService struct {
	ctx context.Context
}

// BtSearchResult BT搜索结果
type BtSearchResult struct {
	Title      string `json:"title"`       // 标题
	Link       string `json:"link"`        // 磁力链接或torrent文件URL
	TorrentURL string `json:"torrent_url"` // torrent文件URL（如果有）
	Size       string `json:"size"`        // 文件大小
	Seeders    int    `json:"seeders"`     // 做种数
	Leechers   int    `json:"leechers"`    // 下载中的用户数
	Trusted    bool   `json:"trusted"`     // 是否受信任
	Date       string `json:"date"`        // 发布日期
	PageLink   string `json:"page_link"`   // 详情页面链接
}

// BtDownloadResult BT下载结果
type BtDownloadResult struct {
	Success bool   `json:"success"` // 是否成功
	Message string `json:"message"` // 结果信息
	TaskID  string `json:"task_id"` // 任务ID（qBittorrent返回）
}

func NewBTDownloadService() *BTDownloadService {
	return &BTDownloadService{}
}

func (s *BTDownloadService) Init(ctx context.Context) {
	s.ctx = ctx
}

// SearchBT 通过RSS搜索BT资源
// searchKey: 搜索关键字
// rssURL: RSS URL，其中 %search_key 会被替换为搜索关键字
func (s *BTDownloadService) SearchBT(searchKey string, rssURL string) ([]BtSearchResult, error) {
	if searchKey == "" || rssURL == "" {
		return nil, fmt.Errorf("search key or rss url is empty")
	}

	// 替换搜索关键字到RSS URL
	searchURL := strings.ReplaceAll(rssURL, "%search_key", url.QueryEscape(searchKey))
	applog.InfoLogSaveAppLog("BTDownloadService: searching with URL: %s", searchURL)

	// 请求RSS feed
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	// 解析RSS XML
	results := parseNyaaRSS(body)
	applog.InfoLogSaveAppLog("BTDownloadService: found %d results", len(results))
	return results, nil
}

// parseNyaaRSS 解析Nyaa RSS XML
func parseNyaaRSS(xmlData []byte) []BtSearchResult {
	var results []BtSearchResult

	// 简单解析RSS XML，提取item
	content := string(xmlData)

	// 提取每个item
	itemParts := strings.Split(content, "<item>")
	for i, part := range itemParts {
		if i == 0 {
			continue // 跳过第一个（header部分）
		}

		result := BtSearchResult{}

		// 提取title
		if title := extractXMLValue(part, "<title>"); title != "" {
			result.Title = title
		}

		// 提取link (可能是磁力链接或torrent文件URL)
		link := extractXMLValue(part, "<link>")
		if link != "" {
			if strings.HasPrefix(link, "magnet:") {
				// 直接是磁力链接
				result.Link = link
				applog.InfoLogSaveAppLog("BTDownloadService: found direct magnet link in <link>")
			} else if strings.HasSuffix(link, ".torrent") {
				// 是torrent文件URL，需要下载后发送
				result.Link = link
				result.TorrentURL = link
				applog.InfoLogSaveAppLog("BTDownloadService: found torrent URL in <link>: %s", link)
			}
		}

		// 如果link不是磁力链接且没有torrent URL，尝试从infoHash构造
		if !strings.HasPrefix(result.Link, "magnet:") && result.TorrentURL == "" {
			infoHash := extractXMLValue(part, "<nyaa:infoHash>")
			if infoHash == "" {
				infoHash = extractXMLValue(part, "infoHash>")
			}
			if infoHash != "" && result.Title != "" {
				// 构造磁力链接: magnet:?xt=urn:btih:<infohash>&dn=<title>
				magnetLink := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infoHash, url.QueryEscape(result.Title))
				result.Link = magnetLink
				applog.InfoLogSaveAppLog("BTDownloadService: constructed magnet link from infoHash=%s", infoHash)
			}
		}

		// 提取guid作为详情页
		if guid := extractXMLValue(part, "<guid>"); guid != "" {
			if strings.HasPrefix(guid, "http") {
				result.PageLink = guid
			}
		}

		// 提取pubDate
		if date := extractXMLValue(part, "<pubDate>"); date != "" {
			result.Date = date
		}

		// 提取size (nyaa格式: size标签)
		size := extractXMLValue(part, "<nyaa:size>")
		if size == "" {
			size = extractXMLValue(part, "<size>")
		}
		if size != "" {
			result.Size = size
		}

		// 提取seeders (nyaa格式)
		seeders := extractXMLValue(part, "<nyaa:seeders>")
		if seeders == "" {
			seeders = extractXMLValue(part, "<seeders>")
		}
		if seeders != "" {
			fmt.Sscanf(seeders, "%d", &result.Seeders)
		}

		// 提取leechers (nyaa格式)
		leechers := extractXMLValue(part, "<nyaa:leechers>")
		if leechers == "" {
			leechers = extractXMLValue(part, "<leechers>")
		}
		if leechers != "" {
			fmt.Sscanf(leechers, "%d", &result.Leechers)
		}

		// 提取trusted (nyaa格式)
		trusted := extractXMLValue(part, "<nyaa:trusted>")
		if trusted == "" {
			trusted = extractXMLValue(part, "<trusted>")
		}
		result.Trusted = trusted == "Yes"

		// 只有包含磁力链接或torrent URL的结果才添加
		if result.Title != "" && result.Link != "" && (strings.HasPrefix(result.Link, "magnet:") || result.TorrentURL != "") {
			results = append(results, result)
			applog.InfoLogSaveAppLog("BTDownloadService: added result - title=%s, link=%s, torrent_url=%s", result.Title, result.Link, result.TorrentURL)
		} else {
			applog.InfoLogSaveAppLog("BTDownloadService: skipped result - title=%s, link=%s", result.Title, result.Link)
		}
	}

	return results
}

// extractXMLValue 提取XML标签内的值
func extractXMLValue(content, tag string) string {
	start := strings.Index(content, tag)
	if start == -1 {
		return ""
	}
	start += len(tag)
	end := strings.Index(content[start:], "</")
	if end == -1 {
		return ""
	}
	value := content[start : start+end]
	// 解码HTML实体
	value = strings.ReplaceAll(value, "&lt;", "<")
	value = strings.ReplaceAll(value, "&gt;", ">")
	value = strings.ReplaceAll(value, "&amp;", "&")
	value = strings.ReplaceAll(value, "&apos;", "'")
	value = strings.ReplaceAll(value, "&quot;", "\"")
	// 解码CDATA
	if strings.HasPrefix(value, "<![CDATA[") && strings.HasSuffix(value, "]]>") {
		value = value[9 : len(value)-3]
	}
	return strings.TrimSpace(value)
}

// extractXMLAttr 提取XML标签的属性值
func extractXMLAttr(content, tag, attr string) string {
	start := strings.Index(content, tag)
	if start == -1 {
		return ""
	}
	tagContent := content[start : start+len(tag)+100] // 简单截取
	attrStr := fmt.Sprintf(`%s="`, attr)
	attrStart := strings.Index(tagContent, attrStr)
	if attrStart == -1 {
		return ""
	}
	attrStart += len(attrStr)
	attrEnd := strings.Index(tagContent[attrStart:], `"`)
	if attrEnd == -1 {
		return ""
	}
	return tagContent[attrStart : attrStart+attrEnd]
}

// DownloadToQBittorrent 发送下载任务到qBittorrent
// server: qBittorrent服务器地址
// port: 端口
// user: 用户名
// password: 密码
// downloadFolder: 下载目录（可选）
// magnetLink: 磁力链接
func (s *BTDownloadService) DownloadToQBittorrent(server, user, password, downloadFolder, magnetLink string, port int) (BtDownloadResult, error) {
	applog.InfoLogSaveAppLog("BTDownloadService: DownloadToQBittorrent called - server=%s, port=%d, user=%s, password_len=%d, folder=%s, magnet=%s",
		server, port, user, len(password), downloadFolder, magnetLink)

	if server == "" || magnetLink == "" {
		applog.InfoLogSaveAppLog("BTDownloadService: missing required parameters - server or magnetLink is empty")
		return BtDownloadResult{Success: false, Message: "missing required parameters (server or magnetLink)"}, fmt.Errorf("missing required parameters")
	}

	// 构建qBittorrent API URL
	baseURL := fmt.Sprintf("http://%s:%d", server, port)
	apiURL := baseURL + "/api/v2/auth/login"
	applog.InfoLogSaveAppLog("BTDownloadService: login URL=%s", apiURL)

	// 登录获取cookie
	client := &http.Client{Timeout: 30 * time.Second}
	loginBody := fmt.Sprintf("username=%s&password=%s", url.QueryEscape(user), url.QueryEscape(password))
	applog.InfoLogSaveAppLog("BTDownloadService: login body=%s", loginBody)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(loginBody))
	if err != nil {
		applog.InfoLogSaveAppLog("BTDownloadService: create login request failed: %v", err)
		return BtDownloadResult{Success: false, Message: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		applog.InfoLogSaveAppLog("BTDownloadService: login request failed: %v", err)
		return BtDownloadResult{Success: false, Message: err.Error()}, err
	}
	defer resp.Body.Close()

	applog.InfoLogSaveAppLog("BTDownloadService: login response status=%d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		applog.InfoLogSaveAppLog("BTDownloadService: login failed, response body=%s", string(body))
		return BtDownloadResult{Success: false, Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))}, fmt.Errorf("login failed")
	}

	// 检查是否登录成功（qBittorrent返回"Ok."表示成功）
	body, _ := io.ReadAll(resp.Body)
	loginResult := string(body)
	applog.InfoLogSaveAppLog("BTDownloadService: login result=%s", loginResult)

	if loginResult != "Ok." {
		applog.InfoLogSaveAppLog("BTDownloadService: login failed, invalid credentials")
		return BtDownloadResult{Success: false, Message: "login failed, invalid credentials"}, fmt.Errorf("login failed")
	}

	// 获取cookie
	cookies := resp.Cookies()
	var cookieStr string
	for _, c := range cookies {
		cookieStr += c.Name + "=" + c.Value + "; "
	}
	applog.InfoLogSaveAppLog("BTDownloadService: cookies=%s", cookieStr)

	if cookieStr == "" {
		// qBittorrent 可能不返回 cookie，而是使用 SID
		applog.InfoLogSaveAppLog("BTDownloadService: no cookie received, trying Referer header approach")
	}

	// 发送下载任务
	downloadURL := baseURL + "/api/v2/torrents/add"
	applog.InfoLogSaveAppLog("BTDownloadService: download URL=%s", downloadURL)

	// qBittorrent 的 urls 参数可以直接接受 magnet 链接
	// 如果是 .torrent 文件 URL，则需要下载后通过 multipart form 上传
	var postBody string
	var req2 *http.Request

	if strings.HasSuffix(magnetLink, ".torrent") {
		// 下载 torrent 文件到本地临时目录
		applog.InfoLogSaveAppLog("BTDownloadService: downloading torrent file from %s", magnetLink)

		// 创建临时文件
		tempDir := os.TempDir()
		tempFile, err := os.CreateTemp(tempDir, "lunabox_*.torrent")
		if err != nil {
			applog.InfoLogSaveAppLog("BTDownloadService: create temp file failed: %v", err)
			return BtDownloadResult{Success: false, Message: err.Error()}, err
		}
		tempPath := tempFile.Name()
		defer os.Remove(tempPath) // 确保函数结束时删除临时文件

		// 下载 torrent 文件
		torrentResp, err := client.Get(magnetLink)
		if err != nil {
			applog.InfoLogSaveAppLog("BTDownloadService: download torrent file failed: %v", err)
			tempFile.Close()
			return BtDownloadResult{Success: false, Message: err.Error()}, err
		}
		defer torrentResp.Body.Close()

		if torrentResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(torrentResp.Body)
			applog.InfoLogSaveAppLog("BTDownloadService: download torrent file failed with status %d: %s", torrentResp.StatusCode, string(body))
			tempFile.Close()
			return BtDownloadResult{Success: false, Message: fmt.Sprintf("download torrent failed: HTTP %d", torrentResp.StatusCode)}, fmt.Errorf("download torrent failed")
		}

		// 写入临时文件
		_, err = io.Copy(tempFile, torrentResp.Body)
		tempFile.Close()
		if err != nil {
			applog.InfoLogSaveAppLog("BTDownloadService: write temp file failed: %v", err)
			return BtDownloadResult{Success: false, Message: err.Error()}, err
		}
		applog.InfoLogSaveAppLog("BTDownloadService: downloaded torrent file to %s, size=%d bytes", tempPath, 0) // 后续获取实际大小

		// 读取文件内容用于日志
		fileInfo, _ := os.Stat(tempPath)
		if fileInfo != nil {
			applog.InfoLogSaveAppLog("BTDownloadService: torrent file size=%d bytes", fileInfo.Size())
		}

		// 构建 multipart form 请求
		req2, err = newMultipartRequest(downloadURL, tempPath, downloadFolder)
		if err != nil {
			applog.InfoLogSaveAppLog("BTDownloadService: create multipart request failed: %v", err)
			return BtDownloadResult{Success: false, Message: err.Error()}, err
		}
	} else {
		// 磁力链接直接发送
		if downloadFolder != "" {
			postBody = fmt.Sprintf("urls=%s&savepath=%s", url.QueryEscape(magnetLink), url.QueryEscape(downloadFolder))
		} else {
			postBody = fmt.Sprintf("urls=%s", url.QueryEscape(magnetLink))
		}
		applog.InfoLogSaveAppLog("BTDownloadService: post body=%s", truncateString(postBody, 500))

		req2, err = http.NewRequest("POST", downloadURL, strings.NewReader(postBody))
		if err != nil {
			applog.InfoLogSaveAppLog("BTDownloadService: create download request failed: %v", err)
			return BtDownloadResult{Success: false, Message: err.Error()}, err
		}
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	req2.Header.Set("User-Agent", "Mozilla/5.0")
	req2.Header.Set("Referer", baseURL)
	if cookieStr != "" {
		req2.Header.Set("Cookie", cookieStr)
	}

	resp2, err := client.Do(req2)
	if err != nil {
		applog.InfoLogSaveAppLog("BTDownloadService: download request failed: %v", err)
		return BtDownloadResult{Success: false, Message: err.Error()}, err
	}
	defer resp2.Body.Close()

	applog.InfoLogSaveAppLog("BTDownloadService: download response status=%d", resp2.StatusCode)

	body2, _ := io.ReadAll(resp2.Body)
	resultBody := string(body2)
	applog.InfoLogSaveAppLog("BTDownloadService: download response body=%s", resultBody)

	if resp2.StatusCode != http.StatusOK {
		return BtDownloadResult{Success: false, Message: fmt.Sprintf("HTTP %d: %s", resp2.StatusCode, resultBody)}, fmt.Errorf("add torrent failed")
	}

	// qBittorrent 成功时返回空响应或 "Ok."
	if resultBody == "" || resultBody == "Ok." {
		applog.InfoLogSaveAppLog("BTDownloadService: download task added successfully")
		return BtDownloadResult{Success: true, Message: "Download task added successfully", TaskID: ""}, nil
	}

	applog.InfoLogSaveAppLog("BTDownloadService: add torrent failed with response: %s", resultBody)
	return BtDownloadResult{Success: false, Message: resultBody}, fmt.Errorf("add torrent failed: %s", resultBody)
}

// GetTitleFromSearchName 从搜索名称中提取标题（全名和短标题）
// 参考helper.go中的getTitlesNum函数
func (s *BTDownloadService) GetTitleFromSearchName(searchName string) (string, string) {
	if searchName == "" {
		return "", ""
	}
	mainTitle, subTitle, _ := getTitlesNum(searchName, false)
	// mainTitle是主标题，subTitle是副标题（如果有的话）
	// 完整标题 = mainTitle + subTitle
	fullName := mainTitle
	if subTitle != "" {
		fullName = mainTitle + " " + subTitle
	}
	return mainTitle, fullName
}

// getTitlesNum 从搜索名称中提取标题和编号
// 参考 internal/utils/helper.go 中的 getTitlesNum 函数
func getTitlesNum(searchName string, onlyNum bool) (mainT string, subT string, number string) {
	hasValidSeparator := regexp.MustCompile(`[－\-~～　＝ ・！：─―_!「\[]`).MatchString(searchName)

	mainTitle := ""

	if !hasValidSeparator {
		// 没有有效分隔符，整个字符串作为主标题处理
		mainTitle = regexp.QuoteMeta(strings.TrimSpace(searchName))
		mainTitle, numStr := extractLastNumberFromString(mainTitle)
		return handleMainTitle(mainTitle), "", numStr
	}

	// 有有效分隔符，尝试分离主标题和副标题
	hasNonEnglish := regexp.MustCompile(`[^a-zA-Z0-9 ]`).MatchString(searchName)
	separatorPattern := `[－\-~～_]+`
	if hasNonEnglish {
		separatorPattern = `[－\-~～＝： ・_!─―「]+`
	}
	parts := regexp.MustCompile(separatorPattern).Split(searchName, -1)
	if len(parts) > 1 {
		mainTitle = strings.TrimSpace(parts[0])
	}
	if mainTitle == "" {
		mainTitle = searchName
	}
	if len(parts) < 2 {
		hasNonEnglish = regexp.MustCompile(`[^a-zA-Z0-9 ]`).MatchString(mainTitle)
		if onlyNum || hasNonEnglish {
			separatorPattern = `[－\-~～　＝ ！・─「\[]+`
		} else {
			separatorPattern = `[－\-~～　！\[]+`
		}
		parts = regexp.MustCompile(separatorPattern).Split(searchName, -1)
	}

	if len(parts) >= 2 {
		mainTitle = strings.TrimSpace(parts[0])
		subTitle := strings.TrimSpace(parts[1])
		numStr := ""
		if mainTitle != "" && subTitle != "" {
			for i, txt := range parts {
				if i == 0 {
					mainTitle, numStr = extractLastNumberFromString(txt)
					if numStr != "" {
						return handleMainTitle(mainTitle), subTitle, numStr
					}
				} else if i == 1 {
					subTitle, numStr = extractLastNumberFromString(txt)
					if numStr != "" {
						return handleMainTitle(mainTitle), subTitle, numStr
					}
				} else {
					_, numStr = extractLastNumberFromString(txt)
					if numStr != "" {
						return handleMainTitle(mainTitle), subTitle, numStr
					}
				}
			}
			if re, err := regexp.MatchString(`^[a-zA-Z]{1,4}$`, mainTitle); err == nil && re && subTitle != "" {
				mainTitle = subTitle
			}
			return handleMainTitle(mainTitle), subTitle, numStr
		}
		if mainTitle == "" && subTitle != "" {
			return handleMainTitle(subTitle), "", ""
		}
	}

	// 默认情况：整个字符串作为主标题
	mainTitle = regexp.QuoteMeta(strings.TrimSpace(searchName))
	mainTitle, numStr := extractLastNumberFromString(mainTitle)

	return handleMainTitle(mainTitle), "", numStr
}

// handleMainTitle 处理主标题
func handleMainTitle(mainTitle string) string {
	title := mainTitle
	lowTitle := strings.ToLower(mainTitle)
	if strings.Contains(lowTitle, "chapter") {
		parts := strings.Split(lowTitle, "chapter")
		if len(parts) > 0 {
			title = strings.TrimSpace(parts[0])
		}
	}
	return title
}

// extractLastNumberFromString 从字符串中提取末尾的数字
func extractLastNumberFromString(s string) (string, string) {
	re := regexp.MustCompile(`(\d+)[^\d]*$`)
	match := re.FindStringSubmatch(s)
	if len(match) > 1 {
		numStr := match[1]
		rest := strings.TrimSuffix(s, match[0])
		return rest, numStr
	}
	return s, ""
}

// newMultipartRequest 创建 multipart form 请求用于上传 torrent 文件
func newMultipartRequest(urlStr, filePath, downloadFolder string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 打开 torrent 文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 添加 torrent 文件
	part, err := writer.CreateFormFile("torrentFile", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	// 添加下载目录（如果指定）
	if downloadFolder != "" {
		err = writer.WriteField("savepath", downloadFolder)
		if err != nil {
			return nil, err
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
