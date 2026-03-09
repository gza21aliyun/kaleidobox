package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"lunabox/internal/applog"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery" // 添加 GoQuery 导入
	"github.com/gocolly/colly/v2"    // 添加 Colly 导入
)

type EroscapeInfoGetter struct {
	client    *http.Client
	timeout   time.Duration
	useMirror bool
}

func NewEroscapeInfoGetter(useMirror bool) *EroscapeInfoGetter {
	return &EroscapeInfoGetter{
		client:    &http.Client{},
		timeout:   10 * time.Second,
		useMirror: useMirror,
	}
}

var _ Getter = (*EroscapeInfoGetter)(nil)

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

	// c.WithTransport(&http.Transport{
	// 	TLSClientConfig: &tls.Config{
	// 		InsecureSkipVerify: true,
	// 	},
	// 	DisableCompression: true, // 禁用自动解压缩
	// })

	// 首先，设置请求前的处理
	c.OnRequest(func(r *colly.Request) {

		applog.InfoLogSaveAppLog("Visiting %s", r.URL.String()) // 打印正在访问的 URL
		cookie3 := &http.Cookie{Name: "age_check_done", Value: "1"}
		cookie4 := &http.Cookie{Name: "adultchecked", Value: "1"}
		cookie5 := &http.Cookie{Name: "locale", Value: "ja-jp"}
		cookie6 := &http.Cookie{Name: "localesuggested", Value: "true"}

		// 将所有cookie组合成一个字符串
		cookies := fmt.Sprintf("%s=%s; %s=%s; %s=%s, %s=%s",
			cookie3.Name, cookie3.Value,
			cookie4.Name, cookie4.Value,
			cookie5.Name, cookie5.Value,
			cookie6.Name, cookie6.Value,
		)

		r.Headers.Set("Cookie", cookies)

		// 设置额外的请求头
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "ja-JP,ja;q=0.8,en-US;q=0.5,en;q=0.3")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
	})
	return c
}

func (b EroscapeInfoGetter) FetchMetadataByName(name string, totken string) (models.Game, error) {
	return b.FetchMetadataByName2(name, true)
}

func (b EroscapeInfoGetter) FetchMetadataByName2(name string, isEnabled bool) (models.Game, error) {
	applog.InfoLogSaveAppLog("FetchMetadataByNameFunc 00\n")
	// mainTitle, num := getTitles(name)
	game, err := b.FetchMetadataByNameFunc(name, isEnabled,
		func(request vo.MetadataRequest) (models.Game, error) {
			// fmt.Printf("FetchMetadataByNameFunc 01")
			gameEntity, err := b.FetchMetadataById(request)

			return gameEntity.Game, err
		})
	if game.SourceID == "" {
		alternativeName := getGameNameAlternative(name)
		if alternativeName != "" {
			game, err = b.FetchMetadataByNameFunc(alternativeName, isEnabled,
				func(request vo.MetadataRequest) (models.Game, error) {
					// fmt.Printf("FetchMetadataByNameFunc 01")
					gameEntity, err := b.FetchMetadataById(request)

					return gameEntity.Game, err
				})
		}
		// fmt.Printf("FetchMetadataByNameFunc Error fetching metadata:%v\n", err)
		return game, err
	}
	return game, err
}

func (b EroscapeInfoGetter) GetBaseUrl() string {
	var mirror string = "https://koko.kyara.top/"
	var original string = "https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/"
	var baseUrl string
	if b.useMirror {
		baseUrl = mirror
	} else {
		baseUrl = original
	}
	return baseUrl
}

func (b EroscapeInfoGetter) GetDomain() string {
	var mirrorDomain = "*kyara.top"
	var baseDomain = "*dyndns.org"
	var domain string
	if b.useMirror {
		domain = mirrorDomain
	} else {
		domain = baseDomain
	}
	return domain
}

