package database

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

const COURSES_STATEMENT = `CREATE TABLE IF NOT EXISTS courses (
    term_crn TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    subject_code TEXT NOT NULL,
    course_number TEXT NOT NULL,
    description TEXT NOT NULL
);`

const INSTRUCTORS_STATEMENT = `CREATE TABLE IF NOT EXISTS instructors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    last_name TEXT NOT NULL,
    first_name TEXT NOT NULL,
    email TEXT NOT NULL,
    term_crn TEXT NOT NULL,
    FOREIGN KEY (term_crn) REFERENCES courses(term_crn)
);`

const MEETINGS_STATEMENT = `CREATE TABLE IF NOT EXISTS meetings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    days TEXT NOT NULL,
    building TEXT NOT NULL,
    room TEXT NOT NULL,
    time TEXT NOT NULL,
    term_crn TEXT NOT NULL,
    FOREIGN KEY (term_crn) REFERENCES courses(term_crn)
);`

const USERS_STATEMENT = `CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    classes TEXT NOT NULL
);`

const INSERT_USER_STATEMENT = `INSERT INTO users (username, password, classes) VALUES (?, ?, ?);`
const INSERT_INSTUCTOR_STATEMENT = `INSERT INTO instructors (last_name, first_name, email, term_crn) VALUES (?, ?, ?, ?);`
const INSERT_MEETING_STATEMENT = `INSERT INTO meetings (days, building, room, time, term_crn) VALUES (?, ?, ?, ?, ?);`
const INSERT_COURSE_STATEMENT = `INSERT INTO courses (term_crn, title, subject_code, course_number, description) VALUES (?, ?, ?, ?, ?);`

const SELECT_USER_STATEMENT = `SELECT id, username, password, classes FROM users WHERE username = ?;`
const SELECT_COUSE_STATEMENT = `SELECT term_crn, title, subject_code, course_number, description FROM courses WHERE term_crn = ?;`
const SELECT_INSTRUCTORS_STATEMENT = `SELECT id, last_name, first_name, email FROM instructors WHERE term_crn = ?;`
const SELECT_MEETINGS_STATEMENT = `SELECT id, days, building, room, time FROM meetings WHERE term_crn = ?;`

const (
	maxRetries = 5
	baseDelay  = 100 * time.Millisecond
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

func Init() {
	db, err := OpenDatabase()
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(USERS_STATEMENT)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(COURSES_STATEMENT)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(INSTRUCTORS_STATEMENT)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(MEETINGS_STATEMENT)
	if err != nil {
		panic(err)
	}
}
