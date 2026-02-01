package enums

type StaffRole string

const (
	Staff       StaffRole = "工作人员"
	CV          StaffRole = "声优"
	Sceneario   StaffRole = "剧本"
	Director    StaffRole = "监督"
	Composer    StaffRole = "音乐"
	CharaDesign StaffRole = "人设"
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
}
