// Package repository предоставляет доступ к хранилищу данных GophKeeper.
package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
)

var (
	// ErrUserNotFound возвращается, если пользователь не найден.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists возвращается при попытке повторной регистрации.
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrEntryNotFound возвращается, если запись не найдена.
	ErrEntryNotFound = errors.New("entry not found")
)

// Storage описывает интерфейс хранилища GophKeeper.
type Storage interface {
	CreateUser(ctx context.Context, login, passwordHash string) (*model.User, error)
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*model.User, error)
	CreateEntry(ctx context.Context, entry *model.Entry) error
	UpdateEntry(ctx context.Context, entry *model.Entry) error
	GetEntry(ctx context.Context, userID, entryID uuid.UUID) (*model.Entry, error)
	ListEntries(ctx context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error)
	DeleteEntry(ctx context.Context, userID, entryID uuid.UUID) error
	Ping(ctx context.Context) error
}

// PostgresStorage реализует Storage поверх PostgreSQL.
type PostgresStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage создаёт подключение к PostgreSQL и применяет миграции.
func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return &PostgresStorage{pool: pool}, nil
}

func runMigrations(dsn string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	migrationsPath := filepath.Join(filepath.Dir(exePath), "migrations")
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		migrationsPath = "migrations"
		if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
			return fmt.Errorf("migrations directory not found: %w", err)
		}
	}

	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// CreateUser регистрирует нового пользователя.
func (s *PostgresStorage) CreateUser(ctx context.Context, login, passwordHash string) (*model.User, error) {
	user := &model.User{
		ID:        uuid.New(),
		Login:     login,
		Password:  passwordHash,
		CreatedAt: time.Now().UTC(),
	}

	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (id, login, password, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at`,
		user.ID, user.Login, user.Password, user.CreatedAt,
	).Scan(&user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

// GetUserByLogin возвращает пользователя по логину.
func (s *PostgresStorage) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	user := &model.User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, login, password, created_at FROM users WHERE login = $1`,
		login,
	).Scan(&user.ID, &user.Login, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by login: %w", err)
	}

	return user, nil
}

// GetUserByID возвращает пользователя по идентификатору.
func (s *PostgresStorage) GetUserByID(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, login, password, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Login, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

// CreateEntry сохраняет новую запись пользователя.
func (s *PostgresStorage) CreateEntry(ctx context.Context, entry *model.Entry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}

	now := time.Now().UTC()
	entry.CreatedAt = now
	entry.UpdatedAt = now
	if entry.Version == 0 {
		entry.Version = 1
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO data_entries
		 (id, user_id, type, metadata, payload, version, created_at, updated_at, deleted)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		entry.ID, entry.UserID, entry.Type, entry.Metadata, entry.Payload,
		entry.Version, entry.CreatedAt, entry.UpdatedAt, entry.Deleted,
	)
	if err != nil {
		return fmt.Errorf("create entry: %w", err)
	}

	return nil
}

// UpdateEntry обновляет существующую запись с проверкой версии.
func (s *PostgresStorage) UpdateEntry(ctx context.Context, entry *model.Entry) error {
	entry.UpdatedAt = time.Now().UTC()
	entry.Version++

	tag, err := s.pool.Exec(ctx,
		`UPDATE data_entries
		 SET type = $1, metadata = $2, payload = $3, version = $4,
		     updated_at = $5, deleted = $6
		 WHERE id = $7 AND user_id = $8`,
		entry.Type, entry.Metadata, entry.Payload, entry.Version,
		entry.UpdatedAt, entry.Deleted, entry.ID, entry.UserID,
	)
	if err != nil {
		return fmt.Errorf("update entry: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrEntryNotFound
	}

	return nil
}

// GetEntry возвращает запись пользователя по идентификатору.
func (s *PostgresStorage) GetEntry(ctx context.Context, userID, entryID uuid.UUID) (*model.Entry, error) {
	entry := &model.Entry{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, type, metadata, payload, version, created_at, updated_at, deleted
		 FROM data_entries WHERE id = $1 AND user_id = $2`,
		entryID, userID,
	).Scan(
		&entry.ID, &entry.UserID, &entry.Type, &entry.Metadata, &entry.Payload,
		&entry.Version, &entry.CreatedAt, &entry.UpdatedAt, &entry.Deleted,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntryNotFound
		}
		return nil, fmt.Errorf("get entry: %w", err)
	}

	return entry, nil
}

// ListEntries возвращает записи пользователя, изменённые после указанного времени.
func (s *PostgresStorage) ListEntries(ctx context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, type, metadata, payload, version, created_at, updated_at, deleted
		 FROM data_entries
		 WHERE user_id = $1 AND updated_at >= $2
		 ORDER BY updated_at ASC`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	var entries []model.Entry
	for rows.Next() {
		var entry model.Entry
		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Type, &entry.Metadata, &entry.Payload,
			&entry.Version, &entry.CreatedAt, &entry.UpdatedAt, &entry.Deleted,
		); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	return entries, nil
}

// DeleteEntry помечает запись как удалённую.
func (s *PostgresStorage) DeleteEntry(ctx context.Context, userID, entryID uuid.UUID) error {
	entry, err := s.GetEntry(ctx, userID, entryID)
	if err != nil {
		return err
	}

	entry.Deleted = true
	return s.UpdateEntry(ctx, entry)
}

// Ping проверяет доступность базы данных.
func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
