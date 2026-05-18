# Architecture Decision Records

## ADR-001: Стек технологий — Go + React + PostgreSQL + Docker

**Дата:** 2026-05-19  
**Статус:** Принято

### Контекст

Нужно выбрать стек для MVP системы записи пациентов частной клиники.
Требования: надёжность, простота деплоя, скорость разработки.

### Решение

- **Backend:** Go 1.22 + chi + pgx  
- **Frontend:** React 18 + Vite + Tailwind  
- **DB:** PostgreSQL 16  
- **Инфраструктура:** Docker Compose  
- **Telegram bot:** Go + telego

### Причины

**Go вместо Python/Node:**
- Статическая типизация снижает класс ошибок в runtime
- Один бинарник без runtime зависимостей → маленький Docker образ
- Высокая производительность для concurrent обработки записей
- Нативный `slog` для структурированного логирования

**pgx вместо ORM:**
- Полный контроль над SQL запросами
- Нет N+1 проблем из коробки
- EXCLUDE USING GIST constraint для защиты от двойных записей требует raw SQL

**PostgreSQL вместо MySQL:**
- EXCLUDE USING GIST — критичный constraint для защиты от overlapping записей
- Нативный JSONB для хранения FSM состояний бота
- tstzrange для работы с временными диапазонами

**React + Vite вместо Next.js:**
- Простота: чистый SPA без SSR (не нужен для admin panel)
- Быстрый dev server
- Меньше сложности для MVP

**Docker Compose вместо K8s:**
- MVP в одной клинике, не нужен горизонтальный скейлинг
- Простота локального запуска и деплоя на одном сервере

### Компромиссы

- Go требует больше boilerplate чем Python, но это разовые затраты
- Без ORM — больше SQL вручную, но полный контроль
- React SPA — нет SEO, но admin panel не нуждается в индексации

---

## ADR-002: Layered architecture (Handler → Service → Repository)

**Дата:** 2026-05-19  
**Статус:** Принято

### Решение

Строгое разделение: HTTP Handler → Service (бизнес-логика) → Repository (SQL).

### Причины

- Testability: Service тестируется с mock Repository без DB
- Maintainability: SQL изменяется только в Repository
- Single responsibility: каждый слой делает одно

---

## ADR-003: Soft delete через is_active

**Дата:** 2026-05-19  
**Статус:** Принято

### Решение

Врачи, услуги, направления никогда не удаляются физически.
`is_active = false` скрывает их из активных списков.

### Причины

- Сохраняется история записей (appointment → service → doctor)
- Нет orphaned FK references
- Простота аудита
