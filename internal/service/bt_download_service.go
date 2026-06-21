package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	Title    string `json:"title"`     // 标题
	Link     string `json:"link"`      // 磁力链接或详情页
	Size     string `json:"size"`      // 文件大小
	Seeders  int    `json:"seeders"`   // 做种数
	Date     string `json:"date"`      // 发布日期
	PageLink string `json:"page_link"` // 详情页面链接
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

		// 提取link (可能是磁力链接)
		if link := extractXMLValue(part, "<link>"); link != "" {
			result.Link = link
		}

		// 提取enclosure (磁力链接)
		if enclosure := extractXMLAttr(part, "enclosure", "url"); enclosure != "" {
			result.Link = enclosure
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
		if size := extractXMLValue(part, "<size>"); size != "" {
			result.Size = size
		}

		// 提取seeders (nyaa格式)
		if seeders := extractXMLValue(part, "<seeders>"); seeders != "" {
			fmt.Sscanf(seeders, "%d", &result.Seeders)
		}

		// 如果有标题和链接，才添加
		if result.Title != "" && result.Link != "" {
			// 确保链接是磁力链接
			if strings.HasPrefix(result.Link, "magnet:") || strings.HasPrefix(result.Link, "http") {
				results = append(results, result)
			}
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
	if server == "" || user == "" || password == "" || magnetLink == "" {
		return BtDownloadResult{Success: false, Message: "missing required parameters"}, fmt.Errorf("missing required parameters")
	}

	// 构建qBittorrent API URL
	baseURL := fmt.Sprintf("http://%s:%d", server, port)
	apiURL := baseURL + "/api/v2/auth/login"

	// 登录获取cookie
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(fmt.Sprintf("username=%s&password=%s", url.QueryEscape(user), url.QueryEscape(password))))
	if err != nil {
		return BtDownloadResult{Success: false, Message: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return BtDownloadResult{Success: false, Message: err.Error()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return BtDownloadResult{Success: false, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}, fmt.Errorf("login failed")
	}

	// 检查是否登录成功（qBittorrent返回"Ok."表示成功）
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Ok." {
		return BtDownloadResult{Success: false, Message: "login failed, invalid credentials"}, fmt.Errorf("login failed")
	}

	// 获取cookie
	cookies := resp.Cookies()
	var cookieStr string
	for _, c := range cookies {
		cookieStr += c.Name + "=" + c.Value + "; "
	}
	if cookieStr == "" {
		return BtDownloadResult{Success: false, Message: "no cookie received"}, fmt.Errorf("no cookie")
	}

	// 发送下载任务
	downloadURL := baseURL + "/api/v2/torrents/add"
	var postBody string
	if downloadFolder != "" {
		postBody = fmt.Sprintf("urls=%s&downloadpath=%s", url.QueryEscape(magnetLink), url.QueryEscape(downloadFolder))
	} else {
		postBody = fmt.Sprintf("urls=%s", url.QueryEscape(magnetLink))
	}

	req2, err := http.NewRequest("POST", downloadURL, strings.NewReader(postBody))
	if err != nil {
		return BtDownloadResult{Success: false, Message: err.Error()}, err
	}
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("Cookie", cookieStr)
	req2.Header.Set("User-Agent", "Mozilla/5.0")

	resp2, err := client.Do(req2)
	if err != nil {
		return BtDownloadResult{Success: false, Message: err.Error()}, err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return BtDownloadResult{Success: false, Message: fmt.Sprintf("HTTP %d", resp2.StatusCode)}, fmt.Errorf("add torrent failed")
	}

	body2, _ := io.ReadAll(resp2.Body)
	resultBody := string(body2)

	if strings.Contains(resultBody, "Ok.") || resultBody == "" {
		return BtDownloadResult{Success: true, Message: "Download task added successfully", TaskID: ""}, nil
	}

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
