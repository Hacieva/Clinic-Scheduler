# ROADMAP — План разработки MVP

> **Как пользоваться этим файлом:**
> 1. Найди текущий этап (отмечен `🔄 ТЕКУЩИЙ`)
> 2. Скопируй промпт целиком и вставь агенту
> 3. После выполнения задачи отметь её `✅ DONE` и переходи к следующей
> 4. **НЕ перескакивай этапы.** Зависимости важны.

---

## 📅 Календарь (при 12ч/день)

| День | Этап | Что получим |
|------|------|-------------|
| 1 | Setup | Структура проекта, Docker запускается |
| 2 | DB схема + миграции | БД с правильными таблицами |
| 3 | AvailabilityCalculator | Расчёт свободных окон + тесты |
| 4 | Auth + Models | JWT работает, есть admin/doctor |
| 5 | CRUD: directions, doctors | Админ может управлять врачами |
| 6 | CRUD: services, schedule | Услуги и расписание |
| 7 | Appointments API | Создание/отмена записей |
| 8 | Telegram bot (часть 1) | Бот показывает direction → doctor → service |
| 9 | Telegram bot (часть 2) | Бот создаёт запись |
| 10 | Admin frontend (часть 1) | Логин, врачи, услуги |
| 11 | Admin frontend (часть 2) | Расписание, записи |
| 12 | Doctor frontend | Кабинет врача |
| 13 | Тесты + полировка | Покрытие тестами, баг-фиксы |
| 14 | Docker prod + deploy | Production-ready |

---

# 🔄 ЭТАП 0: Setup проекта

**Статус:** 🔄 ТЕКУЩИЙ

**Цель:** Инициализировать проект так, чтобы Docker Compose поднимал PostgreSQL и заглушки backend/bot/frontend.

**Зависимости:** Нет.

## Промпт для агента

```
Ты работаешь в проекте Clinic Scheduler. Прочитай CLAUDE.md перед началом.

ЗАДАЧА: Setup нового проекта.

1. Создай структуру папок согласно CLAUDE.md секция 6:
   backend/, bot/, frontend/, docs/, migrations/

2. Создай Go module для backend:
   cd backend && go mod init github.com/USERNAME/clinic-scheduler/backend
   Добавь зависимости в go.mod:
     - github.com/go-chi/chi/v5
     - github.com/jackc/pgx/v5
     - github.com/jackc/pgx/v5/pgxpool
     - github.com/pressly/goose/v3
     - github.com/golang-jwt/jwt/v5
     - github.com/stretchr/testify
     - github.com/joho/godotenv
   Запусти go mod tidy.

3. Создай backend/cmd/api/main.go — простейший HTTP сервер на chi
   с endpoint GET /health -> {"status":"ok"}. Слушает порт 8000.

4. Аналогично создай Go module для bot:
   cd bot && go mod init github.com/USERNAME/clinic-scheduler/bot
   Зависимости:
     - github.com/mymmrac/telego
   bot/cmd/bot/main.go — пока просто печатает "bot starting..." и спит.

5. Создай frontend через Vite:
   cd frontend && npm create vite@latest . -- --template react
   Установи Tailwind, axios, react-router-dom, @tanstack/react-query, zustand.
   Удали стартовый код, оставь главную страницу с "Clinic Scheduler".

6. Создай Dockerfile для каждого сервиса:
   - backend/Dockerfile (multi-stage build, alpine)
   - bot/Dockerfile (multi-stage build, alpine)
   - frontend/Dockerfile (multi-stage build, nginx serve)

7. Создай docker-compose.yml в корне:
   Services: postgres, backend, bot, frontend
   Networks: один bridge network
   Volumes: postgres_data
   Healthcheck для postgres
   depends_on: postgres healthy

8. Создай docker-compose.dev.yml — overrides для разработки:
   - backend с air для hot reload
   - frontend с npm run dev
   - все порты проброшены

9. Создай .env.example в корне с переменными:
   POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB
   JWT_SECRET (с инструкцией openssl rand -hex 32)
   BOT_TOKEN
   API_URL=http://backend:8000

10. Создай .gitignore (Go, Node, Docker, IDE).

11. Создай README.md с разделами:
    - Описание
    - Требования (Docker, Go 1.22+, Node 20+)
    - Quick start (docker compose up)
    - Структура проекта

12. Создай docs/DECISIONS.md и запиши первое решение:
    "ADR-001: Стек Go + React + PostgreSQL + Docker. Причина: ..."

13. Инициализируй git, сделай commit "chore: project setup".

КРИТЕРИИ ПРИЁМКИ:
- docker compose up поднимает все сервисы
- curl http://localhost:8000/health возвращает {"status":"ok"}
- curl http://localhost:5173 показывает React страницу
- В логах bot видно "bot starting..."
- psql может подключиться к localhost:5432

В КОНЦЕ покажи мне:
- tree -L 3 (структуру)
- содержимое docker-compose.yml
- git log
```

**Когда готово:**
```bash
git checkout -b feature/setup
git push origin feature/setup
```

---

# ЭТАП 1: DB схема + миграции

**Статус:** ⏳ Ожидает

**Цель:** Все таблицы созданы, миграции работают, EXCLUDE constraint защищает от пересечений.

**Зависимости:** Этап 0.

## Промпт для агента

