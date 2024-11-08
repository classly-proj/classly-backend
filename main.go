package main

import "net/http"

func main() {
	// Basic http server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	http.ListenAndServe("127.0.0.1:80", nil)
}
