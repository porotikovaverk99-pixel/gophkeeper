package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	router *chi.Mux
	addr   string
}

func New(addr string) *Server {
	return &Server{
		router: chi.NewRouter(),
		addr:   addr,
	}
}

func (s *Server) HandleFunc(pattern string, handler http.HandlerFunc) {
	s.router.HandleFunc(pattern, handler)
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.router.Handle(pattern, handler)
}

func (s *Server) Post(pattern string, handler http.HandlerFunc) {
	s.router.Post(pattern, handler)
}

func (s *Server) Get(pattern string, handler http.HandlerFunc) {
	s.router.Get(pattern, handler)
}

func (s *Server) Put(pattern string, handler http.HandlerFunc) {
	s.router.Put(pattern, handler)
}

func (s *Server) Delete(pattern string, handler http.HandlerFunc) {
	s.router.Delete(pattern, handler)
}

// Router возвращает chi-роутер сервера для расширенной настройки маршрутов.
func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) Run() error {
	server := &http.Server{
		Addr:              s.addr,
		Handler:           s.router,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	return server.ListenAndServe()
}
