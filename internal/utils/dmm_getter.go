package utils

import (
	"encoding/json"
	"fmt"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery" // 添加 GoQuery 导入
	"github.com/gocolly/colly/v2"    // 添加 Colly 导入
)

type DmmInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

type IdFunction func(request vo.MetadataRequest) (models.Game, error)

// DMMSearchResponse DMM 搜索结果响应
type DMMSearchResponse struct {
	Error interface{}           `json:"error"`
	Body  DMMSearchResponseBody `json:"body"`
}

var _ Getter = (*DmmInfoGetter)(nil)
var _ Reviewer = (*DmmInfoGetter)(nil)

// DMMSearchResponseBody 响应体
type DMMSearchResponseBody struct {
	ProductArray []DMMProductArray `json:"productArray"`
}

// DMMProductArray 产品数组
type DMMProductArray struct {
	CardProduct DMMCardProduct `json:"cardProduct"`
}

// DMMCardProduct 卡片产品信息
type DMMCardProduct struct {
	Title               string      `json:"title"`
	ContentID           string      `json:"contentId"`
	DetailURL           string      `json:"detailUrl"`
	PackageImageURL     string      `json:"packageImageUrl"`
	ThumbnailImageURLPs string      `json:"thumbnailImageUrlPs"`
	ThumbnailImageURLPl string      `json:"thumbnailImageUrlPl"`
	BrandName           string      `json:"brandName"`
	BrandURL            string      `json:"brandUrl"`
	OriginalPrice       string      `json:"originalPrice"`
	SellingPrice        string      `json:"sellingPrice"`
	IsSellingPriceMore  bool        `json:"isSellingPriceMore"`
	DiscountRate        interface{} `json:"discountRate"`
	IsDifferentDiscount bool        `json:"isDifferentDiscount"`
	DiscountEndDate     interface{} `json:"discountEndDate"`
	ReturnPoint         string      `json:"returnPoint"`
	ReturnPointRate     string      `json:"returnPointRate"`
	// ContentCharacteristicArray []ContentCharacteristic `json:"contentCharacteristicArray"`
	IsCouponTargetProduct bool   `json:"isCouponTargetProduct"`
	IsBulkProduct         bool   `json:"isBulkProduct"`
	IsReserve             bool   `json:"isReserve"`
	PriorityProductID     string `json:"priorityProductId"`
	FloorProductID        string `json:"floorProductId"`
}

// DMMReviewResponse DMM 评论响应
type DMMReviewResponse struct {
	Error *string               `json:"error"`
	Body  DMMReviewResponseBody `json:"body"`
}

// DMMReviewResponseBody 评论响应体
type DMMReviewResponseBody struct {
	Page       int         `json:"page"`
	Sort       string      `json:"sort"`
	MaxPage    int         `json:"maxPage"`
	Limit      int         `json:"limit"`
	ReviewList []DMMReview `json:"reviewList"`
}

// DMMReview 评论信息
type DMMReview struct {
	ReviewId         string `json:"reviewId"`
	Title            string `json:"title"`
	Comment          string `json:"comment"`
	IsExposure       bool   `json:"isExposure"`
	DisplayShopName  string `json:"displayShopName"`
	Score            int    `json:"score"`
	ReviewerName     string `json:"reviewerName"`
	ReviewerUrl      string `json:"reviewerUrl"`
	ReleaseDate      string `json:"releaseDate"`
	IsPurchased      bool   `json:"isPurchased"`
	PublicReviewerId string `json:"publicReviewerId"`
	EvaluateCount    int    `json:"evaluateCount"`
	YesCount         int    `json:"yesCount"`
}

func (b DmmInfoGetter) FetchMetadataByName(name string, totken string) (models.Game, error) {
	return b.FetchMetadataByName2(name)
}

func (b DmmInfoGetter) FetchMetadataByName2(name string) (models.Game, error) {
	game, err := b.FetchByNameImpl(name, false,
		func(request vo.MetadataRequest) (models.Game, error) {
			fmt.Println("FetchMetadataByName 34")
			gameEntity, err := b.FetchMetadataById(request)
			return gameEntity.Game, err
		})
	if game.SourceID != "" {
		game, err = b.FetchByNameImpl(name, true,
			func(request vo.MetadataRequest) (models.Game, error) {
				fmt.Println("FetchMetadataByName 34")
				gameEntity, err := b.FetchMetadataById(request)
				return gameEntity.Game, err
			})
	}
	return game, err
}

