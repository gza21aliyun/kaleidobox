package utils

import (
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"testing"
)

// 用站点真实数据验证外部源 ID 提取
func TestApplyExternalSourceIDs(t *testing.T) {
	g := HikarinagiInfoGetter{}
	game := &models.Game{}
	vndbID := int64(16560)
	bgmid := int64(121488)
	gameDto := &HikarinagiGalgame{
		ID:            1,
		VndbID:        &vndbID,
		BangumiGameID: &bgmid,
		ExternalLinks: []HikarinagiExternalLink{
			{Name: "getchu", URL: "http://www.getchu.com/soft.phtml?id=842090"},
			{Name: "getchu", URL: "http://www.getchu.com/soft.phtml?id=981425"},
			{Name: "dmm", URL: "https://dlsoft.dmm.co.jp/detail/will_0167pack/a2aaaeaaiiia-001"},
			{Name: "dmm", URL: "https://www.dmm.co.jp/mono/pcgame/-/detail/=/cid=1075will381set/"},
			{Name: "dlsite", URL: "https://www.dlsite.com/pro/work/=/product_id/VJ010255.html"},
			{Name: "steam", URL: "https://store.steampowered.com/app/745960/"},
			{Name: "website", URL: "http://www.pulltop.com/full-of-stars/"},
		},
	}
	g.applyExternalSourceIDs(game, gameDto)

	if got := game.GetSourceID(enums.VNDB); got != "v16560" {
		t.Errorf("vndb id = %q, want v16560", got)
	}
	if got := game.GetSourceID(enums.Bangumi); got != "121488" {
		t.Errorf("bangumi id = %q, want 121488", got)
	}
	if got := game.GetSourceID(enums.Getchu); got != "842090" {
		t.Errorf("getchu id = %q, want 842090（取首个，重复不覆盖）", got)
	}
	if got := game.GetSourceID(enums.Dmm); got != "will_0167pack" {
		t.Errorf("dmm id = %q, want will_0167pack（mono 实体页不取）", got)
	}
	if got := game.GetSourceID(enums.Dlsite); got != "VJ010255" {
		t.Errorf("dlsite id = %q, want VJ010255", got)
	}

	// 已有 ID 时不覆盖：模拟 SourceList 已带 vndb
	game2 := &models.Game{SourceList: "vndb:v99999"}
	g.applyExternalSourceIDs(game2, gameDto)
	if got := game2.GetSourceID(enums.VNDB); got != "v99999" {
		t.Errorf("vndb id = %q, want v99999（已有不覆盖）", got)
	}

	// 无关联数据时不报错、不写入任何来源（hikarinagi 自身 ID 由 convertToModel 负责）
	game3 := &models.Game{}
	g.applyExternalSourceIDs(game3, &HikarinagiGalgame{ID: 2})
	if got := game3.SourceList; got != "" {
		t.Errorf("source list = %q, want empty", got)
	}
}
