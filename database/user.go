package database

import (
	"strings"
)

func CreateUser(email, first, last, password string) (*User, int) {
	// Check if user already exists
	_, err := GetUser(email)

	if err == nil {
		return nil, CREATE_USER_ERROR_IMUsed
	}

	err = QueuedExec(INSERT_USER_STATEMENT, email, first, last, HashPassword(password), "")
	if err != nil {
		return nil, CREATE_USER_ERROR_InternalServerError
	}

	if user, err := GetUser(email); err == nil {
		return user, 0
	} else {
		return nil, CREATE_USER_ERROR_InternalServerError
	}
}

func GetUser(email string) (*User, error) {
	row := QueuedQueryRow(SELECT_USER_STATEMENT, email)

	var user User
	var courses string
	err := row.Scan(&user.Email, &user.FirstName, &user.LastName, &user.PasswordHash, &courses, &user.Privilege)
	if err != nil {
		return nil, err
	}

	for _, class := range strings.Split(courses, ",") {
		if class != "" {
			user.Courses = append(user.Courses, class)
		}
	}

	return &user, nil
}

func DeleteUser(username string) error {
	return QueuedExec("DELETE FROM users WHERE username = ?;", username)
}

func AllUsers() ([]User, error) {
	rows, err := QueuedQuery("SELECT id, email, first_name, last_name, password, classes, privilege FROM users;")
	if err != nil {
		return nil, err
	}

	users := make([]User, 0)

	for rows.Next() {
		var user User
		var courses string
		err := rows.Scan(&user.Email, &user.FirstName, &user.LastName, &user.PasswordHash, &courses, &user.Privilege)
		if err != nil {
			return nil, err
		}

		for _, class := range strings.Split(courses, ",") {
			if class != "" {
				user.Courses = append(user.Courses, class)
			}
		}

		users = append(users, user)
	}

	return users, nil
}
