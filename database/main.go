package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hacknhbackend.eparker.dev/courseload"
	_ "modernc.org/sqlite"
)

type User struct {
	Username, PasswordHash string
	Classes                []string
}

func (u *User) JSON() []byte {
	bytes, _ := json.Marshal(map[string]interface{}{
		"username": u.Username,
		"classes":  u.Classes,
	})

	return bytes
}

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

	return &User{
		Username:     name,
		PasswordHash: password,
		Classes:      strings.Split(classes, ","),
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

		users = append(users, User{
			Username:     name,
			PasswordHash: password,
			Classes:      strings.Split(classes, ","),
		})
	}

	return users, nil
}

func InsertCourse(course courseload.Course) error {
	_, err := db.Exec(INSERT_COURSE_STATEMENT, course.TERM_CRN, course.COURSE_DATA.SYVSCHD_CRSE_LONG_TITLE, course.COURSE_DATA.SYVSCHD_SUBJ_CODE, course.COURSE_DATA.SYVSCHD_CRSE_NUMB, course.COURSE_DATA.SYVSCHD_CRSE_DESC)
	if err != nil {
		return err
	}

	for _, instructor := range course.COURSE_DATA.INSTRUCTORS {
		_, err := db.Exec(INSERT_INSTUCTOR_STATEMENT, instructor.LAST_NAME, instructor.FIRST_NAME, instructor.EMAIL, course.TERM_CRN)
		if err != nil {
			return err
		}
	}

	for _, meeting := range course.COURSE_DATA.MEETINGS {
		_, err := db.Exec(INSERT_MEETING_STATEMENT, meeting.DAYS, meeting.BUILDING, meeting.ROOM, meeting.TIME, course.TERM_CRN)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetCourse(term_crn string) (*courseload.Course, error) {
	row := db.QueryRow(SELECT_COUSE_STATEMENT, term_crn)

	var title, subject_code, course_number, description string
	err := row.Scan(&term_crn, &title, &subject_code, &course_number, &description)
	if err != nil {
		return nil, err
	}

	instructors := make([]courseload.Instructor, 0)
	rows, err := db.Query(SELECT_INSTRUCTORS_STATEMENT, term_crn)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var id int
		var last_name, first_name, email string
		err = rows.Scan(&id, &last_name, &first_name, &email)
		if err != nil {
			return nil, err
		}

		instructors = append(instructors, courseload.Instructor{
			LAST_NAME:  last_name,
			FIRST_NAME: first_name,
			EMAIL:      email,
		})
	}

	meetings := make([]courseload.Meeting, 0)
	rows, err = db.Query(SELECT_MEETINGS_STATEMENT, term_crn)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var id int
		var days, building, room, time string
		err = rows.Scan(&id, &days, &building, &room, &time)
		if err != nil {
			return nil, err
		}

		meetings = append(meetings, courseload.Meeting{
			DAYS:     days,
			BUILDING: building,
			ROOM:     room,
			TIME:     time,
		})
	}

	return &courseload.Course{
		TERM_CRN: term_crn,
		COURSE_DATA: courseload.CourseData{
			SYVSCHD_CRSE_LONG_TITLE: title,
			SYVSCHD_SUBJ_CODE:       subject_code,
			SYVSCHD_CRSE_NUMB:       course_number,
			SYVSCHD_CRSE_DESC:       description,
			INSTRUCTORS:             instructors,
			MEETINGS:                meetings,
		},
	}, nil
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
