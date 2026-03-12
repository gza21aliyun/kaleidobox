package models

import "time"

type GameReview struct {
	Id          string   `json:"id"`
	Points      string   `json:"points"`
	TotalPoints string   `json:"total_points"`
	Reviews     []Review `json:"reviews"`
}

type Review struct {
	Id          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Link        string    `json:"link"`
	Points      string    `json:"points"`
	TotalPoints string    `json:"total_points"`
	Date        time.Time `json:"date"`
}
