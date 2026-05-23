# Database — Миграции и схема

## Инструмент

**goose** v3 — SQL-миграции с поддержкой Up/Down.

Установка локально:
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

## Переменная окружения

Все команды goose и make используют `DATABASE_URL`:

```bash
export DATABASE_URL="postgres://clinic:clinic_pass@localhost:5432/clinic_db?sslmode=disable"
```

## Команды (из backend/)

```bash
# Применить все миграции
make migrate-up

# Откатить последнюю миграцию
make migrate-down

# Статус применённых миграций
make migrate-status

# Создать новую миграцию (подставить имя через NAME=)
make migrate-create NAME=add_refresh_tokens
```

Напрямую через goose:
```bash
goose -dir migrations postgres "$DATABASE_URL" up
goose -dir migrations postgres "$DATABASE_URL" down
goose -dir migrations postgres "$DATABASE_URL" status
goose -dir migrations create my_migration sql
```

## Автозапуск в Docker

При старте контейнера `backend` entrypoint (`scripts/entrypoint.sh`) автоматически:
1. Запускает `goose ... up` перед стартом сервера
2. Запускает `/app/api`

Порядок гарантируется: backend зависит от `postgres: condition: service_healthy`.

## Сброс БД (dev)

```bash
# Из корня проекта:
./scripts/db-reset.sh
```

Скрипт: `docker compose down -v` → `docker compose up -d postgres` → `make migrate-up`

## Структура файлов

```
backend/migrations/
  20260519100000_init.sql       -- все таблицы, индексы, EXCLUDE constraint
  20260519100001_seed_admin.sql -- первый admin@clinic.local / changeme123
```

Новые файлы создаются в этой же папке с форматом `YYYYMMDDHHMMSS_name.sql`.

## Правило именования

```
20260519100000_init.sql
^             ^
timestamp     название (snake_case)
```

Goose применяет миграции строго по возрастанию timestamp.

## Schema Diagram

```
directions
  id, name*, description, is_active, timestamps
  * UNIQUE

users
  id, email* (lower), password_hash, role(admin|doctor), is_active, timestamps
  * UNIQUE

doctors
  id, user_id? → users, first_name, last_name, middle_name,
  cabinet, branch_address, description, photo_url, is_active, timestamps
  user_id UNIQUE

doctor_directions  [M2M]
  id, doctor_id → doctors CASCADE, direction_id → directions CASCADE
  UNIQUE(doctor_id, direction_id)

services
  id, doctor_id → doctors CASCADE, direction_id → directions CASCADE,
  name, description, duration_minutes (>0), price BIGINT (kopecks, nullable), is_active, timestamps

doctor_working_hours
  id, doctor_id → doctors CASCADE, day_of_week(1-7),
  start_time, end_time, is_active, timestamps
  CHECK(start_time < end_time)
  UNIQUE(doctor_id, day_of_week, start_time, end_time)

doctor_schedule_exceptions
  id, doctor_id → doctors CASCADE, date,
  type(day_off|custom_working_hours), start_time?, end_time?,
  comment, timestamps
  UNIQUE(doctor_id, date)

patients
  id, telegram_user_id? UNIQUE, telegram_username,
  full_name, phone, timestamps
  CHECK phone ~ '^\+?[0-9\s\-\(\)]{7,20}$'

appointments
  id, patient_id → patients, doctor_id → doctors,
  service_id → services, direction_id → directions,
  start_at TIMESTAMPTZ, end_at TIMESTAMPTZ,
  status(created|confirmed|cancelled_*|completed|no_show),
  source(telegram_bot|admin_panel), patient_comment, timestamps
  EXCLUDE USING GIST (doctor_id WITH =, tstzrange(start_at, end_at) WITH &&)
    WHERE status IN ('created', 'confirmed')

appointment_status_history
  id, appointment_id → appointments CASCADE,
  old_status?, new_status, changed_by_user_id → users SET NULL,
  changed_at, comment

audit_logs
  id, user_id → users, action, entity_type, entity_id,
  old_values JSONB, new_values JSONB, ip_address, user_agent, created_at

bot_sessions
  id, telegram_user_id UNIQUE, state, data JSONB, updated_at
```

## EXCLUDE constraint — защита от двойной записи

```sql
EXCLUDE USING GIST (
    doctor_id WITH =,
    tstzrange(start_at, end_at) WITH &&
) WHERE (status IN ('created', 'confirmed'))
```

Требует расширения `btree_gist` (включено в `20260519100000_init.sql`).

