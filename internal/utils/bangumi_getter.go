package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
)

type BangumiInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

func NewBangumiInfoGetter() *BangumiInfoGetter {
	return &BangumiInfoGetter{
		client:  &http.Client{},
		timeout: 10 * time.Second,
	}
}

var _ Getter = (*BangumiInfoGetter)(nil)

const bangumiIdQueryAPIURL = "https://api.bgm.tv/v0/subjects"

type bangumiImages struct {
	Large  string `json:"large"`
	Common string `json:"common"`
	Medium string `json:"medium"`
	Small  string `json:"small"`
	Grid   string `json:"grid"`
}

func (b *bangumiImages) GetImage() string {
	if b.Large != "" {
		return b.Large
	}
	if b.Common != "" {
		return b.Common
	}
	return b.Medium
}

type bangumiInfoboxItem struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type bangumiRating struct {
	Rank  int            `json:"rank"`
	Total int            `json:"total"`
	Count map[string]int `json:"count"`
	Score float64        `json:"score"`
}

type bangumiCollection struct {
	Wish    int `json:"wish"`
	Collect int `json:"collect"`
	Doing   int `json:"doing"`
	OnHold  int `json:"on_hold"`
	Dropped int `json:"dropped"`
}

type bangumiTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type bangumiResponse struct {
	ID            int                  `json:"id"`
	Type          int                  `json:"type"`
	Name          string               `json:"name"`
	NameCN        string               `json:"name_cn"`
	Summary       string               `json:"summary"`
	Series        bool                 `json:"series"`
	NSFW          bool                 `json:"nsfw"`
	Locked        bool                 `json:"locked"`
	Date          string               `json:"date"`
	Platform      string               `json:"platform"`
	Images        bangumiImages        `json:"images"`
	Infobox       []bangumiInfoboxItem `json:"infobox"`
	Volumes       int                  `json:"volumes"`
	Eps           int                  `json:"eps"`
	TotalEpisodes int                  `json:"total_episodes"`
	Rating        bangumiRating        `json:"rating"`
	Collection    bangumiCollection    `json:"collection"`
	MetaTags      []string             `json:"meta_tags"`
	Tags          []bangumiTag         `json:"tags"`
}

// 角色结构体
type bangumiCharacter struct {
	Images       bangumiImages  `json:"images"`
	Name         string         `json:"name"`
	ShortSummary string         `json:"short_summary"`
	Summary      string         `json:"summary"`
	Relation     string         `json:"relation"`
	Actors       []bangumiStaff `json:"actors"`
	Type         int            `json:"type"`
	ID           int            `json:"id"`
}

// 工作人员结构体
type bangumiStaff struct {
	Images       bangumiImages `json:"images"`
	ShortSummary string        `json:"short_summary"`
	Name         string        `json:"name"`
	Relation     string        `json:"relation"`
	Career       []string      `json:"career"`
	Type         int           `json:"type"`
	ID           int           `json:"id"`
	Eps          string        `json:"eps"`
	Summary      string        `json:"summary"`
	Locked       bool          `json:"locked"`
}

func (b BangumiInfoGetter) FetchMetadata(id string, token string) (models.Game, error) {
	gameEntity, err := b.FetchMetadataReq(vo.MetadataRequest{
		Source: enums.Bangumi,
		ID:     id,
	}, token)
	return gameEntity.Game, err
}

func (b BangumiInfoGetter) GetStaffRoleFromRelation(relation string) enums.StaffRole {
	switch relation {
	case "企画":
		return enums.Director
	case "剧本":
		return enums.Sceneario
	case "声优":
		return enums.CV
	case "音乐":
		return enums.Composer
	case "原画":
		return enums.CharaDesign
	case "背景":
		return enums.Art

	}
	return enums.Staff
}

