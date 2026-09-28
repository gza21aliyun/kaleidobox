package enums

import "encoding/json"

type SourceType string

const (
	Local      SourceType = "local"
	Bangumi    SourceType = "bangumi"
	VNDB       SourceType = "vndb"
	Ymgal      SourceType = "ymgal"
	Dmm        SourceType = "dmm"
	Eroscape   SourceType = "批评空间"
	Dlsite     SourceType = "dlsite"
	Getchu     SourceType = "getchu"
	Hikarinagi SourceType = "hikarinagi"
	Kungal     SourceType = "kungal"
)

var AllSourceTypes = []struct {
	Value  SourceType
	TSName string
}{
	{Local, "LOCAL"},
	{Bangumi, "BANGUMI"},
	{VNDB, "VNDB"},
	{Ymgal, "YMGAL"},
	{Dmm, "DMM"},
	{Eroscape, "EROSCAPE"},
	{Dlsite, "DLSITE"},
	{Getchu, "GETCHU"},
	{Hikarinagi, "HIKARINAGI"},
	{Kungal, "KUNGAL"},
}

// Key 返回用于 source_list 编码的稳定 ascii 标识。
// 大多数来源的枚举 value 本身即 ascii，可直接复用；
// 只有 Eroscape 的 value 是中文"批评空间"，这里统一映射为 "eroscape"。
func (s SourceType) Key() string {
	if s == Eroscape {
		return "eroscape"
	}
	return string(s)
}

// SourceTypeFromKey 将 source_list 中的 ascii 标识还原为 SourceType
func SourceTypeFromKey(key string) SourceType {
	if key == "eroscape" {
		return Eroscape
	}
	return SourceType(key)
}

func (s SourceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func (s *SourceType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = SourceType(str)
	return nil
}

type GuideSource string

const (
	Seiya GuideSource = "seiya"
	Fool  GuideSource = "fool"
)

var AllGuideSources = []struct {
	Value  GuideSource
	TSName string
}{
	{Seiya, "SEIYA"},
	{Fool, "FOOL"},
}