```
Этап 1 из ROADMAP. Прочитай CLAUDE.md и docs/TZ.md секции 5, 6.

ЗАДАЧА: Создать SQL миграции (goose) для всех таблиц MVP.

1. Установи goose как dependency tool:
   cd backend && go install github.com/pressly/goose/v3/cmd/goose@latest

2. Создай папку backend/migrations/

3. Создай первую миграцию: goose create init sql
   Файл будет: migrations/YYYYMMDDHHMMSS_init.sql

4. В этой миграции создай таблицы (в правильном порядке для FK):

   a) directions: id, name (unique), description, is_active, timestamps
   
   b) users: id, email (unique, lower), password_hash, role (admin|doctor), is_active, timestamps
   
   c) doctors: id, user_id (FK users, nullable, unique), first_name, last_name, 
      middle_name, cabinet, branch_address, description, photo_url, is_active, timestamps
   
   d) doctor_directions: id, doctor_id (FK doctors CASCADE), direction_id (FK directions CASCADE),
      created_at. UNIQUE(doctor_id, direction_id).
   
   e) services: id, doctor_id (FK doctors CASCADE), direction_id (FK directions CASCADE),
      name, description, duration_minutes (>0), price (decimal 10,2), is_active, timestamps
   
   f) doctor_working_hours: id, doctor_id (FK doctors CASCADE), day_of_week (1-7),
      start_time, end_time, is_active, timestamps. CHECK start_time < end_time.
   
   g) doctor_schedule_exceptions: id, doctor_id (FK doctors CASCADE), date, 
      type (day_off|custom_working_hours), start_time, end_time, comment, timestamps.
      UNIQUE(doctor_id, date).
   
   h) patients: id, telegram_user_id (unique nullable), telegram_username, 
      full_name, phone, timestamps. CHECK phone matches regex.
   
   i) appointments: id, patient_id (FK), doctor_id (FK), service_id (FK), direction_id (FK),
      start_at (timestamptz), end_at (timestamptz), status, source, patient_comment, timestamps.
      
      CRITICAL — добавь EXCLUDE constraint:
      
      ALTER TABLE appointments ADD CONSTRAINT no_overlapping_appointments
        EXCLUDE USING GIST (
          doctor_id WITH =,
          tstzrange(start_at, end_at) WITH &&
        ) WHERE (status IN ('created', 'confirmed'));
      
      Для GIST с integer нужно: CREATE EXTENSION btree_gist;
   
   j) appointment_status_history: id, appointment_id (FK CASCADE), old_status, new_status,
      changed_by_user_id (FK users SET NULL), changed_at, comment
   
   k) audit_logs: id, user_id (FK), action, entity_type, entity_id, old_values jsonb,
      new_values jsonb, ip_address, user_agent, created_at
   
   l) bot_sessions: id, telegram_user_id (unique), state, data jsonb, updated_at
      (это для FSM состояний бота)

5. Добавь индексы:
   - doctors(is_active) WHERE is_active=true
   - appointments(doctor_id, start_at)
   - appointments(patient_id)
   - appointments(status)
   - patients(telegram_user_id)
   - audit_logs(user_id, created_at DESC)

6. В -- +goose Down напиши DROP TABLE в обратном порядке.

7. Создай вторую миграцию: goose create seed_admin sql
   В upgrade — INSERT admin пользователя с email=admin@clinic.local 
   и password_hash от пароля "changeme123" (bcrypt).
   В downgrade — DELETE этого пользователя.

8. Создай Makefile в backend/ с целями:
   - migrate-up: goose -dir migrations postgres "$(DATABASE_URL)" up
   - migrate-down: goose -dir migrations postgres "$(DATABASE_URL)" down
   - migrate-status: ...
   - migrate-create: создать новую миграцию

9. Обнови docker-compose чтобы backend ждал postgres healthy и
   автоматически прогонял миграции при старте (entrypoint script).

10. Создай скрипт scripts/db-reset.sh:
    docker compose down -v && docker compose up -d postgres && 
    sleep 5 && make migrate-up

11. Напиши документацию миграций в docs/database.md:
    - Как создавать новые миграции
    - Как откатывать
    - Schema diagram (можно текстом)

КРИТЕРИИ ПРИЁМКИ:
- make migrate-up создаёт все таблицы
- make migrate-down удаляет все
- psql -c "\d appointments" показывает EXCLUDE constraint
- Тест: вставь 2 appointment с пересекающимся временем — второй должен fail
- Admin создаётся через миграцию seed

В КОНЦЕ:
- Покажи мне CREATE TABLE для appointments (с EXCLUDE)
- Покажи make migrate-status после применения
- Покажи как тест на пересечение fail'ится
```

**Тест который должен пройти:**
```sql
-- Это должно сработать:
INSERT INTO appointments(patient_id, doctor_id, service_id, direction_id, 
  start_at, end_at, status, source) VALUES 
  (1, 1, 1, 1, '2026-05-20 10:00+00', '2026-05-20 11:00+00', 'created', 'telegram_bot');

-- А это должно упасть с EXCLUDE violation:
INSERT INTO appointments(patient_id, doctor_id, service_id, direction_id, 
  start_at, end_at, status, source) VALUES 
  (2, 1, 1, 1, '2026-05-20 10:30+00', '2026-05-20 11:30+00', 'created', 'telegram_bot');
```

