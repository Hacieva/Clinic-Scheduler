# Clinic Scheduler

Система управления записью пациентов в частную клинику.

**Компоненты:**
- **Backend** — Go REST API (chi + pgx)
- **Bot** — Go Telegram bot (telego)
- **Frontend** — React admin + doctor panel (Vite + Tailwind)
- **DB** — PostgreSQL 16 (goose migrations)

## Требования

- [Docker](https://www.docker.com/) 24+
- [Docker Compose](https://docs.docker.com/compose/) v2+
- Go 1.22+ (для локальной разработки)
- Node.js 20+ (для локальной разработки)

## Quick Start

```bash
# 1. Скопируй конфиг
cp .env.example .env

# 2. Заполни JWT_SECRET и BOT_TOKEN в .env
# JWT_SECRET: openssl rand -hex 32

# 3. Запусти все сервисы
docker compose up --build

# Проверки:
# curl http://localhost:8000/health  → {"status":"ok"}
# http://localhost:5173              → React UI
# psql postgresql://clinic:clinic_pass@localhost:5432/clinic_db
```

## Разработка с hot reload

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up
```

## Структура проекта

```
clinic-scheduler/
├── backend/         Go REST API
│   ├── cmd/api/     точка входа
│   ├── internal/    бизнес-логика, репозитории, хендлеры
│   └── migrations/  goose SQL миграции
├── bot/             Telegram bot
│   └── cmd/bot/     точка входа
├── frontend/        React SPA
│   └── src/
├── docs/            ТЗ, роадмап, решения
└── docker-compose.yml
```

## Миграции

```bash
cd backend
make migrate-up    # применить
make migrate-down  # откатить
make migrate-status
```
