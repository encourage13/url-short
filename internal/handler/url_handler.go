package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/encourage13/url-short/internal/domain"
)

const (
	maxRequestBody = 64 << 10

	defaultListLimit = 50
	maxListLimit     = 100

	healthTimeout = 2 * time.Second
)

type URLService interface {
	CreateShortURL(ctx context.Context, originalURL string) (*domain.URL, error)
	GetOriginalURL(ctx context.Context, code string) (*domain.URL, error)
	GetURLInfo(ctx context.Context, code string) (*domain.URL, error)
	ListURLs(ctx context.Context, limit, offset int) ([]*domain.URL, error)
	Health(ctx context.Context) error
}

type URLHandler struct {
	service    URLService
	baseURL    string
	adminToken string
	log        *slog.Logger
}

func NewURLHandler(svc URLService, baseURL, adminToken string, log *slog.Logger) *URLHandler {
	return &URLHandler{
		service:    svc,
		baseURL:    strings.TrimRight(baseURL, "/"),
		adminToken: adminToken,
		log:        log,
	}
}

func (h *URLHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", h.CreateShortURL)
	mux.HandleFunc("GET /info/{code}", h.GetURLInfo)
	mux.HandleFunc("GET /urls", h.ListURLs)
	mux.HandleFunc("GET /health", h.HealthCheck)
	mux.HandleFunc("GET /{code}", h.RedirectToOriginal)
	return withRecover(h.log, withLogging(h.log, mux))
}

type CreateShortURLRequest struct {
	URL string `json:"url"`
}

type URLResponse struct {
	Code        string    `json:"code"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	Clicks      int64     `json:"clicks"`
	CreatedAt   time.Time `json:"created_at"`
}

type ListURLsResponse struct {
	Items  []URLResponse `json:"items"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *URLHandler) toResponse(u *domain.URL) URLResponse {
	return URLResponse{
		Code:        u.Code,
		ShortURL:    h.baseURL + "/" + u.Code,
		OriginalURL: u.OriginalURL,
		Clicks:      u.Clicks,
		CreatedAt:   u.CreatedAt,
	}
}

func (h *URLHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	var req CreateShortURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Неверный формат запроса")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "URL обязателен")
		return
	}

	url, err := h.service.CreateShortURL(r.Context(), req.URL)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, h.toResponse(url))
}

func (h *URLHandler) RedirectToOriginal(w http.ResponseWriter, r *http.Request) {
	url, err := h.service.GetOriginalURL(r.Context(), r.PathValue("code"))
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	http.Redirect(w, r, url.OriginalURL, http.StatusFound)
}

func (h *URLHandler) GetURLInfo(w http.ResponseWriter, r *http.Request) {
	url, err := h.service.GetURLInfo(r.Context(), r.PathValue("code"))
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, h.toResponse(url))
}

func (h *URLHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
	defer cancel()

	if err := h.service.Health(ctx); err != nil {
		h.log.Error("health check failed", "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *URLHandler) ListURLs(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeError(w, http.StatusUnauthorized, "Доступ запрещен")
		return
	}

	limit, err := queryInt(r, "limit", defaultListLimit)
	if err != nil || limit < 1 || limit > maxListLimit {
		writeError(w, http.StatusBadRequest, "limit должен быть от 1 до "+strconv.Itoa(maxListLimit))
		return
	}
	offset, err := queryInt(r, "offset", 0)
	if err != nil || offset < 0 {
		writeError(w, http.StatusBadRequest, "offset должен быть неотрицательным числом")
		return
	}

	urls, err := h.service.ListURLs(r.Context(), limit, offset)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	resp := ListURLsResponse{Items: make([]URLResponse, len(urls)), Limit: limit, Offset: offset}
	for i, u := range urls {
		resp.Items[i] = h.toResponse(u)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *URLHandler) isAdmin(r *http.Request) bool {
	if h.adminToken == "" {
		return false
	}
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return ok && subtle.ConstantTimeCompare([]byte(token), []byte(h.adminToken)) == 1
}

func (h *URLHandler) handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "Ссылка не найдена")
	case errors.Is(err, domain.ErrInvalidURL):
		writeError(w, http.StatusBadRequest, "Некорректный URL: нужен абсолютный http(s)-адрес")
	default:
		h.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "Внутренняя ошибка")
	}
}

func queryInt(r *http.Request, key string, fallback int) (int, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback, nil
	}
	return strconv.Atoi(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
