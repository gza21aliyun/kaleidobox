package utils

import (
	"fmt"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery" // 添加 GoQuery 导入
	"github.com/gocolly/colly/v2"    // 添加 Colly 导入
	"golang.org/x/text/encoding/japanese"
)

type GetchuInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

var _ Getter = (*GetchuInfoGetter)(nil)

const getchuCoverUrl = "https://www.getchu.com/brandnew/%s/rc%spackage.jpg"

func (b GetchuInfoGetter) FetchMetadataByName(name string, totken string) (models.Game, error) {
	return b.FetchMetadataByName2(name)
}

func (b GetchuInfoGetter) FetchMetadataByName2(name string) (models.Game, error) {
	game, err := b.FetchByNameImpl(name, false,
		func(request vo.MetadataRequest) (models.Game, error) {
			fmt.Println("FetchMetadataByName 34")
			gameEntity, err := b.FetchMetadataById(request)
			return gameEntity.Game, err
		})
	if game.SourceID == "" {
		game, err = b.FetchByNameImpl(name, true,
			func(request vo.MetadataRequest) (models.Game, error) {
				fmt.Println("FetchMetadataByName 34")
				gameEntity, err := b.FetchMetadataById(request)
				return gameEntity.Game, err
			})
	}
	return game, err
}

func (b GetchuInfoGetter) FetchMetadataByNameQuick(name string) (models.Game, error) {
	game, err := b.FetchByNameImpl(name, false, nil)
	if game.SourceID == "" {
		game, err = b.FetchByNameImpl(name, true, nil)
	}
	return game, err
}

