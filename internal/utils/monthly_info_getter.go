package utils

import (
	"bytes"
	"fmt"
	"lunabox/internal/applog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// MonthlyReleaseGame 每月发售游戏中的单个游戏条目
type MonthlyReleaseGame struct {
	Name      string `json:"name"`       // 游戏名称
	GetchuID  string `json:"getchu_id"`  // Getchu ID
	CoverURL  string `json:"cover_url"`  // 封面图片 URL（大图）
	ReleaseAt string `json:"release_at"` // 发售日期 (YYYY/MM/DD)
	Company   string `json:"company"`    // 品牌/公司名称
	DetailURL string `json:"detail_url"` // 详情页链接
}

// MonthlyReleaseGroup 按发售日期分组的游戏列表
type MonthlyReleaseGroup struct {
	ReleaseDate string               `json:"release_date"` // 发售日期 (YYYY/MM/DD)
	Games       []MonthlyReleaseGame `json:"games"`        // 该日期的游戏列表
}

// MonthlyReleaseResult 每月发售列表的完整结果
type MonthlyReleaseResult struct {
	Year   int                   `json:"year"`   // 年份
	Month  int                   `json:"month"`  // 月份
	Groups []MonthlyReleaseGroup `json:"groups"` // 按日期分组的游戏列表
}

// MonthlyInfoGetter 获取 Getchu 每月游戏发售列表
type MonthlyInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

func NewMonthlyInfoGetter() *MonthlyInfoGetter {
	return &MonthlyInfoGetter{
		client:  &http.Client{},
		timeout: 30 * time.Second,
	}
}

// FetchMonthlyReleases 获取指定年月的 Getchu 游戏发售列表
// year: 年份（如 2026），month: 月份（如 6）
func (m *MonthlyInfoGetter) FetchMonthlyReleases(year, month int, age string) (MonthlyReleaseResult, error) {
	result := MonthlyReleaseResult{
		Year:  year,
		Month: month,
	}

	targetURL := fmt.Sprintf(
		"https://www.getchu.com/all/month_title.html?genre=pc_soft&gage=%s&year=%d&month=%02d",
		age, year, month,
	)

	applog.InfoLogSaveAppLog("FetchMonthlyReleases: visiting %s", targetURL)

	var groups []MonthlyReleaseGroup

	c := CreateCollector("*getchu.com")

	// Colly 自动根据 Content-Type charset 将 EUC-JP 解码为 UTF-8
	// 在 OnResponse 中拿到解码后的 body，用 goquery 手动解析 DOM，完全掌控遍历逻辑
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			applog.InfoLogSaveAppLog("FetchMonthlyReleases: unexpected status code: %d", r.StatusCode)
			return
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
		if err != nil {
			applog.InfoLogSaveAppLog("FetchMonthlyReleases: failed to parse HTML: %s", err)
			return
		}

		datePattern := regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日`)

		// 调试: 输出页面中所有 class 包含 category 的 div
		classDebug := []string{}
		doc.Find("div[class*='category']").Each(func(_ int, s *goquery.Selection) {
			cls, _ := s.Attr("class")
			txt := strings.TrimSpace(s.Text())
			if len(txt) > 60 {
				txt = txt[:60]
			}
			classDebug = append(classDebug, fmt.Sprintf("class=%q text=%q", cls, txt))
		})
		applog.InfoLogSaveAppLog("FetchMonthlyReleases: found %d category divs: %v", len(classDebug), classDebug)

		// 调试: 输出页面中所有 class 包含 div_product 或 display 的 div
		prodDebug := []string{}
		doc.Find("div[class*='div_product'], div[class*='display']").Each(func(_ int, s *goquery.Selection) {
			cls, _ := s.Attr("class")
			html, _ := s.Html()
			if len(html) > 80 {
				html = html[:80]
			}
			prodDebug = append(prodDebug, fmt.Sprintf("class=%q html=%q", cls, html))
		})
		applog.InfoLogSaveAppLog("FetchMonthlyReleases: found %d product divs", len(prodDebug))

		// 找到所有包含发售日期的元素，然后收集其后续的游戏条目
		doc.Find("div.category_pc_t2").Each(func(_ int, dateDiv *goquery.Selection) {
			dateText := strings.TrimSpace(dateDiv.Text())
			if !strings.Contains(dateText, "発売タイトル") {
				return
			}

			matches := datePattern.FindStringSubmatch(dateText)
			if len(matches) < 4 {
				return
			}
			releaseDate := fmt.Sprintf("%s/%02s/%02s", matches[1], padZero(matches[2]), padZero(matches[3]))
			applog.InfoLogSaveAppLog("FetchMonthlyReleases: found release date: %s", releaseDate)

			// 从日期 div 的父元素开始，向后查找兄弟节点中的游戏条目
			var games []MonthlyReleaseGame
			parent := dateDiv.Parent()

			// 先检查父元素内部是否有 div_product（父子关系）
			parent.Find("div.div_product").Each(func(_ int, s *goquery.Selection) {
				game := extractGameFromProduct(s)
				if game != nil {
					games = append(games, *game)
				}
			})

			// 如果父元素内部没找到，向后遍历兄弟节点
			if len(games) == 0 {
				for next := parent.Next(); next.Length() > 0; next = next.Next() {
					// 遇到下一个日期块则停止
					if next.HasClass("category_pc_t") || next.Find("div.category_pc_t2").Length() > 0 {
						break
					}

					if next.HasClass("div_product") {
						game := extractGameFromProduct(next)
						if game != nil {
							games = append(games, *game)
						}
					} else {
						next.Find("div.div_product").Each(func(_ int, s *goquery.Selection) {
							game := extractGameFromProduct(s)
							if game != nil {
								games = append(games, *game)
							}
						})
					}
				}
			}

			if len(games) > 0 {
				groups = append(groups, MonthlyReleaseGroup{
					ReleaseDate: releaseDate,
					Games:       games,
				})
			}
		})
	})

	err := c.Visit(targetURL)
	if err != nil {
		return result, fmt.Errorf("failed to visit Getchu monthly page: %w", err)
	}

	c.Wait()

	result.Groups = groups
	applog.InfoLogSaveAppLog("FetchMonthlyReleases: total %d date groups found", len(groups))

	return result, nil
}

// extractGameFromProduct 从单个 div_product 元素中提取游戏信息
func extractGameFromProduct(s *goquery.Selection) *MonthlyReleaseGame {
	linkEl := s.Find("a[href*='soft.phtml?id=']").First()
	href := linkEl.AttrOr("href", "")
	getchuID := extractGetchuIDFromHref(href)
	if getchuID == "" {
		return nil
	}

	detailURL := "https://www.getchu.com" + strings.TrimPrefix(href, "https://www.getchu.com")

	imgEl := s.Find("img[src*='package']").First()
	imgSrc := imgEl.AttrOr("src", "")
	coverURL := buildFullCoverURL(imgSrc)

	titleLink := s.Find("a.black").First()
	gameName, company := parseGameNameAndCompany(titleLink)

	applog.InfoLogSaveAppLog("FetchMonthlyReleases: found game [%s] %s (%s)", getchuID, gameName, company)

	return &MonthlyReleaseGame{
		Name:      gameName,
		GetchuID:  getchuID,
		CoverURL:  coverURL,
		Company:   company,
		DetailURL: detailURL,
	}
}

// extractGetchuIDFromHref 从链接中提取 Getchu ID
// 示例: "/soft.phtml?id=1365262" -> "1365262"
func extractGetchuIDFromHref(href string) string {
	re := regexp.MustCompile(`id=(\d+)`)
	matches := re.FindStringSubmatch(href)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// buildFullCoverURL 将缩略图 URL 转换为完整大图 URL
// 输入: "./xxx_files/c1365262package_s.jpg" 或 "/brandnew/xxx/c1365262package_s.jpg"
// 输出: "https://www.getchu.com/brandnew/1365262/c1365262package.jpg"
func buildFullCoverURL(imgSrc string) string {
	if imgSrc == "" {
		return ""
	}

	// 提取 ID 部分，例如 "c1365262package_s.jpg" -> "1365262"
	re := regexp.MustCompile(`c(\d+)package`)
	matches := re.FindStringSubmatch(imgSrc)
	if len(matches) >= 2 {
		id := matches[1]
		return fmt.Sprintf("https://www.getchu.com/brandnew/%s/c%spackage.jpg", id, id)
	}

	// 回退：直接拼接
	if strings.HasPrefix(imgSrc, "/") {
		return "https://www.getchu.com" + imgSrc
	}
	return imgSrc
}

// parseGameNameAndCompany 从 a.black 元素中解析游戏名称和品牌
// HTML 结构: <a class="black">游戏名称<br>(品牌名)</a>
func parseGameNameAndCompany(sel *goquery.Selection) (string, string) {
	// 获取完整 HTML 以解析 <br> 分隔的内容
	html, err := sel.Html()
	if err != nil || html == "" {
		// 回退到纯文本
		text := strings.TrimSpace(sel.Text())
		return parseGameNameAndCompanyFromText(text)
	}

	// 按 <br> 或 <br/> 分割
	parts := regexp.MustCompile(`<br\s*/?>`).Split(html, -1)
	if len(parts) >= 2 {
		gameName := cleanHTMLText(parts[0])
		companyRaw := cleanHTMLText(parts[1])
		// 去除括号: "(品牌名)" -> "品牌名"
		company := strings.TrimPrefix(strings.TrimSuffix(companyRaw, ")"), "(")
		company = strings.TrimPrefix(strings.TrimSuffix(company, "）"), "（")
		return strings.TrimSpace(gameName), strings.TrimSpace(company)
	}

	// 回退
	text := strings.TrimSpace(sel.Text())
	return parseGameNameAndCompanyFromText(text)
}

// parseGameNameAndCompanyFromText 从纯文本中解析名称和品牌
// 示例: "わんたま☆らいふ ... (Rabbitfoot)"
func parseGameNameAndCompanyFromText(text string) (string, string) {
	// 尝试匹配末尾的 "(品牌名)"
	re := regexp.MustCompile(`^(.*?)\s*[（(](.+?)[）)]\s*$`)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 3 {
		return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
	}
	return strings.TrimSpace(text), ""
}

// cleanHTMLText 清理 HTML 片段中的标签和空白
func cleanHTMLText(s string) string {
	// 移除所有 HTML 标签
	re := regexp.MustCompile(`<[^>]+>`)
	cleaned := re.ReplaceAllString(s, "")
	// 移除多余的空白
	cleaned = strings.TrimSpace(cleaned)
	// 压缩连续空白
	spaceRe := regexp.MustCompile(`\s+`)
	cleaned = spaceRe.ReplaceAllString(cleaned, " ")
	return cleaned
}

// padZero 为月份/日期补零
func padZero(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}
