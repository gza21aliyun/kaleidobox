package migrations

import (
	"database/sql"
	"fmt"
)

func InitSchema(db *sql.DB) error {
	queries := SchemaQueries()

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			fmt.Printf("创建数据库表失败: %v\n", err)
			return err
		}
	}
	return nil
}

func SchemaQueries() []string {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			created_at TIMESTAMPTZ,
			default_backup_target TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id TEXT PRIMARY KEY,
			name TEXT,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ,
			is_system BOOLEAN
		)`,
		`CREATE TABLE IF NOT EXISTS games (
			id TEXT PRIMARY KEY,
			name TEXT,
			cover_url TEXT,
			company TEXT,
			summary TEXT,
			path TEXT,
			save_path TEXT,
			status TEXT DEFAULT 'not_started',
			source_type TEXT,
			cached_at TIMESTAMPTZ,
			source_id TEXT,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ,
			tags TEXT,
			arguments TEXT,
			images TEXT,
			bangumi_id TEXT,
			dmm_id TEXT,
			ymgal_id TEXT,
			eroscape_id TEXT,
			search_name TEXT,
			dlsite_id TEXT,
			release_at TIMESTAMPTZ,
			related_games TEXT,
			use_locale_emulator BOOLEAN DEFAULT FALSE,
			use_magpie BOOLEAN DEFAULT FALSE,
			process_name TEXT,
			getchu_id TEXT,
			pv_path TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS game_categories (
			game_id TEXT,
			category_id TEXT,
			PRIMARY KEY (game_id, category_id)
		)`,
		`CREATE TABLE IF NOT EXISTS play_sessions (
			id TEXT PRIMARY KEY,
			game_id TEXT,
			start_time TIMESTAMPTZ,
			end_time TIMESTAMPTZ,
			duration INTEGER,
			pid INTEGER,
		    process_name TEXT
		)`,
		// 新增 Task 表
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			name TEXT,
			status TEXT,
			type TEXT,
			completed INTEGER,
			total INTEGER,
			working_on TEXT,
			description TEXT,
			warning TEXT,
			deley INTEGER,
			json_data TEXT,
			item_id TEXT,
		)`,

		// 新增 Charactor 表
		`CREATE TABLE IF NOT EXISTS charactors (
			id TEXT PRIMARY KEY,
			name TEXT,
			other_names TEXT,
			image_path TEXT,
			images TEXT,
			source_charactor_id TEXT,
            source_type TEXT,
			game_ids TEXT,
			summary TEXT,
			gender INTEGER,
			measurements TEXT,
			height TEXT,
			sort INTEGER
		)`,
		// 新增 Staff 表
		`CREATE TABLE IF NOT EXISTS staffs (
			id TEXT PRIMARY KEY,
			name TEXT,
			other_names TEXT,
			roles TEXT,
			source_staff_id TEXT,
            source_type TEXT,
			game_ids TEXT,
			summary TEXT,
			gender INTEGER,
			image TEXT
		)`,
		// 新增 Work 表
		`CREATE TABLE IF NOT EXISTS works (
			id TEXT PRIMARY KEY,
			game_id TEXT,
			staff_id TEXT,
			role TEXT,
			charactor_id TEXT,
			charactor_name TEXT,
			staff_name TEXT,
			work_summary TEXT,
            source_type TEXT,
			source_staff_id TEXT,
			source_charactor_id TEXT,
			source_game_id TEXT,
			images TEXT,
			game_name TEXT,
			game_cover TEXT,
			sort INTEGER
		)`,
		// 新增 Tag 表
		`CREATE TABLE IF NOT EXISTS tags (
			name TEXT PRIMARY KEY,
			category TEXT,
			group_name TEXT,
			is_h BOOLEAN DEFAULT FALSE,
			is_spoiler BOOLEAN DEFAULT FALSE,
			block_modify BOOLEAN DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS image_backups (
			url TEXT PRIMARY KEY,
			local_path TEXT,
			subject_id TEXT,
			subject_type INTEGER,
			image_type INTEGER,
			game_id TEXT,
			created_at TIMESTAMPTZ 
		)`,
		// 新增统一的快捷键配置表
		`CREATE TABLE IF NOT EXISTS hotkeys (
			id TEXT PRIMARY KEY,
			game_id TEXT,              -- 当为 'global' 时表示全局配置
			name TEXT NOT NULL,
			device_type TEXT NOT NULL, -- keyboard | dualsense | dualshock4 | joycon | xinput
			key_code TEXT NOT NULL,    -- 按键码
			modifiers TEXT,            -- 修饰键 (ctrl,shift,alt,win等)
			action_type TEXT NOT NULL, -- start_game | stop_game | toggle_pause | screenshot | custom
			action_params TEXT,        -- 动作参数 (JSON格式)
			is_enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ,
		)`,
		// 新增已连接设备表
		`CREATE TABLE IF NOT EXISTS connected_devices (
			id TEXT PRIMARY KEY,
			device_type TEXT NOT NULL, -- keyboard | dualsense | dualshock4 | joycon | xinput
			device_name TEXT NOT NULL,
			device_id TEXT NOT NULL,   -- 设备唯一标识
			is_active BOOLEAN DEFAULT TRUE,
			connected_at TIMESTAMPTZ,
			last_seen_at TIMESTAMPTZ
		)`,
	}
	return queries
}
