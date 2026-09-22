// dbcheck: temporary diagnostic tool to inspect created_at / timestamp
// columns in gline.db (tasks/messages) and sessions.db (ADK sessions/events).
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
)

func q(db *sql.DB, label, query string) {
	fmt.Printf("--- %s ---\n%s\n", label, query)
	rows, err := db.Query(query)
	if err != nil {
		fmt.Println("  ERROR:", err)
		return
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	n := 0
	for rows.Next() {
		_ = rows.Scan(ptrs...)
		if n < 5 {
			for i, c := range cols {
				v := vals[i]
				if b, ok := v.([]byte); ok {
					v = string(b)
				}
				fmt.Printf("  %s=%v", c, v)
			}
			fmt.Println()
		}
		n++
	}
	fmt.Printf("  (%d rows sampled)\n", n)
}

func main() {
	fix := len(os.Args) > 1 && os.Args[1] == "-fix"
	home, _ := os.UserHomeDir()

	glineDB, err := sql.Open("sqlite", filepath.Join(home, ".gline", "gline.db"))
	if err != nil {
		panic(err)
	}
	defer glineDB.Close()

	fmt.Println("========== gline.db ==========")
	q(glineDB, "tables", `SELECT name FROM sqlite_master WHERE type='table'`)
	q(glineDB, "tasks: created_at null count", `SELECT COUNT(*) AS total, SUM(CASE WHEN created_at IS NULL OR created_at='' THEN 1 ELSE 0 END) AS null_created, SUM(CASE WHEN updated_at IS NULL OR updated_at='' THEN 1 ELSE 0 END) AS null_updated FROM tasks`)
	q(glineDB, "messages: created_at null count", `SELECT COUNT(*) AS total, SUM(CASE WHEN created_at IS NULL OR created_at='' THEN 1 ELSE 0 END) AS null_created, SUM(CASE WHEN created_at LIKE '0001-01-01%' THEN 1 ELSE 0 END) AS zero_time FROM messages`)
	q(glineDB, "sample tasks", `SELECT id, substr(title,1,30) AS title, created_at, updated_at FROM tasks ORDER BY rowid DESC LIMIT 5`)
	q(glineDB, "sample messages", `SELECT id, task_id, role, created_at FROM messages ORDER BY id DESC LIMIT 5`)

	if fix {
		res, err := glineDB.Exec(`UPDATE messages SET created_at = (
			SELECT t.created_at FROM tasks t WHERE t.id = messages.task_id
		) WHERE created_at LIKE '0001-01-01%'`)
		if err != nil {
			fmt.Println("BACKFILL ERROR:", err)
		} else {
			n, _ := res.RowsAffected()
			fmt.Printf("BACKFILL: %d messages updated from task created_at\n", n)
		}
	}

	sessDB, err := sql.Open("sqlite", filepath.Join(home, ".gline", "sessions.db"))
	if err != nil {
		panic(err)
	}
	defer sessDB.Close()

	fmt.Println("========== sessions.db ==========")
	q(sessDB, "tables", `SELECT name FROM sqlite_master WHERE type='table'`)
	q(sessDB, "sessions: create_time null count", `SELECT COUNT(*) AS total, SUM(CASE WHEN create_time IS NULL THEN 1 ELSE 0 END) AS null_create, SUM(CASE WHEN update_time IS NULL THEN 1 ELSE 0 END) AS null_update FROM sessions`)
	q(sessDB, "events: timestamp null count", `SELECT COUNT(*) AS total, SUM(CASE WHEN timestamp IS NULL THEN 1 ELSE 0 END) AS null_ts FROM events`)
	q(sessDB, "sample sessions", `SELECT id, create_time, update_time FROM sessions ORDER BY rowid DESC LIMIT 5`)
	q(sessDB, "sample events", `SELECT id, author, timestamp FROM events ORDER BY rowid DESC LIMIT 5`)
}
