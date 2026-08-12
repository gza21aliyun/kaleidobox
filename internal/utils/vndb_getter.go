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
	"strings"
	"time"

	"github.com/labstack/gommon/log"
)

// VNDBInfoGetter 获取 VNDB 信息
type VNDBInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

var _ Getter = (*VNDBInfoGetter)(nil)

const vndbAPIURL = "https://api.vndb.org/kana/vn"

type vndbRequest struct {
	Filters []interface{} `json:"filters"`
	Fields  string        `json:"fields"`
	Sort    string        `json:"sort,omitempty"`
	Results int           `json:"results,omitempty"`
}

type vndbImage struct {
	URL    string  `json:"url"`
	Sexual float64 `json:"sexual"`
}

type vndbCharacter struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Original    string    `json:"original"`
	Description string    `json:"description"`
	Gender      []string  `json:"gender"`
	Height      int       `json:"height"`
	Bust        int       `json:"bust"`
	Waist       int       `json:"waist"`
	Hips        int       `json:"hips"`
	Image       vndbImage `json:"image"`
}

type vndbStaff struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Original string `json:"original"`
	Role     string `json:"role"`
}

type vndbVA struct {
	Character vndbCharacter `json:"character"`
	Staff     vndbStaff     `json:"staff"`
}

type vndbDeveloper struct {
	Name string `json:"name"`
}

type vndbTag struct {
	Name    string  `json:"name"`
	Rating  float64 `json:"rating"`
	Spoiler int     `json:"spoiler"` // 0=无剧透, 1=轻微, 2=重度
	Lie     bool    `json:"lie"`
}

type vndbTitle struct {
	Lang     string `json:"lang"`
	Title    string `json:"title"`
	Latin    string `json:"latin"`
	Official bool   `json:"official"`
	Main     bool   `json:"main"`
}

type vndbQueryResult struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Aliases     []string        `json:"aliases"`
	Titles      []vndbTitle     `json:"titles"`
	Image       vndbImage       `json:"image"`
	Description string          `json:"description"`
	Rating      float64         `json:"rating"`
	Released    string          `json:"released"`
	Developers  []vndbDeveloper `json:"developers"`
	Tags        []vndbTag       `json:"tags"`
	Screenshots []vndbImage     `json:"screenshots"`
	Staff       []vndbStaff     `json:"staff"`
	VA          []vndbVA        `json:"va"`
}

type vndbResponse struct {
	Results []vndbQueryResult `json:"results"`
}

func (V VNDBInfoGetter) FetchMetadata(id string, token string) (models.Game, error) {
	filters := []interface{}{"id", "=", id}
	mtReq := vo.MetadataRequest{}
	result, err := V.queryVNDB(filters, mtReq)
	if err != nil {
		return models.Game{}, err
	}
	return result.Game, nil
}

func (V VNDBInfoGetter) FetchEntity(req vo.MetadataRequest, token string) (models.GameEntity, error) {
	filters := []interface{}{"id", "=", req.ID}
	result, err := V.queryVNDB(filters, req)
	if err != nil {
		return models.GameEntity{}, err
	}
	return result, nil
}

func (V VNDBInfoGetter) FetchMetadataByName(name string, token string) (models.Game, error) {
	filters := []interface{}{"search", "=", name}
	mtReq := vo.MetadataRequest{}
	result, err := V.queryVNDB(filters, mtReq)
	if err != nil {
		return models.Game{}, err
	}
	return result.Game, nil
}

