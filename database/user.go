package database

import "strings"

func CreateUser(username, password string) (*User, int) {
	// Check if user already exists
	_, err := GetUser(username)

	if err == nil {
		return nil, CREATE_USER_ERROR_IMUsed
	}

	_, err = db.Exec(INSERT_USER_STATEMENT, username, HashPassword(password), "")
	if err != nil {
		return nil, CREATE_USER_ERROR_InternalServerError
	}

	if user, err := GetUser(username); err == nil {
		return user, 0
	} else {
		return nil, CREATE_USER_ERROR_InternalServerError
	}
}

func GetUser(username string) (*User, error) {
	row := db.QueryRow(SELECT_USER_STATEMENT, username)

	var id int
	var name, password, classes string
	err := row.Scan(&id, &name, &password, &classes)
	if err != nil {
		return nil, err
	}

	var classesArr []string

	for _, class := range strings.Split(classes, ",") {
		if class != "" {
			classesArr = append(classesArr, class)
		}
	}

	return &User{
		Username:     name,
		PasswordHash: password,
		Classes:      classesArr,
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

		var classesArr []string

		for _, class := range strings.Split(classes, ",") {
			if class != "" {
				classesArr = append(classesArr, class)
			}
		}

		users = append(users, User{
			Username:     name,
			PasswordHash: password,
			Classes:      classesArr,
		})
	}

	return users, nil
}
