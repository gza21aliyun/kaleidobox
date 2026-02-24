package models

import (
	"lunabox/internal/enums"
	"time"
)

type Game struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	CoverURL          string           `json:"cover_url"`
	Company           string           `json:"company"`
	Summary           string           `json:"summary"`
	Path              string           `json:"path"`        // 启动路径
	SavePath          string           `json:"save_path"`   // 存档目录路径
	Status            enums.GameStatus `json:"status"`      // 游戏状态: not_started, playing, completed, on_hold
	SourceType        enums.SourceType `json:"source_type"` // "local", "bangumi", "vndb"
	CachedAt          time.Time        `json:"cached_at"`
	SourceID          string           `json:"source_id"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	Tags              string           `json:"tags"`
	Arguments         string           `json:"arguments"`
	Images            string           `json:"images"`
	BangumiId         string           `json:"bangumi_id"`
	DmmId             string           `json:"dmm_id"`
	EroscapeId        string           `json:"eroscape_id"`
	YmgalId           string           `json:"ymgal_id"`
	SearchName        string           `json:"search_name"`
	Staffs            string           `json:"staffs"`
	ReleaseAt         time.Time        `json:"release_at"`
	RelatedGames      string           `json:"related_games"`
	UseLocaleEmulator bool             `json:"use_locale_emulator"` // 是否使用 Locale Emulator 转区启动
	UseMagpie         bool             `json:"use_magpie"`          // 是否使用 Magpie 超分辨率缩放

}

type GameEntity struct {
	Game     Game                       `json:"game"`
	WorksMap map[enums.StaffRole][]Work `json:"worksMap"`
	Tags     map[string][]Tag           `json:"tags"`
}

// GameBackup 游戏存档备份记录（基于文件系统，不使用数据库）
type GameBackup struct {
	Path      string    `json:"path"` // 备份文件路径（作为唯一标识）
	Name      string    `json:"name"` // 文件名
	GameID    string    `json:"game_id"`
	Size      int64     `json:"size"`       // 备份文件大小（字节）
	CreatedAt time.Time `json:"created_at"` // 创建时间（来自文件修改时间）
}

type ImageBackup struct {
	Url       string `json:"url"`
	LocalPath string `json:"local_path"`
}

type GameFilter struct {
}
