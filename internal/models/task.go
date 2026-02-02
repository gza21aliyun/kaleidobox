package models

import (
	"lunabox/internal/enums"
)

type Task struct {
	Name        string           `json:"name"`
	Status      enums.TaskStatus `json:"status"`
	Type        enums.TaskType   `json:"type"`
	Completed   int              `json:"completed"`
	Total       int              `json:"total"`
	WorkingOn   string           `json:"working_on"`
	Description string           `json:"description"`
	Warning     string           `json:"warning"`
	Deley       int              `json:"deley"`
}
