package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

func (u *User) JSON() []byte {
	bytes, _ := json.Marshal(map[string]interface{}{
		"username": u.Username,
		"classes":  u.Classes,
	})

	return bytes
}

const USERS_STATEMENT = `CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL,
	classes TEXT NOT NULL
);`

const CLASSES_STATEMENT = `CREATE TABLE IF NOT EXISTS classes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	description TEXT NOT NULL,
	instructor TEXT NOT NULL,
	building TEXT NOT NULL,
	room INTEGER NOT NULL
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
	// Check if user already exists
	_, err := GetUser(username)

	if err == nil {
		return nil, fmt.Errorf("user already exists")
	}

	_, err = db.Exec(INSERT_USER_STATEMENT, username, HashPassword(password), "")
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
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

func DeleteUser(username string) error {
	_, err := db.Exec("DELETE FROM users WHERE username = ?;", username)
	return err
}

func AllUsers() ([]User, error) {
	rows, err := db.Query("SELECT id, username, password, classes FROM users;")
	if err != nil {
		return nil, err
	}

	users := make([]User, 0)

	for rows.Next() {
		var id int
		var name, password, classes string
		err = rows.Scan(&id, &name, &password, &classes)
		if err != nil {
			return nil, err
		}

		var classesList []int = make([]int, 0)

		for _, class := range strings.Split(classes, ",") {
			num, _ := strconv.ParseInt(class, 10, 64)
			classesList = append(classesList, int(num))
		}

		users = append(users, User{
			Username:     name,
			PasswordHash: password,
			Classes:      classesList,
		})
	}

	return users, nil
}

func CreateClass(name, description, instructor, building string, room int) (*Class, error) {
	_, err := db.Exec(INSERT_CLASS_STATEMENT, name, description, instructor, building, room)
	if err != nil {
		return nil, err
	}

	return GetClass(name)
}

func GetClass(name string) (*Class, error) {
	row := db.QueryRow(SELECT_CLASS_STATEMENT, name)

	var id, room int
	var n, desc, instr, build string
	err := row.Scan(&id, &n, &desc, &instr, &build, &room)
	if err != nil {
		return nil, err
	}

	return &Class{
		Name:        n,
		Description: desc,
		Instructor:  instr,
		Building:    build,
		Room:        room,
	}, nil
}

func LoadAllClasses() ([]Class, error) {
	rows, err := db.Query("SELECT id, name, description, instructor, building, room FROM classes;")
	if err != nil {
		return nil, err
	}

	classes := make([]Class, 0)

	for rows.Next() {
		var id, room int
		var name, desc, instr, build string
		err = rows.Scan(&id, &name, &desc, &instr, &build, &room)
		if err != nil {
			return nil, err
		}

		classes = append(classes, Class{
			ID:          id,
			Name:        name,
			Description: desc,
			Instructor:  instr,
			Building:    build,
			Room:        room,
		})
	}

	return classes, nil
}

func Init() {
	db, err := OpenDatabase()
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(USERS_STATEMENT)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(CLASSES_STATEMENT)
	if err != nil {
		panic(err)
	}
}
