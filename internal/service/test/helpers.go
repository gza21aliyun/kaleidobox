package test

import (
	"database/sql"
	"lunabox/internal/migrations"
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
	queries := migrations.SchemaQueries()

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			t.Fatalf("创建测试表失败: %v", err)
		}
	}
}
