# Clinic Scheduler — Project Rules for AI Agent

> **ВАЖНО:** Этот файл — обязательное чтение перед любой работой в проекте.
> Каждый агент (Claude Code, Cursor, Aider, etc.) должен прочесть его первым.

---

## 1. Что это за проект

**Clinic Scheduler** — система управления записью пациентов в частную клинику.

**Компоненты:**
- `backend/` — Go REST API (FastAPI заменён на Go)
- `bot/` — Go Telegram bot
- `frontend/` — React admin + doctor panel
- `db/` — PostgreSQL миграции (goose)
- `docker-compose.yml` — оркестрация

**Главная цель MVP:**
Админ управляет врачами/услугами/расписанием → Telegram бот показывает свободные окна → Пациент записывается → Врач видит расписание.

---
---

# PRODUCT & UX RULES (CRITICAL)

This system is NOT a generic admin dashboard.

This project is a clinic operational MIS inspired by Medlock workflow logic.

The most important thing in the system is NOT CRUD.
The most important thing is registrar speed and clinic workflow usability.

The UI must feel like a real clinic workspace.

---

## Core workflow

The main workflow is:

patient → doctor → service → time slot → appointment → payment → reporting

The schedule grid is the heart of the system.

Everything must optimize:
- speed of patient registration
- quick navigation
- visibility of doctor workload
- visibility of free slots
- operational work during clinic hours

---

## UX priorities

Priority order:

1. Schedule / appointment workflow
2. Patient flow
3. Doctor workload visibility
4. Fast registration
5. Reports / analytics
6. Settings

Settings pages are secondary.
The schedule is primary.

---

## Medlock-inspired behavior

The project should follow operational UX patterns similar to Medlock.

This means:

- compact operational UI
- dense information layout
- minimal empty whitespace
- fast filtering
- quick schedule editing
- clear current-time visibility
- strong visual differentiation of states

Do NOT build generic SaaS dashboards.

---

## Schedule grid requirements

The schedule page is the core page of the system.

### MUST HAVE

- doctor columns
- fixed visible timeline
- current time indicator
- past time visually dimmed
- blocked/non-working hours visually disabled
- hidden doctors on non-working days
- fast doctor search
- filtering by:
  - specialization
  - branch
  - service
  - doctor name
- compact doctor headers
- visible appointment cards
- visual differentiation:
  - booked
  - completed
  - cancelled
  - live queue
  - lunch break
  - blocked time

### MUST NOT

- huge empty spaces
- giant cards
- long checkbox lists of doctors
- placeholder-style layouts
- generic calendar widgets without clinic workflow adaptation

---

## Schedule editing behavior

Doctor schedule editing must support:

- weekly templates
- exceptions
- interactive day editing
- extending hours for a single day
- blocking specific hours
- lunch breaks
- vacations/day off
- emergency closure

The registrar/admin must be able to click directly on schedule cells.

Do not rely only on forms with checkboxes and time inputs.

---

## Patients page

Patients page must support:

- quick add patient
- phone search
- FIO search
- filtering:
  - gender
  - age
  - branch
  - source
  - doctor
- visit history
- quick appointment creation

---

## Services structure

Services should be organized as:

category → service → price → duration

Avoid exposing technical entities that confuse clinic staff.

Do not force users to understand internal database relationships.

---

## Specializations vs directions

“Directions” as a technical entity should not dominate UI.

Use:
- specializations for doctors
- categories for services

If directions exist internally in DB logic, keep them hidden unless truly necessary.

---

## External referrers

External referrers are important business entities.

Referrers must support:

- full name
- specialization
- workplace/LPU
- phone
- type
- referral statistics
- service commission %
- lab commission %
- exceptions per service/lab category
- future payout reports

This feature is business-critical.

---

## Reports

Reports must reflect real clinic operations.

Important reports:
- by doctors
- by services
- by referrers
- by branch
- cash register
- average bill
- cancellations/no-shows
- lab commissions
- referral payouts

Reports are not decorative widgets.

---

## UI philosophy

The system should feel:
- operational
- compact
- fast
- clinic-oriented

