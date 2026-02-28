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

func (s *Server) Start(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", s.handlerRedirect)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		return err
	}
	return nil
}

func (s *Server) handlerRedirect(w http.ResponseWriter, r *http.Request) {
	//code := r.PathValue("code")
	//ctx := r.Context()
	//ip := r.Header.Get("X-Forwarded-For")
	//userAgent := r.UserAgent()
	//referer := r.Referer()
	// TODO
}