---

# ЭТАП 2: AvailabilityCalculator (САМЫЙ ВАЖНЫЙ)

**Статус:** ⏳ Ожидает

**Цель:** Чистая функция рассчитывает свободные слоты с учётом расписания, исключений, существующих записей. Полностью покрыта тестами.

**Зависимости:** Этап 1.

## Промпт для агента

```
Этап 2. Прочитай CLAUDE.md секцию 7.1 и docs/TZ.md секцию про availability.

ЗАДАЧА: Реализовать AvailabilityCalculator с полным test coverage.

ВАЖНО: Сначала пишем ТЕСТЫ (TDD подход), потом реализацию.

1. Создай backend/internal/availability/types.go:

   type Slot struct {
       Start time.Time
       End   time.Time
   }
   
   type WorkingInterval struct {
       Start time.Time
       End   time.Time
   }
   
   type DaySchedule struct {
       Date          time.Time
       IsWorkingDay  bool
       Intervals     []WorkingInterval
       BookedSlots   []Slot
   }
   
   type CalculatorInput struct {
       Date                time.Time      // нач. даты в локальном TZ
       ServiceDuration     time.Duration
       RegularSchedule     []RegularSchedule // weekly
       Exceptions          []Exception
       ExistingAppointments []Slot
       SlotStep            time.Duration // обычно 30 минут
   }
   
   type RegularSchedule struct {
       DayOfWeek time.Weekday
       Start     time.Time // только time-of-day
       End       time.Time
   }
   
   type Exception struct {
       Date  time.Time
       Type  string // "day_off" | "custom_working_hours"
       Start *time.Time
       End   *time.Time
   }

2. Создай backend/internal/availability/calculator_test.go — НАПИШИ ТЕСТЫ ПЕРВЫМИ:

   TestCalculateSlots_BasicDay — врач работает 10:00-12:00, услуга 60мин, 
     нет записей, шаг 30мин → ожидаем слоты [10:00-11:00, 10:30-11:30, 11:00-12:00]
   
   TestCalculateSlots_WithBookedSlot — врач 10:00-12:00, услуга 60мин,
     занято 10:00-11:00 → ожидаем [11:00-12:00] 
     (10:30-11:30 не подходит т.к. пересекается с занятым)
   
   TestCalculateSlots_DayOff — есть exception type=day_off → пустой результат
   
   TestCalculateSlots_CustomWorkingHours — exception 12:00-15:00 → 
     слоты только в этом окне, регулярное игнорируется
   
   TestCalculateSlots_NonWorkingDay — день не в расписании → пустой результат
   
   TestCalculateSlots_TwoIntervals — врач работает 10:00-13:00 и 14:00-17:00 (обед),
     услуга 60мин → слоты в обоих интервалах, обед не используется
   
   TestCalculateSlots_ServiceTooLong — услуга 90мин, рабочий интервал 60мин →
     пустой результат
   
   TestCalculateSlots_BoundaryConditions — занято 9:30-10:00 (граница),
     рабочее 10:00-12:00, услуга 60мин → [10:00-11:00, 10:30-11:30, 11:00-12:00]
     (занятый слот не пересекается)
   
   TestCalculateSlots_BackToBackBookings — занято 10:00-10:30 и 10:30-11:00,
     услуга 60мин → 11:00-12:00 (если только до 12)
   
   Для каждого теста: arrange, act, assert через testify.

3. Создай backend/internal/availability/calculator.go — реализация:

   func Calculate(input CalculatorInput) []Slot {
       // 1. Определить какое расписание применить (exception > regular)
       // 2. Если день нерабочий → return nil
       // 3. Для каждого working interval:
       //    - сгенерировать слоты с шагом SlotStep
       //    - проверить что слот помещается до конца interval
       //    - проверить отсутствие пересечения с booked slots
       //    - добавить в результат
       // 4. Сортировать по времени
   }

4. Реализуй до зелёных тестов: go test ./internal/availability/ -v

5. Создай backend/internal/availability/service.go — обёртка которая 
   ходит в repository и собирает Input для Calculator:

   type Service struct {
       doctorRepo    DoctorRepository
       scheduleRepo  ScheduleRepository
       apptRepo      AppointmentRepository
       serviceRepo   ServiceRepository
   }
   
   func (s *Service) GetAvailability(ctx, doctorID, serviceID, from, to time.Time) ([]DayAvailability, error)

6. Bench-тест (опционально):
   BenchmarkCalculate_30Days — генерируем входные данные на 30 дней
   с 20 записями в день, проверяем что < 10ms.

КРИТЕРИИ ПРИЁМКИ:
- go test -v ./internal/availability/ — все тесты зелёные
- go test -cover ./internal/availability/ — coverage > 90%
- Calculate чистая функция (без БД, без I/O)
- Все edge cases покрыты

В КОНЦЕ:
- Покажи go test -v output
- Покажи go test -cover output
- Покажи структуру calculator.go (краткий обзор)
```

**Почему это критично:** Если AvailabilityCalculator сломан — бот будет показывать пациентам несуществующие окна, или прятать существующие. Это **ядро бизнеса**.

---

# ЭТАП 3: Auth + Models

**Статус:** ⏳ Ожидает

**Цель:** Регистрация/логин админа и врача через JWT. Базовые модели.

