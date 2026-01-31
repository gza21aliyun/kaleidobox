package utils

import (
	"fmt"
	"log"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery" // 添加 GoQuery 导入
	"github.com/gocolly/colly/v2"    // 添加 Colly 导入
)

type EroscapeInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

func (b EroscapeInfoGetter) FetchMetadataByName(name string) (models.Game, error) {
	var mirror string = "https://koko.kyara.top/"
	// var base string = "https://koko.kyara.top/"
	var searchPart = "kensaku.php?category=game&word_category=name&mode=normal&word="
	var gamePart = "game.php?game="
	var url string = mirror + searchPart
	var gameUrl = mirror + gamePart
	var mirrorDomain = "*kyara.top"
	url += name
	var game = models.Game{}
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		colly.Async(),
	)

	// 设置请求超时
	c.SetRequestTimeout(30 * time.Second)

	// 设置限速
	c.Limit(&colly.LimitRule{
		DomainGlob:  mirrorDomain,
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

	var potentialGames []struct {
		Title  string
		Link   string
		GameId string
	}

	// 处理搜索结果页面中的游戏条目
	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {

		title := e.ChildText("td a.tooltip")
		idParts := strings.Split(strings.Split(e.ChildAttr("td a.tooltip", "href"), "#")[0], "=")
		gameId := idParts[len(idParts)-1]
		link := gameUrl + gameId

		if title != "" {
			log.Print("OnHTML 网页列表 ：", e.Text)
			log.Print("OnHTML 网页列表2 ：", e.ChildAttr("td a.tooltip", "innerHtml"))
			log.Print("网页3", idParts)
			log.Print("网页4", title)
			potentialGames = append(potentialGames, struct {
				Title  string
				Link   string
				GameId string
			}{
				Title:  title,
				Link:   e.Request.AbsoluteURL(link),
				GameId: gameId,
			})
		}

	})

	// 在访问完搜索页面后进行过滤和处理
	c.OnScraped(func(r *colly.Response) {
		for _, gameFound := range potentialGames {
			// 应用过滤条件
			if strings.Contains(gameFound.Title, "セット") {
				continue
			}
			if gameFound.Link == "" {
				continue
			}
			game.Name = gameFound.Title
			game.SourceID = gameFound.GameId
			c.Visit(gameFound.Link)
			return
		}
	})

	// 处理游戏详情页面
	c.OnHTML("div#main", func(e *colly.HTMLElement) {
		// 使用 GoQuery 进一步解析 HTML
		// log.Print("OnHTML 网页详情 ：", e.Text)
		// doc, err := goquery.NewDocumentFromReader(strings.NewReader(e.Text))
		// if err != nil {
		// 	fmt.Printf("Error creating goquery document: %v\n", err)
		// 	return
		// }

		// 提取游戏名称
		// gameName := e.ChildText("h1.page-title") // 尝试使用 Colly 提取
		// if gameName == "" {
		// 	// 使用 GoQuery 提取标题
		// 	doc.Find("h1.page-title").Each(func(i int, s *goquery.Selection) {
		// 		gameName = strings.TrimSpace(s.Text())
		// 	})
		// }
		// game.Name = gameName

		// 提取封面图片
		coverURL := e.ChildAttr("div#main_image a img", "src")
		// if coverURL == "" {
		// 	doc.Find("div.product-main-image img").Each(func(i int, s *goquery.Selection) {
		// 		coverURL, _ = s.Attr("src")
		// 	})
		// }
		game.CoverURL = coverURL

		// 提取公司信息
		company := e.ChildText("tr#brand a")
		// if company == "" {
		// 	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		// 		if strings.Contains(s.Text(), "メーカー") {
		// 			s.Find("td").Each(func(j int, td *goquery.Selection) {
		// 				if j == 1 { // 假设厂商在第二列
		// 					company = strings.TrimSpace(td.Text())
		// 				}
		// 			})
		// 		}
		// 	})
		// }
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
		// summary := e.ChildAttr("div.area-detail-read", "innerHTML")
		// if summary == "" {
		// 	doc.Find("div.product-introduction").Each(func(i int, s *goquery.Selection) {
		// 		summary = strings.TrimSpace(s.Text())
		// 	})
		// }
		game.Summary = summary

		// 提取标签
		var tags []string
		e.DOM.Find("table#att_pov_table a").Each(func(i int, s *goquery.Selection) {
			tag := strings.TrimSpace(s.Text())
			if !strings.Contains(tag, "還元") && !strings.Contains(tag, "クーポン") {
				tags = append(tags, tag)
			}

		})
		// e.DOM.C
		// log.Print("OnHTML 网页标签1 ：", e.ChildText("div.productLayout__secondaryColumn div.contentsDetailBottom__tableRow--container"))
		// log.Print("OnHTML 网页标签2 ：", doc.Find("div.productLayout__secondaryColumn div.contentsDetailBottom__tableRow--container").Text())

		// doc.Find("div.productLayout__secondaryColumn div.contentsDetailBottom__tableRow--container li.contentsDetailBottom__tableDataItem").Each(func(i int, s *goquery.Selection) {
		// 	tags = append(tags, strings.TrimSpace(s.Text()))
		// })
		game.Tags = strings.Join(tags, ",")
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

	// 检查是否成功获取了数据
	if game.Name == "" {
		return models.Game{}, fmt.Errorf("game not found: %s", name)
	}

	// 设置其他必要字段
	game.SourceType = enums.Eroscape // 假设你有这个枚举
	// game.SourceID = name
	game.CachedAt = time.Now()

	return game, nil
}
