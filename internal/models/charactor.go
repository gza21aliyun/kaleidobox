package models

import "lunabox/internal/enums"

type Charactor struct {
	Id                string           `json:"id"`
	Name              string           `json:"name"`
	OtherNames        string           `json:"other_names"`
	ImagePath         string           `json:"image_path"`
	Images            string           `json:"images"`
	SourceCharactorId string           `json:"source_charactor_id"`
	SourceType        enums.SourceType `json:"source_type"`
	GameIds           string           `json:"game_ids"`
	Summary           string           `json:"summary"`
	Gender            int              `json:"gender"`
}

type Staff struct {
	Id            string           `json:"id"`
	Name          string           `json:"name"`
	OtherNames    string           `json:"other_names"`
	Roles         string           `json:"roles"`
	SourceStaffId string           `json:"source_staff_id"`
	SourceType    enums.SourceType `json:"source_type"`
	GameIds       string           `json:"game_ids"`
	Summary       string           `json:"summary"`
	Gender        int              `json:"gender"`
}

type Work struct {
	Id                string           `json:"id"`
	GameId            string           `json:"game_id"`
	StaffId           string           `json:"staff_id"`
	Role              enums.StaffRole  `json:"role"`
	CharactorId       string           `json:"charactor_id"`
	CharactorName     string           `json:"charactor_name"`
	StaffName         string           `json:"staff_name"`
	WorkSummary       string           `json:"work_summary"`
	SourceType        enums.SourceType `json:"source_type"`
	SourceStaffId     string           `json:"source_staff_id"`
	SourceCharactorId string           `json:"source_charactor_id"`
	SourceGameId      string           `json:"source_game_id"`
	Images            string           `json:"images"`
}
