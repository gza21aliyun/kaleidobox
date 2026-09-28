package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"net/http"
	"strings"
	"time"
)

// KungalInfoGetter 获取 Kungal（kungal.com）Galgame 信息
//
// Kungal 的 /api/* 端点均需登录（返回 code 205 用户登录失效），匿名只能抓取
// 详情页 https://www.kungal.com/galgame/{id} 的 SSR 内嵌 __NUXT_DATA__
// （devalue 扁平数组格式），其中包含完整的 work 对象。
// 站点搜索接口需要登录且搜索页 SSR 不含结果数据，因此不支持按名称搜索。
// 站点改版可能破坏解析。
type KungalInfoGetter struct {
	client   *http.Client
	timeout  time.Duration
	searchCn bool
}

func NewKungalInfoGetter(searchCn bool) *KungalInfoGetter {
	return &KungalInfoGetter{
		client:   &http.Client{},
		timeout:  15 * time.Second,
		searchCn: searchCn,
	}
}

var _ Getter = (*KungalInfoGetter)(nil)

const kungalSiteBaseURL = "https://www.kungal.com"

// getHTML 抓取 SSR 页面原始 HTML（匿名即可）
func (g KungalInfoGetter) getHTML(path string) (string, error) {
	reqURL := kungalSiteBaseURL + path
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.4.0 (desktop) (https://github.com/Saramanda9988/LunaBox)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	client := &http.Client{Timeout: g.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close kungal response body: %v\n", err)
		}
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("kungal 页面返回状态: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	return string(bodyBytes), nil
}

// parseNuxtData 从 SSR 页面 HTML 中提取并解析 __NUXT_DATA__（devalue 扁平数组）
func parseNuxtData(html string) ([]interface{}, error) {
	marker := `id="__NUXT_DATA__"`
	start := strings.Index(html, marker)
	if start < 0 {
		return nil, fmt.Errorf("页面未找到 __NUXT_DATA__")
	}
	jsonStart := strings.Index(html[start:], ">")
	if jsonStart < 0 {
		return nil, fmt.Errorf("__NUXT_DATA__ 标签解析失败")
	}
	jsonStart += start + 1
	jsonEnd := strings.Index(html[jsonStart:], "</script>")
	if jsonEnd < 0 {
		return nil, fmt.Errorf("__NUXT_DATA__ 未正常闭合")
	}
	var data []interface{}
	if err := json.Unmarshal([]byte(html[jsonStart:jsonStart+jsonEnd]), &data); err != nil {
		return nil, fmt.Errorf("__NUXT_DATA__ 解析失败: %w", err)
	}
	return data, nil
}

// findKungalWork 在 devalue 扁平数组中定位详情页主 work 对象。
// 相关作品卡片同样是 object=work 但不含 intros 等详情字段，以此区分。
func findKungalWork(data []interface{}) map[string]interface{} {
	for _, item := range data {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if nuxtAsString(data, obj["object"]) != "work" {
			continue
		}
		if _, has := obj["intros"]; has {
			return obj
		}
	}
	return nil
}

// nuxtMap 解引用并断言为对象
func nuxtMap(data []interface{}, v interface{}) map[string]interface{} {
	if m, ok := nuxtResolve(data, v).(map[string]interface{}); ok {
		return m
	}
	return nil
}

// nuxtList 解引用并断言为数组
func nuxtList(data []interface{}, v interface{}) []interface{} {
	if l, ok := nuxtResolve(data, v).([]interface{}); ok {
		return l
	}
	return nil
}

// nuxtLocalizedValue 读取 localized[locale].value（Kungal 的本地化名称/简介结构）
func nuxtLocalizedValue(data []interface{}, localized interface{}, locale string) string {
	m := nuxtMap(data, localized)
	if m == nil {
		return ""
	}
	entry := nuxtMap(data, m[locale])
	if entry == nil {
		return ""
	}
	return nuxtAsString(data, entry["value"])
}

// pickKungalTitle 按 SearchCn 开关选择标题：zh-Hans 为译文，display_name 为原文
func (g KungalInfoGetter) pickKungalTitle(data []interface{}, work map[string]interface{}) string {
	origin := nuxtAsString(data, work["display_name"])
	if g.searchCn {
		if trans := nuxtLocalizedValue(data, work["localized"], "zh-Hans"); trans != "" {
			return trans
		}
	}
	if origin == "" {
		if ja := nuxtLocalizedValue(data, work["localized"], "ja"); ja != "" {
			return ja
		}
	}
	return origin
}

// pickKungalIntro 按 SearchCn 开关选择简介：intros 按 locale 区分原文(ja)/译文(zh-Hans)
func (g KungalInfoGetter) pickKungalIntro(data []interface{}, work map[string]interface{}) string {
	origin := ""
	trans := ""
	for _, item := range nuxtList(data, work["intros"]) {
		entry := nuxtMap(data, item)
		if entry == nil {
			continue
		}
		switch nuxtAsString(data, entry["locale"]) {
		case "ja":
			origin = nuxtAsString(data, entry["value"])
		case "zh-Hans":
			trans = nuxtAsString(data, entry["value"])
		}
	}
	if g.searchCn && trans != "" {
		return trans
	}
	if origin != "" {
		return origin
	}
	return trans
}

func (g KungalInfoGetter) parseKungalReleaseDate(data []interface{}, work map[string]interface{}) time.Time {
	releaseDate := nuxtAsString(data, work["release_date"])
	if releaseDate == "" {
		return time.Time{}
	}
	var releaseAt time.Time
	var err error
	switch nuxtAsString(data, work["release_date_precision"]) {
	case "year":
		releaseAt, err = time.Parse("2006", releaseDate)
	case "month":
		releaseAt, err = time.Parse("2006-01", releaseDate)
	default:
		releaseAt, err = time.Parse("2006-01-02", releaseDate)
	}
	if err != nil {
		fmt.Printf("failed to parse kungal release date: %v, releaseDate: %s\n", err, releaseDate)
		return time.Time{}
	}
	return releaseAt
}

// applyKungalExternalSourceIDs 从 external_refs 提取本应用支持的其他源的 ID 写入 SourceList；
// 该源已有 ID 时不覆盖直接跳过（与合并层 MergeSourceList 的不覆盖语义一致）。
func applyKungalExternalSourceIDs(game *models.Game, data []interface{}, work map[string]interface{}) {
	for _, item := range nuxtList(data, work["external_refs"]) {
		entry := nuxtMap(data, item)
		if entry == nil {
			continue
		}
		site := nuxtAsString(data, entry["site"])
		id := nuxtAsString(data, entry["external_id"])
		if id == "" {
			continue
		}
		var st enums.SourceType
		switch site {
		case "bangumi":
			st = enums.Bangumi
		case "dmm":
			st = enums.Dmm
		case "dlsite":
			st = enums.Dlsite
		case "erogamescape":
			// 批评空间使用数字 ID，与 Eroscape 抓取器的 GameId 格式一致
			st = enums.Eroscape
		default:
			// steam 等本应用不支持的源跳过
			continue
		}
		if game.GetSourceID(st) != "" {
			continue
		}
		game.SetSourceID(st, id)
	}
}

func (g KungalInfoGetter) convertToModel(data []interface{}, work map[string]interface{}, workID string) (models.Game, error) {
	name := g.pickKungalTitle(data, work)
	summary := g.pickKungalIntro(data, work)
	releaseAt := g.parseKungalReleaseDate(data, work)

	coverURL := ""
	if cover := nuxtMap(data, work["cover"]); cover != nil {
		coverURL = nuxtAsString(data, cover["url"])
	}

	company := ""
	if maker := nuxtMap(data, work["maker"]); maker != nil {
		company = nuxtAsString(data, maker["display_name"])
	}

	images := make([]string, 0)
	for _, item := range nuxtList(data, work["screenshots"]) {
		entry := nuxtMap(data, item)
		if entry == nil {
			continue
		}
		if img := nuxtMap(data, entry["image"]); img != nil {
			if u := nuxtAsString(data, img["url"]); u != "" {
				images = append(images, u)
			}
		}
	}

	game := models.Game{
		Name:       name,
		CoverURL:   coverURL,
		Company:    company,
		Summary:    summary,
		Images:     strings.Join(images, ","),
		SourceType: enums.Kungal,
		SourceID:   workID,
		ReleaseAt:  releaseAt,
		CachedAt:   time.Now(),
	}
	game.SetSourceID(enums.Kungal, workID)
	applyKungalExternalSourceIDs(&game, data, work)
	return game, nil
}

func (g KungalInfoGetter) buildTags(data []interface{}, work map[string]interface{}, game models.Game) map[string][]models.Tag {
	tagMap := make(map[string][]models.Tag)
	if game.Company != "" {
		tagMap[models.TagCategoryBrand] = []models.Tag{
			{Category: models.TagCategoryBrand, Name: game.Company, BlockModify: true},
		}
	}
	genreTags := make([]models.Tag, 0)
	for _, item := range nuxtList(data, work["tags"]) {
		entry := nuxtMap(data, item)
		if entry == nil {
			continue
		}
		name := nuxtAsString(data, entry["display_name"])
		if name == "" {
			continue
		}
		genreTags = append(genreTags, models.Tag{Category: models.TagCategoryGenre, Name: name, BlockModify: true})
	}
	if len(genreTags) > 0 {
		tagMap[models.TagCategoryGenre] = genreTags
	}
	return tagMap
}

// mapKungalStaffRole 将 Kungal credits 的 role_key 映射为应用内 staff 角色。
// voice-actor 由 roster 的 voices 处理，developer/publisher 已在 Company 中体现，均跳过。
func mapKungalStaffRole(roleKey string) (enums.StaffRole, bool) {
	switch roleKey {
	case "scenario":
		return enums.Sceneario, true
	case "director":
		return enums.Director, true
	case "music":
		return enums.Composer, true
	case "character-design":
		return enums.CharaDesign, true
	case "illustration":
		return enums.Art, true
	case "singer", "theme-song":
		return enums.Singer, true
	case "voice-actor", "developer", "publisher":
		return "", false
	default:
		// other-staff 等
		return enums.Staff, true
	}
}

func (g KungalInfoGetter) FetchEntity(request vo.MetadataRequest, token string) (models.GameEntity, error) {
	gameEntity := models.GameEntity{}
	if request.ID == "" {
		return gameEntity, fmt.Errorf("kungal ID is required to fetch metadata")
	}

	html, err := g.getHTML("/galgame/" + request.ID)
	if err != nil {
		return gameEntity, fmt.Errorf("failed to fetch kungal page %s: %w", request.ID, err)
	}
	data, err := parseNuxtData(html)
	if err != nil {
		return gameEntity, fmt.Errorf("failed to parse kungal page %s: %w", request.ID, err)
	}
	work := findKungalWork(data)
	if work == nil {
		return gameEntity, fmt.Errorf("kungal 页面未找到作品数据（ID 可能不存在）: %s", request.ID)
	}

	game, err := g.convertToModel(data, work, request.ID)
	if err != nil {
		return gameEntity, err
	}
	game.ID = request.DbGameId

	gameEntity.Game = game
	gameEntity.Tags = g.buildTags(data, work, game)

	works := make(map[enums.StaffRole][]models.Work)

	// 角色 + 声优（来自 roster）
	if request.ShouldFetchCharactors {
		for i, item := range nuxtList(data, work["roster"]) {
			entry := nuxtMap(data, item)
			if entry == nil {
				continue
			}
			charaName := nuxtAsString(data, entry["display_name"])
			if g.searchCn {
				if trans := nuxtLocalizedValue(data, entry["localized"], "zh-Hans"); trans != "" {
					charaName = trans
				}
			}
			if charaName == "" {
				continue
			}
			charaID := nuxtAsString(data, entry["id"])
			workItem := models.Work{
				CharactorName:     charaName,
				SourceCharactorId: charaID,
				SourceGameId:      request.ID,
				GameId:            request.DbGameId,
				Sort:              i,
				Role:              enums.Charactor,
			}
			if img := nuxtMap(data, entry["image"]); img != nil {
				workItem.CharactorImage = nuxtAsString(data, img["url"])
			}
			works[workItem.Role] = append(works[workItem.Role], workItem)

			// 配音演员（取首个）
			for _, voiceItem := range nuxtList(data, entry["voices"]) {
				voice := nuxtMap(data, voiceItem)
				if voice == nil {
					continue
				}
				cvName := nuxtAsString(data, voice["display_name"])
				if cvName == "" {
					continue
				}
				cvWork := models.Work{
					StaffName:         cvName,
					SourceStaffId:     nuxtAsString(data, voice["id"]),
					SourceGameId:      request.ID,
					GameId:            request.DbGameId,
					Role:              enums.CV,
					SourceCharactorId: charaID,
				}
				works[enums.CV] = append(works[enums.CV], cvWork)
				break
			}
		}
	}

	// 制作人员（来自 credits，跳过声优/开发/发行）
	if request.ShouldFetchStaffs {
		for _, creditItem := range nuxtList(data, work["credits"]) {
			credit := nuxtMap(data, creditItem)
			if credit == nil {
				continue
			}
			roleKey := nuxtAsString(data, credit["role_key"])
			role, ok := mapKungalStaffRole(roleKey)
			if !ok {
				continue
			}
			for _, personItem := range nuxtList(data, credit["people"]) {
				person := nuxtMap(data, personItem)
				if person == nil {
					continue
				}
				staffName := nuxtAsString(data, person["display_name"])
				if staffName == "" {
					continue
				}
				workItem := models.Work{
					StaffName:     staffName,
					SourceStaffId: nuxtAsString(data, person["id"]),
					SourceGameId:  request.ID,
					GameId:        request.DbGameId,
					Role:          role,
				}
				works[role] = append(works[role], workItem)
			}
		}
	}

	gameEntity.WorksMap = works
	return gameEntity, nil
}

func (g KungalInfoGetter) FetchMetadata(id string, token string) (models.Game, error) {
	req := vo.MetadataRequest{
		ID:     id,
		Source: enums.Kungal,
	}
	gameEntity, err := g.FetchEntity(req, token)
	if err != nil {
		return models.Game{}, err
	}
	return gameEntity.Game, nil
}

// FetchMetadataByName Kungal 搜索接口需要登录且搜索页 SSR 不含结果数据，无法匿名按名称搜索
func (g KungalInfoGetter) FetchMetadataByName(name string, token string) (models.Game, error) {
	return models.Game{}, fmt.Errorf("kungal 不支持按名称搜索（站点搜索需要登录），请使用条目 ID（如 %s/galgame/210350 中的 210350）", kungalSiteBaseURL)
}
