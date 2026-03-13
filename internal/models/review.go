package models

import (
	"lunabox/internal/enums"
	"time"
)

type GameReview struct {
	Id          string           `json:"id"`
	Points      string           `json:"points"`
	TotalPoints string           `json:"total_points"`
	SourceType  enums.SourceType `json:"source_type"`
	Reviews     []Review         `json:"reviews"`
	HasNext     bool             `json:"has_next"`
}

type Review struct {
	Id          string    `json:"id"`
	Title       string    `json:"title"`
	Reviewer    string    `json:"reviewer"`
	Content     string    `json:"content"`
	Link        string    `json:"link"`
	Points      string    `json:"points"`
	TotalPoints string    `json:"total_points"`
	Date        time.Time `json:"date"`
}
