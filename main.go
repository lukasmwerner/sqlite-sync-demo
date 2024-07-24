package main

import (
	"database/sql"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-sqlite3"
)

func mustNot(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	sql.Register("cr-sqlite", &sqlite3.SQLiteDriver{
		Extensions: []string{"crsqlite"},
	})

	db, err := sql.Open("cr-sqlite", "./todos.db")
	mustNot(err)
	defer func(db *sql.DB) {
		db.Exec(`select crsql_finalize();`) // Clean up after cr-sqlite
		db.Close()
	}(db)

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS todos (id PRIMARY KEY NOT NULL, description, completed);")
	mustNot(err)

	items := []item{}

	rows, err := db.Query("select id, description, completed from todos;")
	mustNot(err)
	for rows.Next() {
		var id int
		var description string
		var completed bool
		err = rows.Scan(&id, &description, &completed)
		mustNot(err)
		items = append(items, item{
			ID:          id,
			Description: description,
			Done:        completed,
		})
	}

	m := NewModel(items)

	m.OnNew = func(i item) {
		_, err := db.Exec("INSERT INTO todos (id, description, completed) VALUES (?, ?, ?)", i.ID, i.Description, i.Done)
		mustNot(err)
	}
	m.OnUpdate = func(i item) {
		_, err := db.Exec("UPDATE todos SET description = ?, completed = ? WHERE id = ?", i.Description, i.Done, i.ID)
		mustNot(err)
	}
	m.OnDelete = func(i item) {
		_, err := db.Exec("DELETE FROM todos WHERE id = ?", i.ID)
		mustNot(err)
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		return
	}

	return
}
