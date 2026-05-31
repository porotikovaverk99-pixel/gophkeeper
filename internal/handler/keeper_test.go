package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/auth"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/handler"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/middleware"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/service"
)

func setupHandler() (*handler.KeeperHandler, *mockStorage, uuid.UUID) {
	storage := newMockStorage()
	authManager := auth.NewManager("secret", time.Hour)
	svc := service.NewKeeperService(storage, authManager)
	h := handler.NewKeeperHandler(svc)

	user, err := storage.CreateUser(context.Background(), "alice", "hash")
	if err != nil {
		panic(err)
	}

	return h, storage, user.ID
}

func withUserAndID(userID uuid.UUID, entryID string) (*http.Request, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/data/"+entryID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", entryID)
	req = req.WithContext(context.WithValue(middleware.WithUserID(context.Background(), userID), chi.RouteCtxKey, rctx))
	return req, httptest.NewRecorder()
}

func TestRegisterAndLoginHandlers(t *testing.T) {
	authManager := auth.NewManager("secret", time.Hour)
	h := handler.NewKeeperHandler(service.NewKeeperService(newMockStorage(), authManager))

	registerRec := httptest.NewRecorder()
	h.Register(registerRec, httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(`{"login":"alice","password":"secret"}`)))
	require.Equal(t, http.StatusCreated, registerRec.Code)

	loginRec := httptest.NewRecorder()
	h.Login(loginRec, httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBufferString(`{"login":"alice","password":"secret"}`)))
	require.Equal(t, http.StatusOK, loginRec.Code)
}

func TestRegisterDuplicateUser(t *testing.T) {
	h, _, _ := setupHandler()

	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(`{"login":"alice","password":"secret"}`)))
	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestLoginInvalidCredentials(t *testing.T) {
	h, _, _ := setupHandler()

	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBufferString(`{"login":"alice","password":"wrong"}`)))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateAndGetEntryHandlers(t *testing.T) {
	h, _, userID := setupHandler()

	body, err := json.Marshal(map[string]any{
		"type": model.EntryTypeText, "metadata": "note", "payload": []byte("encrypted"),
	})
	require.NoError(t, err)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/data", bytes.NewReader(body))
	createReq = createReq.WithContext(middleware.WithUserID(context.Background(), userID))
	createRec := httptest.NewRecorder()
	h.CreateEntry(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var created model.Entry
	require.NoError(t, json.NewDecoder(createRec.Body).Decode(&created))

	getReq, getRec := withUserAndID(userID, created.ID.String())
	h.GetEntry(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)
}

func TestUpdateDeleteSyncAndPingHandlers(t *testing.T) {
	h, storage, userID := setupHandler()

	entry := &model.Entry{
		ID: uuid.New(), UserID: userID, Type: model.EntryTypeText,
		Metadata: "note", Payload: []byte("data"), Version: 1,
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, storage.CreateEntry(context.Background(), entry))

	updateBody, _ := json.Marshal(map[string]any{
		"type": model.EntryTypeText, "metadata": "updated", "payload": []byte("data2"), "version": entry.Version,
	})
	updateReq, updateRec := withUserAndID(userID, entry.ID.String())
	updateReq.Method = http.MethodPut
	updateReq.Body = ioNopCloser(updateBody)
	h.UpdateEntry(updateRec, updateReq)
	require.Equal(t, http.StatusOK, updateRec.Code)

	syncBody := bytes.NewBufferString(`{"since":"1970-01-01T00:00:00Z"}`)
	syncReq := httptest.NewRequest(http.MethodPost, "/api/v1/sync", syncBody)
	syncReq = syncReq.WithContext(middleware.WithUserID(context.Background(), userID))
	syncRec := httptest.NewRecorder()
	h.SyncEntries(syncRec, syncReq)
	require.Equal(t, http.StatusOK, syncRec.Code)

	deleteReq, deleteRec := withUserAndID(userID, entry.ID.String())
	deleteReq.Method = http.MethodDelete
	h.DeleteEntry(deleteRec, deleteReq)
	require.Equal(t, http.StatusNoContent, deleteRec.Code)

	pingRec := httptest.NewRecorder()
	h.Ping(pingRec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.Equal(t, http.StatusOK, pingRec.Code)
}

func TestGetEntryNotFound(t *testing.T) {
	h, _, userID := setupHandler()
	req, rec := withUserAndID(userID, uuid.New().String())
	h.GetEntry(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestJSONError(t *testing.T) {
	rec := httptest.NewRecorder()
	handler.JSONError(rec, "boom", http.StatusTeapot)
	require.Equal(t, http.StatusTeapot, rec.Code)
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func ioNopCloser(data []byte) nopCloser { return nopCloser{bytes.NewReader(data)} }
