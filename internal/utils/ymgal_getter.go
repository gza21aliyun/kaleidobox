package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/gommon/log"
)

// YmgalInfoGetter 获取月幕Galgame信息
type YmgalInfoGetter struct {
	client   *http.Client
	timeout  time.Duration
	searchCn bool
}

func NewYmgalInfoGetter(searchCn bool) *YmgalInfoGetter {
	return &YmgalInfoGetter{
		client:   &http.Client{},
		timeout:  10 * time.Second,
		searchCn: searchCn,
	}
}

var _ Getter = (*YmgalInfoGetter)(nil)

const (
	ymgalAPIURL       = "https://www.ymgal.games/open/archive"
	ymgalTokenURL     = "https://www.ymgal.games/oauth/token"
	ymgalClientID     = "ymgal"
	ymgalClientSecret = "luna0327"
)

var ymgalTokenCache struct {
	token     string
	expiresAt time.Time
	mu        sync.Mutex
}

type ymgalTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type ymgalGame struct {
	Gid          int64                `json:"gid"`
	Name         string               `json:"name"`
	ChineseName  string               `json:"chineseName"`
	Introduction string               `json:"introduction"`
	MainImg      string               `json:"mainImg"`
	ReleaseDate  string               `json:"releaseDate"`
	Publisher    int                  `json:"publisher"`
	DeveloperID  int64                `json:"developerId"`
	Characters   []YmgalGameCharacter `json:"characters"`
	Staff        []YmgalGameStaff     `json:"staff"`
	Type         string               `json:"type"`
}

type YmgalGameStaff struct {
	Sid     int    `json:"sid"`
	Pid     int    `json:"pid"`
	EmpName string `json:"empName"`
	EmpDesc string `json:"empDesc"`
	JobName string `json:"jobName"`
}

type YmgalGameCharacter struct {
	Cid               int `json:"cid"`
	CvId              int `json:"cvId"`
	CharacterPosition int `json:"characterPosition"`
}

type ymgalResponse struct {
	Data    *ymgalData `json:"data"`
	Success *bool      `json:"success"`
	Code    int        `json:"code"`
	Msg     string     `json:"msg"`
}

type YmgalCharacter struct {
	Cid         int    `json:"cid"`
	Name        string `json:"name"`
	ChineseName string `json:"chineseName"`
	MainImg     string `json:"mainImg"`
	State       string `json:"state"`
	Freeze      bool   `json:"freeze"`
}

// YmgalStaff YMGal 制作人员信息
type YmgalStaff struct {
	Pid         int    `json:"pid"`
	Name        string `json:"name"`
	ChineseName string `json:"chineseName,omitempty"`
	MainImg     string `json:"mainImg"`
	State       string `json:"state"`
	Freeze      bool   `json:"freeze"`
}

type ymgalData struct {
	Game       *ymgalGame                `json:"game"`
	CidMapping map[string]YmgalCharacter `json:"cidMapping"`
	PidMapping map[string]YmgalStaff     `json:"pidMapping"`
}

type ymgalOrgData struct {
	Game       *ymgalGame                `json:"game"`
	Org        *ymgalOrganization        `json:"org"`
	CidMapping map[string]YmgalCharacter `json:"cidMapping"`
	PidMapping map[string]YmgalStaff     `json:"pidMapping"`
}

type ymgalOrganization struct {
	PublishVersion int    `json:"publishVersion"`
	PublishTime    string `json:"publishTime"`
	Publisher      int    `json:"publisher"`
	Name           string `json:"name"`
	ChineseName    string `json:"chineseName"`
	// ExtensionName  []YmgalExtensionName   `json:"extensionName"`
	Introduction string `json:"introduction"`
	State        string `json:"state"`
	Weights      int    `json:"weights"`
	MainImg      string `json:"mainImg"`
	// MoreEntry      []interface{}          `json:"moreEntry"`
	OrgId   int    `json:"orgId"`
	Country string `json:"country"`
	// Website        []YmgalWebsite         `json:"website"`
	Type   string `json:"type"`
	Freeze bool   `json:"freeze"`
}

type ymgalOrgResponse struct {
	Data    *ymgalOrgData `json:"data"`
	Success *bool         `json:"success"`
	Code    int           `json:"code"`
	Msg     string        `json:"msg"`
}

func (y YmgalInfoGetter) getAccessToken() (string, error) {
	ymgalTokenCache.mu.Lock()
	defer ymgalTokenCache.mu.Unlock()

	if ymgalTokenCache.token != "" && time.Now().UTC().Before(ymgalTokenCache.expiresAt) {
		return ymgalTokenCache.token, nil
	}

	params := url.Values{}
	params.Add("grant_type", "client_credentials")
	params.Add("client_id", ymgalClientID)
	params.Add("client_secret", ymgalClientSecret)
	params.Add("scope", "public")

	reqURL := fmt.Sprintf("%s?%s", ymgalTokenURL, params.Encode())
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.4.0 (desktop) (https://github.com/Saramanda9988/LunaBox)")

	resp, err := y.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warnf("Error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ymgal token API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp ymgalTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	ymgalTokenCache.token = tokenResp.AccessToken
	// 提前 60 秒过期，以防万一
	ymgalTokenCache.expiresAt = time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn)*time.Second - 60*time.Second)

	return ymgalTokenCache.token, nil
}

