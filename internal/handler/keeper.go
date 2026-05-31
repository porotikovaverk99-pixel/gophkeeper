// Package handler содержит HTTP-обработчики REST API GophKeeper.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/middleware"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/repository"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/service"
)

// KeeperHandler обрабатывает HTTP-запросы к API GophKeeper.
type KeeperHandler struct {
	service *service.KeeperService
}

// NewKeeperHandler создаёт HTTP-обработчик GophKeeper.
func NewKeeperHandler(svc *service.KeeperService) *KeeperHandler {
	return &KeeperHandler{service: svc}
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	Login string `json:"login"`
}

type entryRequest struct {
	ID       *uuid.UUID      `json:"id,omitempty"`
	Type     model.EntryType `json:"type"`
	Metadata string          `json:"metadata"`
	Payload  []byte          `json:"payload"`
	Version  int64           `json:"version,omitempty"`
	Deleted  bool            `json:"deleted,omitempty"`
}

type syncRequest struct {
	Since time.Time `json:"since"`
}

// Register обрабатывает регистрацию нового пользователя.
func (h *KeeperHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, user, err := h.service.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			JSONError(w, "user already exists", http.StatusConflict)
			return
		}
		JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{Token: token, Login: user.Login})
}

// Login обрабатывает аутентификацию пользователя.
func (h *KeeperHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, user, err := h.service.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			JSONError(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, Login: user.Login})
}

// CreateEntry создаёт новую запись пользователя.
func (h *KeeperHandler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req entryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entry := &model.Entry{
		Type:     req.Type,
		Metadata: req.Metadata,
		Payload:  req.Payload,
	}
	if req.ID != nil {
		entry.ID = *req.ID
	}

	if err := h.service.CreateEntry(r.Context(), userID, entry); err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, entry)
}

// UpdateEntry обновляет существующую запись пользователя.
func (h *KeeperHandler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		JSONError(w, "invalid entry id", http.StatusBadRequest)
		return
	}

	var req entryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entry := &model.Entry{
		ID:       entryID,
		Type:     req.Type,
		Metadata: req.Metadata,
		Payload:  req.Payload,
		Version:  req.Version,
		Deleted:  req.Deleted,
	}

	if err := h.service.UpdateEntry(r.Context(), userID, entry); err != nil {
		if errors.Is(err, repository.ErrEntryNotFound) {
			JSONError(w, "entry not found", http.StatusNotFound)
			return
		}
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updated, err := h.service.GetEntry(r.Context(), userID, entryID)
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// GetEntry возвращает запись пользователя по идентификатору.
func (h *KeeperHandler) GetEntry(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		JSONError(w, "invalid entry id", http.StatusBadRequest)
		return
	}

	entry, err := h.service.GetEntry(r.Context(), userID, entryID)
	if err != nil {
		if errors.Is(err, repository.ErrEntryNotFound) {
			JSONError(w, "entry not found", http.StatusNotFound)
			return
		}
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

// SyncEntries возвращает записи, изменённые после указанного времени.
func (h *KeeperHandler) SyncEntries(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entries, err := h.service.SyncEntries(r.Context(), userID, req.Since)
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, entries)
}

// DeleteEntry удаляет запись пользователя.
func (h *KeeperHandler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		JSONError(w, "invalid entry id", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteEntry(r.Context(), userID, entryID); err != nil {
		if errors.Is(err, repository.ErrEntryNotFound) {
			JSONError(w, "entry not found", http.StatusNotFound)
			return
		}
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Ping проверяет доступность сервера и хранилища.
func (h *KeeperHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		JSONError(w, "database unavailable", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
