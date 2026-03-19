package models

import "time"

type PlaySession struct {
	ID          string    `json:"id"`
	GameID      string    `json:"game_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Duration    int       `json:"duration"` // seconds
	Pid         int       `json:"pid"`
	ProcessName string    `json:"process_name"`
}