NOT:
- marketing dashboard
- startup SaaS admin panel
- empty analytics template

---

## Before implementing UI

Before any major UI implementation:

1. inspect existing workflow
2. compare with Medlock behavior
3. explain workflow problems
4. define acceptance criteria
5. then implement

Do not jump directly into coding.

---

## Definition of done for UI tasks

A UI task is NOT done unless:

- workflow can be completed quickly
- registrar can understand interface without explanation
- layout scales to many doctors/patients
- page works with realistic clinic load
- visual hierarchy is clear
- Playwright visual verification is performed

---

## 2. Главные бизнес-правила (НЕЛЬЗЯ нарушать)

1. **Только 2 роли:** `admin` и `doctor`. Никаких `patient`, `registrar`, `superadmin`.
2. **Пациент не имеет аккаунта** в БД. Он идентифицируется только по `telegram_user_id` и `phone`.
3. **Услуга принадлежит конкретному врачу** (`services.doctor_id`), не клинике.
4. **Один врач может иметь несколько направлений** (M2M через `doctor_directions`).
5. **Защита от двойной записи на 2 уровнях:**
   - Backend: транзакция + проверка пересечений
   - PostgreSQL: `EXCLUDE USING GIST` constraint на `tstzrange(start_at, end_at)`
6. **Расписание = недельный шаблон + исключения.** Исключения имеют приоритет.
7. **Минимальный шаг записи: 30 минут.**
8. **Soft delete** для врачей, услуг, направлений (через `is_active`). Никогда не удалять физически.
9. **Telegram bot НЕ ходит в БД напрямую.** Только через Backend API.
10. **Все datetime — TIMESTAMPTZ** (timezone-aware).

---

## 3. Архитектурные правила

### 3.1. Layered architecture

```
HTTP Handler → Service → Repository → Database
                  ↓
              Business logic ТОЛЬКО здесь
```

- **Handler** (`internal/api/`): парсит request, валидирует, вызывает Service
- **Service** (`internal/service/`): бизнес-логика, транзакции
- **Repository** (`internal/repository/`): только SQL-запросы, никакой логики
- **Model** (`internal/model/`): структуры данных

**НЕЛЬЗЯ:** SQL в Handler, бизнес-логика в Repository, прямой доступ к БД в Handler.

### 3.2. Ошибки

```go
// Доменные ошибки в internal/errors/
var (
    ErrNotFound       = errors.New("not found")
    ErrSlotTaken      = errors.New("time slot already taken")
    ErrOutsideHours   = errors.New("outside working hours")
    ErrDoctorInactive = errors.New("doctor is inactive")
)
```

Service возвращает доменные ошибки. Handler конвертирует их в HTTP коды:
- `ErrNotFound` → 404
- `ErrSlotTaken` → 409
- `ErrOutsideHours` → 422

### 3.3. Имена

- **Packages:** lowercase, single word (`appointment`, `doctor`)
- **Types:** PascalCase (`AppointmentService`)
- **Functions:** PascalCase для public, camelCase для private
- **DB tables:** plural snake_case (`doctors`, `doctor_working_hours`)
- **DB columns:** snake_case (`first_name`, `created_at`)

---

## 4. Правила работы агента

### 4.1. Перед началом любой задачи

1. **Прочитай `docs/TZ.md`** — там полное ТЗ.
2. **Прочитай `docs/ROADMAP.md`** — там на каком этапе сейчас.
3. **Прочитай `docs/DECISIONS.md`** — там принятые решения.
4. **Проверь `git status`** — нет ли незакоммиченных изменений.
5. **Прочитай существующие файлы** в зоне задачи. НЕ переписывай не глядя.

### 4.2. Scope изменений

- **Малая задача (1-3 файла):** делай сразу.
- **Средняя (4-10 файлов):** сначала покажи план, дождись подтверждения.
- **Большая (10+ файлов):** обязательно разбей на подзадачи.

### 4.3. После каждой задачи

