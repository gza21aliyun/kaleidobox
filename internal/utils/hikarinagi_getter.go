package utils

import (
	"context"
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
	"sync"
	"time"

	"github.com/labstack/gommon/log"
)

// ErrHikarinagiNotAuthorized 未授权错误（HIKARINAGI_NOT_AUTHORIZED 为稳定标记，
// 前端据此识别并弹出"去授权"引导，请勿改动标记文本）
var ErrHikarinagiNotAuthorized = errors.New("HIKARINAGI_NOT_AUTHORIZED: Hikarinagi 未授权，请先在设置中完成 Hikarinagi 授权")

// HikarinagiInfoGetter 获取 Hikarinagi（hikarinagi.org）Galgame 信息
// 开放平台文档: https://www.hikarinagi.org/developers/guide
type HikarinagiInfoGetter struct {
	client   *http.Client
	timeout  time.Duration
	searchCn bool
}

func NewHikarinagiInfoGetter(searchCn bool) *HikarinagiInfoGetter {
	return &HikarinagiInfoGetter{
		client:   &http.Client{},
		timeout:  10 * time.Second,
		searchCn: searchCn,
	}
}

var _ Getter = (*HikarinagiInfoGetter)(nil)

const (
	hikarinagiTokenURL   = "https://id.hikarinagi.org/oidc/token"
	hikarinagiAPIBaseURL = "https://api.hikarinagi.org/v3"
	hikarinagiClientID   = "hkn_6psvrnuwQJ7cwmRe" // 公共客户端，不持有 client_secret
)

// hikarinagiTokenState 令牌状态（刷新令牌每次刷新均会轮换，须覆盖保存）
var hikarinagiTokenState struct {
	refreshToken string
	accessToken  string
	expiresAt    time.Time
	saver        func(refreshToken string) error // 轮换后的持久化回调
	mu           sync.Mutex
}

// SetHikarinagiRefreshToken 设置刷新令牌（应用启动或配置更新时由服务层调用）
func SetHikarinagiRefreshToken(refreshToken string) {
	hikarinagiTokenState.mu.Lock()
	defer hikarinagiTokenState.mu.Unlock()
	hikarinagiTokenState.refreshToken = refreshToken
	hikarinagiTokenState.accessToken = ""
	hikarinagiTokenState.expiresAt = time.Time{}
}

// SetHikarinagiTokenSaver 设置刷新令牌轮换后的持久化回调
func SetHikarinagiTokenSaver(saver func(refreshToken string) error) {
	hikarinagiTokenState.mu.Lock()
	defer hikarinagiTokenState.mu.Unlock()
	hikarinagiTokenState.saver = saver
}

// HasHikarinagiToken 是否已完成过授权
func HasHikarinagiToken() bool {
	hikarinagiTokenState.mu.Lock()
	defer hikarinagiTokenState.mu.Unlock()
	return hikarinagiTokenState.refreshToken != ""
}

// OpenMediaDto 游戏截图 / 图片
type HikarinagiMedia struct {
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Sexual   int    `json:"sexual"`
	Violence int    `json:"violence"`
}

// OpenGalgameCoverDto 封面（按得票数排序返回）
type HikarinagiCover struct {
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Kind     string `json:"kind"`
	Language string `json:"language"`
	Sexual   int    `json:"sexual"`
	Violence int    `json:"violence"`
	Votes    int    `json:"votes"`
}

type HikarinagiTag struct {
	Name  string `json:"name"`
	Likes int    `json:"likes"`
}

// OpenGalgameDto Galgame 详情
type HikarinagiGalgame struct {
	ID             int64             `json:"id"`
	OriginTitle    string            `json:"origin_title"`
	TransTitle     *string           `json:"trans_title"`
	EnTitle        *string           `json:"en_title"`
	Aliases        []string          `json:"aliases"`
	Developer      *string           `json:"developer"`
	OriginIntro    *string           `json:"origin_intro"`
	TransIntro     *string           `json:"trans_intro"`
	EnIntro        *string           `json:"en_intro"`
	AdvType        *string           `json:"adv_type"`
	DevStatus      *string           `json:"dev_status"`
	ReleaseDate    *string           `json:"release_date"`
	ReleaseDateTbd bool              `json:"release_date_tbd"`
	NSFW           bool              `json:"nsfw"`
	Homepage       *string           `json:"homepage"`
	Covers         []HikarinagiCover `json:"covers"`
	Images         []HikarinagiMedia `json:"images"`
	Tags           []HikarinagiTag   `json:"tags"`
	Platforms      []string          `json:"platforms"`
	OriginLang     *string           `json:"origin_lang"`
}

