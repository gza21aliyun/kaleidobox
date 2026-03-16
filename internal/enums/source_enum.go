package enums

import "encoding/json"

type SourceType string

const (
	Local    SourceType = "local"
	Bangumi  SourceType = "bangumi"
	VNDB     SourceType = "vndb"
	Ymgal    SourceType = "ymgal"
	Dmm      SourceType = "dmm"
	Eroscape SourceType = "批评空间"
	Dlsite   SourceType = "dlsite"
	Getchu   SourceType = "getchu"
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