```
✅ Готово.

Изменённые файлы:
  ➕  created   internal/service/appointment.go
  ✏️  modified  internal/api/handler.go
  🗑️  deleted   internal/old_thing.go

Что проверить:
  - go test ./internal/service/... 
  - curl http://localhost:8000/api/v1/appointments
  
Следующий шаг по roadmap: [ROADMAP.md строка X]
```

### 4.4. Что АГЕНТ ДОЛЖЕН делать всегда

- ✅ Писать тесты для бизнес-логики (Service layer)
- ✅ Добавлять `context.Context` первым параметром везде
- ✅ Использовать `slog` для логирования (не fmt.Println)
- ✅ Валидировать input в Handler через struct tags
- ✅ Использовать prepared statements
- ✅ Закрывать ресурсы через `defer`
- ✅ Возвращать ошибки, а не panic

### 4.5. Что АГЕНТ НЕ ДОЛЖЕН делать

- ❌ Использовать `ORM` (никакого GORM). Только sqlc или pgx + raw SQL.
- ❌ Хардкодить значения (часы работы, пути, токены)
- ❌ Делать `init()` функции с глобальным состоянием
- ❌ Использовать глобальные переменные кроме конфигурации
- ❌ Catch all errors (`_ =`)
- ❌ Создавать "удобные" wrapper'ы без необходимости
- ❌ Использовать reflection без крайней нужды
- ❌ Писать `interface{}` (используй `any` и только когда реально нужно)
- ❌ Делать миграции через `AutoMigrate` или `create_all()`. Только goose.

### 4.6. Запрет самодеятельности

**Правило одного промта:** агент выполняет ровно то, что написано в задаче. Не больше.

#### Что НЕЛЬЗЯ без явного разрешения:

- ❌ Создавать файлы, не упомянутые в задаче
- ❌ Устанавливать пакеты, не упомянутые в задаче
- ❌ Изменять файлы вне зоны задачи («попутный» рефакторинг, cleanup)
- ❌ Добавлять `docs/*`, README, changelog «для полноты»
- ❌ Запускать деструктивные команды: `docker compose down -v`, `git reset --hard`, `rm -rf`
- ❌ Делать `git push` или открывать PR
- ❌ Переходить к следующему этапу ROADMAP без подтверждения
- ❌ Исправлять «соседние» проблемы, не входящие в задачу
- ❌ Изменять структуру проекта, указанную в секции 6, без явного разрешения

#### Перед созданием любого нового файла агент обязан:

1. Проверить, существует ли файл/директория уже в секции 6.
2. Объяснить, зачем нужен новый файл.
3. Получить подтверждение, если файл не указан в структуре проекта.

#### Как действовать в пограничных случаях:

**Заметил проблему рядом с задачей** → упомяни в конце ответа, не исправляй молча.

**Промт допускает несколько трактовок** → выбери самую узкую. Спроси про остальные.

**Действие имеет необратимый побочный эффект** (пересоздание контейнера, удаление данных) → назови его явно одним предложением и жди подтверждения.

**Задача оказалась шире, чем казалось** → стопни, объясни, предложи разбить.

#### Шаблон завершения каждой задачи:

```
✅ Готово.

Изменённые файлы:
  ➕ created  ...
  ✏️ modified ...

Что проверить: [команды]

Следующий шаг по roadmap: [ссылка]. Жду подтверждения.
```

Агент **не переходит к следующему шагу сам** — только после явного «продолжай» или «да».

---

## 5. Стек технологий (зафиксировано)

### Backend
- **Go 1.22+**
- **chi router** (`github.com/go-chi/chi/v5`)
- **pgx v5** (`github.com/jackc/pgx/v5`) — НЕ database/sql
- **sqlc** для type-safe SQL (опционально, но рекомендуется)
- **goose** для миграций (`github.com/pressly/goose/v3`)
- **golang-jwt v5** для JWT
- **slog** для логирования (стандартная либа)
- **testify** для тестов (`github.com/stretchr/testify`)
- **swag** для Swagger из аннотаций

### Frontend
- **React 18** + **Vite 5**
- **react-router-dom v6**
- **@tanstack/react-query v5**
- **zustand v4** для глобального state
- **axios** для HTTP
- **tailwindcss** для стилей
- **lucide-react** для иконок
- **react-hook-form** + **zod** для форм
- **date-fns** для дат