// OpenEntityRefDto 人物/角色引用
type HikarinagiEntityRef struct {
	ID        int64            `json:"id"`
	Name      string           `json:"name"`
	TransName *string          `json:"trans_name"`
	Image     *HikarinagiMedia `json:"image"`
}

// Galgame 角色条目
type HikarinagiCharacterEntry struct {
	Character HikarinagiEntityRef   `json:"character"`
	Actors    []HikarinagiEntityRef `json:"actors"`
	Role      string                `json:"role"`
}

// Galgame staff 条目
type HikarinagiStaffEntry struct {
	Person HikarinagiEntityRef `json:"person"`
	Role   *string             `json:"role"`
}

// 搜索结果条目
type HikarinagiSearchHit struct {
	ID          int64            `json:"id"`
	Title       string           `json:"title"`
	OriginTitle string           `json:"origin_title"`
	TransTitle  *string          `json:"trans_title"`
	OriginIntro *string          `json:"origin_intro"`
	TransIntro  *string          `json:"trans_intro"`
	Subtitle    *string          `json:"subtitle"`
	Type        string           `json:"type"`
	Developer   *string          `json:"developer"`
	Cover       *HikarinagiCover `json:"cover"`
}

type hikarinagiEnvelope struct {
	Success *bool            `json:"success"`
	Data    json.RawMessage  `json:"data"`
	Error   *hikarinagiError `json:"error"`
}

type hikarinagiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type hikarinagiSearchData struct {
	Items []HikarinagiSearchHit `json:"items"`
}

func (g HikarinagiInfoGetter) getAccessToken() (string, error) {
	hikarinagiTokenState.mu.Lock()
	defer hikarinagiTokenState.mu.Unlock()

	if hikarinagiTokenState.accessToken != "" && time.Now().UTC().Before(hikarinagiTokenState.expiresAt) {
		return hikarinagiTokenState.accessToken, nil
	}

	if hikarinagiTokenState.refreshToken == "" {
		return "", ErrHikarinagiNotAuthorized
	}

	tokenResp, err := RefreshHikarinagiToken(context.Background(), hikarinagiTokenState.refreshToken)
	if err != nil {
		return "", err
	}

	// 访问令牌有效期为 1 小时
	hikarinagiTokenState.accessToken = tokenResp.AccessToken
	hikarinagiTokenState.expiresAt = time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn)*time.Second - 60*time.Second)

	// 刷新令牌轮换：覆盖保存新值
	if tokenResp.RefreshToken != "" && tokenResp.RefreshToken != hikarinagiTokenState.refreshToken {
		hikarinagiTokenState.refreshToken = tokenResp.RefreshToken
		if hikarinagiTokenState.saver != nil {
			if err := hikarinagiTokenState.saver(tokenResp.RefreshToken); err != nil {
				fmt.Printf("failed to save rotated hikarinagi refresh token: %v\n", err)
			}
		}
	}

	return hikarinagiTokenState.accessToken, nil
}

func (g HikarinagiInfoGetter) invalidateToken() {
	hikarinagiTokenState.mu.Lock()
	defer hikarinagiTokenState.mu.Unlock()
	hikarinagiTokenState.accessToken = ""
	hikarinagiTokenState.expiresAt = time.Time{}
}