**Зависимости:** Этап 1.

## Промпт для агента

```
Этап 3. Прочитай CLAUDE.md секции 3, 5.

ЗАДАЧА: Реализовать JWT auth + базовые модели.

1. Создай internal/model/ с структурами для всех таблиц:
   - user.go: User
   - doctor.go: Doctor
   - direction.go: Direction
   - service.go: Service
   - schedule.go: WorkingHours, ScheduleException
   - patient.go: Patient
   - appointment.go: Appointment, AppointmentStatusHistory

   Каждая структура — точное отражение таблицы. Поля JSON tags, 
   но БЕЗ password_hash в JSON (json:"-").

2. internal/auth/password.go:
   - HashPassword(plain string) (string, error) — bcrypt cost 12
   - VerifyPassword(hash, plain string) bool
   - ValidatePasswordStrength(plain string) error — мин 8 символов

3. internal/auth/jwt.go:
   - GenerateAccessToken(userID, role) → 7 дней
   - GenerateRefreshToken(userID) → 30 дней
   - ValidateToken(token) → claims
   Secret из env JWT_SECRET.

4. internal/repository/user.go — pgx запросы:
   - GetByEmail(ctx, email) (*User, error)
   - GetByID(ctx, id) (*User, error)
   - Create(ctx, user) error
   - UpdatePassword(ctx, userID, hash) error

5. internal/service/auth.go:
   - Login(ctx, email, password) (accessToken, refreshToken, *User, error)
   - Refresh(ctx, refreshToken) (newAccessToken, error)
   - ChangePassword(ctx, userID, old, new) error
   - GetMe(ctx, userID) (*User, error)

6. internal/api/middleware/auth.go:
   - RequireAuth — проверяет JWT, кладёт claims в context
   - RequireRole(roles...) — проверяет роль из context

7. internal/api/handler/auth.go:
   - POST /api/v1/auth/login {email, password} → {access_token, refresh_token, user}
   - POST /api/v1/auth/refresh {refresh_token} → {access_token}
   - POST /api/v1/auth/logout — пока заглушка (нет blacklist в MVP)
   - GET  /api/v1/auth/me — текущий user
   - POST /api/v1/auth/change-password {old, new}

8. Подключи routes в cmd/api/main.go. 
   Структурируй так: chi с группами, RequireAuth на protected, 
   RequireRole("admin") на админских.

9. Тесты:
   - internal/auth/password_test.go
   - internal/auth/jwt_test.go (валидный, истёкший, неправильная подпись)
   - internal/service/auth_test.go (mock repository, login flow)

10. Интеграционный тест через httptest:
    - POST /login с правильными данными → 200
    - POST /login с неправильным паролем → 401
    - GET /auth/me без токена → 401
    - GET /auth/me с токеном → 200 + данные

КРИТЕРИИ ПРИЁМКИ:
- curl с admin@clinic.local / changeme123 возвращает токены
- curl с битым токеном возвращает 401
- Unit tests все зелёные
- go vet и go fmt чистые

В КОНЦЕ:
- Покажи curl примеры login → me с реальным токеном
- Покажи go test -v ./internal/auth/...
```

---

# ЭТАП 4: CRUD Directions + Doctors

**Статус:** ⏳ Ожидает

**Зависимости:** Этап 3.

## Промпт для агента

```
Этап 4. Прочитай CLAUDE.md секции 3, 4.

ЗАДАЧА: Реализовать CRUD для directions и doctors с проверкой ролей.

DIRECTIONS:

1. internal/repository/direction.go:
   - List(ctx, activeOnly bool) ([]Direction, error)
   - GetByID(ctx, id) (*Direction, error)
   - Create(ctx, *Direction) error
   - Update(ctx, *Direction) error
   - SoftDelete(ctx, id) error — is_active = false

2. internal/service/direction.go — оборачивает repository, без сложной логики.

3. internal/api/handler/direction.go:
   - GET    /api/v1/directions?active=true (RequireAuth)
   - GET    /api/v1/directions/:id (RequireAuth)
   - POST   /api/v1/directions (RequireRole admin) {name, description}
   - PATCH  /api/v1/directions/:id (RequireRole admin) {name?, description?, is_active?}
   - DELETE /api/v1/directions/:id (RequireRole admin) — soft delete

DOCTORS:

4. internal/repository/doctor.go:
   - List(ctx, activeOnly bool, directionID *int) ([]DoctorWithDirections, error)
   - GetByID(ctx, id) (*DoctorWithDirections, error)
   - Create(ctx, doctor, directionIDs) error — транзакция
   - Update(ctx, doctor, directionIDs) error — транзакция, обновляет directions
   - SoftDelete(ctx, id) error
   - CreateAccount(ctx, doctorID, email, passwordHash) (*User, error)
   - GetByUserID(ctx, userID) (*Doctor, error)

5. internal/service/doctor.go — оборачивает.
   В Create/Update проверяет что все directionIDs существуют и активны.

6. internal/api/handler/doctor.go:
   - GET    /api/v1/doctors (RequireAuth)
   - GET    /api/v1/doctors/:id (RequireAuth)
   - POST   /api/v1/doctors (RequireRole admin) — {first_name, last_name, middle_name, 
     cabinet, branch_address, description, photo_url, direction_ids: []}
   - PATCH  /api/v1/doctors/:id (RequireRole admin)
   - DELETE /api/v1/doctors/:id (RequireRole admin)
   - POST   /api/v1/doctors/:id/account (RequireRole admin) {email, password}
     Создаёт User с role=doctor, связывает с Doctor через user_id

7. Валидация:
   - first_name, last_name — required, max 255
   - direction_ids — массив существующих direction id
   - При создании account: email уникален, password >= 8

8. Тесты:
   - Unit для service (mock repo)
   - Integration через httptest:
     a) Создать direction, потом doctor с этим direction
     b) Получить список — увидеть doctor с правильными directions
     c) Обновить doctor: убрать direction → видим только одно
     d) Soft delete → активные не показывают его, но он есть в БД
     e) Создать account → доктор может залогиниться

9. Документация в docs/api.md — все endpoints с примерами.

КРИТЕРИИ ПРИЁМКИ:
- Тест: создал direction → doctor → account → залогинился как doctor → curl /auth/me возвращает role=doctor
- DELETE doctor → его нет в GET /doctors?active=true, но он есть с active=false
- Невалидный direction_id → 400 с понятной ошибкой

В КОНЦЕ:
- curl полная цепочка: login admin → create direction → create doctor → create account → login as doctor
```

