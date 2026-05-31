// Package model определяет доменные типы данных GophKeeper.
package model

import (
	"time"

	"github.com/google/uuid"
)

// EntryType описывает тип хранимой записи.
type EntryType string

const (
	// EntryTypeCredentials — пара логин/пароль.
	EntryTypeCredentials EntryType = "credentials"
	// EntryTypeText — произвольный текст.
	EntryTypeText EntryType = "text"
	// EntryTypeBinary — произвольные бинарные данные.
	EntryTypeBinary EntryType = "binary"
	// EntryTypeCard — данные банковской карты.
	EntryTypeCard EntryType = "card"
)

// Entry представляет приватную запись пользователя на сервере.
type Entry struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      EntryType `json:"type"`
	Metadata  string    `json:"metadata"`
	Payload   []byte    `json:"payload"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Deleted   bool      `json:"deleted"`
}

// CredentialsPayload — расшифрованное содержимое записи типа credentials.
type CredentialsPayload struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TextPayload — расшифрованное содержимое текстовой записи.
type TextPayload struct {
	Text string `json:"text"`
}

// BinaryPayload — расшифрованное содержимое бинарной записи.
type BinaryPayload struct {
	Data []byte `json:"data"`
}

// CardPayload — расшифрованное содержимое записи банковской карты.
type CardPayload struct {
	Number     string `json:"number"`
	Holder     string `json:"holder"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
}

// User представляет зарегистрированного пользователя системы.
type User struct {
	ID        uuid.UUID `json:"id"`
	Login     string    `json:"login"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
