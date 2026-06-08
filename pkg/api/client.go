// Package api предоставляет HTTP-клиент для взаимодействия с сервером GophKeeper.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
)

// Client выполняет REST-запросы к серверу GophKeeper.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewClient создаёт HTTP-клиент GophKeeper.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SetToken сохраняет JWT-токен для авторизованных запросов.
func (c *Client) SetToken(token string) {
	c.token = token
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	Login string `json:"login"`
}

// Register регистрирует нового пользователя на сервере.
func (c *Client) Register(ctx context.Context, login, password string) (string, error) {
	var resp authResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/register", authRequest{
		Login:    login,
		Password: password,
	}, &resp, false); err != nil {
		return "", err
	}

	c.token = resp.Token
	return resp.Token, nil
}

// Login выполняет аутентификацию пользователя на сервере.
func (c *Client) Login(ctx context.Context, login, password string) (string, error) {
	var resp authResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/login", authRequest{
		Login:    login,
		Password: password,
	}, &resp, false); err != nil {
		return "", err
	}

	c.token = resp.Token
	return resp.Token, nil
}

type entryRequest struct {
	ID       *uuid.UUID      `json:"id,omitempty"`
	Type     model.EntryType `json:"type"`
	Metadata string          `json:"metadata"`
	Payload  []byte          `json:"payload"`
	Version  int64           `json:"version,omitempty"`
	Deleted  bool            `json:"deleted,omitempty"`
}

// CreateEntry создаёт запись на сервере.
func (c *Client) CreateEntry(ctx context.Context, entry *model.Entry) error {
	req := entryRequest{
		ID:       &entry.ID,
		Type:     entry.Type,
		Metadata: entry.Metadata,
		Payload:  entry.Payload,
	}
	var created model.Entry
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/data", req, &created, true); err != nil {
		return err
	}
	*entry = created
	return nil
}

// UpdateEntry обновляет запись на сервере.
func (c *Client) UpdateEntry(ctx context.Context, entry *model.Entry) error {
	req := entryRequest{
		Type:     entry.Type,
		Metadata: entry.Metadata,
		Payload:  entry.Payload,
		Version:  entry.Version,
		Deleted:  entry.Deleted,
	}
	var updated model.Entry
	path := fmt.Sprintf("/api/v1/data/%s", entry.ID)
	if err := c.doJSON(ctx, http.MethodPut, path, req, &updated, true); err != nil {
		return err
	}
	*entry = updated
	return nil
}

// GetEntry возвращает запись с сервера.
func (c *Client) GetEntry(ctx context.Context, entryID uuid.UUID) (*model.Entry, error) {
	var entry model.Entry
	path := fmt.Sprintf("/api/v1/data/%s", entryID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &entry, true); err != nil {
		return nil, err
	}
	return &entry, nil
}

type syncRequest struct {
	Since time.Time `json:"since"`
}

// SyncEntries возвращает записи, изменённые после указанного времени.
func (c *Client) SyncEntries(ctx context.Context, since time.Time) ([]model.Entry, error) {
	var entries []model.Entry
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/sync", syncRequest{Since: since}, &entries, true); err != nil {
		return nil, err
	}
	return entries, nil
}

// DeleteEntry удаляет запись на сервере.
func (c *Client) DeleteEntry(ctx context.Context, entryID uuid.UUID) error {
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/data/%s", entryID), nil, nil, true)
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, respBody any, authRequired bool) error {
	var body io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if authRequired {
		if c.token == "" {
			return fmt.Errorf("authorization required")
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("read error response: %w", readErr)
		}

		var errResp struct {
			Error string `json:"error"`
		}
		if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &errResp)
		}

		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    errResp.Error,
		}
	}

	if respBody != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
