package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func mustNot(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	db, err := sql.Open("sqlite3", "./todos.db")
	mustNot(err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE todos (id PRIMARY KEY NOT NULL, description, completed);`)
	mustNot(err)
	_, err = db.Exec(`INSERT INTO todos (id, description, completed) VALUES (?, ?, ?);`, 1, "Get Toast", true)
	mustNot(err)
	_, err = db.Exec(`INSERT INTO todos (id, description, completed) VALUES (?, ?, ?);`, 2, "Get Butter", false)
	mustNot(err)
	_, err = db.Exec(`INSERT INTO todos (id, description, completed) VALUES (?, ?, ?);`, 3, "Get Nutella", false)
	mustNot(err)
	_, err = db.Exec(`INSERT INTO todos (id, description, completed) VALUES (?, ?, ?);`, 4, "Make a slice of Nutella toast", false)
	mustNot(err)
}