func (b EroscapeInfoGetter) FetchMetadataByNameFunc(name string, isEnabled bool, fn IdFunction) (models.Game, error) {
	log.Println("Fetching 01 metadata by name:", name)
	if !isEnabled { // 禁用的话，就返回一个空游戏
		return models.Game{}, nil
	}
	log.Println("Fetching 02 metadata by name:", name)

	var searchPart = "kensaku.php?category=game&word_category=name&mode=normal&word="
	// var gamePart = "game.php?game="
	var url string = b.GetBaseUrl() + searchPart
	// var gameUrl = baseUrl + gamePart

	mainTitle, _, _ := getTitles(name)
	// url += mainTitle

	url += mainTitle
	var game = models.Game{}
	c := CreateCollector(b.GetDomain())

	var potentialGames []struct {
		Title string
		// Link   string
		GameId string
	}

	// 处理搜索结果页面中的游戏条目
	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {

		title := e.DOM.Find("td").Eq(0).Text()
		href := e.ChildAttr("td a.tooltip", "href")
		idParts := strings.Split(strings.Split(href, "#")[0], "=")
		gameId := idParts[len(idParts)-1]

		// link := gameUrl + gameId

		if title != "" {
			applog.InfoLogSaveAppLog("title:", title)
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
		applog.InfoLogSaveAppLog("games found:%d\n", len(potentialGames))
		gameFound := searchNameByRegex(potentialGames, name, []string{"セット", "PSV", "PS4", "PSP"}, func(t1 struct {
			Title  string
			GameId string
		}) string {
			return t1.Title
		})
		if gameFound != nil {
			game.Name = gameFound.Title
			game.SourceID = gameFound.GameId
			game.SourceType = enums.Eroscape
			game.EroscapeId = gameFound.GameId
		}

		// sort.Slice(potentialGames, func(i, j int) bool {
		// 	return len(potentialGames[i].Title) < len(potentialGames[j].Title)
		// })
		// for _, gameFound := range potentialGames {
		// 	// 应用过滤条件
		// 	if strings.Contains(gameFound.Title, "セット") || strings.Contains(gameFound.Title, "PSV") || strings.Contains(gameFound.Title, "PS4") {
		// 		continue
		// 	}
		// 	// if gameFound.Link == "" {
		// 	// 	continue
		// 	// }
		// 	game.Name = gameFound.Title
		// 	game.SourceID = gameFound.GameId
		// 	game.SourceType = enums.Eroscape
		// 	game.EroscapeId = gameFound.GameId
		// 	// c.Visit(gameFound.Link)
		// 	return
		// }
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		applog.ErrorLogSaveAppLog("Request error, \n", err)
	})

	// 访问构建的 URL
	err := c.Visit(url)
	if err != nil {
		return models.Game{}, err
	}

	// 等待收集完成
	c.Wait()
	if game.SourceID == "" {
		applog.InfoLogSaveAppLog("id is empty for game %s", game.Name)
		err = errors.New("id is empty")
		return game, err
	}
	game, _ = fn(GetReqEntity(&game))

	return game, nil
}

func (b EroscapeInfoGetter) FetchImages(request vo.MetadataRequest, gameEntity models.GameEntity) (models.GameEntity, error) {
	if request.ShouldFetchImages == false {
		return gameEntity, nil
	}
	var imagesPart = "game_dmm.php?game="
	var cUrl = b.GetBaseUrl() + imagesPart + request.ID
	c := CreateCollector(b.GetDomain())
	var err error = nil
	game := gameEntity.Game
	game.Images = ""

	c.OnHTML("div#images div", func(e *colly.HTMLElement) {
		// applog.InfoLogSaveAppLog("图库：", game.Images)
		src := e.ChildAttr("img", "src")
		if src != "" {
			game.Images = MergeStrings(game.Images, src)
		}
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		applog.ErrorLogSaveAppLog("Request error\n", err)
	})

	// 访问构建的 URL
	err = c.Visit(cUrl)
	if err != nil {
		return gameEntity, err
	}

	// 等待收集完成
	c.Wait()
	gameEntity.Game = game

	return gameEntity, err
}

