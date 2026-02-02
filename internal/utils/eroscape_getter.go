package utils

import (
	"fmt"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery" // 添加 GoQuery 导入
	"github.com/gocolly/colly/v2"    // 添加 Colly 导入
)

type EroscapeInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

func CreateCollector(domain string) *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		colly.Async(),
	)

	// 设置请求超时
	c.SetRequestTimeout(30 * time.Second)

	// 设置限速
	c.Limit(&colly.LimitRule{
		DomainGlob:  domain,
		Parallelism: 1,               // 限制并发数
		Delay:       2 * time.Second, // 延迟请求
	})

	// 首先，设置请求前的处理
	c.OnRequest(func(r *colly.Request) {

		fmt.Println("Visiting", r.URL.String()) // 打印正在访问的 URL
		cookie3 := &http.Cookie{Name: "age_check_done", Value: "1"}

		r.Headers.Set("Cookie", cookie3.Name+"="+cookie3.Value)

		// 设置额外的请求头
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "ja-JP,ja;q=0.8,en-US;q=0.5,en;q=0.3")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
	})
	return c
}

func (b EroscapeInfoGetter) FetchMetadataByName(name string, isEnabled bool, useMirror bool) (models.Game, error) {
	if !isEnabled { // 禁用的话，就返回一个空游戏
		return models.Game{}, nil
	}

	var mirror string = "https://koko.kyara.top/"
	var original string = "https://erogamescape.dyndns.org/"
	var baseUrl string
	if useMirror {
		baseUrl = mirror
	} else {
		baseUrl = original
	}
	var searchPart = "kensaku.php?category=game&word_category=name&mode=normal&word="
	// var gamePart = "game.php?game="
	var url string = baseUrl + searchPart
	// var gameUrl = baseUrl + gamePart
	var mirrorDomain = "*kyara.top"
	var baseDomain = "*dyndns.org"
	var domain string
	if useMirror {
		domain = mirrorDomain
	} else {
		domain = baseDomain
	}
	url += name
	var game = models.Game{}
	c := CreateCollector(domain)

	var potentialGames []struct {
		Title string
		// Link   string
		GameId string
	}

	// 处理搜索结果页面中的游戏条目
	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {

		title := e.ChildText("td a.tooltip")
		href := e.ChildAttr("td a.tooltip", "href")
		idParts := strings.Split(strings.Split(href, "#")[0], "=")
		gameId := idParts[len(idParts)-1]

		// link := gameUrl + gameId

		if title != "" {
			fmt.Println("title:", title)
			// fmt.Println("href", href)
			// fmt.Println("idParts:", idParts)
			// fmt.Println("gameId:", gameId)

			potentialGames = append(potentialGames, struct {
				Title string
				// Link   string
				GameId string
			}{
				Title: title,
				// Link:   e.Request.AbsoluteURL(link),
				GameId: gameId,
			})
		}

	})

	// 在访问完搜索页面后进行过滤和处理
	c.OnScraped(func(r *colly.Response) {
		sort.Slice(potentialGames, func(i, j int) bool {
			return len(potentialGames[i].Title) < len(potentialGames[j].Title)
		})
		for _, gameFound := range potentialGames {
			// 应用过滤条件
			if strings.Contains(gameFound.Title, "セット") {
				continue
			}
			// if gameFound.Link == "" {
			// 	continue
			// }
			game.Name = gameFound.Title
			game.SourceID = gameFound.GameId
			game.EroscapeId = gameFound.GameId
			// c.Visit(gameFound.Link)
			return
		}
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err := c.Visit(url)
	if err != nil {
		return models.Game{}, err
	}

	// 等待收集完成
	c.Wait()
	game, _ = b.FetchMetadataById(game, useMirror)

	return game, nil
}

func (b EroscapeInfoGetter) FetchMetadataById(game models.Game, useMirror bool) (models.Game, error) {
	var mirror string = "https://koko.kyara.top/"
	var original string = "https://erogamescape.dyndns.org/"
	var baseUrl string
	if useMirror {
		baseUrl = mirror
	} else {
		baseUrl = original
	}
	var gamePart = "game.php?game="
	var gameUrl = baseUrl + gamePart + game.EroscapeId
	var mirrorDomain = "*kyara.top"
	var baseDomain = "*dyndns.org"
	var domain string
	if useMirror {
		domain = mirrorDomain
	} else {
		domain = baseDomain
	}
	c := CreateCollector(domain)
	c.OnHTML("div#main", func(e *colly.HTMLElement) {

		// 提取封面图片
		coverURL := e.ChildAttr("div#main_image a img", "src")
		game.CoverURL = coverURL

		// 提取公司信息
		company := e.ChildText("tr#brand a")
		game.Company = company

		genre := ""
		e.DOM.Find("table#att_pov_table tr:contains('ジャンル') a").Each(func(i int, s *goquery.Selection) {
			g := strings.TrimSpace(s.Text())
			if i == 0 {
				genre += g
			} else {
				genre += "," + g
			}

		})
		game.MetaTags = genre

		// 提取简介
		summary := e.ChildText("div.area-detail-read")
		game.Summary = summary

		// 提取标签
		var tags []string
		e.DOM.Find("table#att_pov_table a").Each(func(i int, s *goquery.Selection) {
			tag := strings.TrimSpace(s.Text())
			if !strings.Contains(tag, "還元") && !strings.Contains(tag, "クーポン") {
				tags = append(tags, tag)
			}

		})
		game.Tags = strings.Join(tags, ",")
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err := c.Visit(gameUrl)
	if err != nil {
		return models.Game{}, err
	}

	// 等待收集完成
	c.Wait()

	// 检查是否成功获取了数据
	if game.Name == "" {
		return models.Game{}, fmt.Errorf("game not found: %s", game.Name)
	}

	// 设置其他必要字段
	game.SourceType = enums.Eroscape // 假设你有这个枚举
	game.CachedAt = time.Now()

	return game, nil
}