func (y YmgalInfoGetter) invalidateToken() {
	ymgalTokenCache.mu.Lock()
	defer ymgalTokenCache.mu.Unlock()
	ymgalTokenCache.token = ""
	ymgalTokenCache.expiresAt = time.Time{}
}

func (y YmgalInfoGetter) FetchEntity(request vo.MetadataRequest, token string) (models.GameEntity, error) {
	accessToken, err := y.getAccessToken()
	gameEntity := models.GameEntity{}
	if err != nil {
		return gameEntity, fmt.Errorf("failed to get access token: %w", err)
	}

	reqURL := fmt.Sprintf("%s?gid=%s", ymgalAPIURL, request.ID)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return gameEntity, err
	}

	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.4.0 (desktop) (https://github.com/Saramanda9988/LunaBox)")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("version", "1")
	req.Header.Set("Accept", "application/json;charset=utf-8")

	resp, err := y.client.Do(req)
	if err != nil {
		return gameEntity, err
	}

	// Handle 401 Unauthorized - Retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		y.invalidateToken()

		accessToken, err = y.getAccessToken()
		if err != nil {
			return gameEntity, fmt.Errorf("failed to refresh access token: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err = y.client.Do(req)
		if err != nil {
			return gameEntity, err
		}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warnf("Error closing response body: %v", err)
		}
	}(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return gameEntity, err
	}

	if resp.StatusCode != http.StatusOK {
		return gameEntity, fmt.Errorf("ymgal API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	fmt.Printf("json:\n%s\n", string(bodyBytes))

	var ymgalResp ymgalResponse
	if err := json.Unmarshal(bodyBytes, &ymgalResp); err != nil {
		return gameEntity, err
	}

	if ymgalResp.Success != nil && !*ymgalResp.Success {
		return gameEntity, fmt.Errorf("ymgal API error: %s (code: %d)", ymgalResp.Msg, ymgalResp.Code)
	}

	if ymgalResp.Data == nil || ymgalResp.Data.Game == nil {
		return gameEntity, fmt.Errorf("ymgal API returned no game data, body: %s", string(bodyBytes))
	}
	org, err := y.FetchOrganization(strconv.FormatInt(ymgalResp.Data.Game.DeveloperID, 10), accessToken)
	game, err := y.convertToModel(ymgalResp.Data.Game)
	game.ID = request.DbGameId
	game.Company = org
	game.Tags = org
	tagMap := gameEntity.Tags
	tagMap = make(map[string][]models.Tag)
	tagMap[models.TagCategoryBrand] = []models.Tag{
		models.Tag{
			Category:    models.TagCategoryBrand,
			Name:        org,
			BlockModify: true,
		},
	}
	gameEntity.Tags = tagMap
	works := gameEntity.WorksMap
	works = make(map[enums.StaffRole][]models.Work)
	for _, cr := range ymgalResp.Data.Game.Characters {
		scId := fmt.Sprintf("%d", cr.Cid)
		charactor := ymgalResp.Data.CidMapping[scId]
		work := models.Work{
			CharactorName:     charactor.Name,
			SourceCharactorId: scId,
			CharactorImage:    charactor.MainImg,
			Sort:              cr.CharacterPosition,
			Role:              enums.Charactor,
			SourceGameId:      request.ID,
			GameId:            request.DbGameId,
		}
		if cr.CvId != 0 {
			ssId := fmt.Sprintf("%d", cr.CvId)
			cv := ymgalResp.Data.PidMapping[ssId]
			if y.searchCn && cv.ChineseName != "" {
				work.StaffName = cv.ChineseName
			} else {
				work.StaffName = cv.Name
			}
			work.StaffImage = cv.MainImg
			work.SourceStaffId = ssId
			work.Role = enums.CV

		}
		works[work.Role] = append(works[work.Role], work)
	}

	for _, staff := range ymgalResp.Data.Game.Staff {
		sid := ""
		if staff.Pid != 0 {
			sid = fmt.Sprintf("%d", staff.Pid)
		}
		work := models.Work{
			StaffName:     staff.EmpName,
			SourceStaffId: sid,
			SourceGameId:  request.ID,
			GameId:        request.DbGameId,
		}
		role := enums.Staff
		if staff.JobName == "人物设计" {
			role = enums.CharaDesign
		} else if staff.JobName == "脚本" {
			role = enums.Sceneario
		} else if staff.JobName == "音乐" {
			role = enums.Composer
		} else if staff.JobName == "原画" {
			role = enums.Art
		} else if staff.JobName == "歌曲" {
			role = enums.Singer
		}
		work.Role = role
		if sid != "" {
			s := ymgalResp.Data.PidMapping[sid]
			if y.searchCn && s.ChineseName != "" {
				work.StaffName = s.ChineseName
			} else {
				work.StaffName = s.Name
			}
			work.StaffImage = s.MainImg
		}
		works[role] = append(works[role], work)

	}

	gameEntity.WorksMap = works

	gameEntity.Game = game
	return gameEntity, nil
}

func (y YmgalInfoGetter) FetchMetadata(id string, token string) (models.Game, error) {
	req := vo.MetadataRequest{
		ID:     id,
		Source: enums.Ymgal,
	}
	gameEntity, err := y.FetchEntity(req, token)
	if err != nil {
		return models.Game{}, err
	}
	return gameEntity.Game, nil
}

func (y YmgalInfoGetter) FetchOrganization(developerId string, token string) (string, error) {
	// accessToken, err := y.getAccessToken()
	// if err != nil {
	// 	return "", fmt.Errorf("failed to get access token: %w", err)
	// }

	reqURL := fmt.Sprintf("%s?orgId=%s", ymgalAPIURL, developerId)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.4.0 (desktop) (https://github.com/Saramanda9988/LunaBox)")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("version", "1")
	req.Header.Set("Accept", "application/json;charset=utf-8")

	resp, err := y.client.Do(req)
	if err != nil {
		return "", err
	}

	// Handle 401 Unauthorized - Retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		y.invalidateToken()

		accessToken, err := y.getAccessToken()
		if err != nil {
			return "", fmt.Errorf("failed to refresh access token: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err = y.client.Do(req)
		if err != nil {
			return "", err
		}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warnf("Error closing response body: %v", err)
		}
	}(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ymgal API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// fmt.Printf("json:\n%s\n", string(bodyBytes))

	var ymgalResp ymgalOrgResponse
	if err := json.Unmarshal(bodyBytes, &ymgalResp); err != nil {
		return "", err
	}

	if ymgalResp.Success != nil && !*ymgalResp.Success {
		return "", fmt.Errorf("ymgal API error: %s (code: %d)", ymgalResp.Msg, ymgalResp.Code)
	}

	// if ymgalResp.Data == nil {
	// 	return "", fmt.Errorf("ymgal API returned no game data, body: %s", string(bodyBytes))
	// }
	if y.searchCn && ymgalResp.Data.Org.ChineseName != "" {
		return ymgalResp.Data.Org.ChineseName, nil
	}
	// fmt.Printf("fo 02:%v\n", ymgalResp.Data)

	return ymgalResp.Data.Org.Name, nil
}

func (y YmgalInfoGetter) FetchMetadataByName(name string, token string) (models.Game, error) {
	accessToken, err := y.getAccessToken()
	if err != nil {
		return models.Game{}, fmt.Errorf("failed to get access token: %w", err)
	}

	searchURL := fmt.Sprintf("%s/search-game", ymgalAPIURL)
	params := url.Values{}
	params.Add("mode", "accurate")
	params.Add("keyword", name)
	params.Add("similarity", "70")
	fullURL := fmt.Sprintf("%s?%s", searchURL, params.Encode())

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return models.Game{}, err
	}

	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.4.0 (desktop) (https://github.com/Saramanda9988/LunaBox)")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("version", "1")
	req.Header.Set("Accept", "application/json;charset=utf-8")
	resp, err := y.client.Do(req)
	if err != nil {
		return models.Game{}, err
	}

	// Handle 401 Unauthorized - Retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		y.invalidateToken()

		accessToken, err = y.getAccessToken()
		if err != nil {
			return models.Game{}, fmt.Errorf("failed to refresh access token: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err = y.client.Do(req)
		if err != nil {
			return models.Game{}, err
		}
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warnf("Error closing response body: %v", err)
		}
	}(resp.Body)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Game{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return models.Game{}, fmt.Errorf("ymgal search API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var ymgalResp ymgalResponse
	if err := json.Unmarshal(bodyBytes, &ymgalResp); err != nil {
		return models.Game{}, err
	}

	if ymgalResp.Success != nil && !*ymgalResp.Success {
		return models.Game{}, fmt.Errorf("ymgal API error: %s (code: %d)", ymgalResp.Msg, ymgalResp.Code)
	}

	if ymgalResp.Data == nil || ymgalResp.Data.Game == nil {
		return models.Game{}, fmt.Errorf("ymgal API returned no game data, body: %s", string(bodyBytes))
	}

	return y.convertToModel(ymgalResp.Data.Game)
}

func (y YmgalInfoGetter) convertToModel(g *ymgalGame) (models.Game, error) {
	name := ""
	if y.searchCn {
		name = g.ChineseName
	}
	if name == "" {
		name = g.Name
	}
	releaseAt, err := time.Parse("2006-01-02", g.ReleaseDate)
	if err != nil {
		fmt.Printf("failed to parse release date: %v, releaseDate: %s\n", err, g.ReleaseDate)
	}

	game := models.Game{
		Name:       name,
		CoverURL:   g.MainImg,
		Company:    "", // Ymgal API response doesn't directly provide company name in the game object
		Summary:    g.Introduction,
		SourceType: enums.Ymgal,
		SourceID:   strconv.FormatInt(g.Gid, 10),
		YmgalId:    strconv.FormatInt(g.Gid, 10),
		ReleaseAt:  releaseAt,
		// CachedAt:   time.Now(),
		CachedAt: time.Now(),
	}
	return game, nil
}
