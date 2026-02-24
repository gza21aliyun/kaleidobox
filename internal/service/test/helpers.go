package test

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupTestDB 创建测试数据库（供所有 service 测试使用）
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// 使用内存数据库进行测试
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("无法打开测试数据库: %v", err)
	}

	// 创建测试表结构
	initTestSchema(t, db)

	// 返回清理函数
	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

// initTestSchema 初始化测试表结构
func initTestSchema(t *testing.T, db *sql.DB) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id TEXT PRIMARY KEY,
			name TEXT,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
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
			cached_at TIMESTAMP,
			source_id TEXT,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			tags TEXT,
			arguments TEXT,
			images TEXT,
			bangumi_id TEXT,
			dmm_id TEXT,
			ymgal_id TEXT,
			eroscape_id TEXT,
			search_name TEXT,
			staffs TEXT,
			release_at TIMESTAMP,
			related_games TEXT,
			use_locale_emulator BOOLEAN DEFAULT FALSE,
			use_magpie BOOLEAN DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS game_categories (
			game_id TEXT,
			category_id TEXT,
			PRIMARY KEY (game_id, category_id)
		)`,
		`CREATE TABLE IF NOT EXISTS play_sessions (
			id TEXT PRIMARY KEY,
			game_id TEXT,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			duration INTEGER
		)`,
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
			gender INTEGER
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
			game_cover TEXT
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
		`CREATE TABLE IF NOT EXISTS image_backup (
			url TEXT PRIMARY KEY,
			local_path TEXT
		)`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			t.Fatalf("创建测试表失败: %v", err)
		}
	}
}
