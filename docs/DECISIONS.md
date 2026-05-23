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

---

## ADR-004: Visit как учётная единица прихода пациента

**Дата:** 2026-05-23  
**Статус:** Принято (v0.3)

### Контекст

Нужно поддержать walk-in пациентов (без предварительной записи) и визиты с несколькими услугами от разных врачей. Прямая связь appointment → receipt недостаточна.

### Решение

Введена сущность `Visit` как учётная единица прихода пациента:
- `type=scheduled` — связан с `appointment_id`
- `type=walk_in` — `appointment_id = NULL`

Один визит содержит N услуг через `receipt_items`.  
Billing единица = visit, не appointment.

### Причины

- Appointment = намерение. Visit = факт прихода.
- Walk-in пациенты не имеют appointment, но должны попадать в статистику и оплату.
- Один визит может включать услуги нескольких врачей (УЗИ + консультация).

### Компромиссы

- Дополнительная сущность увеличивает сложность data model.
- Cashier должен явно открыть visit перед созданием receipt — дополнительный шаг в UI.

---

## ADR-005: Receipt создаётся на Visit, не на Appointment

**Дата:** 2026-05-23  
**Статус:** Принято (v0.3)

### Решение

```
visits → receipts → receipt_items
```

Запрещённая связь: `appointments.receipt_id` — не использовать.

### Причины

- Walk-in визиты не имеют appointment.
- Один визит теоретически может иметь несколько receipt (например, поправить позже).
- Финансовый документ должен быть привязан к факту визита, не к намерению.

### Как найти receipt для appointment

```sql
SELECT r.* FROM receipts r
JOIN visits v ON v.id = r.visit_id
WHERE v.appointment_id = $1;
```

---

## ADR-006: Payout врача только после performed, не после paid

**Дата:** 2026-05-23  
**Статус:** Принято (v0.3)

### Контекст

В клинике: оплата всегда до оказания услуги (политика). Но это не означает, что врач получает процент от оплаты — только от реально оказанных услуг.

### Решение

```
receipt_item.status = 'paid'      ← не существует (это receipt уровень)
receipt_item.status = 'performed' ← триггер для payout eligibility
```

Backend `POST /payouts` включает только `receipt_items WHERE status = 'performed'`.  
Refunded/cancelled items не участвуют в выплатах.

### Причины

- Врач не должен получать за отменённые или возвращённые услуги.
- Защита от ошибок: кассир оплатил, но врач не принял пациента.
- Юридическая корректность: выплата = за оказанную услугу.

### Компромиссы

- Cashier должен явно отметить каждую услугу как выполненную (extra click).
- UI должен напоминать о не отмеченных услугах в открытых визитах.

---

## ADR-007: Referrer как отдельная таблица, не FK к doctors

**Дата:** 2026-05-23  
**Статус:** Принято (v0.3)

### Контекст

Направитель может быть: врач нашей клиники, врач другого ЛПУ, организация, внешний специалист. Жёсткий FK к `doctors` не покрывает внешних направителей.

### Решение

Отдельная таблица `referrers` с полем `type`:
- `internal_doctor` → `linked_doctor_id` nullable FK к `doctors`
- `external_doctor` → только текстовые поля
- `clinic`, `other` → только текстовые поля

`appointments.referrer_id` → FK к `referrers` (nullable).

### Причины

- Единая модель для всех типов направителей.
- Внешние направители хранятся в системе (для аналитики кто направляет больше).
- Не ломается аналитика при удалении внутреннего врача (ON DELETE SET NULL).

### Компромиссы

- Сложнее JOIN для отчётов по внутренним направителям (нужен LEFT JOIN doctors).
- Нет FK-гарантии что `linked_doctor_id` соответствует `type=internal_doctor` (enforced в application layer).

---

## ADR-008: Финансовые записи иммутабельны — только смена статуса

**Дата:** 2026-05-23  
**Статус:** Принято (v0.3)

### Решение

Receipts, receipt_items, doctor_payouts физически не удаляются никогда.  
Только допустимые переходы статусов:

```
receipt:       draft → paid → refunded
               draft → cancelled

receipt_item:  pending → performed
               pending → cancelled
               performed → refunded (только через receipt refund)

doctor_payout: draft → approved → paid
                                → partially_paid
```

Все изменения логируются в `audit_logs` с old_values/new_values.

### Причины

- Финансовые документы = юридически значимые записи.
- Возможность аудита любого изменения.
- Защита от случайного удаления.

### Компромиссы

- Накапливаются "мёртвые" записи в draft/cancelled статусе.
- Нужна периодическая очистка/архивация (v0.4).
