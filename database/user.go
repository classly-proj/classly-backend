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

	var intPtr int
	var user User
	var courses string
	err := row.Scan(&intPtr, &user.Email, &user.FirstName, &user.LastName, &user.PasswordHash, &courses, &user.Privilege)
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
		var idPtr int
		var user User
		var courses string
		err := rows.Scan(&idPtr, &user.Email, &user.FirstName, &user.LastName, &user.PasswordHash, &courses, &user.Privilege)
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

func intersect(a, b []string) []string {
	m := make(map[string]bool)
	ab := make([]string, 0)

	for _, item := range a {
		m[item] = true
	}

	for _, item := range b {
		if _, ok := m[item]; ok {
			ab = append(ab, item)
		}
	}

	return ab
}

// Every user with 1+ courses in common with the given user
func UsersWithSimilarCourses(user *User) ([]User, error) {
	rows, err := QueuedQuery("SELECT id, email, first_name, last_name, password, classes, privilege FROM users;")
	if err != nil {
		return nil, err
	}

	users := make([]User, 0)

	for rows.Next() {
		var idPtr int
		var u User
		var courses string
		err := rows.Scan(&idPtr, &u.Email, &u.FirstName, &u.LastName, &u.PasswordHash, &courses, &u.Privilege)
		if err != nil {
			return nil, err
		}

		for _, class := range strings.Split(courses, ",") {
			if class != "" {
				u.Courses = append(u.Courses, class)
			}
		}

		if len(intersect(user.Courses, u.Courses)) > 0 {
			users = append(users, u)
		}
	}

	return users, nil
}
