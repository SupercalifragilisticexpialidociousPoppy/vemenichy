package api

import (
	"net/http"
)

func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/ping", HandlePing)
	mux.HandleFunc("/add", HandleAdd)
	mux.HandleFunc("/search", HandleSearch)
	mux.HandleFunc("/download", HandleDownload)

	mux.HandleFunc("/pause", HandlePause)
	mux.HandleFunc("/skip", HandleSkip)
	mux.HandleFunc("/queue", HandleGetQueue)

	return mux
}
