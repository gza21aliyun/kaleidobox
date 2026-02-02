package enums

type TaskType string

const (
	Games      TaskType = "游戏"
	Charactors TaskType = "角色"
	Staffs     TaskType = "工作人员"
	Images     TaskType = "图片"
	Relations  TaskType = "关系"
)

var AllTaskTypes = []struct {
	Value  TaskType
	TSName string
}{
	{Games, "GAMES"},
	{Charactors, "CHARACTORS"},
	{Staffs, "STAFFS"},
	{Images, "IMAGES"},
	{Relations, "RELATIONS"},
}
