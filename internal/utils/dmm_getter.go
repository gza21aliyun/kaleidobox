package utils

import (
	"encoding/json"
	"fmt"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery" // 添加 GoQuery 导入
	"github.com/gocolly/colly/v2"    // 添加 Colly 导入
	"github.com/labstack/gommon/log"
)

type DmmInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

type IdFunction func(request vo.MetadataRequest) (models.Game, error)

func (b DmmInfoGetter) FetchMetadataByName(name string, dmmIsEnabled bool) (models.Game, error) {
	game, err := b.FetchByNameImpl(name, dmmIsEnabled,
		func(request vo.MetadataRequest) (models.Game, error) {
			fmt.Println("FetchMetadataByName 34")
			gameEntity, err := b.FetchMetadataById(request)
			return gameEntity.Game, err
		})
	return game, err
}

func (b DmmInfoGetter) FetchByNameImpl(name string, dmmIsEnabled bool, fn IdFunction) (models.Game, error) {
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

		if !strings.Contains(title, "動画版") && !strings.Contains(title, "音楽") {
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
		}

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
			fmt.Println("dmm详情：" + gameFound.Link)
			linkParts := strings.Split(gameFound.Link, "/")
			id := linkParts[len(linkParts)-2]
			fmt.Println("05 id: " + id)
			game.SourceID = id
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
	game, err = fn(GetReqEntity(&game))

	return game, err
}

func (b DmmInfoGetter) FetchMetadataById(request vo.MetadataRequest) (models.GameEntity, error) {
	fmt.Println("开始获取DMM游戏信息 36 " + request.Source)
	var game models.Game = request.GetGame()
	var gameEntity models.GameEntity = models.GameEntity{}
	gameEntity.Game = game
	if request.ID == "" {
		return gameEntity, fmt.Errorf("DMM ID is required to fetch metadata by ID")
	}
	fmt.Println("开始获取DMM游戏信息 37 " + request.ID)

	dmmUrl := fmt.Sprintf("https://dlsoft.dmm.co.jp/detail/%s/", request.ID)
	c := CreateCollector("*dmm.co.jp")

	// 处理游戏详情页面
	c.OnHTML("div.pageLayout__contentWrapper", func(e *colly.HTMLElement) {
		name := e.ChildText("h1.productTitle__item--headline")
		fmt.Println("开始获取DMM游戏信息 41 " + name)
		game.Name = name

		// 提取公司信息
		company := e.ChildText("div.productLayout__secondaryColumn div.contentsDetailTop__tableRow:contains('ブランド') div.contentsDetailTop__tableDataRight a")

		game.Company = company

		genre := e.ChildText("div.productLayout__secondaryColumn div.contentsDetailBottom__tableRow:contains('ゲームジャンル') div.contentsDetailBottom__tableDataRight p")
		game.MetaTags = genre

		// 提取简介
		summary, _ := e.DOM.Find("div.area-detail-read").Html()
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
		// 获取图片
		var images []string
		e.DOM.Find("div.productLayout__primaryColumn div.slider-area li img").Each(func(i int, s *goquery.Selection) {
			image, _ := s.Attr("src")
			if image == "" {
				return
			} else if strings.Contains(image, "pl.jpg") {
				game.CoverURL = image
			}
			images = append(images, image)
		})
		game.Images = strings.Join(images, ",")
		var worksMap map[enums.StaffRole][]models.Work = make(map[enums.StaffRole][]models.Work)
		e.DOM.Find("div.contentsDetailBottom__table div.contentsDetailBottom__tableRow:contains('原画') a").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					fmt.Println("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("author")
				sId := "author=" + staffId
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.CharaDesign,
					StaffName:     staffName,
					SourceStaffId: sId,
					SourceType:    enums.Dmm,
					GameName:      game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		e.DOM.Find("div.contentsDetailBottom__table div.contentsDetailBottom__tableRow:contains('シナリオ') a").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					fmt.Println("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("scenario")
				sId := "scenario=" + staffId
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.Sceneario,
					StaffName:     staffName,
					SourceStaffId: sId,
					SourceType:    enums.Dmm,
					GameName:      game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		e.DOM.Find("div.contentsDetailBottom__table div.contentsDetailBottom__tableRow:contains('声優') a").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					fmt.Println("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("voice_actor")
				sId := "voice_actor=" + staffId
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.CV,
					StaffName:     staffName,
					SourceStaffId: sId,
					SourceType:    enums.Dmm,
					GameName:      game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})

		e.DOM.Find("div.detailGuide__sect div.detailGuide__box-chr").Each(func(i int, s *goquery.Selection) {

			charactorName := strings.TrimSpace(s.Find("span.detailGuide__lin-hgt").Text())
			image := strings.TrimSpace(s.Find("img").AttrOr("src", ""))
			summary, _ := s.Find("p").Eq(1).Html()

			summary = strings.TrimSpace(summary)

			if charactorName != "" {

				work := models.Work{
					GameId:        game.ID,
					Role:          enums.Charactor,
					CharactorName: charactorName,
					WorkSummary:   summary,
					Images:        image,
					SourceType:    enums.Dmm,
					GameName:      game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		jstr, _ := json.Marshal(worksMap)
		log.Printf("worksMap: " + string(jstr))
		gameEntity.WorksMap = worksMap

	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err := c.Visit(dmmUrl)
	if err != nil {
		fmt.Println("开始获取DMM游戏信息 40 " + game.CoverURL)
		return gameEntity, err
	}

	// 等待收集完成
	c.Wait()
	fmt.Println("开始获取DMM游戏信息 39 " + game.CoverURL)

	// 检查是否成功获取了数据
	if game.Name == "" {
		fmt.Println("开始获取DMM游戏信息 40 " + game.CoverURL)
		return gameEntity, fmt.Errorf("game not found: %s", game.SourceID)
	}

	// 设置其他必要字段
	game.SourceType = enums.Dmm // 假设你有这个枚举
	// game.SourceID = name
	game.CachedAt = time.Now()
	gameEntity.Game = game
	fmt.Println("开始获取DMM游戏信息 38 " + game.CoverURL)

	return gameEntity, nil
}