### Telegram bot
- **go-telegram-bot-api/telegram-bot-api/v5** ИЛИ
- **mymmrac/telego** (новее, чище API) — выбираем второе

### Infrastructure
- **PostgreSQL 16**
- **Docker Compose**

---

## 6. Структура проекта (обязательная)

```
clinic-scheduler/
├── CLAUDE.md                    # ← этот файл
├── README.md                    # обзор + quickstart
├── docker-compose.yml
├── docker-compose.dev.yml       # overrides для dev
├── .env.example
├── .gitignore
│
├── docs/
│   ├── TZ.md                    # полное техзадание
│   ├── ROADMAP.md               # план этапов
│   ├── DECISIONS.md             # лог решений (ADR-style)
│   └── api.md                   # API контракты
│
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go          # точка входа API
│   ├── internal/
│   │   ├── api/                 # HTTP handlers + middleware
│   │   ├── service/             # бизнес-логика
│   │   ├── repository/          # доступ к БД
│   │   ├── model/               # структуры данных
│   │   ├── errors/              # доменные ошибки
│   │   ├── auth/                # JWT, password hashing
│   │   ├── config/              # загрузка из env
│   │   └── availability/        # КРИТИЧНО: расчёт свободных окон
│   ├── migrations/              # goose SQL миграции
│   ├── go.mod
│   └── Dockerfile
│
├── bot/
│   ├── cmd/
│   │   └── bot/
│   │       └── main.go
│   ├── internal/
│   │   ├── handler/             # обработчики команд/callback'ов
│   │   ├── flow/                # FSM сценарии (запись)
│   │   ├── client/              # HTTP клиент к backend API
│   │   └── keyboard/            # построение inline keyboards
│   ├── go.mod
│   └── Dockerfile
│
└── frontend/
    ├── src/
    │   ├── api/                 # axios клиент
    │   ├── components/          # переиспользуемые компоненты
    │   ├── pages/               # страницы (1 файл = 1 роут)
    │   ├── stores/              # zustand stores
    │   ├── hooks/               # React hooks
    │   ├── lib/                 # утилиты
    │   └── App.jsx
    ├── package.json
    └── Dockerfile
```

---

## 7. Критические компоненты

### 7.1. AvailabilityCalculator

Это **сердце системы**. Файл: `backend/internal/availability/calculator.go`.

**Что должен уметь:**
1. На вход: `doctor_id`, `service_id`, `date_from`, `date_to`
2. Прочитать рабочие часы врача (`doctor_working_hours`)
3. Применить исключения (`doctor_schedule_exceptions`)
4. Прочитать длительность услуги (`services.duration_minutes`)
5. Прочитать активные записи врача в диапазоне
6. Сгенерировать слоты с шагом 30 минут
7. Отфильтровать только те, где помещается услуга и нет пересечений с записями

**Обязательно покрыть тестами:**
- Базовый случай (1 день, без записей)
- День с записями (показать только свободные)
- День с исключением `day_off`
- День с `custom_working_hours`
- Услуга длиннее свободного интервала
- Граничные случаи (записи в самом начале/конце дня)

### 7.2. Appointment creation

Файл: `backend/internal/service/appointment.go`.

**Алгоритм:**
```go
func (s *AppointmentService) Create(ctx, input) error {
    tx, _ := db.BeginTx(ctx, ...)
    defer tx.Rollback()
    
    // 1. Проверить что врач активен
    doctor := repo.GetDoctor(tx, input.DoctorID)
    if !doctor.IsActive { return ErrDoctorInactive }
    
    // 2. Проверить что услуга активна и принадлежит врачу
    service := repo.GetService(tx, input.ServiceID)
    if service.DoctorID != doctor.ID { return ErrServiceMismatch }
    
    // 3. Проверить пересечения через SQL FOR UPDATE
    exists := repo.HasConflictLocked(tx, doctor.ID, input.StartAt, input.EndAt)
    if exists { return ErrSlotTaken }
    
    // 4. Создать пациента (по telegram_user_id, или найти существующего)
    patient := repo.UpsertPatientByTelegram(tx, input.Patient)
    
    // 5. Создать запись
    appt := repo.CreateAppointment(tx, ...)
    
    // 6. Создать запись в истории статусов
    repo.CreateStatusHistory(tx, appt.ID, "", "created", input.SourceUserID)
    
    return tx.Commit()
}
```