func (V VNDBInfoGetter) queryVNDB(filters []interface{}, mtReq vo.MetadataRequest) (models.GameEntity, error) {
	gameEntity := models.GameEntity{}

	reqBody := vndbRequest{
		Filters: filters,
		Fields:  "id, title, aliases, titles.lang, titles.title, titles.latin, titles.official, titles.main, image.url, image.sexual, screenshots.url, staff.id, staff.name, staff.original, staff.role, staff.gender, va.character.id, va.character.name, va.character.original, va.character.gender, va.character.description, va.character.height, va.character.bust, va.character.waist, va.character.hips, va.character.image.url, va.staff.id, va.staff.name, va.staff.original, description, rating, released, developers.name, tags.name, tags.rating, tags.spoiler, tags.lie",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return gameEntity, err
	}

	req, err := http.NewRequest("POST", vndbAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return gameEntity, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := V.client.Do(req)
	if err != nil {
		return gameEntity, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warnf("Error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return gameEntity, fmt.Errorf("VNDB API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var vndbResp vndbResponse
	if err := json.NewDecoder(resp.Body).Decode(&vndbResp); err != nil {
		return gameEntity, err
	}

	if len(vndbResp.Results) == 0 {
		return gameEntity, errors.New("no results found")
	}

	result := vndbResp.Results[0]

	var company string
	if len(result.Developers) > 0 {
		var devs []string
		for _, d := range result.Developers {
			devs = append(devs, d.Name)
		}
		company = strings.Join(devs, ", ")
	}

	var coverURL string
	if result.Image.URL != "" {
		coverURL = result.Image.URL
	}
	var tags []string
	var tagList []models.Tag
	for _, t := range result.Tags {
		tags = append(tags, t.Name)
		tagList = append(tagList, models.Tag{
			Name:      t.Name,
			IsSpoiler: t.Spoiler > 0,
		})
	}
	gameEntity.Tags = ArrayToMap(tagList, func(t1 models.Tag) string { return t1.Category })
	var screenshots []string
	for _, s := range result.Screenshots {
		screenshots = append(screenshots, s.URL)
	}
	var releaseAt time.Time
	if result.Released != "" {
		releaseAt, _ = time.Parse("2006-01-02", result.Released)
	}

	game := models.Game{
		ID:         mtReq.DbGameId,
		Name:       result.Title,
		CoverURL:   coverURL,
		Company:    company,
		Summary:    result.Description,
		SourceType: enums.VNDB,
		Tags:       strings.Join(tags, ","),
		Images:     strings.Join(screenshots, ","),
		SourceID:   result.ID,
		ReleaseAt:  releaseAt,
		CachedAt:   time.Now(),
	}
	if len(result.Titles) > 0 {
		titleja := Find(result.Titles, func(t vndbTitle) bool { return t.Lang == "ja" })
		if titleja != nil {
			game.Name = titleja.Title
		}
	}
	jsonData2, err := json.MarshalIndent(game, "", "  ")
	if err == nil {
		fmt.Printf("VNDB API response: %s\n", string(jsonData2))

	}
	gameEntity.Game = game
	gameEntity.WorksMap = make(map[enums.StaffRole][]models.Work)
	for _, st := range result.Staff {

		var role enums.StaffRole = enums.Staff
		switch st.Role {
		case "director":
			role = enums.Director
		case "scenario":
			role = enums.Sceneario
		case "Music":
			role = enums.Composer
		case "art":
			role = enums.Art
		case "artist":
			role = enums.CharaDesign
		case "songs":
			role = enums.Singer
		}
		gameEntity.WorksMap[role] = append(gameEntity.WorksMap[role], models.Work{
			GameId:        game.ID,
			Role:          role,
			SourceStaffId: st.ID,
			SourceType:    enums.VNDB,
			StaffName:     strings.ReplaceAll(st.Original, " ", ""),
			GameName:      game.Name,
		})
	}
	for _, va := range result.VA {
		image := ""
		image = va.Character.Image.URL
		gameEntity.WorksMap[enums.CV] = append(gameEntity.WorksMap[enums.CV], models.Work{
			GameId:            game.ID,
			Role:              enums.CV,
			SourceStaffId:     va.Staff.ID,
			StaffName:         strings.ReplaceAll(va.Staff.Original, " ", ""),
			SourceCharactorId: va.Character.ID,
			CharactorName:     strings.ReplaceAll(va.Character.Original, " ", ""),
			CharactorImage:    image,
			WorkSummary:       va.Character.Description,
			Height:            fmt.Sprintf("%d", va.Character.Height),
			Measurements:      fmt.Sprintf("%d/%d/%d", va.Character.Bust, va.Character.Waist, va.Character.Hips),
			SourceType:        enums.VNDB,
			GameName:          game.Name,
		})
	}

	return gameEntity, nil
}
