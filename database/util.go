package database

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"hacknhbackend.eparker.dev/util"
)

func HashPassword(password string) string {
	hash := sha256.New()
	hash.Write([]byte(util.Config.Database.PasswordSalt + password))
	return string(hash.Sum(nil))
}

type User struct {
	Email, FirstName, LastName, PasswordHash string
	Courses                                  []string
	Privilege                                int
}

func (u *User) AddClass(crn string) {
	if _, err := GetCourse(crn); err != nil {
		return
	}

	for _, class := range u.Courses {
		if class == crn {
			return
		}
	}

	u.Courses = append(u.Courses, crn)

	QueuedExec("UPDATE users SET classes = ? WHERE email = ?;", strings.Join(u.Courses, ","), u.Email)
}

func (u *User) RemoveClass(crn string) {
	for i, class := range u.Courses {
		if class == crn {
			u.Courses = append(u.Courses[:i], u.Courses[i+1:]...)
			break
		}
	}

	QueuedExec("UPDATE users SET classes = ? WHERE email = ?;", strings.Join(u.Courses, ","), u.Email)
}

func (u *User) JSON() []byte {
	bytes, _ := json.Marshal(map[string]interface{}{
		"email":   u.Email,
		"first":   u.FirstName,
		"last":    u.LastName,
		"courses": u.Courses,
		"priv":    u.Privilege,
	})

	return bytes
}

const (
	CREATE_USER_SUCCESS = iota
	CREATE_USER_ERROR_IMUsed
	CREATE_USER_ERROR_InternalServerError
	CREATE_USER_ERROR_BadRequest
)

var ErrorQueueTimeout error = fmt.Errorf("queue timeout")
