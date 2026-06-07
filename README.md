# GophKeeper

Клиент (CLI) и сервер (REST API). Данные шифруются на клиенте мастер-паролем, на сервер уходит уже ciphertext. Для входа на сервер требуется отдельный пароль аккаунта.

## Возможности

Хранится 4 типа записей: логин/пароль, текст, файл, банковская карта. У каждой записи есть `metadata` — произвольная строка вроде «github.com» или «рабочая почта».

Синхронизация между клиентами одного пользователя — через `sync`: клиент забирает всё, что изменилось на сервере с момента последней синхронизации.

## Как устроено

- `cmd/gophkeeper-server` — HTTP-сервер, chi, JWT, PostgreSQL
- `cmd/gophkeeper-client` — CLI на cobra
- `pkg/vault` — шифрование (PBKDF2 + AES-GCM)
- `pkg/storage` — локальный кеш в `~/.gophkeeper/vault.json`
- `internal/repository/migrations/` — SQL-миграции, встроены в бинарник сервера через `go:embed`

## Запуск

Нужны Go 1.24+ и PostgreSQL.

```bash
# база (или docker compose up -d)
export DATABASE_DSN="postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper?sslmode=disable"
export JWT_SECRET="dev-secret"

# сервер
go run ./cmd/gophkeeper-server

# клиент — в другом терминале
go run ./cmd/gophkeeper-client register --login alice
go run ./cmd/gophkeeper-client add-login --login user@site.com --password secret --metadata "mysite.com"
go run ./cmd/gophkeeper-client list
```

При регистрации и многих командах клиент спросит два пароля:
- **Account password** — для сервера
- **Master password** — для шифрования записей

Сборка клиента с версией:

```bash
go build -ldflags "-X github.com/porotikovaverk99-pixel/gophkeeper/pkg/version.Version=1.0.0 -X github.com/porotikovaverk99-pixel/gophkeeper/pkg/version.BuildDate=$(date -u +%Y-%m-%d)" -o gophkeeper-client ./cmd/gophkeeper-client
./gophkeeper-client version
```

## Команды клиента

```
register --login USER     регистрация
login --login USER        вход
sync                      синхронизация
list                      список записей
get ID                    показать запись
update ID [flags]         изменить (только переданные поля)
delete ID                 удалить

add-login    --login --password [--metadata]
add-text     --text [--metadata]
add-card     --number --holder --expiry --cvv [--metadata]
add-binary   --file PATH [--metadata]
```

Флаги `--server` (default `http://localhost:8080`) и `--storage` (default `~/.gophkeeper/vault.json`) работают для всех команд.

Пример update:

```bash
go run ./cmd/gophkeeper-client update <id> --password "newpass"
go run ./cmd/gophkeeper-client update <id> --text "другой текст"
```

## API сервера

Без авторизации: `GET /ping`, `POST /api/v1/register`, `POST /api/v1/login`.

С заголовком `Authorization: Bearer <token>`:

```
POST   /api/v1/data          создать
GET    /api/v1/data/{id}     получить
PUT    /api/v1/data/{id}     обновить
DELETE /api/v1/data/{id}     удалить (soft delete)
POST   /api/v1/sync          синхронизация, body: {"since": "2024-01-01T00:00:00Z"}
```

Тело записи:

```json
{
  "type": "credentials",
  "metadata": "mysite.com",
  "payload": "<зашифрованные байты>"
}
```

## Тесты

```bash
go test ./... -cover
```

В CI (`/.github/workflows/ci.yml`) — vet, тесты с `-race`, проверка покрытия ≥ 70%, сборка обоих бинарников.

## Переменные окружения сервера

```
SERVER_ADDRESS   адрес, default :8080
DATABASE_DSN     строка подключения к postgres
JWT_SECRET       секрет для токенов
LOG_LEVEL        уровень логов
```

Те же параметры можно передать флагами: `-a`, `-d`, `-j`, `-l`.
