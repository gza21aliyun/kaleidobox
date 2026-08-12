package models

import (
	"lunabox/internal/enums"
)

// 给TaskService持有数据用
type Task struct {
	Id          string           `json:"id"`
	Name        string           `json:"name"`
	Status      enums.TaskStatus `json:"status"`
	Type        enums.TaskType   `json:"type"`
	Completed   int              `json:"completed"`
	Total       int              `json:"total"`
	WorkingOn   string           `json:"working_on"`
	Description string           `json:"description"`
	Title       string           `json:"title"`
	Warning     string           `json:"warning"`
	Deley       int              `json:"deley"`
	JsonData    string           `json:"json_data"`
	Data        interface{}      `json:"data"`
	ItemId      string           `json:"item_id"`
	ItemStatus  enums.TaskStatus `json:"item_status"`
	ItemData    interface{}      `json:"item_data"`
	ResultGames []ResultGames    `json:"result_games"`
}

// 发给前端用
type TaskNotice struct {
	Id          string           `json:"id"`
	Name        string           `json:"name"`
	Status      enums.TaskStatus `json:"status"`
	Type        enums.TaskType   `json:"type"`
	Completed   int              `json:"completed"`
	Total       int              `json:"total"`
	Title       string           `json:"title"`
	WorkingOn   string           `json:"working_on"`
	Description string           `json:"description"`
	Warning     string           `json:"warning"`
	ItemId      string           `json:"item_id"`
	ItemStatus  enums.TaskStatus `json:"item_status"`
	ItemData    interface{}      `json:"item_data"`
	ResultGames []ResultGames    `json:"result_games"`
}

type ResultGames struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      int      `json:"status"` //200 成功, 400 失败, 300 跳过
	GameIds     []string `json:"game_ids"`
}