---

# ЭТАП 5: Services + Schedule

**Статус:** ⏳ Ожидает

**Зависимости:** Этап 4.

## Промпт для агента

```
Этап 5. Прочитай CLAUDE.md и доделай Этап 4 перед началом.

ЗАДАЧА: CRUD для services врачей + расписание врачей.

SERVICES:

1. internal/repository/service.go:
   - ListByDoctor(ctx, doctorID, activeOnly bool) ([]Service, error)
   - GetByID(ctx, id) (*Service, error)
   - Create, Update, SoftDelete как раньше

2. internal/api/handler/service.go:
   - GET    /api/v1/doctors/:doctorId/services
   - POST   /api/v1/doctors/:doctorId/services (admin)
     {name, description, duration_minutes, price, direction_id}
   - PATCH  /api/v1/doctors/:doctorId/services/:serviceId (admin)
   - DELETE /api/v1/doctors/:doctorId/services/:serviceId (admin)

3. Бизнес-правило: при создании service проверить что direction_id 
   входит в directions этого врача.

WORKING HOURS:

4. internal/repository/schedule.go:
   - ListWorkingHours(ctx, doctorID) ([]WorkingHours, error)
   - ReplaceWorkingHours(ctx, doctorID, []WorkingHours) error — транзакция:
     удалить все, вставить новые (PUT семантика)
   
   - ListExceptions(ctx, doctorID, from, to time.Time) ([]Exception, error)
   - CreateException(ctx, *Exception) error
   - UpdateException(ctx, *Exception) error
   - DeleteException(ctx, id) error

5. internal/api/handler/schedule.go:
   - GET /api/v1/doctors/:doctorId/working-hours
   - PUT /api/v1/doctors/:doctorId/working-hours (admin) — массив интервалов
   
   - GET    /api/v1/doctors/:doctorId/schedule-exceptions?from=&to=
   - POST   /api/v1/doctors/:doctorId/schedule-exceptions (admin)
   - PATCH  /api/v1/doctors/:doctorId/schedule-exceptions/:id (admin)
   - DELETE /api/v1/doctors/:doctorId/schedule-exceptions/:id (admin)

6. Валидация:
   - day_of_week ∈ [1,7]
   - start_time < end_time
   - Type exception ∈ {day_off, custom_working_hours}
   - Если type=custom_working_hours, start_time и end_time обязательны
   - Если type=day_off, должны быть NULL

AVAILABILITY:

7. internal/api/handler/availability.go:
   - GET /api/v1/availability?doctor_id=&service_id=&date_from=&date_to=
     Вызывает AvailabilityService (Этап 2) и возвращает структуру:
     {
       "doctor_id": 1,
       "service_id": 10,
       "service_duration_minutes": 60,
       "availability": [
         {"date": "2026-05-20", "day_name": "вторник", "slots": [...]}
       ]
     }
   
   - Доступно для роли admin И для bot (через X-Bot-Token header)

8. Тесты integration:
   - Create doctor + service + working_hours → GET availability возвращает слоты
   - Add day_off exception → этого дня нет в результате
   - Create appointment в слот → этот слот пропадает из availability
   - Service_id чужого врача → 400

КРИТЕРИИ ПРИЁМКИ:
- Полный flow: создаёшь все → запрашиваешь availability → получаешь правильные окна
- Edge case: занимаешь слот → availability обновляется

В КОНЦЕ:
- curl полная цепочка с реальной availability в выводе
```

---

# ЭТАП 6: Appointments API

**Статус:** ⏳ Ожидает

**Зависимости:** Этап 5.

## Промпт для агента