func (b BangumiInfoGetter) FetchWorks(request vo.MetadataRequest, gameEntity models.GameEntity, token string) (models.GameEntity, error) {
	worksMap := make(map[enums.StaffRole][]models.Work)
	gameEntity.WorksMap = worksMap
	fmt.Printf("FetchWorks gameId:%s\n", gameEntity.Game.ID)

	if request.ShouldFetchStaffs {
		staffUrl := fmt.Sprintf("%s/%s/persons", bangumiIdQueryAPIURL, request.ID)
		fmt.Println(staffUrl)
		resp3, err := getResp(*b.client, staffUrl, fmt.Sprintf("Bearer %s", token))
		if err != nil || resp3 == nil {
			fmt.Println("error FetchWorks 12: %v", err)
		}
		// defer func(Body io.ReadCloser) {
		// 	err := Body.Close()
		// 	if err != nil {
		// 		log.Warnf("Error closing response body: %v", err)
		// 	}
		// }(resp3.Body)
		if resp3 == nil {
			fmt.Println("resp3 is  nil")
		}
		var staffs []bangumiStaff
		if err := json.NewDecoder(resp3.Body).Decode(&staffs); err != nil {
			// fmt.Println("BangumiInfoGetter FetchMetadata 05 error: %v", err)
			resp3.Body.Close()
			return gameEntity, err
		} else {
			resp3.Body.Close()
		}
		for _, staff := range staffs {
			work := models.Work{
				StaffName:     staff.Name,
				GameId:        gameEntity.Game.ID,
				SourceStaffId: strconv.Itoa(staff.ID),
				Role:          b.GetStaffRoleFromRelation(staff.Relation),
				SourceType:    enums.Bangumi,
				SourceGameId:  request.ID,
				GameName:      gameEntity.Game.Name,
				StaffImage:    staff.Images.GetImage(),
			}
			worksMap[work.Role] = append(worksMap[work.Role], work)
		}
	}

	if request.ShouldFetchCharactors {
		charactorsUrl := fmt.Sprintf("%s/%s/characters", bangumiIdQueryAPIURL, request.ID)
		resp2, err := getResp(*b.client, charactorsUrl, fmt.Sprintf("Bearer %s", token))
		if err != nil {
			fmt.Sprintf("error FetchWorks 11: %v", err)
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Warnf("Error closing response body: %v", err)
			}
		}(resp2.Body)
		var characters []bangumiCharacter
		if err := json.NewDecoder(resp2.Body).Decode(&characters); err != nil {
			// fmt.Println("BangumiInfoGetter FetchMetadata 05 error: %v", err)
			return gameEntity, err
		}
		for _, charactor := range characters {
			t := FindMatch(charactor.Actors, worksMap[enums.CV], func(t1 bangumiStaff, t2 models.Work) bool { return strconv.Itoa(t1.ID) == t2.SourceStaffId })
			sourceStaffId := ""
			staffName := ""
			if t != nil {
				sourceStaffId = strconv.Itoa(t.ID)
				staffName = t.Name
			}
			if len(charactor.Actors) > 0 {
				sourceStaffId = strconv.Itoa(charactor.Actors[0].ID)
				staffName = charactor.Actors[0].Name
			}
			work := models.Work{
				CharactorName:     charactor.Name,
				GameId:            gameEntity.Game.ID,
				Images:            charactor.Images.GetImage(),
				Role:              enums.Charactor,
				SourceCharactorId: strconv.Itoa(charactor.ID),
				SourceType:        enums.Bangumi,
				SourceStaffId:     sourceStaffId,
				WorkSummary:       charactor.Summary,
				StaffName:         staffName,
				SourceGameId:      request.ID,
				GameName:          gameEntity.Game.Name,
			}
			worksMap[enums.Charactor] = append(worksMap[enums.Charactor], work)
		}
	}

	return gameEntity, nil
}

