package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"hacknhbackend.eparker.dev/database"
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

func withCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
}

func main() {
	database.Init()

	go database.CourseUpdates()

	// Basic http server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		withCors(w)
		http.ServeFile(w, r, "index.html")
	})

	// All users (SAFE)
	http.HandleFunc("/user/all", func(w http.ResponseWriter, r *http.Request) {
		withCors(w)
		users, err := database.AllUsers()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		var strs []string = make([]string, len(users))

		for i, user := range users {
			strs[i] = string(user.JSON())
		}

		w.Write([]byte("[" + strings.Join(strs, ",") + "]"))
	})

	// Get user (SAFE)
	http.HandleFunc("/user/get", func(w http.ResponseWriter, r *http.Request) {
		withCors(w)
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "application/json" {
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
		withCors(w)

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "application/json" {
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

		user, err := database.CreateUser(obj.Username, obj.Password)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(user.JSON())
	})

	// Login
	http.HandleFunc("/user/login", func(w http.ResponseWriter, r *http.Request) {
		withCors(w)
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "application/json" {
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
			HttpOnly: true,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    TokenFor(obj.Username),
			Path:     "/",
			HttpOnly: true,
		})

		w.WriteHeader(http.StatusOK)
	})

	// Logout
	http.HandleFunc("/user/logout", func(w http.ResponseWriter, r *http.Request) {
		withCors(w)
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
		withCors(w)
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
	})

	// Private resource
	http.HandleFunc("/private", func(w http.ResponseWriter, r *http.Request) {
		withCors(w)
		if !withAuth(w, r) {
			return
		}

		w.Write([]byte("You can see this because you are logged in!"))
	})

	http.ListenAndServe("127.0.0.1:8000", nil)
}