```
Этап 6. Прочитай CLAUDE.md секцию 7.2.

ЗАДАЧА: API для создания, отмены, просмотра appointments.

1. internal/repository/appointment.go:
   - HasConflictLocked(ctx, tx, doctorID, startAt, endAt, excludeID *int) (bool, error)
     — SQL запрос с FOR UPDATE
   - Create(ctx, tx, *Appointment) error
   - GetByID(ctx, id) (*AppointmentDetail, error) — с JOIN на patient, doctor, service
   - List(ctx, filter ListFilter) ([]AppointmentDetail, error)
   - UpdateStatus(ctx, id, newStatus, userID, comment) error — пишет в history
   
   - UpsertPatientByTelegram(ctx, tx, telegramID, name, phone) (*Patient, error)
     — INSERT ... ON CONFLICT (telegram_user_id) DO UPDATE

2. internal/service/appointment.go — главная логика:

   func (s *Service) Create(ctx, input CreateInput) (*Appointment, error) {
       tx, err := s.db.BeginTx(ctx, ...)
       defer tx.Rollback()
       
       // 1. Получить doctor → проверить is_active
       // 2. Получить service → проверить doctor_id совпадает, is_active
       // 3. Вычислить end_at = start_at + service.duration_minutes
       // 4. Проверить что start_at в будущем
       // 5. Проверить конфликт через HasConflictLocked
       // 6. Upsert patient
       // 7. Создать appointment
       // 8. Создать status history
       // 9. Commit
   }
   
   func (s *Service) Cancel(ctx, id, userID int) error {
       // Только админ может отменить
       // Только статусы created, confirmed можно отменять
       // status = cancelled_by_admin
       // status history
   }
   
   func (s *Service) Complete(ctx, id, userID int) error
   func (s *Service) MarkNoShow(ctx, id, userID int) error

3. internal/api/handler/appointment.go:
   - GET    /api/v1/appointments (admin: все, doctor: только свои)
   - GET    /api/v1/appointments/:id
   - POST   /api/v1/appointments (admin)
   - POST   /api/v1/appointments/:id/cancel (admin)
   - POST   /api/v1/appointments/:id/complete (admin)
   
   - GET /api/v1/doctor/appointments (RequireRole doctor) — свои записи
     doctor_id берётся из JWT claims

4. Для bot:
   - POST /api/v1/bot/appointments (X-Bot-Token)
     {telegram_user_id, telegram_username, patient_name, patient_phone,
      doctor_id, service_id, start_at}
     Возвращает полную информацию для отправки в Telegram

5. Тесты:
   - Unit для service (mock repo) — все edge cases
   - КРИТИЧЕСКИЙ тест: concurrent creation
     запускаем 100 goroutines пытающихся создать запись на одно время
     → ровно 1 успешна, 99 получают ErrSlotTaken
   - Cancel non-existent → 404
   - Cancel completed → 422
   - Doctor request /doctor/appointments видит только свои

6. Логирование:
   - Создание записи (info): "appointment created" with appt_id, doctor_id, patient_id
   - Отмена (info): "appointment cancelled"
   - Конфликт (warn): "slot conflict prevented" with details
   - Ошибки БД (error)

КРИТЕРИИ ПРИЁМКИ:
- Concurrent тест зелёный
- Полный happy path работает
- Doctor не может видеть чужие записи
- Все статус-переходы валидируются

В КОНЦЕ:
- Покажи concurrent тест в выводе
- Покажи happy path: create direction → doctor → service → working_hours →
  availability → create appointment → list as doctor → cancel as admin
```

---

# ЭТАП 7: Telegram Bot

**Статус:** ⏳ Ожидает

**Зависимости:** Этап 6.

## Промпт для агента

