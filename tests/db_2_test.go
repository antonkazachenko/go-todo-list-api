package tests

import (
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

type Task struct {
	ID      int64  `db:"id"`
	Date    string `db:"date"`
	Title   string `db:"title"`
	Comment string `db:"comment"`
	Repeat  string `db:"repeat"`
}

func count(db *sqlx.DB) (int, error) {
	var count int
	return count, db.Get(&count, `SELECT count(id) FROM scheduler`)
}

func openDB(t *testing.T) *sqlx.DB {
	dbfile := DBFile
	envFile := os.Getenv("TODO_DBFILE")
	if len(envFile) > 0 {
		dbfile = envFile
	}
	db, err := sqlx.Connect("sqlite3", dbfile)
	assert.NoError(t, err)
	return db
}

func TestDB(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	before, err := count(db)
	assert.NoError(t, err)

	today := time.Now().Format(`20060102`)

	res, err := db.Exec(`INSERT INTO scheduler (date, title, comment, repeat) 
	VALUES (?, 'Todo', 'Комментарий', '')`, today)
	assert.NoError(t, err)

	id, err := res.LastInsertId()

	var task Task
	err = db.Get(&task, `SELECT * FROM scheduler WHERE id=?`, id)
	assert.NoError(t, err)
	assert.Equal(t, id, task.ID)
	assert.Equal(t, `Todo`, task.Title)
	assert.Equal(t, `Комментарий`, task.Comment)

	_, err = db.Exec(`DELETE FROM scheduler WHERE id = ?`, id)
	assert.NoError(t, err)

	after, err := count(db)
	assert.NoError(t, err)

	assert.Equal(t, before, after)
}
