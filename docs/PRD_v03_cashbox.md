# PRD v0.3 — Cashbox, Visit, Payout

**Версия:** 0.3  
**Дата:** 2026-05-23  
**Статус:** Draft — Requirements only. Nothing implemented.  
**Зависит от:** v0.2 (branches, owner role, patients module)

---

## Содержание

1. [Scope](#1-scope)
2. [Ключевые концепции](#2-ключевые-концепции)
3. [Entities — полный список](#3-entities--полный-список)
4. [DB Tables — новые](#4-db-tables--новые)
5. [DB Tables — изменения существующих](#5-db-tables--изменения-существующих)
6. [Workflow diagrams](#6-workflow-diagrams)
7. [API plan](#7-api-plan)
8. [Роли и доступ](#8-роли-и-доступ)
9. [Риски и митигации](#9-риски-и-митигации)
10. [Phased implementation order](#10-phased-implementation-order)
11. [Gaps — требуют уточнения](#11-gaps--требуют-уточнения)
12. [Out of scope — v0.4+](#12-out-of-scope--v04)

---

## 1. Scope

v0.3 добавляет кассовый модуль (cashbox) для работы клиники:

| Что добавляем | Почему |
|---|---|
| **Visit (визит)** | Учётная единица прихода пациента. Связывает appointment, услуги, оплату. |
| **Receipt (квитанция)** | Финансовый документ на визит. Заменяет прямую связь appointments → оплата. |
| **Walk-in workflow** | Пациенты по живой очереди — без предварительной записи. |
| **Doctor payout** | Архитектура выплат врачам. Выплата = только после реально оказанной услуги. |
| **Referrer (направитель)** | Внешний источник направления пациента. Отдельная таблица, не FK к doctors. |
| **Expanded audit logging** | Все финансовые изменения, отмены, возвраты — в audit_logs. |

**Не входит в v0.3 (см. раздел 12):** salary schemes, lab module, ATS, WhatsApp, analytics dashboards.

---

## 2. Ключевые концепции

### 2.1. Visit (Визит)

**Один приход пациента = один визит.**

```
Visit
 ├─ type: scheduled  ←→  appointment_id → appointments
 │                        (пациент пришёл по записи)
 └─ type: walk_in         (appointment_id = NULL)
                          (пациент пришёл без записи)
```

Один визит может содержать:
- несколько услуг (receipt_items)
- услуги разных врачей
- УЗИ, анализы, консультации, процедуры — все как receipt_items

Walk-in пациенты обязательно попадают в:
- статистику врача (через receipt_items.doctor_id)
- статистику филиала (через receipts.branch_id)
- аналитику услуг (через receipt_items.service_id)

### 2.2. Receipt (Квитанция)

**Receipt создаётся на visit, НЕ на appointment.**

```
visits (1) → (N) receipts → (N) receipt_items
```

Запрещённая связь: `appointments.receipt_id` — не использовать.

Receipt жизненный цикл:
```
draft → paid → (refunded)
             → (cancelled, если ещё не paid)
```

Статусы неизменяемы назад. Физическое удаление запрещено.

### 2.3. paid ≠ performed

**Ключевое правило системы:**

```
receipt.status = 'paid'          ← пациент заплатил
receipt_item.status = 'performed' ← врач оказал услугу
```

Это разные события. Оплата всегда до оказания (политика клиники).

**Payout врача появляется ТОЛЬКО после `performed`, не после `paid`.**

```
paid → [пациент идёт к врачу] → performed → eligible for payout
```

Refunded/cancelled receipt_items НЕ участвуют в выплатах.

### 2.4. Doctor Payout

Система хранит:
- Кто оказал услугу (performed_by_doctor_id)
- Когда (performed_at)
- Стоимость (gross_amount — снапшот цены на момент создания квитанции)
- Схему процента (payout_rate — снапшот на момент создания выплаты)
- Начисленная сумма (payout_amount)
- Когда выплачено (paid_at)
- Кто выплатил (paid_by_user_id)

Salary schemes (автоматический расчёт payout_rate из договора) — v0.4.  
В v0.3: payout_rate вводится вручную при создании выплаты.

### 2.5. Referrer (Направитель)

Внешний источник направления пациента. Не является пользователем системы.

Типы:
- `internal_doctor` — врач нашей клиники (linked_doctor_id nullable FK)
- `external_doctor` — врач другого ЛПУ
- `clinic` — организация / ЛПУ
- `other` — прочее

В форме записи:
- Autocomplete по full_name + specialization + workplace
- Вариант "Нет в базе" → appointment.referrer_id = NULL

---

## 3. Entities — полный список

### Новые сущности (v0.3)

| Сущность | Описание | Approval |
|---|---|---|
| `visits` | Визит пациента (scheduled или walk_in) | MEDIUM |
| `receipts` | Квитанция на визит | MEDIUM |
| `receipt_items` | Строки квитанции (одна услуга) | MEDIUM |
| `referrers` | Направители (внешние/внутренние) | MEDIUM |
| `doctor_payouts` | Сводная выплата врачу за период | DANGEROUS* |
| `doctor_payout_items` | Строки выплаты (одна услуга) | DANGEROUS* |

*DANGEROUS: изменяет финансовую логику. Каждый шаг требует явного подтверждения.

### Изменения существующих (v0.3)

| Таблица | Изменение | Риск |
|---|---|---|
| `appointments` | ADD COLUMN referrer_id nullable FK | LOW — additive, nullable |
| `audit_logs` | Расширить покрытие логирования (без schema changes) | SAFE |

### Уже существующие (переиспользуем без изменений)

`patients`, `doctors`, `services`, `branches`, `users`, `appointments`,  
`appointment_status_history`, `bot_sessions`, `directions`, `doctor_working_hours`

---

## 4. DB Tables — новые

> Все суммы хранятся в **копейках** (BIGINT), аналогично существующим `services.price`.

### visits

```sql
CREATE TABLE visits (
  id                BIGSERIAL PRIMARY KEY,
  patient_id        BIGINT NOT NULL REFERENCES patients(id),
  branch_id         BIGINT REFERENCES branches(id),
  appointment_id    BIGINT REFERENCES appointments(id) ON DELETE SET NULL,
  type              VARCHAR(20) NOT NULL
                    CHECK (type IN ('scheduled', 'walk_in')),
  status            VARCHAR(20) NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'in_progress', 'completed', 'cancelled')),
  cashier_user_id   BIGINT REFERENCES users(id) ON DELETE SET NULL,
  opened_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_at         TIMESTAMPTZ,
  notes             TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Один appointment → не более одного визита
CREATE UNIQUE INDEX visits_appointment_id_key
  ON visits(appointment_id)
  WHERE appointment_id IS NOT NULL;

CREATE INDEX visits_patient_id_idx ON visits(patient_id);
CREATE INDEX visits_branch_id_opened_at_idx ON visits(branch_id, opened_at DESC);
```

### receipts

```sql
CREATE TABLE receipts (
  id                    BIGSERIAL PRIMARY KEY,
  visit_id              BIGINT NOT NULL REFERENCES visits(id),
  branch_id             BIGINT REFERENCES branches(id),
  cashier_user_id       BIGINT REFERENCES users(id) ON DELETE SET NULL,
  status                VARCHAR(20) NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'paid', 'cancelled', 'refunded')),
  payment_method        VARCHAR(20)
                        CHECK (payment_method IN ('cash', 'card', 'online')),
  subtotal              BIGINT NOT NULL DEFAULT 0,   -- сумма items до скидки
  discount              BIGINT NOT NULL DEFAULT 0,   -- общая скидка
  total                 BIGINT NOT NULL DEFAULT 0,   -- subtotal - discount
  paid_at               TIMESTAMPTZ,
  cancelled_at          TIMESTAMPTZ,
  refunded_at           TIMESTAMPTZ,
  cancel_reason         TEXT,
  cancelled_by_user_id  BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX receipts_visit_id_idx ON receipts(visit_id);
CREATE INDEX receipts_branch_id_paid_at_idx
  ON receipts(branch_id, paid_at DESC)
  WHERE status = 'paid';
```

### receipt_items

```sql
CREATE TABLE receipt_items (
  id                      BIGSERIAL PRIMARY KEY,
  receipt_id              BIGINT NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
  service_id              BIGINT NOT NULL REFERENCES services(id),
  doctor_id               BIGINT NOT NULL REFERENCES doctors(id),
  quantity                INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
  unit_price              BIGINT NOT NULL,          -- снапшот цены на момент создания
  discount                BIGINT NOT NULL DEFAULT 0,
  total                   BIGINT NOT NULL,          -- (unit_price - discount) * quantity
  status                  VARCHAR(20) NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'performed', 'cancelled', 'refunded')),
  performed_at            TIMESTAMPTZ,
  performed_by_doctor_id  BIGINT REFERENCES doctors(id) ON DELETE SET NULL,
  cancel_reason           TEXT,
  cancelled_by_user_id    BIGINT REFERENCES users(id) ON DELETE SET NULL,
  cancelled_at            TIMESTAMPTZ,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX receipt_items_receipt_id_idx ON receipt_items(receipt_id);
CREATE INDEX receipt_items_doctor_id_idx ON receipt_items(doctor_id);
CREATE INDEX receipt_items_service_id_idx ON receipt_items(service_id);
-- Быстрый поиск выполненных, не включённых в выплату
CREATE INDEX receipt_items_performed_idx
  ON receipt_items(doctor_id, performed_at)
  WHERE status = 'performed';
```

### referrers

```sql
CREATE TABLE referrers (
  id                BIGSERIAL PRIMARY KEY,
  type              VARCHAR(30) NOT NULL
                    CHECK (type IN ('internal_doctor', 'external_doctor', 'clinic', 'other')),
  full_name         VARCHAR(300) NOT NULL,
  specialization    VARCHAR(200),
  workplace         VARCHAR(300),
  phone             VARCHAR(30),
  linked_doctor_id  BIGINT REFERENCES doctors(id) ON DELETE SET NULL,
                    -- заполняется только для type='internal_doctor'
  is_active         BOOLEAN NOT NULL DEFAULT true,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX referrers_full_name_idx ON referrers(full_name);
CREATE INDEX referrers_active_idx ON referrers(is_active) WHERE is_active = true;
```

### doctor_payouts

```sql
CREATE TABLE doctor_payouts (
  id                    BIGSERIAL PRIMARY KEY,
  doctor_id             BIGINT NOT NULL REFERENCES doctors(id),
  branch_id             BIGINT REFERENCES branches(id),
  period_from           DATE NOT NULL,
  period_to             DATE NOT NULL,
  total_amount          BIGINT NOT NULL DEFAULT 0,  -- сумма payout_items.payout_amount
  paid_amount           BIGINT NOT NULL DEFAULT 0,  -- фактически выплачено
  status                VARCHAR(20) NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'approved', 'paid', 'partially_paid')),
  approved_by_user_id   BIGINT REFERENCES users(id) ON DELETE SET NULL,
  approved_at           TIMESTAMPTZ,
  paid_at               TIMESTAMPTZ,
  paid_by_user_id       BIGINT REFERENCES users(id) ON DELETE SET NULL,
  notes                 TEXT,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

  CHECK (period_from <= period_to),
  CHECK (paid_amount <= total_amount)
);

CREATE INDEX doctor_payouts_doctor_id_period_idx
  ON doctor_payouts(doctor_id, period_from);
```

### doctor_payout_items

```sql
CREATE TABLE doctor_payout_items (
  id                BIGSERIAL PRIMARY KEY,
  payout_id         BIGINT NOT NULL REFERENCES doctor_payouts(id) ON DELETE CASCADE,
  receipt_item_id   BIGINT NOT NULL REFERENCES receipt_items(id),
  doctor_id         BIGINT NOT NULL REFERENCES doctors(id),
  service_id        BIGINT NOT NULL REFERENCES services(id),
  service_name      VARCHAR(300) NOT NULL,   -- снапшот названия
  performed_at      TIMESTAMPTZ NOT NULL,
  gross_amount      BIGINT NOT NULL,          -- снапшот receipt_item.unit_price
  payout_rate       DECIMAL(5,4) NOT NULL,   -- 0.3500 = 35%, вводится вручную в v0.3
  payout_amount     BIGINT NOT NULL,          -- floor(gross_amount * payout_rate)
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Каждая строка квитанции — в одной выплате максимум
CREATE UNIQUE INDEX doctor_payout_items_receipt_item_key
  ON doctor_payout_items(receipt_item_id);
```

---

## 5. DB Tables — изменения существующих

```sql
-- appointments: добавить referrer_id
ALTER TABLE appointments
  ADD COLUMN referrer_id BIGINT REFERENCES referrers(id) ON DELETE SET NULL;

CREATE INDEX appointments_referrer_id_idx
  ON appointments(referrer_id)
  WHERE referrer_id IS NOT NULL;
```

**Не добавлять** `appointments.visit_id` — обратная связь найдёт через  
`SELECT * FROM visits WHERE appointment_id = $1` (один запрос, индекс есть).

**Не добавлять** `appointments.receipt_id` — явно запрещено в концепции.

**audit_logs** — схема не меняется. Расширяем покрытие в application logic:

| Entity | Actions to log |
|---|---|
| receipts | create, pay, cancel, refund |
| receipt_items | perform, cancel, refund |
| doctor_payouts | create, approve, paid |
| appointments | status_change, referrer_set |
| users | role_change, password_change |
| doctors / services | create, update, deactivate |

---

## 6. Workflow diagrams

### Scheduled patient — полный цикл

```
appointment (created/confirmed)
          │
          ▼
  patient arrives at clinic
          │
          ▼
cashier opens visit
  visit.type = 'scheduled'
  visit.appointment_id = X
  visit.status = 'open'
          │
          ▼
cashier creates receipt
  receipt.status = 'draft'
  receipt.visit_id = V
          │
          ▼
cashier adds receipt_items
  per service: doctor_id, service_id, quantity, unit_price
  receipt.subtotal recalculated
          │
          ▼
    patient pays
  receipt.status = 'paid'
  receipt.payment_method = 'cash' | 'card'
  receipt.paid_at = now()
          │
          ▼        ← ПРАВИЛО: оплата ВСЕГДА до оказания
  patient sees doctor
  doctor performs service
          │
          ▼
  receipt_item.status = 'performed'
  receipt_item.performed_at = now()
  receipt_item.performed_by_doctor_id = D
          │
          ▼   (все items performed → visit closed)
  visit.status = 'completed'
  visit.closed_at = now()
          │
          ▼
  item eligible for payout
          │
  [async — owner creates payout for period]
          │
          ▼
  doctor_payout created (status='draft')
  doctor_payout_items added (payout_rate entered manually)
          │
          ▼
  owner approves → payout.status = 'approved'
          │
          ▼
  owner pays → payout.status = 'paid'
               payout.paid_at, paid_by_user_id set
```

### Walk-in patient — полный цикл

```
patient arrives without appointment
          │
          ▼
cashier searches patient
  → found by phone/name → use existing
  → not found → create patient (source='admin_panel', upsert by phone)
          │
          ▼
cashier creates walk-in visit
  visit.type = 'walk_in'
  visit.appointment_id = NULL
  visit.status = 'open'
          │
          ▼
  [same as scheduled: receipt → items → pay → perform → payout]
```

### Refund

```
receipt.status = 'paid'
          │
  [DANGEROUS action — требует подтверждения]
          │
          ▼
admin/owner initiates refund
  receipt.status = 'refunded'
  receipt.refunded_at, cancelled_by_user_id, cancel_reason
          │
          ▼
  per item: receipt_item.status = 'refunded'
            (если item уже в payout → требует ручной корректировки payout)
          │
          ▼
  audit_log: action='receipt_refunded', entity_type='receipt', entity_id=R
             old_values={status:'paid'}, new_values={status:'refunded', reason:...}
```

### Cashier shift report (read-only)

```
GET /api/v1/reports/cashier/shift?date=&cashier_user_id=&branch_id=

Агрегация за смену:
  cash_total      = SUM(receipt.total WHERE payment_method='cash' AND status='paid')
  card_total      = SUM(receipt.total WHERE payment_method='card' AND status='paid')
  refunds_total   = SUM(receipt.total WHERE status='refunded')
  items_count     = COUNT(receipt_items WHERE status IN ('performed','pending'))
  [expenses_total = pending — requires expenses table, see Gaps]
  gross_total     = cash_total + card_total - refunds_total
```

---

## 7. API plan

### Visits

```
POST   /api/v1/visits
       Body: {patient_id, branch_id, type, appointment_id?, notes?}
       Auth: admin | owner

GET    /api/v1/visits
       Query: patient_id?, branch_id?, date?, type?, status?
       Auth: admin | owner

GET    /api/v1/visits/:id
       Auth: admin | owner

PATCH  /api/v1/visits/:id/status
       Body: {status, notes?}
       Auth: admin | owner
```

### Receipts

```
POST   /api/v1/visits/:visitId/receipts
       Body: {branch_id?}
       Auth: admin | owner

GET    /api/v1/visits/:visitId/receipts
       Auth: admin | owner

GET    /api/v1/receipts/:id
       Auth: admin | owner

POST   /api/v1/receipts/:id/items
       Body: {service_id, doctor_id, quantity?, discount?}
       Auth: admin | owner
       Validation: receipt.status = 'draft' only

DELETE /api/v1/receipts/:id/items/:itemId
       Auth: admin | owner
       Validation: receipt.status = 'draft' only

POST   /api/v1/receipts/:id/pay
       Body: {payment_method: 'cash' | 'card' | 'online'}
       Auth: admin | owner

POST   /api/v1/receipts/:id/cancel
       Body: {reason}
       Auth: admin | owner
       Validation: status must be 'draft'

POST   /api/v1/receipts/:id/refund
       Body: {reason}
       Auth: owner only   ← возвраты только owner
       Validation: status must be 'paid'

POST   /api/v1/receipts/:id/items/:itemId/perform
       Body: {performed_by_doctor_id?, performed_at?}
       Auth: admin | owner
```

### Referrers

```
GET    /api/v1/referrers?search=&is_active=true
       Auth: admin | owner

GET    /api/v1/referrers/:id
       Auth: admin | owner

POST   /api/v1/referrers
       Body: {type, full_name, specialization?, workplace?, phone?, linked_doctor_id?}
       Auth: admin | owner

PATCH  /api/v1/referrers/:id
       Auth: admin | owner

DELETE /api/v1/referrers/:id   (soft: is_active=false)
       Auth: owner only
```

### Doctor Payouts

```
GET    /api/v1/payouts?doctor_id=&from=&to=&status=&branch_id=
       Auth: owner only

GET    /api/v1/payouts/:id
       Auth: owner only

POST   /api/v1/payouts
       Body: {doctor_id, branch_id?, period_from, period_to}
       → сервер автоматически находит все eligible receipt_items
       Auth: owner only

POST   /api/v1/payouts/:id/items/add
       Body: [{receipt_item_id, payout_rate}]  ← rate вводится вручную в v0.3
       Auth: owner only

POST   /api/v1/payouts/:id/approve
       Auth: owner only
       Validation: status = 'draft'

POST   /api/v1/payouts/:id/mark-paid
       Body: {paid_amount, notes?}
       Auth: owner only
       Validation: status = 'approved'
```

### Reports

```
GET    /api/v1/reports/cashier/shift
       Query: date, cashier_user_id?, branch_id?
       Auth: admin | owner   (admin видит только свою смену)

GET    /api/v1/reports/revenue
       Query: from, to, branch_id?
       Auth: owner only

GET    /api/v1/reports/doctors/payout-summary
       Query: from, to, doctor_id?, branch_id?
       Auth: owner only

GET    /api/v1/reports/audit
       Query: entity_type?, entity_id?, from?, to?
       Auth: owner only
```

---

## 8. Роли и доступ

В v0.3 предполагается роль `admin` = регистратор = кассир (один человек).  
Отдельной роли `cashier` не вводим.

| Операция | owner | admin | doctor |
|---|---|---|---|
| Открыть визит | ✅ | ✅ | ❌ |
| Создать/изменить квитанцию | ✅ | ✅ | ❌ |
| Отметить услугу выполненной | ✅ | ✅ | ❌ |
| Провести возврат | ✅ | ❌ | ❌ |
| Управлять направителями | ✅ | ✅ | ❌ |
| Создать/утвердить выплату врачу | ✅ | ❌ | ❌ |
| Просмотр смены (своей) | ✅ | ✅ | ❌ |
| Просмотр отчётов owner | ✅ | ❌ | ❌ |
| Просмотр audit_logs | ✅ | ❌ | ❌ |

---

## 9. Риски и митигации

| Риск | Severity | Митигация |
|---|---|---|
| **Circular FK**: visit ↔ appointment | HIGH | One-way: `visits.appointment_id → appointments`. Обратный поиск: `SELECT * FROM visits WHERE appointment_id = $1`. Не добавлять `appointments.visit_id`. |
| **paid ≠ performed**: кассир не отмечает выполнение | HIGH | UI блокирует создание payout для item без `performed` статуса. Backend: `POST /payouts` включает только items WHERE status='performed'. |
| **Partial refund**: часть items выполнена, часть отменена | MEDIUM | Статус на уровне receipt_item, не receipt. Receipt.status='refunded' только если ВСЕ items refunded или cancelled. |
| **Payout rate изменяется** со временем | LOW | Снапшот payout_rate хранится в doctor_payout_items. Rate — input при создании, не FK к конфигу. |
| **Receipt item попадает в 2 payout** | MEDIUM | `UNIQUE INDEX doctor_payout_items(receipt_item_id)` на уровне БД. |
| **Walk-in дубликат пациента** | MEDIUM | Поиск по phone перед созданием. AdminCreate: upsert by phone (не INSERT). |
| **Скорость 20-40 сек** для кассира | HIGH | Все autocomplete < 50ms. Индексы на `patients(phone)`, `services(is_active)`, `referrers(full_name)`. Нет N+1 в cashier API. |
| **Audit log объём** | LOW | Индекс (entity_type, entity_id), (created_at DESC). Партиционирование по месяцу — v0.4. |
| **Expenses (расходы) не определены** | MEDIUM | Gap — см. раздел 11. Смена-отчёт будет неполным без `expenses` таблицы. |
| **Выплата после возврата** | HIGH | После refund: если item уже включён в payout → система предупреждает, ручная корректировка. Backend НЕ удаляет payout_item автоматически — финансовые записи иммутабельны. |

---

## 10. Phased implementation order

### Phase 1 — DB migrations (MEDIUM, additive, no code changes)

```
20260521100000_add_referrers.sql
  → CREATE TABLE referrers
  → ALTER TABLE appointments ADD COLUMN referrer_id

20260521100001_add_visits_receipts.sql
  → CREATE TABLE visits
  → CREATE TABLE receipts
  → CREATE TABLE receipt_items
  → CREATE INDEX (все перечисленные выше)

20260521100002_add_payouts.sql
  → CREATE TABLE doctor_payouts
  → CREATE TABLE doctor_payout_items
```

### Phase 2 — Backend: Referrers (MEDIUM, standalone, no dependencies)

```
internal/model/referrer.go
internal/repository/referrer.go  — List(search), GetByID, Create, Update, Deactivate
internal/service/referrer.go
internal/api/handler/referrer.go
```

Включить в `main.go`. Тест: CRUD + autocomplete search.

### Phase 3 — Backend: Visits + Receipts (MEDIUM)

```
internal/model/visit.go
internal/model/receipt.go
internal/repository/visit.go
internal/repository/receipt.go
internal/service/visit.go     — open/close, walk_in create, scheduled linkage
internal/service/receipt.go   — create, add_item, pay, cancel, refund, perform_item
internal/api/handler/visit.go
internal/api/handler/receipt.go
```

Бизнес-правила в service:
- `pay` требует хотя бы одного item
- `cancel` запрещён если status='paid'
- `refund` запрещён если status != 'paid'
- `add_item` запрещён если status != 'draft'
- `perform_item` требует receipt.status='paid'

### Phase 4 — Frontend: Cashier module (SAFE/MEDIUM)

```
src/pages/admin/cashbox/
  OpenVisitPage.jsx      — поиск/создание пациента + выбор типа визита
  ReceiptPage.jsx        — добавление услуг, оплата
  ShiftReportPage.jsx    — отчёт за смену

src/api/visits.js
src/api/receipts.js
src/api/referrers.js
```

UX требования (walk-in ≤ 40 сек):
- Patient search: autocomplete с debounce 200ms
- Service selection: autocomplete + быстрый список «частые услуги»
- Payment: один клик на метод оплаты → подтверждение → готово

### Phase 5 — Backend + Frontend: Payouts (DANGEROUS)

```
internal/model/payout.go
internal/repository/payout.go  — с поиском eligible items
internal/service/payout.go     — create, approve, mark_paid, calculate eligible
internal/api/handler/payout.go
src/pages/admin/settings/PayoutsPage.jsx  — owner only
```

Каждый шаг (create, approve, mark_paid) = отдельное явное подтверждение.

### Phase 6 — Reports (MEDIUM)

```
internal/api/handler/reports.go
  GET /reports/cashier/shift
  GET /reports/revenue
  GET /reports/doctors/payout-summary
  GET /reports/audit
```

Нет новых таблиц. Только SQL агрегации по существующим.

---

## 11. Gaps — требуют уточнения

### 11.1. Expenses (Расходы)

Смена-отчёт кассира включает «расходы», но сущность не определена.

Нужно уточнить:
- Что такое расходы в кассе? (хозяйственные, закупки, аренда?)
- Кто их вводит? (кассир, бухгалтер?)
- Нужна ли привязка к филиалу/смене?
- Требуется ли чек/документ?

Предварительная структура (НЕ финализировано):
```sql
expenses: id, branch_id, category VARCHAR, amount BIGINT, 
          description TEXT, cashier_user_id, created_at
```

### 11.2. Cashier role vs admin role

Текущее решение: admin = кассир (один человек). Если в клинике кассир и регистратор — разные люди — потребуется отдельная роль `cashier` в v0.4.

### 11.3. Salary schemes (v0.4)

В v0.3 payout_rate вводится вручную при создании выплаты.  
В v0.4 нужна таблица `doctor_salary_schemes`:
```
id, doctor_id, service_id?, direction_id?, 
rate DECIMAL(5,4), valid_from DATE, valid_to DATE nullable
```

### 11.4. Reminder / notification

После оплаты / после выполнения услуги — нужны ли уведомления пациенту?  
(Telegram / SMS — не определено для v0.3)

---

## 12. Out of scope — v0.4+

| Что | Почему отложено |
|---|---|
| Salary schemes (автоматический payout_rate) | Требует contracts/rates таблицы, сложная логика, не блокирует v0.3 |
| Lab module (заказы, комплексы, PDF результаты) | Отдельная система, receipt_item.type='lab' достаточно для v0.3 |
| Day hospital / procedure room | Отдельный workflow, receipt_items покрывает базовые случаи |
| ATS интеграция (МегаФон) | Требует enterprise договора |
| WhatsApp / SMS уведомления | Additive, не блокирует кассовый модуль |
| Google Sheets sync | Удобство, не критично для MVP |
| Medlock / Dikidi sync | Зависит от их API |
| Analytics dashboards | Достаточно shift report + revenue report в v0.3 |
| Drag & drop schedule | UX улучшение, не блокирует |
| Realtime / WebSocket | Нет требований на v0.3 |
| Partial payout (частичная выплата) | Поддержано архитектурой (paid_amount ≤ total_amount), UI — v0.4 |
