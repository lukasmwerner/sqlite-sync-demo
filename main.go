package main

import (
	"database/sql"
	"log"

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
		db.Exec(`select crsql_finalize();`) // clean up after cr-sqlite
		db.Close()
	}(db)

	_, err = db.Exec(`CREATE TABLE todos (id PRIMARY KEY NOT NULL, description, completed);`) // make our todos table
	mustNot(err)
	_, err = db.Exec(`select crsql_as_crr('todos');`) //make it CRDT
	mustNot(err)

	// Lets make some toast!
	_, err = db.Exec(`INSERT INTO todos (id, description, completed) VALUES (?, ?, ?);`, 1, "Get Toast", true)
	mustNot(err)
	_, err = db.Exec(`INSERT INTO todos (id, description, completed) VALUES (?, ?, ?);`, 2, "Get Butter", false)
	mustNot(err)
	_, err = db.Exec(`INSERT INTO todos (id, description, completed) VALUES (?, ?, ?);`, 3, "Get Nutella", false)
	mustNot(err)
	_, err = db.Exec(`INSERT INTO todos (id, description, completed) VALUES (?, ?, ?);`, 4, "Make a slice of Nutella toast", false)
	mustNot(err)
}
