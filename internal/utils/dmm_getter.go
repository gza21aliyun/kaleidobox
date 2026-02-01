package utils

import (
	"fmt"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery" // 添加 GoQuery 导入
	"github.com/gocolly/colly/v2"    // 添加 Colly 导入
)

type DmmInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

func (b DmmInfoGetter) FetchMetadataByName(name string, dmmIsEnabled bool) (models.Game, error) {
	if !dmmIsEnabled {
		return models.Game{}, fmt.Errorf("DMM is not enabled")
	}
	var url string = "https://dlsoft.dmm.co.jp/search/?service=pcgame&searchstr="
	url += name
	var game = models.Game{}
	c := CreateCollector("*dmm.co.jp")

	var potentialGames []struct {
		Title    string
		Link     string
		Review   string
		CoverUrl string
	}

	// 处理搜索结果页面中的游戏条目
	c.OnHTML("li.component-legacy-productTile__item", func(e *colly.HTMLElement) {
		title := e.ChildText(".component-legacy-productTile__title")
		link := e.ChildAttr("a.component-legacy-productTile__detailLink", "href")
		price := e.ChildText(".component-legacy-productTile__review")
		// log.Print("OnHTML 网页列表 ：", e.Text)

		potentialGames = append(potentialGames, struct {
			Title    string
			Link     string
			Review   string
			CoverUrl string
		}{
			Title:    title,
			Link:     e.Request.AbsoluteURL(link),
			Review:   price,
			CoverUrl: e.ChildAttr("span.component-legacy-productTile__thumbnail img", "src"),
		})
	})

	// 在访问完搜索页面后进行过滤和处理
	c.OnScraped(func(r *colly.Response) {
		for _, gameFound := range potentialGames {
			// 应用过滤条件
			if gameFound.Review == "" {
				continue
			}
			if strings.Contains(gameFound.Title, "セット") {
				continue
			}
			if gameFound.Link == "" {
				continue
			}
			game.Name = gameFound.Title
			linkParts := strings.Split(gameFound.Link, "/")
			game.SourceID = linkParts[len(linkParts)-2]
			game.DmmId = game.SourceID
			game.CoverURL = gameFound.CoverUrl
			// c.Visit(gameFound.Link) // 不在这里访问详情页
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

	game, _ = b.FetchMetadataById(game)

	return game, nil
}

func (b DmmInfoGetter) FetchMetadataById(game models.Game) (models.Game, error) {
	if game.DmmId == "" {
		return game, fmt.Errorf("DMM ID is required to fetch metadata by ID")
	}

	url := fmt.Sprintf("https://dlsoft.dmm.co.jp/detail/%s/", game.DmmId)
	c := CreateCollector("*dmm.co.jp")

	// 处理游戏详情页面
	c.OnHTML("div.pageLayout__contentWrapper", func(e *colly.HTMLElement) {

		// 提取公司信息
		company := e.ChildText("div.productLayout__secondaryColumn div.contentsDetailTop__tableRow:contains('ブランド') div.contentsDetailTop__tableDataRight a")

		game.Company = company

		genre := e.ChildText("div.productLayout__secondaryColumn div.contentsDetailBottom__tableRow:contains('ゲームジャンル') div.contentsDetailBottom__tableDataRight p")
		game.MetaTags = genre

		// 提取简介
		summary := e.ChildText("div.area-detail-read")
		game.Summary = summary

		// 提取标签
		var tags []string
		e.DOM.Find("div.productLayout__secondaryColumn div.contentsDetailBottom__tableRow--container li").Each(func(i int, s *goquery.Selection) {
			tag := strings.TrimSpace(s.Text())
			if !strings.Contains(tag, "還元") && !strings.Contains(tag, "クーポン") {
				tags = append(tags, tag)
			}

		})
		game.Tags = strings.Join(tags, ",")
		var images []string
		e.DOM.Find("div.productLayout__primaryColumn div.slider-area li img").Each(func(i int, s *goquery.Selection) {
			image, _ := s.Attr("src")
			images = append(images, image)
		})
		game.Images = strings.Join(images, ",")
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
		return models.Game{}, fmt.Errorf("game not found: %s", game.SourceID)
	}

	// 设置其他必要字段
	game.SourceType = enums.Dmm // 假设你有这个枚举
	// game.SourceID = name
	game.CachedAt = time.Now()

	return game, nil
}
