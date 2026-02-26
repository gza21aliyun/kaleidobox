package models

type Tag struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Group       string `json:"group"`
	IsH         bool   `json:"is_h"`
	IsSpoiler   bool   `json:"is_spoiler"`
	BlockModify bool   `json:"block_modify"`
}

func (t Tag) GetName() string {
	return t.Name
}

const (
	TagCategoryBrand        = "品牌"
	TagCategoryGenre        = "游戏类型"
	TagCategoryPlatform     = "平台"
	TagCategoryPublisher    = "发行"
	TagCategoryPlayerNumber = "玩家人数"
	TagCategoryOther        = "其他"
	TagCategoryProperty     = "属性"
	TagCategoryCustom       = "自定义"
	TagCategorySeries       = "系列"
)
