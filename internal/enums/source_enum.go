package enums

type SourceType string

const (
	Local    SourceType = "local"
	Bangumi  SourceType = "bangumi"
	VNDB     SourceType = "vndb"
	Ymgal    SourceType = "ymgal"
	Dmm      SourceType = "dmm"
	Eroscape SourceType = "批评空间"
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
}
