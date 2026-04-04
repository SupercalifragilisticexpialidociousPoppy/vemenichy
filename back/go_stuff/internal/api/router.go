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

	mux.HandleFunc("/queue", HandleGetQueue)
	mux.HandleFunc("/status", HandleStatus)
	mux.HandleFunc("/logs", HandleLogs)

	mux.HandleFunc("/pause", HandlePause)
	mux.HandleFunc("/skip", HandleSkip)
	mux.HandleFunc("/volume", HandleVolume)

	mux.HandleFunc("/global/enable", HandleEnableGlobal)
	mux.HandleFunc("/global/disable", HandleDisableGlobal)

	mux.Handle("/", http.FileServer(http.Dir("../../front/")))

	return mux
}
