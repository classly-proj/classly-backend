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
	Username, PasswordHash string
	Classes                []string
}

func (u *User) AddClass(crn string) {
	if _, err := GetCourse(crn); err != nil {
		return
	}

	for _, class := range u.Classes {
		if class == crn {
			return
		}
	}

	u.Classes = append(u.Classes, crn)

	QueuedExec("UPDATE users SET classes = ? WHERE username = ?;", strings.Join(u.Classes, ","), u.Username)
}

func (u *User) RemoveClass(crn string) {
	for i, class := range u.Classes {
		if class == crn {
			u.Classes = append(u.Classes[:i], u.Classes[i+1:]...)
			break
		}
	}

	QueuedExec("UPDATE users SET classes = ? WHERE username = ?;", strings.Join(u.Classes, ","), u.Username)
}

func (u *User) JSON() []byte {
	bytes, _ := json.Marshal(map[string]interface{}{
		"username": u.Username,
		"classes":  u.Classes,
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