func (b BangumiInfoGetter) FetchMetadataReq(request vo.MetadataRequest, token string) (models.GameEntity, error) {
	var gameEntity models.GameEntity = models.GameEntity{}
	var game models.Game = request.GetGame()
	fmt.Println("FetchMetadataReq gameId:" + request.DbGameId)
	gameEntity.Game = game
	if token == "" {
		return gameEntity, errors.New("bangumi API requires Bearer token")
	}

	url := fmt.Sprintf("%s/%s", bangumiIdQueryAPIURL, request.ID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("BangumiInfoGetter FetchMetadata 01 error: %v", err)
		return gameEntity, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.3.2 (desktop) (https://github.com/Saramanda9988/LunaBox)")

	resp, err := b.client.Do(req)
	if err != nil {
		fmt.Println("BangumiInfoGetter FetchMetadata 02 error: %v", err)
		return gameEntity, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("BangumiInfoGetter FetchMetadata 03 error: %v", err)
			log.Warnf("Error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return gameEntity, fmt.Errorf("bangumi API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var bangumiResp bangumiResponse
	if err := json.NewDecoder(resp.Body).Decode(&bangumiResp); err != nil {
		fmt.Println("BangumiInfoGetter FetchMetadata 04 error: %v", err)
		return gameEntity, err
	}

	if bangumiResp.Type != 4 { // 4 代表游戏
		return gameEntity, errors.New("the provided ID does not correspond to a game")
	}

	gameEntity, err = b.GetDataFromResp(gameEntity, bangumiResp)

	return gameEntity, err
}

func (b BangumiInfoGetter) GetDataFromResp(gameEntity models.GameEntity, bangumiResp bangumiResponse) (models.GameEntity, error) {

	game := gameEntity.Game

	// 使用中文名，如果没有则使用原名
	name := bangumiResp.NameCN
	if name == "" {
		name = bangumiResp.Name
	}

	// 选择最佳的封面图片 (优先使用 large，然后是 common)
	coverURL := bangumiResp.Images.Large
	if coverURL == "" {
		coverURL = bangumiResp.Images.Common
	}

	var tagsMap map[string][]models.Tag = make(map[string][]models.Tag)

	game.Name = name
	if game.CoverURL == "" {
		game.CoverURL = coverURL
	}

	var err error = nil
	for _, metaTagText := range bangumiResp.MetaTags {
		tag := models.Tag{
			Name:     metaTagText,
			Category: models.TagCategoryGenre,
		}
		tagsMap[models.TagCategoryGenre] = append(tagsMap[models.TagCategoryGenre], tag)
	}
	game.Summary = bangumiResp.Summary
	game.SourceType = enums.Bangumi
	game.ReleaseAt, err = time.Parse("2006-01-02", bangumiResp.Date)
	game.CachedAt = time.Now()
	game.BangumiId = strconv.Itoa(bangumiResp.ID)
	game.SourceID = game.BangumiId
	game.SourceType = enums.Bangumi
	gameEntity.Tags = tagsMap

	// 从 infobox 中提取开发商信息
	err = b.extractCompanyFromInfobox(bangumiResp.Infobox, &gameEntity, &game)
	for _, tag := range bangumiResp.Tags {
		gameEntity.Tags[models.TagCategoryOther] = append(gameEntity.Tags[models.TagCategoryOther], models.Tag{Name: tag.Name})
	}
	game.Tags = JoinString(MapToArray(gameEntity.Tags), ",",
		func(tag models.Tag) string { return tag.Name })
	fmt.Println("bangumi标签 01", len(gameEntity.Tags[models.TagCategoryOther]))
	gameEntity.Tags = tagsMap
	gameEntity.Game = game

	fmt.Println("01 02 " + game.Company)
	return gameEntity, err
}

func (b BangumiInfoGetter) FetchMetadataByName(name string, token string) (models.Game, error) {
	fmt.Println("FetchMetadataByName" + name)
	if token == "" {
		return models.Game{}, errors.New("bangumi API requires Bearer token")
	}

	searchURL := "https://api.bgm.tv/v0/search/subjects"

	params := url.Values{}
	params.Add("limit", "1")
	params.Add("offset", "0")
	fullURL := fmt.Sprintf("%s?%s", searchURL, params.Encode())

	reqBody := map[string]interface{}{
		"keyword": name,
		"sort":    "rank",
		"filter": map[string]interface{}{
			"type": []int{4},
			"nsfw": true,
		},
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return models.Game{}, err
	}

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return models.Game{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.3.2 (desktop) (https://github.com/Saramanda9988/LunaBox)")
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return models.Game{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warnf("Error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return models.Game{}, fmt.Errorf("bangumi search API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var searchResp struct {
		Data []bangumiResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return models.Game{}, err
	}

	if len(searchResp.Data) == 0 {
		return models.Game{}, errors.New("no results found")
	}

	bangumiResp := searchResp.Data[0]

	if bangumiResp.Type != 4 { // 4 代表游戏
		return models.Game{}, errors.New("the provided ID does not correspond to a game")
	}
	gameEntity := models.GameEntity{}
	fmt.Println("游戏名01：" + bangumiResp.Name + " " + bangumiResp.NameCN)
	gameEntity, err = b.GetDataFromResp(gameEntity, bangumiResp)
	game := gameEntity.Game

	return game, err
}

// extractCompanyFromInfobox 从 infobox 中提取开发商信息
func (b BangumiInfoGetter) extractCompanyFromInfobox(infobox []bangumiInfoboxItem, gameEntity *models.GameEntity, game *models.Game) error {
	var tagsMap map[string][]models.Tag = gameEntity.Tags
	for _, item := range infobox {
		// 查找开发商相关的字段
		if strings.Contains(item.Key, "开发商") || strings.Contains(item.Key, "开发") {

		}
		tag := models.Tag{}
		switch v := item.Value.(type) {
		case string:
			tag.Name = v
			if isValidDateFormat(v) {
				break
			}

			if strings.Contains(item.Key, "平台") {
				tag.Category = models.TagCategoryPlatform
				tag.BlockModify = true
				tagsMap[tag.Category] = append(tagsMap[tag.Category], tag)
			} else if strings.Contains(item.Key, "发行商") {

				tag.Category = models.TagCategoryPublisher
				tag.BlockModify = true
				tagsMap[tag.Category] = append(tagsMap[tag.Category], tag)
			} else if strings.Contains(item.Key, "游戏类型") {
				tag.Category = models.TagCategoryGenre
				tag.BlockModify = true
				tagsMap[tag.Category] = append(tagsMap[tag.Category], tag)
			} else if strings.Contains(item.Key, "开发") {
				tag.Category = models.TagCategoryBrand
				tag.BlockModify = true
				tagsMap[tag.Category] = append(tagsMap[tag.Category], tag)
				game.Company = v
				fmt.Println("developer 01:", v)
			}
			break
		case []interface{}:

			if strings.Contains(item.Key, "开发商") || strings.Contains(item.Key, "开发") {
				// 如果是数组，尝试提取第一个值
				if len(v) > 0 {
					if str, ok := v[0].(string); ok {
						tag.Name = str
						tag.Category = models.TagCategoryBrand
						tag.BlockModify = true
						tagsMap[tag.Category] = append(tagsMap[tag.Category], tag)
						game.Company = str
						fmt.Println("developer 02:", str)
					}
					// 处理可能的对象格式 {"v": "value"}
					if obj, ok := v[0].(map[string]interface{}); ok {
						if val, exists := obj["v"]; exists {
							if str, ok := val.(string); ok {
								tag.Name = str
								tag.Category = models.TagCategoryBrand
								tag.BlockModify = true
								tagsMap[tag.Category] = append(tagsMap[tag.Category], tag)
								game.Company = str
								fmt.Println("developer 03:", str)
							}
						}
					}
				}
			}

		}
	}
	gameEntity.Tags = tagsMap
	return nil
}
