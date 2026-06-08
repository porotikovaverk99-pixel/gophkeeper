package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/crypto"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/api"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/vault"
)

func printSuccess(format string, args ...any) {
	fmt.Fprintf(os.Stdout, "✓ "+format+"\n", args...)
}

func printError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
}

func humanizeError(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, vault.ErrWrongMasterPassword),
		errors.Is(err, crypto.ErrInvalidCiphertext):
		return "неверный мастер-пароль"
	}

	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			return "неверный логин или пароль аккаунта"
		case 409:
			return "пользователь с таким логином уже зарегистрирован"
		case 404:
			return "запись не найдена"
		}
		if apiErr.Message != "" {
			return apiErr.Message
		}
		return fmt.Sprintf("ошибка сервера (код %d)", apiErr.StatusCode)
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "invalid credentials"):
		return "неверный логин или пароль аккаунта"
	case strings.Contains(msg, "user already exists"):
		return "пользователь с таким логином уже зарегистрирован"
	case strings.Contains(msg, "not authenticated"):
		return "вы не авторизованы — выполните login или register"
	case strings.Contains(msg, "entry not found"):
		return "запись не найдена"
	case strings.Contains(msg, "authorization required"):
		return "требуется авторизация — выполните login"
	case strings.Contains(msg, "invalid token"):
		return "сессия истекла — выполните login повторно"
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "connect: connection"):
		return "не удалось подключиться к серверу — проверьте, что сервер запущен"
	}

	return err.Error()
}