// getAPI 发起带令牌的 API 请求并解包统一响应信封，data 内容解码到 out
func (g HikarinagiInfoGetter) getAPI(path string, out interface{}) error {
	accessToken, err := g.getAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	reqURL := hikarinagiAPIBaseURL + path
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.4.0 (desktop) (https://github.com/Saramanda9988/LunaBox)")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}

	// Handle 401 Unauthorized - Retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		g.invalidateToken()

		accessToken, err = g.getAccessToken()
		if err != nil {
			return fmt.Errorf("failed to refresh access token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err = g.client.Do(req)
		if err != nil {
			return err
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
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hikarinagi API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var envelope hikarinagiEnvelope
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return err
	}

	if envelope.Success != nil && !*envelope.Success {
		if envelope.Error != nil {
			return fmt.Errorf("hikarinagi API error: %s (code: %s)", envelope.Error.Message, envelope.Error.Code)
		}
		return fmt.Errorf("hikarinagi API error, body: %s", string(bodyBytes))
	}

	if len(envelope.Data) == 0 {
		return fmt.Errorf("hikarinagi API returned no data, body: %s", string(bodyBytes))
	}

	return json.Unmarshal(envelope.Data, out)
}

// resolveAssetURL 处理相对路径的静态资源地址
func resolveHikarinagiAssetURL(u string) string {
	if u == "" || strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	if !strings.HasPrefix(u, "/") {
		u = "/" + u
	}
	return "https://www.hikarinagi.org" + u
}

func (g HikarinagiInfoGetter) pickTitle(originTitle string, transTitle *string) string {
	if g.searchCn && transTitle != nil && *transTitle != "" {
		return *transTitle
	}
	return originTitle
}

func (g HikarinagiInfoGetter) pickIntro(originIntro *string, transIntro *string) string {
	if g.searchCn && transIntro != nil && *transIntro != "" {
		return *transIntro
	}
	if originIntro != nil {
		return *originIntro
	}
	return ""
}

func (g HikarinagiInfoGetter) convertToModel(gameDto *HikarinagiGalgame) (models.Game, error) {
	name := g.pickTitle(gameDto.OriginTitle, gameDto.TransTitle)
	summary := g.pickIntro(gameDto.OriginIntro, gameDto.TransIntro)

	releaseAt := time.Time{}
	if gameDto.ReleaseDate != nil && *gameDto.ReleaseDate != "" {
		releaseDate := *gameDto.ReleaseDate
		var err error
		// 兼容 "2020-01-01" 与 ISO8601 两种格式
		if len(releaseDate) == 10 {
			releaseAt, err = time.Parse("2006-01-02", releaseDate)
		} else {
			releaseAt, err = time.Parse(time.RFC3339, releaseDate)
		}
		if err != nil {
			fmt.Printf("failed to parse hikarinagi release date: %v, releaseDate: %s\n", err, releaseDate)
		}
	}

	coverURL := ""
	if len(gameDto.Covers) > 0 {
		coverURL = resolveHikarinagiAssetURL(gameDto.Covers[0].URL)
	}

	game := models.Game{
		Name:       name,
		CoverURL:   coverURL,
		Company:    "",
		Summary:    summary,
		SourceType: enums.Hikarinagi,
		SourceID:   strconv.FormatInt(gameDto.ID, 10),
		ReleaseAt:  releaseAt,
		CachedAt:   time.Now(),
	}
	if gameDto.Developer != nil {
		game.Company = *gameDto.Developer
	}

	images := make([]string, 0, len(gameDto.Images))
	for _, img := range gameDto.Images {
		if u := resolveHikarinagiAssetURL(img.URL); u != "" {
			images = append(images, u)
		}
	}
	game.Images = strings.Join(images, ",")

	game.SetSourceID(enums.Hikarinagi, strconv.FormatInt(gameDto.ID, 10))
	return game, nil
}

func (g HikarinagiInfoGetter) buildTags(gameDto *HikarinagiGalgame, game models.Game) map[string][]models.Tag {
	tagMap := make(map[string][]models.Tag)
	if game.Company != "" {
		tagMap[models.TagCategoryBrand] = []models.Tag{
			{Category: models.TagCategoryBrand, Name: game.Company, BlockModify: true},
		}
	}
	if len(gameDto.Tags) > 0 {
		genreTags := make([]models.Tag, 0, len(gameDto.Tags))
		for _, t := range gameDto.Tags {
			if t.Name == "" {
				continue
			}
			genreTags = append(genreTags, models.Tag{Category: models.TagCategoryGenre, Name: t.Name, BlockModify: true})
		}
		tagMap[models.TagCategoryGenre] = genreTags
	}
	return tagMap
}

func (g HikarinagiInfoGetter) FetchEntity(request vo.MetadataRequest, token string) (models.GameEntity, error) {
	gameEntity := models.GameEntity{}
	if request.ID == "" {
		return gameEntity, fmt.Errorf("hikarinagi ID is required to fetch metadata")
	}

	var gameDto HikarinagiGalgame
	if err := g.getAPI("/galgames/"+request.ID, &gameDto); err != nil {
		return gameEntity, err
	}

	game, err := g.convertToModel(&gameDto)
	if err != nil {
		return gameEntity, err
	}
	game.ID = request.DbGameId

	gameEntity.Game = game
	gameEntity.Tags = g.buildTags(&gameDto, game)

	works := make(map[enums.StaffRole][]models.Work)

	// 角色
	if request.ShouldFetchCharactors {
		var charEntries []HikarinagiCharacterEntry
		if err := g.getAPI("/galgames/"+request.ID+"/characters", &charEntries); err != nil {
			log.Warnf("failed to fetch hikarinagi characters for game %s: %v", request.ID, err)
		} else {
			for i, entry := range charEntries {
				character := entry.Character
				work := models.Work{
					CharactorName:     g.pickTitle(character.Name, character.TransName),
					SourceCharactorId: strconv.FormatInt(character.ID, 10),
					SourceGameId:      request.ID,
					GameId:            request.DbGameId,
					Sort:              i,
					Role:              enums.Charactor,
				}
				if character.Image != nil {
					work.CharactorImage = resolveHikarinagiAssetURL(character.Image.URL)
				}
				works[work.Role] = append(works[work.Role], work)

				// 配音演员
				if len(entry.Actors) > 0 {
					actor := entry.Actors[0]
					cvWork := models.Work{
						StaffName:         g.pickTitle(actor.Name, actor.TransName),
						SourceStaffId:     strconv.FormatInt(actor.ID, 10),
						SourceGameId:      request.ID,
						GameId:            request.DbGameId,
						Role:              enums.CV,
						SourceCharactorId: strconv.FormatInt(character.ID, 10),
					}
					if actor.Image != nil {
						cvWork.StaffImage = resolveHikarinagiAssetURL(actor.Image.URL)
					}
					works[enums.CV] = append(works[enums.CV], cvWork)
				}
			}
		}
	}

	// 制作人员
	if request.ShouldFetchStaffs {
		var staffEntries []HikarinagiStaffEntry
		if err := g.getAPI("/galgames/"+request.ID+"/staff", &staffEntries); err != nil {
			log.Warnf("failed to fetch hikarinagi staff for game %s: %v", request.ID, err)
		} else {
			for _, entry := range staffEntries {
				person := entry.Person
				work := models.Work{
					StaffName:     g.pickTitle(person.Name, person.TransName),
					SourceStaffId: strconv.FormatInt(person.ID, 10),
					SourceGameId:  request.ID,
					GameId:        request.DbGameId,
					Role:          mapHikarinagiStaffRole(entry.Role),
				}
				if person.Image != nil {
					work.StaffImage = resolveHikarinagiAssetURL(person.Image.URL)
				}
				works[work.Role] = append(works[work.Role], work)
			}
		}
	}

	gameEntity.WorksMap = works
	return gameEntity, nil
}

func mapHikarinagiStaffRole(role *string) enums.StaffRole {
	if role == nil {
		return enums.Staff
	}
	switch *role {
	case "SCENARIO":
		return enums.Sceneario
	case "CHARACTER_DESIGN":
		return enums.CharaDesign
	case "MUSIC":
		return enums.Composer
	case "ART", "SD_ART", "GRAPHICS", "BACKGROUND":
		return enums.Art
	case "THEME_PERFORMANCE", "INSERT_PERFORMANCE":
		return enums.Singer
	case "DIRECTOR":
		return enums.Director
	default:
		return enums.Staff
	}
}

func (g HikarinagiInfoGetter) FetchMetadata(id string, token string) (models.Game, error) {
	req := vo.MetadataRequest{
		ID:     id,
		Source: enums.Hikarinagi,
	}
	gameEntity, err := g.FetchEntity(req, token)
	if err != nil {
		return models.Game{}, err
	}
	return gameEntity.Game, nil
}

func (g HikarinagiInfoGetter) FetchMetadataByName(name string, token string) (models.Game, error) {
	params := url.Values{}
	params.Add("q", name)
	params.Add("types", "galgame")
	params.Add("page", "1")
	params.Add("page_size", "10")

	var searchData hikarinagiSearchData
	if err := g.getAPI("/search?"+params.Encode(), &searchData); err != nil {
		return models.Game{}, err
	}

	for _, hit := range searchData.Items {
		if hit.Type != "galgame" {
			continue
		}
		game := models.Game{
			Name:       g.pickTitle(hit.OriginTitle, hit.TransTitle),
			Summary:    g.pickIntro(hit.OriginIntro, hit.TransIntro),
			SourceType: enums.Hikarinagi,
			SourceID:   strconv.FormatInt(hit.ID, 10),
			CachedAt:   time.Now(),
		}
		if hit.Developer != nil {
			game.Company = *hit.Developer
		}
		if hit.Cover != nil {
			game.CoverURL = resolveHikarinagiAssetURL(hit.Cover.URL)
		}
		game.SetSourceID(enums.Hikarinagi, strconv.FormatInt(hit.ID, 10))
		return game, nil
	}

	return models.Game{}, fmt.Errorf("hikarinagi search returned no galgame for %q", name)
}

// ---------- 评论抓取 ----------
// Hikarinagi 开放 API（/v3）未提供公开短评端点，短评数据取自条目评分页
// https://www.hikarinagi.org/galgames/{id}/rates 的 SSR 页面内嵌 __NUXT_DATA__（devalue 扁平数组格式）。
// 站点改版可能破坏解析。

const hikarinagiSiteBaseURL = "https://www.hikarinagi.org"

type hikarinagiRater struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
}

type hikarinagiRate struct {
	Id          string                  `json:"id"`
	Rate        int                     `json:"rate"`
	Status      string                  `json:"status"`
	IsSpoiler   bool                    `json:"is_spoiler"`
	RateContent *string                 `json:"rate_content"`
	LikeCount   int                     `json:"like_count"`
	CreatedAt   string                  `json:"created_at"`
	Rater       hikarinagiRater         `json:"rater"`
	LongReview  *map[string]interface{} `json:"long_review"`
}

type hikarinagiRatesMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	ItemCount  int `json:"item_count"`
	TotalPages int `json:"total_pages"`
}

// nuxtResolve 解引用 devalue 扁平数组：对象属性值是数组下标，取 data[idx] 即实际值
func nuxtResolve(data []interface{}, v interface{}) interface{} {
	if idx, ok := v.(float64); ok {
		i := int(idx)
		if i >= 0 && i < len(data) {
			return data[i]
		}
	}
	return v
}

// nuxtAsString 解引用并断言为字符串
func nuxtAsString(data []interface{}, v interface{}) string {
	if s, ok := nuxtResolve(data, v).(string); ok {
		return s
	}
	return ""
}

// nuxtAsFloat 解引用并断言为数字
func nuxtAsFloat(data []interface{}, v interface{}) (float64, bool) {
	if f, ok := nuxtResolve(data, v).(float64); ok {
		return f, true
	}
	return 0, false
}

// parseHikarinagiNuxtData 从 SSR 页面 HTML 中提取并解析 __NUXT_DATA__
func parseHikarinagiNuxtData(html string) ([]interface{}, error) {
	marker := `id="__NUXT_DATA__"`
	start := strings.Index(html, marker)
	if start < 0 {
		return nil, fmt.Errorf("hikarinagi 页面未找到 __NUXT_DATA__")
	}
	jsonStart := strings.Index(html[start:], ">")
	if jsonStart < 0 {
		return nil, fmt.Errorf("hikarinagi __NUXT_DATA__ 标签解析失败")
	}
	jsonStart += start + 1
	jsonEnd := strings.Index(html[jsonStart:], "</script>")
	if jsonEnd < 0 {
		return nil, fmt.Errorf("hikarinagi __NUXT_DATA__ 未正常闭合")
	}
	var data []interface{}
	if err := json.Unmarshal([]byte(html[jsonStart:jsonStart+jsonEnd]), &data); err != nil {
		return nil, fmt.Errorf("hikarinagi __NUXT_DATA__ 解析失败: %w", err)
	}
	return data, nil
}

// extractHikarinagiRates 从 devalue 扁平数组中提取评分对象列表与分页 meta
func extractHikarinagiRates(data []interface{}) ([]hikarinagiRate, hikarinagiRatesMeta) {
	rates := make([]hikarinagiRate, 0)
	meta := hikarinagiRatesMeta{}
	for _, item := range data {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		// 评分对象：同时具备 rate/rater/created_at/rate_content 字段
		if _, hasRate := obj["rate"]; hasRate {
			if _, hasRater := obj["rater"]; hasRater {
				if _, hasAt := obj["created_at"]; hasAt {
					rate := hikarinagiRate{}
					if id, ok := nuxtAsFloat(data, obj["id"]); ok {
						rate.Id = strconv.FormatInt(int64(id), 10)
					}
					if r, ok := nuxtAsFloat(data, obj["rate"]); ok {
						rate.Rate = int(r)
					}
					if s, ok := nuxtResolve(data, obj["status"]).(string); ok {
						rate.Status = s
					}
					if b, ok := nuxtResolve(data, obj["is_spoiler"]).(bool); ok {
						rate.IsSpoiler = b
					}
					if s, ok := nuxtResolve(data, obj["rate_content"]).(string); ok {
						rate.RateContent = &s
					}
					if l, ok := nuxtAsFloat(data, obj["like_count"]); ok {
						rate.LikeCount = int(l)
					}
					if s, ok := nuxtResolve(data, obj["created_at"]).(string); ok {
						rate.CreatedAt = s
					}
					if rater, ok := nuxtResolve(data, obj["rater"]).(map[string]interface{}); ok {
						rate.Rater = hikarinagiRater{
							Id:       int64(raterId(rater)),
							Name:     nuxtAsString(data, rater["name"]),
							Nickname: nuxtAsString(data, rater["nickname"]),
						}
					}
					if lr, ok := nuxtResolve(data, obj["long_review"]).(map[string]interface{}); ok {
						rate.LongReview = &lr
					}
					rates = append(rates, rate)
				}
			}
			continue
		}
		// 分页 meta：具备 page/total_items/total_pages 字段
		if _, hasTotal := obj["total_items"]; hasTotal {
			if _, hasPages := obj["total_pages"]; hasPages {
				if p, ok := nuxtAsFloat(data, obj["page"]); ok {
					meta.Page = int(p)
				}
				if p, ok := nuxtAsFloat(data, obj["page_size"]); ok {
					meta.PageSize = int(p)
				}
				if p, ok := nuxtAsFloat(data, obj["total_items"]); ok {
					meta.TotalItems = int(p)
				}
				if p, ok := nuxtAsFloat(data, obj["item_count"]); ok {
					meta.ItemCount = int(p)
				}
				if p, ok := nuxtAsFloat(data, obj["total_pages"]); ok {
					meta.TotalPages = int(p)
				}
			}
		}
	}
	return rates, meta
}

func raterId(rater map[string]interface{}) float64 {
	if f, ok := rater["id"].(float64); ok {
		return f
	}
	return 0
}

// FetchGameReviews 抓取 Hikarinagi 短评
// reviewType "0" 返回带短评内容的评分；"1" 返回附带长评的评分
// 注意：SSR 页面不支持翻页参数，仅能取到第一页（20 条）
func (g HikarinagiInfoGetter) FetchGameReviews(id string, page int, reviewType string) (models.GameReview, error) {
	review := models.GameReview{
		SourceType:  enums.Hikarinagi,
		Id:          id,
		TotalPoints: "10",
	}

	pageURL := fmt.Sprintf("%s/galgames/%s/rates", hikarinagiSiteBaseURL, id)
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return review, err
	}
	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.4.0 (desktop) (https://github.com/Saramanda9988/LunaBox)")
	req.Header.Set("Accept", "text/html")

	resp, err := g.client.Do(req)
	if err != nil {
		return review, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warnf("Error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return review, fmt.Errorf("hikarinagi rates page returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return review, err
	}

	data, err := parseHikarinagiNuxtData(string(bodyBytes))
	if err != nil {
		return review, err
	}

	rates, meta := extractHikarinagiRates(data)
	// SSR 不支持翻页参数：请求页大于返回页时视为无数据，避免重复展示第一页
	if page > 1 && meta.Page < page {
		return review, nil
	}

	for _, rate := range rates {
		content := ""
		if rate.RateContent != nil {
			content = *rate.RateContent
		}
		if reviewType == "0" && content == "" {
			continue
		}
		if reviewType == "1" && rate.LongReview == nil {
			continue
		}
		reviewer := rate.Rater.Nickname
		if reviewer == "" {
			reviewer = rate.Rater.Name
		}
		date := time.Time{}
		if rate.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, rate.CreatedAt); err == nil {
				date = t
			}
		}
		review.Reviews = append(review.Reviews, models.Review{
			Id:          rate.Id,
			Content:     content,
			Reviewer:    reviewer,
			Date:        date,
			TotalPoints: "10",
			Points:      strconv.Itoa(rate.Rate),
		})
	}
	review.HasNext = meta.Page < meta.TotalPages
	return review, nil
}