func (b GetchuInfoGetter) FetchByNameImpl(name string, isAl bool, fn IdFunction) (models.Game, error) {
	var gUrl string = "https://www.getchu.com/php/search.phtml?genre=pc_soft&search_keyword="
	mainTitle, _, _ := getTitles(name)
	// url += mainTitle
	if isAl {
		mainTitle = getGameNameAlternative(mainTitle)
	}
	shiftJISEncoder := japanese.ShiftJIS.NewEncoder()
	encodedForURL, err := shiftJISEncoder.String(mainTitle)
	encodedForURL = url.QueryEscape(encodedForURL)
	gUrl += encodedForURL
	var game = models.Game{}
	c := CreateCollector("*getchu.com")

	var potentialGames []struct {
		Title string
		Link  string
		Cover string
	}
	fmt.Println("FetchByNameImpl: " + gUrl)

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("visiting %s\n", gUrl)
		fmt.Printf("OnResponse 搜索结果页面：\n %s:%d\n", gUrl, r.StatusCode)
	})

	// 处理搜索结果页面中的游戏条目
	c.OnHTML("div.search_container ul.display li", func(e *colly.HTMLElement) {
		block := e.DOM.Find("div.content_block tr").Eq(0).Find("a")
		title := block.Text()
		link := block.AttrOr("href", "")
		// log.Print("OnHTML 网页列表 ：", e.Text)

		if !strings.Contains(title, "動画版") && !strings.Contains(title, "音楽") {
			potentialGames = append(potentialGames, struct {
				Title string
				Link  string
				Cover string
			}{
				Title: title,
				Link:  e.Request.AbsoluteURL(link),
				Cover: e.DOM.Find("div.package_block img").AttrOr("src", ""),
			})
		}

	})

	// 在访问完搜索页面后进行过滤和处理
	c.OnScraped(func(r *colly.Response) {
		fmt.Printf("found %d \n", len(potentialGames))
		gameFound := searchNameByRegex(potentialGames, name, []string{"セット"}, func(t1 struct {
			Title string
			Link  string
			Cover string
		}) string {
			return t1.Title
		})

		if gameFound != nil {
			linkParts := strings.Split(gameFound.Link, "id=")
			id := linkParts[1]
			game.Name = gameFound.Title
			game.SourceID = id
			game.SourceType = enums.Getchu
			game.GetchuId = id
			game.CoverURL = fmt.Sprintf(getchuCoverUrl, id, id)

			fmt.Println("meta 61 getchu cover:", game.CoverURL)
		}

	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err = c.Visit(gUrl)
	if err != nil {
		return models.Game{}, err
	}

	// 等待收集完成
	c.Wait()
	if fn == nil {
		return game, err
	}
	game, err = fn(GetReqEntity(&game))

	return game, err
}

func (b GetchuInfoGetter) GetIdFromLink(link string) string {
	linkParts := strings.Split(link, "id=")
	id := linkParts[1]
	return id
}

func (b GetchuInfoGetter) FetchMetadata(id string, token string) (models.Game, error) {
	gameEntity, err := b.FetchMetadataById(vo.MetadataRequest{ID: id})
	return gameEntity.Game, err
}

func (b GetchuInfoGetter) FetchMetadataById(request vo.MetadataRequest) (models.GameEntity, error) {
	fmt.Println("开始获取getchu游戏信息 36 " + request.Source)
	var game models.Game = request.GetGame()
	var gameEntity models.GameEntity = models.GameEntity{}
	gameEntity.Game = game
	var err error = nil
	if request.ID == "" {
		return gameEntity, fmt.Errorf("getchu ID is required to fetch metadata by ID")
	}
	fmt.Println("开始获取getchu游戏信息 37 " + request.ID)
	game.GetchuId = request.ID
	getchuUrl := fmt.Sprintf("https://www.getchu.com/item/%s/", request.ID)
	fmt.Println("FetchByNameImpl: " + getchuUrl)
	c := CreateCollector("*getchu.com")

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("OnResponse 搜索结果页面：\n %s:%d\n", getchuUrl, r.StatusCode)
		// if len(r.Body) > 0 {
		// 	fmt.Printf("前20字节(hex): ")
		// 	for i := 0; i < min(20, len(r.Body)); i++ {
		// 		fmt.Printf("%02x ", r.Body[i])
		// 	}
		// 	fmt.Println()
		// }
		// decoder := japanese.ShiftJIS.NewDecoder()
		// reader := transform.NewReader(bytes.NewReader(r.Body), decoder)

		// decodedBytes, err := io.ReadAll(reader)
		// if err != nil {
		// 	fmt.Printf("Shift-JIS解码错误: %v\n", err)
		// 	return
		// }

		// decodedText := string(r.Body)
		// fmt.Printf("ggg utf8解码后前200字符: %s\n", decodedText)

		// 解码处理
		// decodedBody, err := decodeJapaneseContent(r.Body, r.Headers.Get("Content-Type"))
		if err != nil {
			fmt.Printf("编码处理错误: %v\n", err)
			return
		}

		// // 保存解码后的内容供分析
		// debugFile := fmt.Sprintf("final_decoded_%s.html", request.ID)
		// // os.WriteFile(debugFile, r.Body, 0644)
		// fmt.Printf("✅ 解码成功，内容已保存到: %s\n", debugFile)

		// // 验证解码结果
		// if strings.Contains(decodedBody, "[编码检测失败]") {
		// 	fmt.Println("⚠️  编码检测可能失败，请检查生成的文件")
		// } else {
		// 	fmt.Printf("解码后内容预览: %.300s...\n", decodedText)
		// }
	})

	// 处理游戏详情页面
	c.OnHTML("div#wrapper", func(e *colly.HTMLElement) {
		// hml, err := e.DOM.Html()s
		// hml, err := e.DOM.Children().Eq(0).Html()
		// fmt.Printf("开始获取getchu游戏信息 21\n%s\n ", hml)
		nameBlock := e.DOM.Find("h2#soft-title")
		nameBlock.Find("nobr").Remove()
		nameBlockHtml, err := nameBlock.Html()
		name := strings.TrimSuffix(strings.TrimSpace(nameBlock.Text()), " 通常版")
		fmt.Printf("nameblock:%s\n", nameBlockHtml)
		fmt.Println("开始获取Getchu信息 41 " + name)
		game.Name = name

		// fmt.Printf("alltext: %s\n", e.Text)

		cover := e.DOM.Find("table#soft_table > tbody > tr").First().Find("td a img").AttrOr("src", "")
		// e.DOM.Find("table#soft_table > tbody > tr").First().Find("td > a > img").AttrOr("href", "")
		game.CoverURL = "https://www.getchu.com" + strings.ReplaceAll(cover, "/rc", "/c")

		fmt.Println("开始获取Getchu cover 22 \n" + cover)
		datablock := e.DOM.Find("table#soft_table > tbody > tr").Eq(1).Find("th > table > tbody > tr")

		// 提取公司信息
		releaseAt := datablock.Eq(2).Find("a").First().Text()
		fmt.Println("release at:" + releaseAt)
		game.ReleaseAt, err = time.Parse("2006/01/02", releaseAt)

		company := datablock.Eq(0).Find("a").First().Text()

		fmt.Println("开始获取Getchu信息 42 \n" + company)

		game.Company = company

		// releaseAt := e.DOM.Find("table#soft_table > tbody > tr").Eq(1).Find("th > table > tbody > tr").Eq(1).Text()
		// game.ReleaseAt, err = time.Parse("2006/01/02", releaseAt)
		// fmt.Printf("发售日11：%v, %s, %v\n", game.ReleaseAt, releaseAt, err)

		var worksMap map[enums.StaffRole][]models.Work = make(map[enums.StaffRole][]models.Work)

		tagList := []models.Tag{}
		tagList = append(tagList, models.Tag{Name: company, Category: models.TagCategoryBrand, BlockModify: true})
		e.DOM.Find("table#soft_table > tbody > tr").Eq(1).Find("th > table > tbody > tr:contains('原画') a").Each(func(i int, s *goquery.Selection) {
			staffName := s.Text()
			fmt.Printf("staff 23:%s\n", staffName)
			if staffName != "" {

				work := models.Work{
					GameId:     game.ID,
					Role:       enums.CharaDesign,
					StaffName:  staffName,
					SourceType: enums.Getchu,
					GameName:   game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		e.DOM.Find("table#soft_table > tbody > tr").Eq(1).Find("th > table > tbody > tr:contains('シナリオ') a").Each(func(i int, s *goquery.Selection) {
			staffName := s.Text()
			fmt.Printf("staff 23:%s\n", staffName)
			if staffName != "" {

				work := models.Work{
					GameId:     game.ID,
					Role:       enums.Sceneario,
					StaffName:  staffName,
					SourceType: enums.Getchu,
					GameName:   game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		genre := e.DOM.Find("table#soft_table > tbody > tr").Eq(1).Find("th > table > tbody > tr:contains('ジャンル') td").Eq(1).Text()
		if genre != "" {
			tagList = append(tagList, models.Tag{Name: genre, Category: models.TagCategoryGenre, BlockModify: true})
		}

		// 提取简介
		summary := ""
		intro := e.DOM.Find("h3:contains('あらすじ')")
		if intro.Length() > 0 {
			summary = intro.Next().Text()
		}
		story := e.DOM.Find("h3:contains('ストーリー')")
		if story.Length() > 0 {
			if summary != "" {
				summary += "\n"
			}
			summary += story.Next().Text()
		}
		gdsIntro := e.DOM.Find("h3:contains('商品紹介')")
		if gdsIntro.Length() > 0 {
			if summary != "" {
				summary += "\n"
			}
			summary += gdsIntro.Next().Text()
		}

		// summary = strings.ReplaceAll(summary, "<br/>", "\n")
		// fmt.Printf("summary 25:\n%s\n", summary)
		game.Summary = summary

		// chars, _ := e.DOM.Find("h3:contains('キャラクター')").Next().Find("tbody tr").Eq(2).Html()
		// fmt.Printf("chars 26:\n%s\n", chars)

		e.DOM.Find("div.item-Samplecard-container img").Each(func(i int, s *goquery.Selection) {
			image := "https://www.getchu.com" + strings.ReplaceAll(s.AttrOr("src", ""), "_s.jpg", ".jpg")
			game.Images = MergeStrings(game.Images, image)
		})

		e.DOM.Find("h3:contains('キャラクター')").Next().Find("td.chara-text").Each(func(i int, s *goquery.Selection) {
			// parent := s.Parent
			// hml, _ := s.Find("h4.chara-name span").Html()
			// fmt.Printf("parent 27:\n%s\n", hml)
			charactorName := ""
			charaNameBlock := s.Find("h4.chara-name span").Contents()
			charaNameTextBlock := strings.Split(s.Find("h4.chara-name").Text(), "CV：")
			var charNameSize = charaNameBlock.Length()
			fmt.Printf("charNameSize:%d\n", charNameSize)
			fmt.Printf("charaNameTextBlock 29 %s\n", JoinString(charaNameTextBlock, ",", func(t string) string { return t }))
			charactorName = strings.TrimSpace(charaNameTextBlock[0])
			if charNameSize == 1 {
				// charactorName = strings.TrimSpace(charaNameTextBlock[0])
			} else {
				lastText := ""
				charaNameBlock.Each(func(i int, ss *goquery.Selection) {
					fmt.Printf("char 06 i: %d,charname=%s,lasttext:%s, ss:=%s\n", i, charactorName, lastText, ss.Text())
					if strings.Contains(ss.Text(), "CV") {
						cnb := strings.Split(ss.Text(), "CV：")
						if len(cnb) > 0 {
							if utf8.RuneCountInString(cnb[0]) > utf8.RuneCountInString(lastText) && lastText != "" {
								charactorName = lastText
								fmt.Printf("char 31 %s\n", charactorName)
							} else {
								charactorName = cnb[0]
								fmt.Printf("char 32,%d,%d %s\n", utf8.RuneCountInString(cnb[0]), utf8.RuneCountInString(lastText), charactorName)
							}

							// fmt.Printf("cnbt:%s\n", cnbt)

						}
					}
					if s.Is("charalist") {
						charactorName = ss.Text()
					}

					lastText = strings.TrimSpace(ss.Text())
					// if charactorName == "" && strings.Contains(ss.Text(), "CV") {
					// 	charactorName = lastText
					// }

				})

			}
			fmt.Printf("char 29 %s\n", charactorName)
			if strings.Contains(charactorName, "（") {
				cnb2 := strings.Split(charactorName, "（")
				if len(cnb2) > 0 {
					charactorName = cnb2[0]
				}
			}
			// fmt.Printf("char 30 %s\n", charactorName)

			if strings.Contains(charactorName, " ") {
				charactorName = strings.ReplaceAll(charactorName, " ", "")
			}
			staffName := ""
			if len(charaNameTextBlock) > 1 {
				staffName = strings.TrimSpace(charaNameTextBlock[1])
			}

			fmt.Printf("chars 28 staffName:%s, charName:%s\n", staffName, charactorName)

			charTextBlock := s.Find("dd").First()
			measurementsBlock := charTextBlock.Find("span > span:contains('身長')")
			fmt.Printf("chars 28 charTextBlock:%s\n", measurementsBlock.Text())
			var role enums.StaffRole
			if staffName == "" {
				role = enums.Charactor
			} else {
				role = enums.CV
			}
			newWork := models.Work{
				Role:          role,
				StaffName:     staffName,
				SourceType:    enums.Getchu,
				CharactorName: charactorName,
				Sort:          i,
				GameId:        game.ID,
			}
			if measurementsBlock.Length() > 0 {
				lines := strings.Split(measurementsBlock.Text(), "\n")

				height := ""
				measurements := ""
				if len(lines) > 1 {
					line1 := strings.TrimSpace(lines[0])
					line1Parts := strings.Split(line1, "スリーサイズ：")
					if len(line1Parts) > 1 {
						height = strings.TrimSpace(strings.ReplaceAll(line1Parts[0], "身長：", ""))
						measurements = strings.TrimSpace(line1Parts[1])
						newWork.Height = height
						newWork.Measurements = measurements

					}
				}
			}

			cimage := strings.TrimSpace(s.Parent().Find("td").First().Find("img").AttrOr("src", ""))
			if cimage != "" {
				newWork.CharactorImage = "https://www.getchu.com" + cimage
			}
			image := s.Parent().Find("td").Eq(2).Find("img").AttrOr("src", "")
			if image != "" {
				image = strings.ReplaceAll(image, "_s.jpg", ".jpg")
				image = "https://www.getchu.com" + image
				newWork.Images = image
			}
			fmt.Printf("chars 33 img:%s\n", image)
			if measurementsBlock.Length() > 0 {
				// fmt.Printf("measurementsBlock:%s\n full:%s\n", measurementsBlock.Text(), charTextBlock.Text())
				measurementsBlock.Remove()
			}
			charSummary := strings.TrimSpace(charTextBlock.Text())
			newWork.WorkSummary = charSummary
			worksMap[newWork.Role] = append(worksMap[newWork.Role], newWork)

		})

		if err != nil {
			// return
		}
		time.Sleep(time.Millisecond * 500)

		// 提取标签
		// var tags []string

		game.Tags = JoinString(tagList, ",", func(tag models.Tag) string { return tag.Name })
		gameEntity.Tags = ArrayToMap(tagList, func(t1 models.Tag) string { return t1.Category })

		// jstr, _ := json.MarshalIndent(worksMap, "", "  ")
		// log.Printf("worksMap: " + string(jstr))
		gameEntity.WorksMap = worksMap

	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	// 访问构建的 URL
	err = c.Visit(getchuUrl)
	if err != nil {
		fmt.Printf("开始获取Getchu游戏信息 40 err: %s\n", err)
		return gameEntity, err
	}

	// 等待收集完成
	c.Wait()
	fmt.Println("开始获取getchu游戏信息 39 " + game.CoverURL)

	// fmt.Printf("related games:%s\n", game.RelatedGames)

	// 检查是否成功获取了数据
	if game.Name == "" {
		fmt.Println("开始获取getchu游戏信息 40 " + game.CoverURL)
		return gameEntity, fmt.Errorf("game not found: %s", game.SourceID)
	}
	gameEntity = combineCharacters(gameEntity)
	fmt.Print("角色人数：", len(gameEntity.WorksMap[enums.Charactor]))

	// 设置其他必要字段
	game.SourceType = enums.Getchu // 假设你有这个枚举
	// game.SourceID = name
	game.CachedAt = time.Now()
	gameEntity.Game = game
	fmt.Println("开始获getchu游戏信息 38 " + game.CoverURL)

	return gameEntity, nil
}
