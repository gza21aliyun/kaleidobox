package enums

type TaskType string

const (
	Games          TaskType = "游戏"
	Charactors     TaskType = "角色"
	Staffs         TaskType = "工作人员"
	Images         TaskType = "图片"
	Relations      TaskType = "关系"
	VideoPaths     TaskType = "视频路径"
	RefreshGames   TaskType = "刷新游戏"
	DownloadFiles  TaskType = "下载文件处理"
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
	{VideoPaths, "VIDEO_PATHS"},
	{RefreshGames, "REFRESH_GAMES"},
	{DownloadFiles, "DOWNLOAD_FILES"},
}
