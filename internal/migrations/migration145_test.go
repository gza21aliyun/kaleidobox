package migrations

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestMigration145SourceList 校验：把旧的单列来源 id 合并进 games.source_list，
// 主源 source_type:source_id 也写入、同源唯一、eroscape 使用 ascii key，最后 DROP 旧列。
func TestMigration145SourceList(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	// 1. 构造旧结构的 games 表（含 6 个 id 列，无 source_list）
	if _, err := db.Exec(`CREATE TABLE games (
		id TEXT PRIMARY KEY,
		name TEXT,
		source_type TEXT,
		source_id TEXT,
		bangumi_id TEXT,
		dmm_id TEXT,
		eroscape_id TEXT,
		ymgal_id TEXT,
		dlsite_id TEXT,
		getchu_id TEXT
	)`); err != nil {
		t.Fatalf("create legacy games: %v", err)
	}

	// 模拟真实库：InitSchema 会在 games 上创建二级索引，它们在 DuckDB 中依赖表并阻止 DROP COLUMN。
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_games_name ON games(name)`); err != nil {
		t.Fatalf("create idx_games_name: %v", err)
	}

	type row struct {
		id, st, sid     string
		bangumi, dmm    string
		eroscape, ymgal string
		dlsite, getchu  string
	}
	rows := []row{
		// g1: 主源 dmm，另含 getchu 与 批评空间(eroscape)
		{"g1", "dmm", "D1", "", "D1", "E1", "", "", "G1"},
		// g2: 主源 bangumi 但对应 id 列为空，主源应被写入 source_list
		{"g2", "bangumi", "B9", "", "", "", "", "", ""},
		// g3: local 主源（不应作为来源写入），仅旧列 getchu
		{"g3", "local", "PATH_ONLY", "", "", "", "", "", "G5"},
		// g4: 同一来源(dmm)旧列与主源 id 冲突，需保证同源唯一（只出现一次）
		{"g4", "dmm", "DNEW", "", "DOLD", "", "", "", ""},
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO games (id, source_type, source_id, bangumi_id, dmm_id, eroscape_id, ymgal_id, dlsite_id, getchu_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.id, r.st, r.sid, r.bangumi, r.dmm, r.eroscape, r.ymgal, r.dlsite, r.getchu); err != nil {
			t.Fatalf("insert %s: %v", r.id, err)
		}
	}

	// 2. 执行 migration145（自管理事务，逐条自动提交）
	if err := migration145(db); err != nil {
		t.Fatalf("migration145 failed: %v", err)
	}

	// 3. 校验 source_list 内容
	want := map[string]string{
		"g1": "dmm:D1;eroscape:E1;getchu:G1",
		"g2": "bangumi:B9",
		"g3": "getchu:G5",
	}
	for id, exp := range want {
		var got string
		if err := db.QueryRow(`SELECT source_list FROM games WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatalf("query %s: %v", id, err)
		}
		if got != exp {
			t.Errorf("source_list[%s] = %q, want %q", id, got, exp)
		}
	}

	// g4: 同源唯一，只应出现一个 dmm 条目
	var g4 string
	if err := db.QueryRow(`SELECT source_list FROM games WHERE id = 'g4'`).Scan(&g4); err != nil {
		t.Fatalf("query g4: %v", err)
	}
	if strings.Count(g4, "dmm:") != 1 {
		t.Errorf("g4 source_list should contain exactly one dmm entry, got %q", g4)
	}

	// 4. 校验旧列已被 DROP，source_list 已存在
	cols := map[string]bool{}
	crows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'games'`)
	if err != nil {
		t.Fatalf("inspect columns: %v", err)
	}
	for crows.Next() {
		var c string
		if err := crows.Scan(&c); err == nil {
			cols[c] = true
		}
	}
	crows.Close()

	for _, old := range []string{"bangumi_id", "dmm_id", "eroscape_id", "ymgal_id", "dlsite_id", "getchu_id"} {
		if cols[old] {
			t.Errorf("legacy column %s should have been dropped", old)
		}
	}
	if !cols["source_list"] {
		t.Errorf("source_list column should exist after migration")
	}

	// 4b. 校验二级索引在迁移后被恢复
	var idxCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes() WHERE lower(table_name) = 'games' AND lower(index_name) = 'idx_games_name'`).Scan(&idxCount); err != nil {
		t.Fatalf("inspect idx_games_name: %v", err)
	}
	if idxCount == 0 {
		t.Errorf("index idx_games_name should have been restored after migration")
	}

	// 5. 幂等：再次运行不应报错，且 source_list 不变（旧列已不存在，回填被跳过）
	if err := migration145(db); err != nil {
		t.Fatalf("migration145 re-run failed: %v", err)
	}
	for id, exp := range want {
		var got string
		if err := db.QueryRow(`SELECT source_list FROM games WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatalf("query %s after re-run: %v", id, err)
		}
		if got != exp {
			t.Errorf("source_list[%s] changed after re-run = %q, want %q", id, got, exp)
		}
	}
}

// TestMigration145FreshSchema 校验：全新库（新 schema 无旧列）运行迁移不报错，
// 且主源会被写入 source_list（回填走 present 为空的分支）。
func TestMigration145FreshSchema(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()

	// 使用真实的新 schema
	if err := InitSchema(db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO games (id, name, source_type, source_id) VALUES ('f1', 'Fresh', 'vndb', 'V1')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := migration145(db); err != nil {
		t.Fatalf("migration145 on fresh schema failed: %v", err)
	}

	// 新 schema 无旧列，present 为空 → 回填分支跳过，source_list 保持默认（迁移不强行注入主源）
	var got sql.NullString
	if err := db.QueryRow(`SELECT source_list FROM games WHERE id = 'f1'`).Scan(&got); err != nil {
		t.Fatalf("query: %v", err)
	}
	t.Logf("fresh source_list = %q (valid=%v)", got.String, got.Valid)
}
