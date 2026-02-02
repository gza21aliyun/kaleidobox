package enums

type TaskStatus string

const (
	Initial   TaskStatus = "初始"
	Started   TaskStatus = "已开始"
	Paused    TaskStatus = "暂停"
	Completed TaskStatus = "完成"
	Error     TaskStatus = "错误"
	Canceled  TaskStatus = "取消"
)

var AllTaskStatus = []struct {
	Value  TaskStatus
	TSName string
}{
	{Initial, "INITIAL"},
	{Started, "STARTED"},
	{Paused, "PAUSED"},
	{Completed, "COMPLETED"},
	{Error, "ERROR"},
	{Canceled, "CANCELED"},
}