```
Этап 7. Прочитай CLAUDE.md секцию 7.3.

ЗАДАЧА: Telegram bot с полным flow записи.

1. bot/internal/client/api.go — HTTP клиент к Backend API:
   - GetDirections(ctx) ([]Direction, error)
   - GetDoctorsByDirection(ctx, directionID) ([]Doctor, error)
   - GetServicesByDoctor(ctx, doctorID, directionID) ([]Service, error)
   - GetAvailability(ctx, doctorID, serviceID, from, to) (...)
   - CreateAppointment(ctx, input) (*AppointmentResult, error)
   
   Каждый запрос с header X-Bot-Token из env.

2. bot/internal/keyboard/keyboard.go — построение InlineKeyboard:
   - DirectionsKeyboard([]Direction) → 2 в ряд
   - DoctorsKeyboard([]Doctor) → 1 в ряд (длинные имена)
   - ServicesKeyboard([]Service) → 1 в ряд (показывает цену)
   - DatesKeyboard([]Date) → 3 в ряд
   - TimesKeyboard([]Slot) → 3 в ряд
   - ConfirmKeyboard → Отменить | Подтвердить
   - PhoneKeyboard → KeyboardButton с request_contact=true

3. bot/internal/flow/booking.go — FSM:
   
   States: StateStart, StateChooseDirection, StateChooseDoctor, 
           StateChooseService, StateChooseDate, StateChooseTime,
           StateEnterName, StateEnterPhone, StateConsent, 
           StateConfirm, StateDone
   
   Session хранится в БД (bot_sessions). Структура data:
   {
     "state": "ChooseDoctor",
     "direction_id": 1,
     "direction_name": "Кардиология",
     "doctor_id": null,
     ...
   }
   
   Каждый handler:
   - Читает session
   - Обрабатывает callback/message
   - Обновляет data
   - Меняет state
   - Сохраняет session
   - Отправляет следующее сообщение

4. bot/internal/handler/start.go:
   /start → создать/сбросить session → показать DirectionsKeyboard
   "Добро пожаловать! Выберите направление:"

5. bot/internal/handler/callback.go — обработка inline button clicks:
   data формата "direction:1", "doctor:5", "service:10", "date:2026-05-20", 
   "time:10:00", "confirm", "cancel"
   
   Routing на основе state + callback data.

6. bot/internal/handler/message.go — обработка текстовых сообщений:
   - StateEnterName → сохранить имя → попросить телефон
   - StateEnterPhone (contact) → сохранить → показать consent
   
7. Подтверждение и согласие:
   ```
   Подтвердите запись:
   
   👨‍⚕️ Врач: Иванов И.И.
   🏥 Направление: Кардиология
   💊 Услуга: Первичная консультация
   📅 Дата: 20 мая 2026
   🕐 Время: 10:00
   🚪 Кабинет: 205
   📍 Адрес: Москва, Примерная 1
   💰 Стоимость: 3000 ₽
   
   Нажимая "Подтвердить", вы соглашаетесь с обработкой персональных данных.
   
   [❌ Отменить]  [✅ Подтвердить]
   ```

8. После подтверждения:
   - Вызвать backend POST /api/v1/bot/appointments
   - Если slot_taken → "К сожалению, время уже занято. Начнём заново?" /start
   - Если ok → "✅ Вы успешно записаны!" + детали

9. Команды:
   /start — главное меню
   /help — справка
   /cancel — сбросить текущий FSM

10. Webhook vs polling:
    Для dev — long polling.
    Для prod — webhook (отдельная задача).
    В env: BOT_MODE=polling|webhook

11. Тесты:
    - Mock backend client → flow тесты
    - Тест навигации между состояниями
    - Тест slot taken handling

КРИТЕРИИ ПРИЁМКИ:
- Полный flow в реальном Telegram: /start → выбрать → ввести → подтвердить → запись в БД
- Несколько одновременных пользователей не мешают друг другу
- /cancel в любом state возвращает в /start
- Если slot taken после выбора → graceful fallback

В КОНЦЕ:
- Запиши видео/скриншоты Telegram flow
- Покажи запись в БД после создания через бот
```

---

# ЭТАП 8: Admin Frontend

**Статус:** ⏳ Ожидает

**Зависимости:** Этап 6.

## Промпт для агента

```
Этап 8. Прочитай CLAUDE.md.

ЗАДАЧА: React админка для управления всем.

Используем готовые компоненты shadcn/ui (через CLI) либо просто Tailwind + lucide.

1. frontend/src/api/client.js — axios instance:
   - baseURL из env
   - interceptor: добавляет JWT из localStorage
   - interceptor response: 401 → редирект на /login

2. frontend/src/api/endpoints/ — функции для каждой группы:
   - auth.js: login, logout, me, refreshToken
   - directions.js
   - doctors.js
   - services.js
   - schedule.js
   - appointments.js
   - availability.js

3. frontend/src/stores/auth.js (zustand):
   - user, accessToken, refreshToken
   - login(email, password)
   - logout()
   - isAuthenticated, isAdmin, isDoctor

4. frontend/src/App.jsx — routes:
   /login → LoginPage
   /admin/* — RequireAuth + RequireAdmin:
     /admin/directions
     /admin/doctors
     /admin/doctors/:id (вкладки: услуги, расписание, исключения)
     /admin/appointments
   /doctor/* — RequireAuth + RequireDoctor:
     /doctor/schedule

5. frontend/src/components/Layout.jsx — sidebar + topbar.

6. frontend/src/pages/admin/DirectionsPage.jsx:
   - Список (DataTable)
   - Кнопка "Добавить"
   - Modal с формой
   - Edit/Delete для каждой строки
   - React Query для кеширования

7. frontend/src/pages/admin/DoctorsPage.jsx:
   - Список с фильтрами (активные/неактивные, по направлению)
   - Создание врача через modal:
     ФИО, кабинет, адрес, описание, multi-select directions
   - Клик на строку → DoctorDetailPage

8. frontend/src/pages/admin/DoctorDetailPage.jsx:
   Tabs: 
   - "Информация" (редактирование основных полей)
   - "Услуги" (CRUD внутри)
   - "Расписание" (weekly view + редактирование)
   - "Исключения" (calendar или список)
   - "Аккаунт" (кнопка "Создать аккаунт" → modal с email/password)

9. frontend/src/components/ScheduleEditor.jsx:
   - 7 строк (дни недели)
   - В каждой можно добавить несколько интервалов (для обеда)
   - Time pickers
   - "Сохранить" → PUT working-hours

10. frontend/src/components/ExceptionsManager.jsx:
    - Calendar (или просто список)
    - Добавить exception: дата, тип, при custom — время
    - Удалить

11. frontend/src/pages/admin/AppointmentsPage.jsx:
    - Список со столбцами: дата/время, пациент, телефон, врач, услуга, статус
    - Фильтры: врач (select), направление, дата from/to, статус
    - Действия: отменить, завершить
    - Пагинация (limit 50, "Загрузить ещё")

12. UX:
    - Toast уведомления (react-hot-toast)
    - Loading skeletons
    - Confirmation для деструктивных действий
    - Optimistic updates где возможно

КРИТЕРИИ ПРИЁМКИ:
- Можно полностью настроить клинику через UI
- Все CRUD работают
- Фильтры и пагинация работают
- Responsive (хотя бы > 1024px)

В КОНЦЕ:
- Screenshots каждой страницы
- Видео полного flow: login → create direction → create doctor → 
  setup schedule → see appointment created via bot
```