func (b DmmInfoGetter) FetchByNameImpl(name string, isAl bool, fn IdFunction) (models.Game, error) {
	var url string = "https://dlsoft.dmm.co.jp/search/?service=pcgame&searchstr="
	mainTitle, _, _ := getTitles(name)
	if isAl {
		mainTitle = getGameNameAlternative(mainTitle)
	}
	url += mainTitle
	// url += name
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

	})

	// 在访问完搜索页面后进行过滤和处理
	c.OnScraped(func(r *colly.Response) {
		gameFound := searchNameByRegex(potentialGames, name, []string{"セット"}, func(t1 struct {
			Title    string
			Link     string
			Review   string
			CoverUrl string
		}) string {
			return t1.Title
		})
		if gameFound != nil {
			linkParts := strings.Split(gameFound.Link, "/")
			id := linkParts[len(linkParts)-2]
			game.Name = gameFound.Title
			game.SourceID = id
			game.SourceType = enums.Dmm
			game.EroscapeId = id
		}
		// for _, gameFound := range potentialGames {
		// 	// 应用过滤条件
		// 	if gameFound.Review == "" {
		// 		continue
		// 	}
		// 	if strings.Contains(gameFound.Title, "セット") {
		// 		continue
		// 	}
		// 	if gameFound.Link == "" {
		// 		continue
		// 	}
		// 	game.Name = gameFound.Title
		// 	fmt.Println("dmm详情：" + gameFound.Link)
		// 	linkParts := strings.Split(gameFound.Link, "/")
		// 	id := linkParts[len(linkParts)-2]
		// 	fmt.Println("05 id: " + id)
		// 	game.SourceID = id
		// 	game.DmmId = game.SourceID
		// 	game.CoverURL = gameFound.CoverUrl
		// 	// c.Visit(gameFound.Link) // 不在这里访问详情页
		// 	return
		// }
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

func (b DmmInfoGetter) FetchMetadata(id string, token string) (models.Game, error) {
	gameEntity, err := b.FetchMetadataById(vo.MetadataRequest{ID: id})
	return gameEntity.Game, err
}

func (b DmmInfoGetter) FetchMetadataById(request vo.MetadataRequest) (models.GameEntity, error) {
	fmt.Println("开始获取DMM游戏信息 36 " + request.Source)
	var game models.Game = request.GetGame()
	var gameEntity models.GameEntity = models.GameEntity{}
	gameEntity.Game = game
	var err error = nil
	if request.ID == "" {
		return gameEntity, fmt.Errorf("DMM ID is required to fetch metadata by ID")
	}
	fmt.Println("开始获取DMM游戏信息 37 " + request.ID)
	game.DmmId = request.ID
	dmmUrl := fmt.Sprintf("https://dlsoft.dmm.co.jp/detail/%s/", request.ID)
	c := CreateCollector("*dmm.co.jp")

	// 处理游戏详情页面
	c.OnHTML("div.pageLayout__contentWrapper", func(e *colly.HTMLElement) {
		time.Sleep(time.Second * 1)
		name := e.ChildText("h1.productTitle__item--headline")
		fmt.Println("开始获取DMM游戏信息 41 " + name)
		game.Name = name

		// fmt.Printf("alltext: %s\n", e.Text)

		// 提取公司信息
		company := e.ChildText("div.productLayout__secondaryColumn div.contentsDetailTop__tableRow:contains('ブランド') div.contentsDetailTop__tableDataRight a")
		// companyLink := e.ChildAttr("div.productLayout__secondaryColumn div.contentsDetailTop__tableRow:contains('ブランド') div.contentsDetailTop__tableDataRight a", "href")

		// companyId := strings.ReplaceAll(strings.ReplaceAll(companyLink, "https://dlsoft.dmm.co.jp/list/?maker=", ""), "&sort=ranking", "")
		// relatedStr, err := b.GetRelatedGames(companyId)
		// game.RelatedGames = relatedStr

		game.Company = company

		tagList := []models.Tag{}
		tagList = append(tagList, models.Tag{Name: company, Category: models.TagCategoryBrand, BlockModify: true})

		genre := e.ChildText("div.productLayout__secondaryColumn div.contentsDetailBottom__tableRow:contains('ゲームジャンル') div.contentsDetailBottom__tableDataRight p")
		// game.Arguments = genre
		tagList = append(tagList, models.Tag{Name: genre, Category: models.TagCategoryGenre, BlockModify: true})

		// 提取简介
		summary, _ := e.DOM.Find("p.text-overflow").Html()
		summary = strings.ReplaceAll(summary, "<br/>", "\n")
		game.Summary = summary
		time.Sleep(time.Millisecond * 500)

		// 提取标签
		// var tags []string
		e.DOM.Find("div.productLayout__secondaryColumn div.contentsDetailBottom__tableRow--container li").Each(func(i int, s *goquery.Selection) {
			tag := strings.TrimSpace(s.Text())
			if !strings.Contains(tag, "還元") && !strings.Contains(tag, "クーポン") && !strings.Contains(tag, "セール") &&
				!strings.Contains(tag, "独占販売") {
				// tags = append(tags, tag)
				tagList = append(tagList, models.Tag{Name: tag, Category: models.TagCategoryOther})
			}

		})

		game.Tags = JoinString(tagList, ",", func(tag models.Tag) string { return tag.Name })
		gameEntity.Tags = ArrayToMap(tagList, func(t1 models.Tag) string { return t1.Category })
		// 获取图片
		var images []string
		e.DOM.Find("div.productLayout__primaryColumn div.slider-area li img").Each(func(i int, s *goquery.Selection) {
			image := s.AttrOr("src", "")
			// image, _ := s.Html()
			fmt.Println("开始获取DMM游戏信息 52 " + image)
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
					Sort:          i + 50,
					GameName:      game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})

		// e.DOM.Find("div.detailGuide__sect div.detailGuide__box-chr").Each(func(i int, s *goquery.Selection) {

		// 	charactorName := strings.ReplaceAll(strings.TrimSpace(s.Find("span.guide-lin-hgt").Text()), " ", "")
		// 	fmt.Printf("char 06 %s\n", charactorName)
		// 	parts := stringCharSplit((charactorName), "(（")
		// 	if len(parts) > 0 {
		// 		charactorName = parts[0]
		// 	}
		// 	boxText := strings.TrimSpace(s.Find("p").Eq(0).Text())
		// 	fmt.Printf("boxtext 022:%s\n", boxText)
		// 	lines := strings.Split(boxText, "\n")
		// 	line1 := strings.TrimSpace(lines[0])
		// 	height := ""
		// 	measurements := ""
		// 	if len(lines) > 1 {
		// 		line2 := strings.TrimSpace(lines[len(lines)-1])
		// 		line1 = strings.TrimSpace(lines[len(lines)-2])
		// 		line2Parts := strings.Split(line2, "スリーサイズ：")
		// 		if len(line2Parts) > 1 {
		// 			height = strings.TrimSpace(strings.ReplaceAll(line2Parts[0], "身長：", ""))
		// 			measurements = strings.TrimSpace(line2Parts[1])
		// 			measurements = strings.TrimSpace(strings.Split(measurements, "特技")[0])
		// 		}
		// 	}
		// 	staffBox := strings.Split(line1, "CV：")
		// 	var staffName string = ""
		// 	if len(staffBox) > 1 {
		// 		staffName = strings.TrimSpace(staffBox[1])
		// 	}

		// 	newWork := Find(gameEntity.WorksMap[enums.CV], func(it models.Work) bool {
		// 		fmt.Printf("boxtext 03:%s\n", it.StaffName)
		// 		return it.StaffName == staffName
		// 	})
		// 	image := strings.TrimSpace(s.Find("img").AttrOr("src", ""))
		// 	summary, _ := s.Find("p").Eq(1).Html()

		// 	summary = strings.ReplaceAll(strings.TrimSpace(summary), "<br/>", "\n")
		// 	summary = strings.ReplaceAll(strings.TrimSpace(summary), "\n", "\n")

		// 	if charactorName != "" {
		// 		if newWork != nil {
		// 			fmt.Printf("boxtext 01:%s\n", staffName)
		// 			newWork.CharactorName = charactorName
		// 			newWork.WorkSummary = summary
		// 			newWork.CharactorImage = image
		// 			newWork.Sort = i

		// 		} else {
		// 			fmt.Printf("boxtext 02:%s\n", staffName)
		// 			work := models.Work{
		// 				GameId:         game.ID,
		// 				Role:           enums.Charactor,
		// 				CharactorName:  charactorName,
		// 				StaffName:      staffName,
		// 				WorkSummary:    summary,
		// 				CharactorImage: image,
		// 				SourceType:     enums.Dmm,
		// 				Sort:           i,
		// 				GameName:       game.Name,
		// 				Measurements:   measurements,
		// 				Height:         height,
		// 			}
		// 			worksMap[work.Role] = append(worksMap[work.Role], work)
		// 		}

		// 	}
		// })

		// e.DOM.Find("div.guide-sect div.guide-box-chr").Each(func(i int, s *goquery.Selection) {
		// 	hml, _ := s.Html()
		// 	fmt.Printf("char 05 %s\n", hml)
		// 	charactorName := strings.ReplaceAll(strings.TrimSpace(s.Find("span.guide-lin-hgt").Text()), " ", "")
		// 	fmt.Printf("char 06 %s\n", charactorName)
		// 	parts := stringCharSplit((charactorName), "(（")
		// 	if len(parts) > 0 {
		// 		charactorName = parts[0]
		// 	}
		// 	fmt.Printf("char 07 %s\n", charactorName)

		// 	boxText := strings.TrimSpace(s.Find("p").Eq(0).Text())
		// 	fmt.Printf("boxtext 022:%s\n", boxText)
		// 	lines := strings.Split(boxText, "\n")
		// 	line1 := strings.TrimSpace(lines[0])
		// 	height := ""
		// 	measurements := ""
		// 	if len(lines) > 1 {
		// 		line1 = strings.TrimSpace(lines[len(lines)-2])
		// 		line2 := strings.TrimSpace(lines[len(lines)-1])
		// 		line2Parts := strings.Split(line2, "スリーサイズ：")
		// 		if len(line2Parts) > 1 {
		// 			height = strings.TrimSpace(strings.ReplaceAll(line2Parts[0], "身長：", ""))
		// 			measurements = strings.TrimSpace(line2Parts[1])
		// 			measurements = strings.TrimSpace(strings.Split(measurements, "特技")[0])
		// 			// if strings.Contains(measurements, "特技") {
		// 			// }
		// 		}
		// 	}
		// 	fmt.Printf("measurements 02:%s\n", measurements)
		// 	staffBox := strings.Split(line1, "CV：")
		// 	var staffName string = ""
		// 	if len(staffBox) > 1 {
		// 		staffName = strings.TrimSpace(staffBox[1])
		// 	}

		// 	newWork := Find(gameEntity.WorksMap[enums.CV], func(it models.Work) bool {
		// 		fmt.Printf("boxtext 03:%s\n", it.StaffName)
		// 		return it.StaffName == staffName
		// 	})
		// 	image := strings.TrimSpace(s.Find("img").AttrOr("src", ""))
		// 	summary, _ := s.Find("p").Eq(1).Html()

		// 	summary = strings.ReplaceAll(strings.TrimSpace(summary), "<br/>", "\n")
		// 	summary = strings.ReplaceAll(strings.TrimSpace(summary), "\n\n", "\n")

		// 	if charactorName != "" {
		// 		if newWork != nil {
		// 			fmt.Printf("boxtext 01:%s\n", staffName)
		// 			newWork.CharactorName = charactorName
		// 			newWork.WorkSummary = summary
		// 			newWork.CharactorImage = image
		// 			newWork.Sort = i

		// 		} else {
		// 			fmt.Printf("boxtext 02:%s\n", staffName)
		// 			work := models.Work{
		// 				GameId:         game.ID,
		// 				Role:           enums.Charactor,
		// 				CharactorName:  charactorName,
		// 				StaffName:      staffName,
		// 				WorkSummary:    summary,
		// 				CharactorImage: image,
		// 				SourceType:     enums.Dmm,
		// 				Sort:           i,
		// 				GameName:       game.Name,
		// 				Measurements:   measurements,
		// 				Height:         height,
		// 			}
		// 			worksMap[work.Role] = append(worksMap[work.Role], work)
		// 		}

		// 	}
		// })

		e.DOM.Find("div.detailGuide__sect div.detailGuide__box-chr").Each(func(i int, s *goquery.Selection) {
			work := extractCharactor(i, s, &gameEntity, game)
			if work != nil {
				worksMap[work.Role] = append(worksMap[work.Role], *work)
			}
		})

		e.DOM.Find("div.guide-sect div.guide-box-chr").Each(func(i int, s *goquery.Selection) {
			work := extractCharactor(i, s, &gameEntity, game)
			if work != nil {
				worksMap[work.Role] = append(worksMap[work.Role], *work)
			}
		})

		e.DOM.Find("div.guide-guide-sect div.guide-guide-box-chr").Each(func(i int, s *goquery.Selection) {
			work := extractCharactor(i, s, &gameEntity, game)
			if work != nil {
				worksMap[work.Role] = append(worksMap[work.Role], *work)
			}
		})
		gameEntity.WorksMap = worksMap

		releaseAt := strings.TrimSpace(e.DOM.Find("div.item-info__release-date__content__date").First().Text()) + ":00"
		game.ReleaseAt, err = time.Parse("2006/01/02 15:04:05", releaseAt)
		fmt.Printf("发售日11：%v, releaseat:%s, err:%v\n", game.ReleaseAt, releaseAt, err)
		if err != nil {
			return
		}

		// jstr, _ := json.Marshal(worksMap)
		// log.Printf("worksMap: " + string(jstr))

	})

	// 在访问完搜索页面后进行过滤和处理
	// c.OnScraped(func(r *colly.Response) {
	// 	// r.Body 包含原始 HTML 内容
	// 	// 使用 goquery 解析 HTML
	// 	// fmt.Println("访问完成 %s\n", string(r.Body))
	// 	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(r.Body)))
	// 	if err != nil {
	// 		fmt.Printf("解析 HTML 失败：%v\n", err)
	// 		return
	// 	}
	// 	doc.Find("div.universalSection a").Each(func(i int, s *goquery.Selection) {
	// 		fmt.Printf("related:%s\n", s.Text())
	// 	})

	// })

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err = c.Visit(dmmUrl)
	if err != nil {
		fmt.Printf("开始获取DMM游戏信息 40 err: %s\n", err)
		return gameEntity, err
	}

	// 等待收集完成
	c.Wait()
	fmt.Println("开始获取DMM游戏信息 39 " + game.CoverURL)

	// fmt.Printf("related games:%s\n", game.RelatedGames)

	// 检查是否成功获取了数据
	if game.Name == "" {
		fmt.Println("开始获取DMM游戏信息 40 " + game.CoverURL)
		return gameEntity, fmt.Errorf("game not found: %s", game.SourceID)
	}
	gameEntity = combineCharacters(gameEntity)
	fmt.Print("角色人数：", len(gameEntity.WorksMap[enums.Charactor]))

	// 设置其他必要字段
	game.SourceType = enums.Dmm // 假设你有这个枚举
	// game.SourceID = name
	game.CachedAt = time.Now()
	gameEntity.Game = game
	fmt.Println("开始获取DMM游戏信息 38 " + game.CoverURL)

	return gameEntity, nil
}

func extractCharactor(i int, s *goquery.Selection, gameEntity *models.GameEntity, game models.Game) *models.Work {
	hml, _ := s.Html()
	fmt.Printf("char 05 %s\n", hml)

	boxText := strings.TrimSpace(s.Find("p").Eq(0).Text())
	fmt.Printf("boxtext 022:%s\n", boxText)
	lines := strings.Split(boxText, "\n")

	line1Index := 0
	line1 := strings.TrimSpace(lines[line1Index])
	fmt.Printf("line1:%s\n", line1)
	height := ""
	measurements := ""
	if len(lines) > 1 {

		line1Index = len(lines) - 2
		line1 = strings.TrimSpace(lines[line1Index])

		fmt.Printf("line1 1:%s\n", line1)
		line2 := strings.TrimSpace(lines[len(lines)-1])
		line2Parts := strings.Split(line2, "スリーサイズ：")
		if len(line2Parts) > 1 {
			height = strings.TrimSpace(strings.ReplaceAll(line2Parts[0], "身長：", ""))
			measurements = strings.TrimSpace(line2Parts[1])
			measurements = strings.TrimSpace(strings.Split(measurements, "特技")[0])
			measurements = strings.TrimSpace(strings.Split(measurements, "血液型")[0])
			// if strings.Contains(measurements, "特技") {
			// }
		} else if len(line2Parts) == 1 {
			measurements = strings.TrimSpace(line2Parts[0])
			measurements = strings.TrimSpace(strings.Split(measurements, "特技")[0])
			measurements = strings.TrimSpace(strings.Split(measurements, "血液型")[0])
		}
	}
	fmt.Printf("measurements 02:%s\n", measurements)
	charactorName := ""
	var staffBox []string = []string{}
	if strings.Contains(line1, "CV：") {
		staffBox = strings.Split(line1, "CV：")
	} else if strings.Contains(line1, "CV:") {
		staffBox = strings.Split(line1, "CV:")
	} else {
		charactorName = strings.TrimSpace(line1)
	}
	var staffName string = ""

	if len(staffBox) > 1 {
		staffName = strings.TrimSpace(staffBox[1])
		charactorName = strings.ReplaceAll(strings.TrimSpace(staffBox[0]), " ", "")
	} else if len(staffBox) == 1 {
		staffName = strings.TrimSpace(staffBox[0])
		staffName = strings.ReplaceAll(staffName, "CV:", "")
	}
	if charactorName == "" && line1Index > 0 {
		charactorName = strings.ReplaceAll(strings.TrimSpace(lines[line1Index-1]), " ", "")

	}

	fmt.Printf("char 06 %s\n", charactorName)
	parts := stringCharSplit((charactorName), "(（")
	if len(parts) > 0 {
		charactorName = parts[0]
	}
	fmt.Printf("char 07 %s\n", charactorName)
	fmt.Printf("char 08 %s\n", staffName)

	newWork := Find(gameEntity.WorksMap[enums.CV], func(it models.Work) bool {
		fmt.Printf("boxtext 03:%s\n", it.StaffName)
		return it.StaffName == staffName
	})
	image := strings.TrimSpace(s.Find("img").AttrOr("src", ""))
	summary, _ := s.Find("p").Eq(1).Html()

	summary = strings.ReplaceAll(strings.TrimSpace(summary), "<br/>", "\n")
	summary = strings.ReplaceAll(strings.TrimSpace(summary), "\n\n", "\n")

	if charactorName != "" {
		if newWork != nil {
			fmt.Printf("boxtext 01:%s\n", staffName)
			newWork.CharactorName = charactorName
			newWork.WorkSummary = summary
			newWork.CharactorImage = image
			newWork.Sort = i

		} else {
			fmt.Printf("boxtext 02:%s\n", staffName)
			work := models.Work{
				GameId:         game.ID,
				Role:           enums.Charactor,
				CharactorName:  charactorName,
				StaffName:      staffName,
				WorkSummary:    summary,
				CharactorImage: image,
				SourceType:     enums.Dmm,
				Sort:           i,
				GameName:       game.Name,
				Measurements:   measurements,
				Height:         height,
			}
			return &work
		}

	}
	return nil
}

func (g *DmmInfoGetter) GetRelatedGames(makerId string) (string, error) {
	return "", nil
	//由于根据品牌找游戏本地可请以做到，暂时不这么搞了
	fmt.Printf("GetRelatedGames 01 \n")
	staffUrl := fmt.Sprintf("https://dlsoft.dmm.co.jp/ajax/article-contents/?floorId=digital_pcgame&articleId=%s&articleType=maker",
		makerId)
	fmt.Println(staffUrl)
	resp3, err := getResp(*g.client, staffUrl, "")
	if err != nil || resp3 == nil {
		fmt.Println("error FetchWorks 12: %v", err)
	}
	if resp3 == nil {
		fmt.Println("resp3 is  nil")
		return "", err
	}
	var res DMMSearchResponse
	if err := json.NewDecoder(resp3.Body).Decode(&res); err != nil {
		// fmt.Println("BangumiInfoGetter FetchMetadata 05 error: %v", err)
		resp3.Body.Close()
		return "", err
	} else {
		resp3.Body.Close()
	}
	fmt.Printf("GetRelatedGames 02\n")
	// rs :=""
	// for _, pd := range res.Body.ProductArray {
	// 	rs = JoinString()
	// }
	return JoinString(res.Body.ProductArray, ",", func(pd DMMProductArray) string {
		return string(enums.Dmm) + ":" + pd.CardProduct.ContentID
	}), nil
}
func combineCharacters(gameEntity models.GameEntity) models.GameEntity {
	worksMap := gameEntity.WorksMap
	if worksMap == nil {
		worksMap = make(map[enums.StaffRole][]models.Work)
	}
	charactors := worksMap[enums.Charactor]
	cvs := worksMap[enums.CV]
	newCvs := []models.Work{}
	remainCvs := []models.Work{}
	newCharactors := []models.Work{}
	for _, c := range charactors {
		newWork := Find(cvs, func(it models.Work) bool {
			fmt.Printf("boxtext 03, c.staffName:%s, cv.staffName:%s\n", c.StaffName, it.StaffName)
			return it.StaffName != "" && it.StaffName == c.StaffName
		})
		if newWork != nil {
			newWork.SourceCharactorId = c.SourceCharactorId
			newWork.CharactorName = c.CharactorName
			newWork.WorkSummary = c.WorkSummary
			newWork.Images = c.Images
			newWork.CharactorImage = c.CharactorImage
			newWork.Measurements = c.Measurements
			newWork.Height = c.Height

			newCvs = append(newCvs, *newWork)
		} else {
			newCharactors = append(newCharactors, c)
		}
	}
	for _, cv := range cvs {
		newWork := Find(newCvs, func(it models.Work) bool {
			return it.StaffName == cv.StaffName
		})
		if newWork == nil {
			remainCvs = append(remainCvs, cv)
		}
	}

	worksMap[enums.Charactor] = newCharactors
	worksMap[enums.CV] = append(newCvs, remainCvs...)
	gameEntity.WorksMap = worksMap

	fmt.Printf("角色人数02：%d, data: %v\n", len(newCharactors), worksMap[enums.CV])
	return gameEntity

}

func (g *DmmInfoGetter) FetchReviews(id string, token string, page int) (models.GameReview, error) {
	gameReview := models.GameReview{}

	reviewUrl := fmt.Sprintf("https://dlsoft.dmm.co.jp/ajax/v1/review/%s/?page=1&sort=value_desc&limit=100",
		id)
	fmt.Println(reviewUrl)
	resp3, err := getResp(*g.client, reviewUrl, "")
	if err != nil || resp3 == nil {
		fmt.Println("error FetchWorks 12: %v", err)
	}
	if resp3 == nil {
		fmt.Println("resp3 is  nil")
		return gameReview, err
	}
	var res DMMReviewResponse
	if err := json.NewDecoder(resp3.Body).Decode(&res); err != nil {
		// fmt.Println("BangumiInfoGetter FetchMetadata 05 error: %v", err)
		resp3.Body.Close()
		return gameReview, err
	} else {
		resp3.Body.Close()
	}
	for _, review := range res.Body.ReviewList {
		date, _ := time.Parse("2006/01/02", review.ReleaseDate)
		fmt.Printf("date:%s\n", review.ReleaseDate)
		gameReview.Reviews = append(gameReview.Reviews, models.Review{
			Id:          review.ReviewId,
			Content:     strings.ReplaceAll(review.Comment, "<br />", ""),
			Title:       review.Title,
			Points:      strconv.Itoa(review.Score),
			TotalPoints: "5",
			Reviewer:    review.ReviewerName,
			Date:        date,
		})
	}

	return gameReview, nil
}

func (g *DmmInfoGetter) FetchReviewDetail(review models.Review, gameId, token string) (models.Review, error) {
	return models.Review{}, nil
}
