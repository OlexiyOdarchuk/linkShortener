package service

import (
	"context"
	"errors"
	"linkshortener/internal/cache"
	"linkshortener/internal/database"
	"net/http"
	"time"
)

type Server struct {
	port      string
	database  *database.Database
	analytics *database.Analytics
	cache     *cache.Cache
}

func NewServer(port string, database *database.Database, cacheDB *cache.Cache, analytics *database.Analytics) *Server {
	return &Server{
		port:      port,
		database:  database,
		analytics: analytics,
		cache:     cacheDB,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{code}", s.handlerRedirect)
	srv := &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}
	errChan := make(chan error, 1)
	go func() { errChan <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errChan:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) handlerRedirect(w http.ResponseWriter, r *http.Request) {
	//code := r.PathValue("code")
	//ctx := r.Context()
	//ip := r.Header.Get("X-Forwarded-For")
	//userAgent := r.UserAgent()
	//referer := r.Referer()
	// TODO
}
