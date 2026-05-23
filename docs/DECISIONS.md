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

## ADR-009: Services — глобальный прайс клиники без привязки к врачу/филиалу

**Дата:** 2026-05-23  
**Статус:** Принято

### Контекст

Исходная модель: `services.doctor_id` — услуга принадлежит одному врачу. Проблема: "Консультация дерматолога" у доктора Иванова не может использоваться доктором Петровым без дублирования записи.

### Решение

```
services     — глобальный прайс клиники (без doctor_id, без branch_id)
  + category: consultation | ultrasound | lab | procedure | other
  + is_system: true для sentinel-записей (не показывать в UI как обычную услугу)

doctor_services  [M2M junction]
  doctor_id → doctors CASCADE
  service_id → services CASCADE
  UNIQUE(doctor_id, service_id)
```

Правила:
- `services` не имеют `branch_id` — branch-специфичность определяется через врачей
- `services.price` — единая цена, per-doctor override **не делаем в v0.3**
- Удаление только через `is_active=false` — история receipts/payouts должна сохраняться
- `is_system=true` — внутренние sentinel-записи (напр. "Прочая услуга"); UI их не показывает в стандартном выборе
- КМН-надбавка решается отдельной строкой прайса: "Консультация дерматолога КМН" ≠ "Консультация дерматолога"

### Изменение валидации appointment

```
Было: service.doctor_id == appointment.doctor_id → ErrServiceMismatch
Стало: EXISTS(doctor_services WHERE doctor_id=? AND service_id=?) → ErrServiceMismatch
```

### UX-правила

- DoctorServicesTab: цена read-only; ссылка "Изменить в прайсе" → ServicesPage
- Кнопка удаления у врача называется "Убрать услугу у врача" (не "Удалить услугу")
- WalkInPage: два flow — doctor-first (getDoctorServices) и service-first (getServices?search=, возвращает doctors[])

### Причины

- Консультации одного типа стоят одинаково независимо от врача — кроме КМН-надбавки, которая решается отдельной строкой
- Один прайс проще поддерживать, проще аудит, проще сравнение
- Doctors = branch-specific; services = clinic-wide; их пересечение = doctor_services

### Компромиссы

- Нельзя задать персональную цену без модели override (решение: v0.4 по необходимости)
- Назначение услуг врачам — ручная операция (нет автоматического наследования по direction)

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
