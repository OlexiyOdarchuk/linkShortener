package service

import (
	"context"
	"database/sql"
	"errors"
	"linkshortener/internal/database"
	"net/http"
	"time"
)

type Server struct {
	port      string
	analytics *database.Analytics
	shortener *Shortener
}

func NewServer(port string, analytics *database.Analytics, shortener *Shortener) *Server {
	return &Server{
		port:      port,
		analytics: analytics,
		shortener: shortener,
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
	code := r.PathValue("code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	ip := r.Header.Get("X-Forwarded-For")
	userAgent := r.UserAgent()
	referer := r.Referer()
	linkCache, err := s.shortener.GetLinkCacheByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		return
	}

	go func() {
		newClickData := database.ClickData{
			UserId:    linkCache.UserID,
			ShortCode: code,
			IP:        ip,
			UserAgent: userAgent,
			Referer:   referer,
		}
		s.analytics.PushClick(newClickData)
	}()

	http.Redirect(w, r, linkCache.OriginalLink, http.StatusFound)
}
