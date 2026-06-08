package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/handler"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/handler/mocks"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/middleware"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/service"
)

func newTestHandler(t *testing.T, svc handler.KeeperService) *handler.KeeperHandler {
	t.Helper()
	return handler.NewKeeperHandler(svc, zap.NewNop())
}

func withUserAndID(userID uuid.UUID, entryID string) (*http.Request, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/data/"+entryID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", entryID)
	req = req.WithContext(contextWithRoute(userID, rctx))
	return req, httptest.NewRecorder()
}

func contextWithRoute(userID uuid.UUID, rctx *chi.Context) context.Context {
	return context.WithValue(middleware.WithUserID(context.Background(), userID), chi.RouteCtxKey, rctx)
}

func TestRegisterAndLoginHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockKeeperService(ctrl)
	user := &model.User{ID: uuid.New(), Login: "alice"}

	svc.EXPECT().Register(gomock.Any(), "alice", "secret").Return("token-alice", user, nil)
	svc.EXPECT().Login(gomock.Any(), "alice", "secret").Return("token-alice", user, nil)

	h := newTestHandler(t, svc)

	registerRec := httptest.NewRecorder()
	h.Register(registerRec, httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(`{"login":"alice","password":"secret"}`)))
	require.Equal(t, http.StatusCreated, registerRec.Code)

	loginRec := httptest.NewRecorder()
	h.Login(loginRec, httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBufferString(`{"login":"alice","password":"secret"}`)))
	require.Equal(t, http.StatusOK, loginRec.Code)
}

func TestRegisterDuplicateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockKeeperService(ctrl)
	svc.EXPECT().Register(gomock.Any(), "alice", "secret").Return("", nil, service.ErrUserAlreadyExists)

	h := newTestHandler(t, svc)

	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(`{"login":"alice","password":"secret"}`)))
	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestLoginInvalidCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockKeeperService(ctrl)
	svc.EXPECT().Login(gomock.Any(), "alice", "wrong").Return("", nil, service.ErrInvalidCredentials)

	h := newTestHandler(t, svc)

	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBufferString(`{"login":"alice","password":"wrong"}`)))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateAndGetEntryHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockKeeperService(ctrl)
	userID := uuid.New()
	entryID := uuid.New()

	svc.EXPECT().CreateEntry(gomock.Any(), userID, gomock.Any()).DoAndReturn(
		func(_ context.Context, uid uuid.UUID, entry *model.Entry) error {
			entry.ID = entryID
			entry.UserID = uid
			return nil
		},
	)
	svc.EXPECT().GetEntry(gomock.Any(), userID, entryID).Return(&model.Entry{
		ID: entryID, UserID: userID, Type: model.EntryTypeText, Metadata: "note", Payload: []byte("encrypted"),
	}, nil)

	h := newTestHandler(t, svc)

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
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockKeeperService(ctrl)
	userID := uuid.New()
	entryID := uuid.New()
	updatedEntry := &model.Entry{
		ID: entryID, UserID: userID, Type: model.EntryTypeText,
		Metadata: "updated", Payload: []byte("data2"), Version: 2,
		UpdatedAt: time.Now().UTC(),
	}

	svc.EXPECT().UpdateEntry(gomock.Any(), userID, gomock.Any()).Return(nil)
	svc.EXPECT().GetEntry(gomock.Any(), userID, entryID).Return(updatedEntry, nil)
	svc.EXPECT().SyncEntries(gomock.Any(), userID, gomock.Any()).Return([]model.Entry{*updatedEntry}, nil)
	svc.EXPECT().DeleteEntry(gomock.Any(), userID, entryID).Return(nil)
	svc.EXPECT().Ping(gomock.Any()).Return(nil)

	h := newTestHandler(t, svc)

	updateBody, _ := json.Marshal(map[string]any{
		"type": model.EntryTypeText, "metadata": "updated", "payload": []byte("data2"), "version": 1,
	})
	updateReq, updateRec := withUserAndID(userID, entryID.String())
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

	deleteReq, deleteRec := withUserAndID(userID, entryID.String())
	deleteReq.Method = http.MethodDelete
	h.DeleteEntry(deleteRec, deleteReq)
	require.Equal(t, http.StatusNoContent, deleteRec.Code)

	pingRec := httptest.NewRecorder()
	h.Ping(pingRec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.Equal(t, http.StatusOK, pingRec.Code)
}

func TestGetEntryNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockKeeperService(ctrl)
	userID := uuid.New()
	entryID := uuid.New()

	svc.EXPECT().GetEntry(gomock.Any(), userID, entryID).Return(nil, service.ErrEntryNotFound)

	h := newTestHandler(t, svc)
	req, rec := withUserAndID(userID, entryID.String())
	h.GetEntry(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRegisterEmptyCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockKeeperService(ctrl)
	svc.EXPECT().Register(gomock.Any(), "", "").Return("", nil, service.ErrLoginPasswordRequired)

	h := newTestHandler(t, svc)

	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(`{"login":"","password":""}`)))
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Equal(t, "login and password are required", resp["error"])
}

func TestRegisterInternalErrorDoesNotLeakDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockKeeperService(ctrl)
	svc.EXPECT().Register(gomock.Any(), "bob", "secret").Return("", nil, errors.New("db connection failed: secret-host"))

	h := newTestHandler(t, svc)

	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(`{"login":"bob","password":"secret"}`)))
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Equal(t, http.StatusText(http.StatusInternalServerError), resp["error"])
	require.NotContains(t, resp["error"], "secret-host")
}

func TestUnauthorizedHandlers(t *testing.T) {
	h := newTestHandler(t, mocks.NewMockKeeperService(gomock.NewController(t)))

	tests := []struct {
		name   string
		method string
		path   string
		call   func(*handler.KeeperHandler, *httptest.ResponseRecorder, *http.Request)
	}{
		{
			name: "create entry", method: http.MethodPost, path: "/api/v1/data",
			call: func(h *handler.KeeperHandler, rec *httptest.ResponseRecorder, req *http.Request) {
				h.CreateEntry(rec, req)
			},
		},
		{
			name: "get entry", method: http.MethodGet, path: "/api/v1/data/" + uuid.New().String(),
			call: func(h *handler.KeeperHandler, rec *httptest.ResponseRecorder, req *http.Request) {
				h.GetEntry(rec, req)
			},
		},
		{
			name: "sync entries", method: http.MethodPost, path: "/api/v1/sync",
			call: func(h *handler.KeeperHandler, rec *httptest.ResponseRecorder, req *http.Request) {
				h.SyncEntries(rec, req)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			tt.call(h, rec, req)
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestRegisterInvalidBody(t *testing.T) {
	h := newTestHandler(t, mocks.NewMockKeeperService(gomock.NewController(t)))

	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(`not-json`)))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPingInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockKeeperService(ctrl)
	svc.EXPECT().Ping(gomock.Any()).Return(errors.New("db unavailable"))

	h := newTestHandler(t, svc)
	rec := httptest.NewRecorder()
	h.Ping(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestJSONError(t *testing.T) {
	rec := httptest.NewRecorder()
	handler.JSONError(rec, "boom", http.StatusTeapot)
	require.Equal(t, http.StatusTeapot, rec.Code)
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func ioNopCloser(data []byte) nopCloser { return nopCloser{bytes.NewReader(data)} }
