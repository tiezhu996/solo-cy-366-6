package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// clauseLocking 返回 MySQL 行锁子句，并发扣款/状态流转时使用 SELECT ... FOR UPDATE。
func clauseLocking() clause.Locking {
	return clause.Locking{Strength: "UPDATE"}
}

// withLocking 对查询追加行锁；MySQL 使用 SELECT ... FOR UPDATE。
// SQLite（仅集成测试环境）不支持行锁语法，直接返回原查询，生产行为不变。
func withLocking(db *gorm.DB) *gorm.DB {
	if db.Dialector.Name() == "sqlite" {
		return db
	}
	return db.Clauses(clauseLocking())
}