Работает в паре с backend-транзакцией (`SELECT ... FOR UPDATE`) — двойная защита.

## Тест пересечения

```sql
-- Вставляем первую запись (должна пройти):
INSERT INTO appointments(patient_id, doctor_id, service_id, direction_id,
  start_at, end_at, status, source) VALUES
  (1, 1, 1, 1, '2026-05-20 10:00+00', '2026-05-20 11:00+00', 'created', 'telegram_bot');

-- Вставляем пересекающуюся (должна упасть с EXCLUDE violation):
INSERT INTO appointments(patient_id, doctor_id, service_id, direction_id,
  start_at, end_at, status, source) VALUES
  (2, 1, 1, 1, '2026-05-20 10:30+00', '2026-05-20 11:30+00', 'created', 'telegram_bot');
-- ERROR: conflicting key value violates exclusion constraint "appointments_doctor_id_tstzrange_excl"
```

---

## Schema v0.3 — Planned (NOT YET MIGRATED)

> Статус: **Requirements only**. Миграции будут созданы при реализации v0.3.  
> Детальный PRD: `docs/PRD_v03_cashbox.md`

### Новые таблицы

```
referrers
  id, type(internal_doctor|external_doctor|clinic|other),
  full_name, specialization, workplace, phone,
  linked_doctor_id? → doctors SET NULL,
  is_active, timestamps

visits
  id, patient_id → patients, branch_id → branches,
  appointment_id? → appointments SET NULL,
  type(scheduled|walk_in),
  status(open|in_progress|completed|cancelled),
  cashier_user_id → users SET NULL,
  opened_at TIMESTAMPTZ, closed_at TIMESTAMPTZ?, notes,
  timestamps
  UNIQUE INDEX(appointment_id) WHERE appointment_id IS NOT NULL

receipts
  id, visit_id → visits, branch_id, cashier_user_id → users SET NULL,
  status(draft|paid|cancelled|refunded),
  payment_method(cash|card|online)?,
  subtotal BIGINT, discount BIGINT, total BIGINT  -- kopecks
  paid_at?, cancelled_at?, refunded_at?,
  cancel_reason?, cancelled_by_user_id → users SET NULL,
  timestamps

receipt_items
  id, receipt_id → receipts CASCADE,
  service_id → services, doctor_id → doctors,
  quantity INT (>0), unit_price BIGINT, discount BIGINT, total BIGINT  -- kopecks
  status(pending|performed|cancelled|refunded),
  performed_at?, performed_by_doctor_id → doctors SET NULL?,
  cancel_reason?, cancelled_by_user_id → users SET NULL?, cancelled_at?,
  timestamps

doctor_payouts
  id, doctor_id → doctors, branch_id → branches,
  period_from DATE, period_to DATE,
  total_amount BIGINT, paid_amount BIGINT  -- kopecks
  status(draft|approved|paid|partially_paid),
  approved_by_user_id → users SET NULL?, approved_at?,
  paid_at?, paid_by_user_id → users SET NULL?,
  notes, timestamps
  CHECK(period_from <= period_to)
  CHECK(paid_amount <= total_amount)

doctor_payout_items
  id, payout_id → doctor_payouts CASCADE,
  receipt_item_id → receipt_items,
  doctor_id → doctors, service_id → services,
  service_name VARCHAR(300),  -- snapshot
  performed_at TIMESTAMPTZ,
  gross_amount BIGINT,        -- kopecks, snapshot
  payout_rate DECIMAL(5,4),   -- e.g. 0.3500 = 35%
  payout_amount BIGINT,       -- kopecks
  created_at
  UNIQUE(receipt_item_id)     -- item в одной выплате максимум
```

### Изменения существующих таблиц

```
appointments
  + referrer_id? → referrers SET NULL  (nullable, additive)
```

### Planned migration files

```
backend/migrations/
  20260521100000_add_referrers.sql
  20260521100001_add_visits_receipts.sql
  20260521100002_add_payouts.sql
  20260521100003_appointments_add_referrer.sql
```

### Ключевые бизнес-правила на уровне БД

| Правило | Реализация |
|---|---|
| Один appointment → один visit | `UNIQUE INDEX visits(appointment_id) WHERE appointment_id IS NOT NULL` |
| Receipt item → одна выплата | `UNIQUE INDEX doctor_payout_items(receipt_item_id)` |
| paid_amount ≤ total_amount | `CHECK(paid_amount <= total_amount)` на doctor_payouts |
| Все суммы в копейках | `BIGINT` для всех денежных полей |
