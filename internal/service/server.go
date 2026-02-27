package service

import (
	"linkshortener/internal/cache"
	"linkshortener/internal/database"
	"net/http"
)

type Server struct {
	database  *database.Database
	analytics *database.Analytics
	cache     *cache.Cache
}

func (s *Server) Start(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", s.handlerRedirect)
	http.ListenAndServe(":"+port, mux)
}

func (s *Server) handlerRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	ctx := r.Context()
	// TODO
}