EXCLUDE constraint в БД — это **дополнительная** защита на случай гонок.

### 7.3. Telegram bot FSM

Файл: `bot/internal/flow/booking.go`.

**States:**
```
StateStart 
  → StateChooseDirection 
  → StateChooseDoctor 
  → StateChooseService 
  → StateChooseDate 
  → StateChooseTime 
  → StateEnterName 
  → StateEnterPhone 
  → StateConsent 
  → StateConfirm 
  → StateDone
```

Хранение state — **в БД** (`bot_sessions` таблица), не в памяти. Иначе при рестарте бота все диалоги потеряются.

---

## 8. Git workflow

### Branch naming
- `main` — production
- `develop` — текущая разработка
- `feature/availability-calculator` — фичи
- `fix/double-booking-race` — баги

### Commit messages (Conventional Commits)
```
feat(availability): add slot calculator with exceptions
fix(appointment): prevent overlap on concurrent requests
docs(api): add availability endpoint spec
test(availability): cover day_off exception case
refactor(service): extract patient upsert
```

### После каждой задачи
```bash
git add .
git commit -m "<type>(<scope>): <description>"
# НЕ push автоматически. Только когда задача полностью готова.
```

---

## 9. Тестирование

### Unit tests (обязательно для):
- `internal/availability/` — все edge cases
- `internal/service/appointment.go` — все бизнес-правила
- `internal/auth/` — пароли, JWT
- Любая чистая функция с логикой

### Integration tests:
- API endpoints с реальной БД (testcontainers)
- Bot flows (mock Telegram API)

### Запуск:
```bash
# Unit tests
go test ./internal/...

# С coverage
go test -cover ./internal/...

# Integration (требует Docker)
go test ./tests/integration/...
```

---

## 10. Когда что-то идёт не так

### Если агент не уверен → СПРАШИВАЙ
Не угадывай. Лучше задать вопрос, чем сгенерировать неправильно.

### Если задача слишком большая → РАЗБИВАЙ
> "Эта задача затрагивает 15 файлов. Предлагаю разбить на:
> 1. Добавить модель
> 2. Создать миграцию
> 3. Добавить repository
> 4. Добавить service
> 5. Добавить handler
> Начать с (1)?"

### Если требование противоречит ТЗ → УКАЖИ
> "Запрос противоречит правилу #3 в CLAUDE.md (роли).
> Может быть имелось в виду X? Подтверди."

### Если код становится сложным → УПРОЩАЙ
Лучше скучный понятный код, чем clever-but-cryptic. Это MVP.

---

## 11. Definition of Done для MVP

MVP считается готовым когда:

- [ ] Админ может создать врача, услугу, направление, расписание
- [ ] Расписание поддерживает несколько интервалов в день (обед)
- [ ] Можно добавить исключение (отпуск)
- [ ] Telegram бот показывает направления → врачей → услуги → даты → окна
- [ ] Запись через бот создаётся в БД
- [ ] Защита от двойной записи работает (тест с concurrent requests)
- [ ] Врач может зайти и увидеть своё расписание
- [ ] Врач НЕ видит телефон пациента
- [ ] Админ может отменить запись
- [ ] Все API endpoints в Swagger
- [ ] `docker-compose up` поднимает всё с нуля
- [ ] Unit tests на availability > 80% coverage
- [ ] README объясняет как запустить

---

## 12. Контакты для агента

Если что-то критически непонятно — стопни задачу и спроси пользователя.

Когда пользователь даёт задачу, агент должен:
1. Прочитать этот файл
2. Прочитать `docs/ROADMAP.md` чтобы понять текущий этап
3. Прочитать релевантные файлы проекта
4. Только потом начинать писать код
