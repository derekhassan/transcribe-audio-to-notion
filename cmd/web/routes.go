package main

import "net/http"

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))

	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /auth/callback", app.notionAuthCallback)
	mux.HandleFunc("GET /upload", app.uploadForm)
	mux.HandleFunc("GET /upload/success", app.uploadSuccessful)
	mux.HandleFunc("POST /transcribe", app.createTranscription)

	return mux
}
