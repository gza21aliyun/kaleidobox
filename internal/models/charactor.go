package models

import "lunabox/internal/enums"

type Charactor struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	OtherNames string `json:"other_names"`
	ImagePath  string `json:"image_path"`
	Images     string `json:"images"`
	BangumiId  string `json:"bangumi_id"`
	EroscapeId string `json:"eroscape_id"`
	YmgalId    string `json:"ymgal_id"`
	VNDBId     string `json:"vndb_id"`
	GameIds    string `json:"game_ids"`
	Summary    string `json:"summary"`
	Gender     int    `json:"gender"`
}

type Staff struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	OtherNames string `json:"other_names"`
	Roles      string `json:"roles"`
	BangumiId  string `json:"bangumi_id"`
	EroscapeId string `json:"eroscape_id"`
	YmgalId    string `json:"ymgal_id"`
	VNDBId     string `json:"vndb_id"`
	GameIds    string `json:"game_ids"`
	Summary    string `json:"summary"`
	Gender     int    `json:"gender"`
}

type Work struct {
	GameId        string          `json:"game_id"`
	StaffId       string          `json:"staff_id"`
	Role          enums.StaffRole `json:"role"`
	CharactorId   string          `json:"charactor_id"`
	CharactorName string          `json:"charactor_name"`
}
