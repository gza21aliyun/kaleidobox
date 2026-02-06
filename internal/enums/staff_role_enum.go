package enums

import "encoding/json"

type StaffRole string

const (
	Staff       StaffRole = "工作人员"
	CV          StaffRole = "声优"
	Sceneario   StaffRole = "剧本"
	Director    StaffRole = "监督"
	Composer    StaffRole = "音乐"
	CharaDesign StaffRole = "人设"
	Charactor   StaffRole = "角色"
	Singer      StaffRole = "歌手"
	Art         StaffRole = "画师"
)

var AllStaffRoles = []struct {
	Value  StaffRole
	TSName string
}{
	{Staff, "STAFF"},
	{CV, "CV"},
	{Sceneario, "SCENEARIO"},
	{Director, "DIRECTOR"},
	{Composer, "COMPOSER"},
	{CharaDesign, "CHARA_DESIGN"},
	{Charactor, "CHARACTOR"},
	{Singer, "SINGER"},
	{Art, "ART"},
}

func (s StaffRole) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func (s *StaffRole) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = StaffRole(str)
	return nil
}
