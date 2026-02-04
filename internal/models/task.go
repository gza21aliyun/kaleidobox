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
	Warning     string           `json:"warning"`
	Deley       int              `json:"deley"`
	Data        interface{}      `json:"data"`
	ItemId      string           `json:"item_id"`
	ItemStatus  enums.TaskStatus `json:"item_status"`
	ItemData    interface{}      `json:"item_data"`
}

// 发给前端用
type TaskNotice struct {
	Id          string           `json:"id"`
	Name        string           `json:"name"`
	Status      enums.TaskStatus `json:"status"`
	Type        enums.TaskType   `json:"type"`
	Completed   int              `json:"completed"`
	Total       int              `json:"total"`
	WorkingOn   string           `json:"working_on"`
	Description string           `json:"description"`
	Warning     string           `json:"warning"`
	ItemId      string           `json:"item_id"`
	ItemStatus  enums.TaskStatus `json:"item_status"`
	ItemData    interface{}      `json:"item_data"`
}
