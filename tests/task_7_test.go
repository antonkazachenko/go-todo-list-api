package tests

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func notFoundTask(t *testing.T, id string) {
	body, err := requestJSON("api/task?id="+id, nil, http.MethodGet)
	assert.NoError(t, err)
	var m map[string]any
	err = json.Unmarshal(body, &m)
	assert.NoError(t, err)
	_, ok := m["error"]
	assert.True(t, ok)
}

func TestDone(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	now := time.Now()
	id := addTask(t, task{
		date:  now.Format(`20060102`),
		title: "Свести баланс",
	})

	ret, err := postJSON("api/task/done?id="+id, nil, http.MethodPost)
	assert.NoError(t, err)
	assert.Empty(t, ret)
	notFoundTask(t, id)

	id = addTask(t, task{
		title:  "Проверить работу /api/task/done",
		repeat: "d 3",
	})

	for i := 0; i < 3; i++ {
		ret, err := postJSON("api/task/done?id="+id, nil, http.MethodPost)
		assert.NoError(t, err)
		assert.Empty(t, ret)

		var task Task
		err = db.Get(&task, `SELECT * FROM scheduler WHERE id=?`, id)
		assert.NoError(t, err)
		now = now.AddDate(0, 0, 3)
		assert.Equal(t, task.Date, now.Format(`20060102`))
	}
}

func TestDelTask(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	id := addTask(t, task{
		title:  "Временная задача",
		repeat: "d 3",
	})
	ret, err := postJSON("api/task?id="+id, nil, http.MethodDelete)
	assert.NoError(t, err)
	assert.Empty(t, ret)

	notFoundTask(t, id)

	ret, err = postJSON("api/task", nil, http.MethodDelete)
	assert.NoError(t, err)
	assert.NotEmpty(t, ret)
	ret, err = postJSON("api/task?id=wjhgese", nil, http.MethodDelete)
	assert.NoError(t, err)
	assert.NotEmpty(t, ret)
}
