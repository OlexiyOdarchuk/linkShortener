package service

import (
	"context"
	"database/sql"
	"errors"
	"linkshortener/internal/database"
	"linkshortener/internal/types"
	"net"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	port      string
	db        *database.Database
	shortener *Shortener
}

func NewServer(port string, db *database.Database, shortener *Shortener) *Server {
	return &Server{
		port:      port,
		db:        db,
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
	ip := s.getClientIP(r)
	userAgent := r.UserAgent()
	referer := r.Referer()
	linkCache, err := s.db.GetLinkCacheByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		return
	}

	go func() {
		newClickData := types.ClickData{
			UserId:    linkCache.UserID,
			ShortCode: code,
			IP:        ip,
			UserAgent: userAgent,
			Referer:   referer,
		}
		s.db.PushClick(newClickData)
	}()

	http.Redirect(w, r, linkCache.OriginalLink, http.StatusFound)
}

func (s *Server) getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
