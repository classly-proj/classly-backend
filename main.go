package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"hacknhbackend.eparker.dev/courseload"
	"hacknhbackend.eparker.dev/database"
	"hacknhbackend.eparker.dev/util"
)

type Token struct {
	Username, Token string
	Expires         time.Time
}

var tokens map[string]*Token = make(map[string]*Token)

func RandomString(length int) string {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)

	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", bytes)
}

func TokenFor(username string) string {
	if token, ok := tokens[username]; ok {
		if token.Expires.After(time.Now()) {
			return token.Token
		}
	}

	tokens[username] = &Token{
		Username: username,
		Token:    RandomString(32),
		Expires:  time.Now().Add(time.Minute * 10),
	}

	return tokens[username].Token
}

func withAuth(w http.ResponseWriter, r *http.Request) bool {
	username, err := r.Cookie("username")

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	token, err := r.Cookie("token")

	if err != nil || token.Value != TokenFor(username.Value) {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	return true
}
func withCors(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
}

func main() {
	util.LoadEnvFile()
	database.Init()

	if util.Config.General.UpdateCourses {
		go database.CourseUpdates()
	}

	// Basic http server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		http.ServeFile(w, r, "index.html")
	})

	// All users (SAFE)
	http.HandleFunc("/user/all", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		users, err := database.AllUsers()

		if err != nil {
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		var strs []string = make([]string, len(users))

		for i, user := range users {
			strs[i] = string(user.JSON())
		}

		w.Write([]byte("[" + strings.Join(strs, ",") + "]"))
	})

	// Get user (SAFE)
	http.HandleFunc("/user/get", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		obj := struct {
			Username string `json:"username"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, err := database.GetUser(obj.Username)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(user.JSON())
	})

	// Create user
	http.HandleFunc("/user/create", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		obj := struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, statusCode := database.CreateUser(obj.Username, obj.Password)

		if statusCode != database.CREATE_USER_SUCCESS {
			switch statusCode {
			case database.CREATE_USER_ERROR_IMUsed:
				w.WriteHeader(http.StatusIMUsed)
			case database.CREATE_USER_ERROR_BadRequest:
				w.WriteHeader(http.StatusBadRequest)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "username",
			Value:    obj.Username,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    TokenFor(obj.Username),
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		w.Header().Set("Content-Type", "application/json")
		w.Write(user.JSON())

		util.Log.AddUser(fmt.Sprintf("User %s created", obj.Username))
	})

	// Login
	http.HandleFunc("/user/login", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		obj := struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, err := database.GetUser(obj.Username)

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if user.PasswordHash != database.HashPassword(obj.Password) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "username",
			Value:    obj.Username,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    TokenFor(obj.Username),
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		w.WriteHeader(http.StatusOK)
	})

	// Me
	http.HandleFunc("/user/me", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		if !withAuth(w, r) {
			return
		}

		username, _ := r.Cookie("username")
		user, err := database.GetUser(username.Value)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(user.JSON())
	})

	// Logout
	http.HandleFunc("/user/logout", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		http.SetCookie(w, &http.Cookie{
			Name:     "username",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
		})

		w.WriteHeader(http.StatusOK)
	})

	// Delete user (self only)
	http.HandleFunc("/user/delete", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		if !withAuth(w, r) {
			return
		}

		if r.Method != "DELETE" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		username, _ := r.Cookie("username")
		err := database.DeleteUser(username.Value)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "username",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
		})

		w.WriteHeader(http.StatusOK)

		util.Log.RemoveUser(fmt.Sprintf("User %s deleted", username.Value))
	})

	http.HandleFunc("/user/addclass", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		if !withAuth(w, r) {
			return
		}

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		var crns []string

		err := json.Unmarshal(body, &crns)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		username, _ := r.Cookie("username")
		user, err := database.GetUser(username.Value)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		for _, crn := range crns {
			user.AddClass(crn)
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/user/removeclass", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		if !withAuth(w, r) {
			return
		}

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		var crns []string

		err := json.Unmarshal(body, &crns)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		username, _ := r.Cookie("username")
		user, err := database.GetUser(username.Value)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		for _, crn := range crns {
			user.RemoveClass(crn)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Get all courses
	http.HandleFunc("/course/all", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		// Require authentication due to expensive operation
		if !withAuth(w, r) {
			return
		}

		courses, err := database.GetCourseCRNs()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if jsonCourses, err := json.Marshal(courses); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonCourses)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// Get course
	http.HandleFunc("/course/get", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		obj := struct {
			CRN string `json:"crn"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		course, err := database.GetCourse(obj.CRN)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(course.JSON())
	})

	// Get courses by subject code
	http.HandleFunc("/course/query/list", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)

		if jsonCourses, err := json.Marshal(database.QueryableKeys); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonCourses)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/course/query", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		obj := struct {
			QueryKey   string `json:"key"`
			QueryValue string `json:"value"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var courses []courseload.Course

		if obj.QueryKey == "subject-number" {
			courses, err = database.QueryCourse(obj.QueryKey, strings.Split(obj.QueryValue, "-")...)
		} else {
			courses, err = database.QueryCourse(obj.QueryKey, obj.QueryValue)
		}

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if jsonCourses, err := json.Marshal(courses); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonCourses)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// Community
	http.HandleFunc("/community/takingmyclasses", func(w http.ResponseWriter, r *http.Request) {

	})

	util.Log.Status(fmt.Sprintf("Server started on port %d", util.Config.Server.Port))
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", util.Config.Server.Port), nil)
}
