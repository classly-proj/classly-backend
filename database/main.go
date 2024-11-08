package db

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Class struct {
	Name, Description, Instructor, Building string
	ID, Room                                int
}

type User struct {
	Username, PasswordHash string
	Classes                []int
}

const USERS_STATEMENT = `CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL,
	classes TEXT NOT NULL,
);`

const CLASSES_STATEMENT = `CREATE TABLE IF NOT EXISTS classes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	description TEXT NOT NULL,
	instructor TEXT NOT NULL,
	building TEXT NOT NULL,
	room INTEGER NOT NULL,
);`

const INSERT_USER_STATEMENT = `INSERT INTO users (username, password, classes) VALUES (?, ?, ?);`
const INSERT_CLASS_STATEMENT = `INSERT INTO classes (name, description, instructor, building, room) VALUES (?, ?, ?, ?, ?);`

const SELECT_USER_STATEMENT = `SELECT id, username, password, classes FROM users WHERE username = ?;`
const SELECT_CLASS_STATEMENT = `SELECT id, name, description, instructor, building, room FROM classes WHERE id = ?;`

const (
	maxRetries = 5
	baseDelay  = 100 * time.Millisecond
	maxDelay   = 2 * time.Second
)

var db *sql.DB

func OpenDatabase() (*sql.DB, error) {
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("sqlite", "hacknh.db")
		if err == nil {
			break
		}

		time.Sleep(baseDelay * time.Duration(i))
	}

	return db, err
}

func CreateUser(username, password string) (*User, error) {
	_, err := db.Exec(INSERT_USER_STATEMENT, username, password, "")
	if err != nil {
		return nil, err
	}

	return GetUser(username)
}

func GetUser(username string) (*User, error) {
	row := db.QueryRow(SELECT_USER_STATEMENT, username)

	var id int
	var name, password, classes string
	err := row.Scan(&id, &name, &password, &classes)
	if err != nil {
		return nil, err
	}

	var classesList []int = make([]int, 0)

	for _, class := range strings.Split(classes, ",") {
		num, _ := strconv.ParseInt(class, 10, 64)
		classesList = append(classesList, int(num))
	}

	return &User{
		Username:     name,
		PasswordHash: password,
		Classes:      classesList,
	}, nil
}