---

# ЭТАП 9: Doctor Frontend

**Статус:** ⏳ Ожидает

**Зависимости:** Этап 8.

## Промпт для агента

```
Этап 9.

ЗАДАЧА: Кабинет врача — только просмотр расписания.

1. frontend/src/pages/doctor/SchedulePage.jsx:
   - Toggle: День | Неделя
   - Calendar grid (можно использовать react-big-calendar или сделать свой)
   - События = appointments
   - Цвет события по статусу:
     created — синий
     confirmed — зелёный
     completed — серый
     cancelled — красный с зачёркиванием
   - Клик на событие → modal с деталями

2. Modal с деталями записи:
   - ФИО пациента (ВИДНО)
   - Услуга, направление, кабинет, время (ВИДНО)
   - Телефон пациента (НЕ ВИДНО — backend не возвращает doctor'у)
   - Telegram (НЕ ВИДНО)

3. Hook useDoctorSchedule:
   const { data } = useQuery({
     queryKey: ['doctor-schedule', from, to],
     queryFn: () => api.get('/doctor/schedule', { params: { from, to }})
   })

КРИТЕРИИ ПРИЁМКИ:
- Доктор логинится → видит своё расписание
- Не видит чужие записи
- Не видит телефон пациента
- Переключение день/неделя работает
```

---

# ЭТАП 10: Тесты + полировка

**Статус:** ⏳ Ожидает

```
Этап 10.

ЗАДАЧА: Финальная подготовка.

1. Coverage report:
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out -o coverage.html
   Цель: > 70% общий, > 90% на internal/availability и internal/service.

2. Линтеры:
   - golangci-lint (создай .golangci.yml)
   - prettier + eslint для frontend
   Зафикси все warnings.

3. Swagger:
   - Установи swag CLI
   - Добавь аннотации к handlers
   - swag init → docs/swagger.json
   - Подключи swagger UI на /swagger/*

4. Smoke test script:
   scripts/smoke-test.sh — полный E2E через curl:
   login → create direction → doctor → service → schedule →
   availability → создаём appointment → видим в /appointments
   
   Должен пройти на чистой БД.

5. Documentation:
   docs/api.md — все endpoints
   docs/deployment.md — как развернуть
   README.md — финальный, с badges

6. Performance:
   k6 / vegeta для load test основных endpoints
   GET /availability — база, цель < 50ms p95
   POST /appointments — цель < 100ms p95

7. Security pass:
   - Все пароли через bcrypt
   - JWT secret из env, не в коде
   - SQL injection невозможна (prepared statements)
   - CORS правильно настроен
   - HTTPS в prod (через nginx)
   - Rate limiting на /auth/login (5 попыток в минуту)

КРИТЕРИИ ПРИЁМКИ:
- coverage > 70%
- linters green
- smoke test passes
- API documented
```

---

# ЭТАП 11: Production deploy

**Статус:** ⏳ Ожидает

```
Этап 11.

ЗАДАЧА: Запустить в production.

1. docker-compose.prod.yml:
   - Без volume mounts кода
   - nginx как reverse proxy
   - SSL через Let's Encrypt (certbot)
   - Restart policies
   - Health checks
   - Logging driver

2. nginx config:
   - /api/* → backend:8000
   - /* → frontend (статика)
   - HTTPS redirect
   - gzip
   - security headers

3. Backup script:
   scripts/backup.sh — pg_dump → S3 (или просто файл)
   Cron: каждый день в 3:00

4. Monitoring (минимум):
   - Health endpoint /health на backend
   - Простой uptime check (uptime-kuma в docker)
   - Логи через docker logs (на старте)

5. Telegram webhook:
   - Установи через curl setWebhook
   - Проверь secret token

6. Deploy script:
   scripts/deploy.sh — git pull && docker compose up -d --build

7. Документация runbook:
   docs/runbook.md — что делать когда:
   - Backend упал
   - БД заполнилась
   - Бот не отвечает
   - SSL истёк

КРИТЕРИИ ПРИЁМКИ:
- Реальный URL отдаёт фронт по HTTPS
- Бот работает через webhook
- Backup создаётся
- Health check зелёный
```

---

## 📌 Финальный чеклист готового MVP

- [ ] Все 11 этапов закрыты
- [ ] Coverage > 70%
- [ ] Все linters зелёные
- [ ] Smoke test проходит
- [ ] Реальный пациент может записаться через бот
- [ ] Реальный врач видит расписание
- [ ] Production задеплоен
- [ ] Backup настроен

---

## ❓ Что делать когда агент застрял

1. **Если агент не понимает задачу:** скопируй промпт ещё раз + добавь "прочитай CLAUDE.md и ROADMAP.md перед началом"

2. **Если агент пишет что-то странное:** прерви → скажи "стоп, это противоречит правилу X в CLAUDE.md"

3. **Если задача оказалась слишком большой:** скажи "разбей на подзадачи и начни с первой"

4. **Если что-то сломалось:** скажи "rollback последние изменения через git, начни заново с другим подходом"

5. **Если непонятно что делать:** вернись к этому файлу, найди где остановился
