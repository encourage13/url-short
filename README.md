# url-short

Сервис коротких ссылок на Go: принимает длинный URL, выдает короткий код и перенаправляет по нему на исходный адрес, считая переходы.

## Технологии

- **Go 1.27**, стандартный `net/http` (роутинг через `ServeMux` с шаблонами методов и путей)
- **PostgreSQL 17** + **GORM** (драйвер pgx)
- **goose** — SQL-миграции, встроены в бинарник и применяются при старте
- **log/slog** — структурированные логи
- **Docker Compose** — приложение и Postgres; образ на `distroless`, запускается не от root
- Хранилище в памяти (`in-memory`) для локальной разработки без БД

## Структура

```
cmd/api/              точка входа: конфиг, выбор хранилища, запуск сервера
db/migrations/        SQL-миграции (goose)
internal/config/      чтение настроек из переменных окружения
internal/domain/      сущность URL и доменные ошибки
internal/generator/   генерация случайных кодов (crypto/rand)
internal/repository/  хранилища: in-memory и PostgreSQL (GORM)
internal/service/     бизнес-логика: валидация URL, генерация кода, счетчик кликов
internal/handler/     HTTP-хендлеры, DTO, middleware (логи, recover)
```

## Запуск

**Локально, без БД** (данные хранятся в памяти до перезапуска):

```bash
make run          # сервер на http://localhost:8080
```

**В Docker с PostgreSQL** — заполнить `.env` (см. ниже), затем:

```bash
make up           # собрать и запустить, сервис на http://localhost:8081
make logs         # логи
make down         # остановить
```

Остальные команды — `make help`.

## Настройки

Переменные окружения (для Docker — в файле `.env`):

| Переменная      | По умолчанию               | Описание                                        |
|-----------------|----------------------------|-------------------------------------------------|
| `SERVER_PORT`   | `8080`                     | порт сервера                                    |
| `BASE_URL`      | `http://localhost:<порт>`  | публичный адрес, из него собираются короткие ссылки |
| `CODE_LENGTH`   | `6`                        | длина кода, от 4 до 10                          |
| `ADMIN_TOKEN`   | —                          | токен для `GET /urls`; без него эндпоинт закрыт |
| `DATABASE_TYPE` | `in-memory`                | `in-memory` или `postgres`                      |
| `PG_HOST`, `PG_PORT`, `PG_USER`, `PG_PASSWORD`, `PG_DBNAME` | — | подключение к PostgreSQL          |
| `PG_SSLMODE`    | `disable`                  | режим SSL для PostgreSQL                        |

## API

| Метод и путь        | Описание                                                  |
|---------------------|-----------------------------------------------------------|
| `POST /shorten`     | создать ссылку, тело `{"url": "https://..."}`             |
| `GET /{code}`       | редирект (302) на исходный URL, +1 к счетчику             |
| `GET /info/{code}`  | информация о ссылке без увеличения счетчика               |
| `GET /urls`         | список ссылок, `?limit=50&offset=0`, нужен `Authorization: Bearer <ADMIN_TOKEN>` |
| `GET /health`       | проверка сервиса и доступности хранилища                  |

Пример:

```bash
curl -X POST http://localhost:8080/shorten -d '{"url": "https://go.dev"}'
```

```json
{
  "code": "upww2O",
  "short_url": "http://localhost:8080/upww2O",
  "original_url": "https://go.dev",
  "clicks": 0,
  "created_at": "2026-09-26T18:56:51.07+02:00"
}
```

Ошибки возвращаются как `{"error": "описание"}` с соответствующим HTTP-статусом.

## Тесты

```bash
make test         # все тесты с детектором гонок
```

Интеграционный тест PostgreSQL-репозитория запускается, только если задана строка подключения:

```bash
TEST_PG_DSN="postgres://user:pass@localhost:5434/db?sslmode=disable" go test ./internal/repository/
```
