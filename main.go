package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
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

func TokenFor(email string) string {
	if token, ok := tokens[email]; ok {
		if token.Expires.After(time.Now()) {
			return token.Token
		}
	}

	tokens[email] = &Token{
		Username: email,
		Token:    RandomString(32),
		Expires:  time.Now().Add(time.Minute * 10),
	}

	return tokens[email].Token
}

func withAuth(w http.ResponseWriter, r *http.Request) bool {
	cookies := r.Cookies()

	// list them all
	for _, cookie := range cookies {
		fmt.Println(cookie.Name, cookie.Value)
	}

	email, err := r.Cookie("email")

	if err != nil {
		util.Log.Error("No email cookie")
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	token, err := r.Cookie("token")

	if err != nil || token.Value != TokenFor(email.Value) {
		if err != nil {
			util.Log.Error("No token cookie")
		} else {
			util.Log.Error("Token mismatch")
		}
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
			Email string `json:"email"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, err := database.GetUser(obj.Email)

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
			Email    string `json:"email"`
			First    string `json:"firstName"`
			Last     string `json:"lastName"`
			Password string `json:"password"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, statusCode := database.CreateUser(obj.Email, obj.First, obj.Last, obj.Password)

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
			Name:     "email",
			Value:    obj.Email,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    TokenFor(obj.Email),
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		w.Header().Set("Content-Type", "application/json")
		w.Write(user.JSON())

		util.Log.AddUser(fmt.Sprintf("User %s created", obj.Email))
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
			Email    string `json:"email"`
			Password string `json:"password"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, err := database.GetUser(obj.Email)

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if user.PasswordHash != database.HashPassword(obj.Password) {
			fmt.Println("Password mismatch")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "email",
			Value:    obj.Email,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    TokenFor(obj.Email),
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		w.WriteHeader(http.StatusOK)

		util.Log.AddUser(fmt.Sprintf("User %s logged in", obj.Email))
	})

	// Me
	http.HandleFunc("/user/me", func(w http.ResponseWriter, r *http.Request) {
		withCors(w, r)
		if !withAuth(w, r) {
			util.Log.Error("Unauthorized")
			return
		}

		email, _ := r.Cookie("email")
		user, err := database.GetUser(email.Value)

		if err != nil {
			util.Log.Error("User not found")
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
			Name:     "email",
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

		email, _ := r.Cookie("email")
		err := database.DeleteUser(email.Value)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "email",
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

		util.Log.RemoveUser(fmt.Sprintf("User %s deleted", email.Value))
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

		email, _ := r.Cookie("email")
		user, err := database.GetUser(email.Value)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		for _, crn := range crns {
			if err := user.AddClass(crn); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
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

		email, _ := r.Cookie("email")
		user, err := database.GetUser(email.Value)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		for _, crn := range crns {
			user.RemoveClass(crn)
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/user/changename", func(w http.ResponseWriter, r *http.Request) {
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

		obj := struct {
			First string `json:"firstName"`
			Last  string `json:"lastName"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		email, _ := r.Cookie("email")
		user, err := database.GetUser(email.Value)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := user.ChangeName(obj.First, obj.Last); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
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

	// Mapbox
	http.HandleFunc("/mapbox/directions", func(w http.ResponseWriter, r *http.Request) {
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

		obj := struct {
			StartX float64 `json:"startX"`
			StartY float64 `json:"startY"`
			EndX   float64 `json:"endX"`
			EndY   float64 `json:"endY"`
		}{}

		err := json.Unmarshal(body, &obj)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		res, err := http.Get(fmt.Sprintf("https://api.mapbox.com/directions/v5/mapbox/walking/%f,%f;%f,%f?alternatives=true&geometries=geojson&language=en&overview=full&steps=true&access_token=%s", obj.StartX, obj.StartY, obj.EndX, obj.EndY, util.Config.Mapbox.AccessToken))

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		defer res.Body.Close()

		w.Header().Set("Content-Type", "application/json")

		if bytes, err := io.ReadAll(res.Body); err == nil {
			w.Write(bytes)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// Community
	http.HandleFunc("/community/takingmyclasses", func(w http.ResponseWriter, r *http.Request) {

	})

	util.Log.Status(fmt.Sprintf("Server started on port %d", util.Config.Server.Port))
	var at string = fmt.Sprintf("%s:%d", util.Config.Server.Host, util.Config.Server.Port)

	if util.Config.Server.TLS != "" {
		http.ListenAndServeTLS(at, util.Config.Server.TLS+"/fullchain.pem", util.Config.Server.TLS+"/privkey.pem", nil)
	} else {
		http.ListenAndServe(at, nil)
	}
}