func (b EroscapeInfoGetter) FetchCharactors(request vo.MetadataRequest, gameEntity models.GameEntity) (models.GameEntity, error) {
	if request.ShouldFetchCharactors == false {
		return gameEntity, nil
	}
	if gameEntity.WorksMap == nil {
		var worksMap map[enums.StaffRole][]models.Work = make(map[enums.StaffRole][]models.Work)
		gameEntity.WorksMap = worksMap
	}
	var charactorPart = "game_character.php?game="
	var cUrl = b.GetBaseUrl() + charactorPart + request.ID
	c := CreateCollector(b.GetDomain())
	var err error = nil
	var cvs []models.Work = gameEntity.WorksMap[enums.CV]
	if cvs == nil {
		cvs = []models.Work{}
	}
	var characters []models.Work = []models.Work{}

	c.OnHTML("div.stage_main", func(e *colly.HTMLElement) {
		e.DOM.Find("div.character").Each(func(i int, s *goquery.Selection) {
			work := models.Work{}
			work.GameId = gameEntity.Game.ID
			work.SourceType = enums.Eroscape
			work.SourceGameId = request.ID
			work.GameName = gameEntity.Game.Name
			work.Role = enums.CV
			work.CharactorImage = s.Find("div.character_image img").AttrOr("src", "")
			work.CharactorName = strings.ReplaceAll(strings.TrimSpace(s.Find("div.character_name").Text()), " ", "")
			work.Height = removeAllChar(s.Find("div.personal_data dl:contains('身長') dd").Eq(0).Text(), " \n")
			work.Measurements = removeAllChar(s.Find("div.personal_data dl:contains('スリーサイズ') dd").Eq(0).Text(), " \n")
			fmt.Printf("三围：%s\n", work.Measurements)
			fmt.Printf("身高:%s\n", work.Height)
			sm, _ := s.Find("div.formal_explanation").Html()
			work.WorkSummary = strings.ReplaceAll(sm, "<br/>", "\n")
			work.Sort = i
			applog.InfoLogSaveAppLog("角色经历 01: " + work.WorkSummary)
			applog.InfoLogSaveAppLog("角色图像 01: " + work.CharactorImage)
			charHref := b.GetBaseUrl() + s.Find("div.character_name a").AttrOr("href", "")

			s.Find("div.eventimage li > img").Each(func(i2 int, s2 *goquery.Selection) {
				img := s2.AttrOr("src", "")
				work.Images = MergeStrings(work.Images, img)
			})

			// var err error = nil
			if charHref != "" {
				parsedURL, err := url.Parse(charHref)
				if err != nil {
					applog.ErrorLogSaveAppLog("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				work.SourceCharactorId = queryParams.Get("character")
				applog.InfoLogSaveAppLog("角色url: " + work.SourceCharactorId)
			}
			// work.WorkSummary, err = s.Find("div.formal_explanation").Html()

			// applog.InfoLogSaveAppLog("角色经历 02: " + work.WorkSummary)
			// if err != nil {
			// 	applog.ErrorLogSaveAppLog("FetchCharactors error \n", err)
			// }
			work.StaffName = s.Find("div.character_name").Text()

			staffHref := b.GetBaseUrl() + s.Find("div.cv a").AttrOr("href", "")
			work.StaffName = s.Find("div.cv a").Text()
			applog.InfoLogSaveAppLog("角色声优url : " + staffHref)
			if charHref != "" {
				parsedURL, err := url.Parse(staffHref)
				if err != nil {
					applog.ErrorLogSaveAppLog("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				work.SourceStaffId = queryParams.Get("creater")
				applog.InfoLogSaveAppLog("角色声优id : " + work.SourceStaffId + ",角色：" + work.CharactorName)
			}
			characters = append(characters, work)

		})
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		applog.ErrorLogSaveAppLog("Request error", err)
	})

	// 访问构建的 URL
	err = c.Visit(cUrl)
	if err != nil {
		return gameEntity, err
	}

	// 等待收集完成
	c.Wait()

	gameEntity.WorksMap[enums.CV] = cvs
	if len(characters) > 0 {
		gameEntity.WorksMap[enums.Charactor] = characters
		combineCharacters(gameEntity)
	}

	jstr, _ := json.Marshal(gameEntity.WorksMap[enums.CV])
	log.Printf("worksMap:" + string(jstr))
	// fmt.Println("cvs:", len(cvs))
	return gameEntity, nil
}

func (b EroscapeInfoGetter) FetchMetadata(id string, token string) (models.Game, error) {
	gameEntity, err := b.FetchMetadataById(vo.MetadataRequest{ID: id})
	return gameEntity.Game, err
}

func (b EroscapeInfoGetter) FetchMetadataById(
	request vo.MetadataRequest) (models.GameEntity, error) {
	var game models.Game = request.GetGame()
	game.SourceType = enums.Eroscape
	game.SourceID = request.ID
	game.EroscapeId = request.ID
	var gameEntity models.GameEntity = models.GameEntity{}
	gameEntity.Game = game
	var gamePart = "game.php?game="
	var gameUrl = b.GetBaseUrl() + gamePart + request.ID
	c := CreateCollector(b.GetDomain())
	c.OnHTML("div#main", func(e *colly.HTMLElement) {
		fmt.Println("开始获取游戏信息 29")

		// 提取封面图片
		coverURL := e.ChildAttr("div#main_image a img", "src")
		game.Name = e.ChildText("div#soft-title span")
		game.CoverURL = coverURL
		var tagsMap map[string][]models.Tag = make(map[string][]models.Tag)
		// 提取公司信息
		company := e.ChildText("tr#brand a")
		game.Company = company
		companyTag := models.Tag{
			Name:        company,
			Category:    models.TagCategoryBrand,
			BlockModify: true,
		}
		releaseAt := e.ChildText("tr#sellday a")
		fmt.Println("发售日1：", releaseAt)
		game.ReleaseAt, _ = time.Parse("2006-01-02", releaseAt)
		fmt.Println("发售日2：", game.ReleaseAt)
		tagsMap[models.TagCategoryBrand] = append(tagsMap[models.TagCategoryBrand], companyTag)

		e.DOM.Find("table#att_pov_table tr:contains('ジャンル') a").Each(func(i int, s *goquery.Selection) {
			g := strings.TrimSpace(s.Text())
			genreTag := models.Tag{
				Name:        g,
				Category:    models.TagCategoryGenre,
				BlockModify: true,
			}
			tagsMap[models.TagCategoryGenre] = append(tagsMap[models.TagCategoryGenre], genreTag)
		})

		// 提取简介
		summary := e.ChildText("div.area-detail-read")
		game.Summary = summary

		// 提取标签
		// var tags []string

		e.DOM.Find("table#att_pov_table tr").Each(func(i int, s *goquery.Selection) {
			category := s.Find("th").Text()
			s.Find("td a").Each(func(i2 int, s2 *goquery.Selection) {
				category := strings.TrimSpace(category)
				blockModify := false
				if category == "ジャンル" {
					category = models.TagCategoryGenre
					blockModify = true

				}
				var newTag models.Tag = models.Tag{
					Name:        strings.TrimSpace(s2.Text()),
					Category:    category,
					IsH:         category == "エロシーン",
					IsSpoiler:   category == "シナリオ",
					BlockModify: blockModify,
				}
				tagsMap[category] = append(tagsMap[category], newTag)
			})

		})

		e.DOM.Find("div#gamegroup > ul > li > a").Each(func(i int, s *goquery.Selection) {
			series := s.Text()
			if series != "" {
				seriesTag := models.Tag{
					Name:        series,
					BlockModify: true,
					Category:    models.TagCategorySeries,
				}
				tagsMap[models.TagCategorySeries] = append(tagsMap[models.TagCategorySeries], seriesTag)
			}
		})

		e.DOM.Find("div#gamegroup > ul > li > ul > li > a").Each(func(i int, s *goquery.Selection) {
			relatedGameLink := s.AttrOr("href", "")
			if relatedGameLink != "" {

				gameId := strings.ReplaceAll(strings.ReplaceAll(relatedGameLink, "game.php?game=", ""), "#ad", "")
				idStr := string(enums.Eroscape) + ":" + gameId
				game.RelatedGames = MergeStrings(game.RelatedGames, idStr)
			}
		})

		gameEntity.Tags = tagsMap
		tagList := []models.Tag{}
		for _, tags := range tagsMap {
			for _, tag := range tags {
				tagList = append(tagList, tag)
			}
		}
		// fmt.Println("标签01 ", len(tagList))

		game.Tags = JoinString(tagList, ",", func(tag models.Tag) string { return tag.Name })
		// fmt.Println("标签011 ", game.Tags)
		// jstr, _ := json.Marshal(tagsMap)
		// log.Printf("tagsMap:" + string(jstr))
		// game.Tags = strings.Join(tags, ",")

		var worksMap map[enums.StaffRole][]models.Work = make(map[enums.StaffRole][]models.Work)
		e.DOM.Find("table#creater_infomation_table tr#genga a").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					applog.ErrorLogSaveAppLog("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("creater")
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.CharaDesign,
					StaffName:     staffName,
					SourceStaffId: staffId,
					SourceType:    enums.Eroscape,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		e.DOM.Find("table#creater_infomation_table tr#shinario a").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					applog.ErrorLogSaveAppLog("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("creater")
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.Sceneario,
					StaffName:     staffName,
					SourceStaffId: staffId,
					SourceType:    enums.Eroscape,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})

		e.DOM.Find("table#creater_infomation_table tr#ongaku a").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					applog.ErrorLogSaveAppLog("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("creater")
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.Composer,
					StaffName:     staffName,
					SourceStaffId: staffId,
					SourceType:    enums.Eroscape,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})

		e.DOM.Find("table#creater_infomation_table tr#kasyu a").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					applog.ErrorLogSaveAppLog("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("creater")
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.Singer,
					StaffName:     staffName,
					SourceStaffId: staffId,
					SourceType:    enums.Eroscape,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})

		e.DOM.Find("table#creater_infomation_table tr#sonota a").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					applog.ErrorLogSaveAppLog("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("creater")
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.Staff,
					StaffName:     staffName,
					SourceStaffId: staffId,
					SourceType:    enums.Eroscape,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})

		var cvWorks []models.Work = []models.Work{}
		e.DOM.Find("table#creater_infomation_table tr#seiyu a").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			charactorName := strings.TrimSpace(s.Find("span").Text())
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					applog.ErrorLogSaveAppLog("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("creater")
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.CV,
					StaffName:     staffName,
					CharactorName: charactorName,
					SourceStaffId: staffId,
					SourceType:    enums.Eroscape,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		cvWorks = worksMap[enums.CV]
		e.DOM.Find("table#creater_infomation_table tr#seiyu span").Each(func(i int, s *goquery.Selection) {
			charactorName := strings.TrimSpace(s.Text())
			charactorName = strings.Trim(charactorName, "()")
			if charactorName == "その他" {
				charactorName = ""
			}
			if len(cvWorks) > i {
				cvWorks[i].CharactorName = charactorName
			}

		})
		worksMap[enums.CV] = cvWorks
		gameEntity.WorksMap = worksMap

	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		//错误的话err已包含url 和错误信息
		applog.ErrorLogSaveAppLog("Request error 36 \n", err)
	})

	// 访问构建的 URL
	err := c.Visit(gameUrl)
	if err != nil {
		return gameEntity, err
	}

	// 等待收集完成
	c.Wait()

	// 检查是否成功获取了数据
	if game.Name == "" {
		// fmt.Println("游戏未找到 37: " + game.Name)
		return gameEntity, applog.ErrorLogSaveTextOutput("game not found: %s", game.Name)
	}

	// 设置其他必要字段
	game.SourceType = enums.Eroscape // 假设你有这个枚举
	game.CachedAt = time.Now()
	// gameEntity.WorksMap = worksMap
	gameEntity.Game = game
	applog.InfoLogSaveAppLog("开始获取Eroscape游戏信息 38 ", gameEntity.Game.Images)
	return gameEntity, nil
}
