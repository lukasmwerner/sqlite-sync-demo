package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/mattn/go-sqlite3"
)

func mustNot(err error, tags ...string) {
	if err != nil {
		log.Fatalln(strings.Join(tags, " "), err)
	}
}

func main() {
	// setup cr-sqlite
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

	_, err = db.Exec("select crsql_as_crr('todos');")
	mustNot(err)

	// setup changes storage
	cwd, err := os.Getwd()
	mustNot(err)
	changesPath := path.Join(cwd, "todo-changes")
	if _, err = os.Stat(changesPath); os.IsNotExist(err) {
		os.Mkdir(changesPath, 0775)
	}

	watcher, err := fsnotify.NewWatcher()
	mustNot(err)
	mustNot(watcher.Add(changesPath))

	hostname, err := os.Hostname()
	mustNot(err)
	hostChangesPath := path.Join(changesPath, hostname)
	err = syncronizeLocalChangesToDisk(db, hostChangesPath)
	mustNot(err, "sync changes DB -> disk")

	syncronizeFromHostsToDB(db, hostname, changesPath)

	// pre-populate data from database
	items := []item{}
	rows, err := db.Query("SELECT id, description, completed FROM todos;")
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

	m := NewModel(hostname, items)

	// handle database interactions
	m.OnNew = func(i item) {
		_, err := db.Exec("INSERT INTO todos (id, description, completed) VALUES (?, ?, ?)", i.ID, i.Description, i.Done)
		mustNot(err)

		err = syncronizeLocalChangesToDisk(db, hostChangesPath)
		mustNot(err)
	}
	m.OnUpdate = func(i item) {
		_, err := db.Exec("UPDATE todos SET description = ?, completed = ? WHERE id = ?", i.Description, i.Done, i.ID)
		mustNot(err)

		err = syncronizeLocalChangesToDisk(db, hostChangesPath)
		mustNot(err)
	}
	m.OnDelete = func(i item) {
		_, err := db.Exec("DELETE FROM todos WHERE id = ?", i.ID)
		mustNot(err)

		err = syncronizeLocalChangesToDisk(db, hostChangesPath)
		mustNot(err)
	}
	m.Refresh = func() []item {
		syncronizeFromHostsToDB(db, hostname, changesPath)
		items := []item{}
		rows, err := db.Query("SELECT id, description, completed FROM todos;")
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
		return items
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		for {
			select {
			case err := <-watcher.Errors:
				fmt.Println("had error watching", err.Error())
			case event := <-watcher.Events:
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
					p.Send(refreshMsg{})
				}
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		return
	}

	err = syncronizeLocalChangesToDisk(db, hostChangesPath)
	mustNot(err)
}

func syncronizeFromHostsToDB(db *sql.DB, hostname, changesPath string) {

	// Synchronize any new changes
	hosts, err := os.ReadDir(changesPath)
	mustNot(err)
	for _, host := range hosts {
		if host.Name() == hostname || path.Ext(host.Name()) != ".changes" {
			continue
		}
		if host.IsDir() {
			continue
		}
		err := syncronizeFromDiskToDB(db, path.Join(changesPath, host.Name()))
		mustNot(err, "sync Disk -> DB", host.Name())
	}
}

type crsql_changes struct {
	Table       string
	Pk          []byte
	Cid         string
	Value       []byte
	Col_version int
	Db_version  int
	Site_id     []byte
	Cl          int
	Seq         int
}

func syncronizeFromDiskToDB(db *sql.DB, hostFile string) error {

	f, err := os.Open(hostFile)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var changes []crsql_changes
	err = json.Unmarshal(b, &changes)
	if err != nil {
		return err
	}
	for _, change := range changes {
		_, err := db.Exec("INSERT INTO crsql_changes VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			change.Table,
			change.Pk,
			change.Cid,
			change.Value,
			change.Col_version,
			change.Db_version,
			change.Site_id,
			change.Cl,
			change.Seq,
		)
		if err != nil {
			return err
		}

	}

	return nil
}

func syncronizeLocalChangesToDisk(db *sql.DB, hostFile string) error {
	rows, err := db.Query("SELECT * FROM crsql_changes where site_id = crsql_site_id();")
	if err != nil {
		return err
	}
	changes := []crsql_changes{}
	for rows.Next() {
		var table string
		var pk []byte
		var cid string
		var value []byte
		var col_version int
		var db_version int
		var site_id []byte
		var cl int
		var seq int
		err := rows.Scan(&table, &pk, &cid, &value, &col_version, &db_version, &site_id, &cl, &seq)
		if err != nil {
			return err
		}
		changes = append(changes, crsql_changes{
			Table:       table,
			Pk:          pk,
			Cid:         cid,
			Value:       value,
			Col_version: col_version,
			Db_version:  db_version,
			Site_id:     site_id,
			Cl:          cl,
			Seq:         seq,
		})
	}
	f, err := os.Create(hostFile+".changes")
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(&changes)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err != nil {
		return err
	}

	return nil
}
