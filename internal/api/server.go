package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/l-eduardo/follow-the-fucking-money/internal/domain"
	"github.com/rs/zerolog/log"
)

// Server provides HTTP REST APIs for graph queries and web visualizations.
type Server struct {
	router chi.Router
	svc    domain.GraphQueryService
	port   int
}

// NewServer builds an API server with Chi routes.
func NewServer(svc domain.GraphQueryService, port int) *Server {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	s := &Server{
		router: r,
		svc:    svc,
		port:   port,
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Get("/health", s.handleHealth)

	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/stats", s.handleStats)
		r.Get("/trail", s.handleTrail)
		r.Get("/entity/{id}", s.handleEntity)
		r.Get("/patterns/quid-pro-quo", s.handleQuidProQuo)
		r.Get("/patterns/ghost-suppliers", s.handleGhostSuppliers)
		r.Get("/patterns/amendments", s.handleAmendments)
	})
}

// Start listens on the configured TCP port.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Info().Str("addr", addr).Msg("Starting FTFM API server")
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "healthy",
		"service":   "follow-the-fucking-money",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.svc.GetDatabaseStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleTrail(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	maxDepthStr := r.URL.Query().Get("max_depth")
	minAmountStr := r.URL.Query().Get("min_amount")

	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "query parameters 'from' and 'to' are required")
		return
	}

	maxDepth := 4
	if d, err := strconv.Atoi(maxDepthStr); err == nil && d > 0 && d <= 8 {
		maxDepth = d
	}

	minAmount := 0.0
	if a, err := strconv.ParseFloat(minAmountStr, 64); err == nil && a >= 0 {
		minAmount = a
	}

	trail, err := s.svc.FindMoneyTrail(r.Context(), from, to, maxDepth, minAmount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, trail)
}

func (s *Server) handleEntity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing entity id")
		return
	}

	res, err := s.svc.GetEntityNeighbors(r.Context(), id, 1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleQuidProQuo(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	year := 2022
	if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
		year = y
	}

	minAmtStr := r.URL.Query().Get("min_amount")
	minAmt := 50000.0
	if a, err := strconv.ParseFloat(minAmtStr, 64); err == nil && a >= 0 {
		minAmt = a
	}

	records, err := s.svc.FindQuidProQuo(r.Context(), year, minAmt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleGhostSuppliers(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	year := 2022
	if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
		year = y
	}

	minExpStr := r.URL.Query().Get("min_expense")
	minExp := 100000.0
	if a, err := strconv.ParseFloat(minExpStr, 64); err == nil && a >= 0 {
		minExp = a
	}

	records, err := s.svc.FindGhostSuppliers(r.Context(), year, minExp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleAmendments(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	year := 2022
	if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
		year = y
	}

	records, err := s.svc.FindAmendmentTriangulation(r.Context(), year)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
