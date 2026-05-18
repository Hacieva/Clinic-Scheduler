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
