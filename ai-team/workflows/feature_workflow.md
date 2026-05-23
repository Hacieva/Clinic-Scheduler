# Feature Workflow

Пошаговый процесс реализации новой фичи. Следовать строго по порядку.

---

## Шаг 0 — Supervisor: приём задачи

```
Supervisor читает:
  1. CLAUDE.md
  2. docs/ROADMAP.md — текущий этап
  3. docs/DECISIONS.md — нет ли конфликтов

Supervisor проверяет:
  - git status — нет незакоммиченных изменений
  - Размер задачи: малая / средняя / большая

Если большая → разбить на subtasks, подтвердить с Product Owner
```

---

## Шаг 1 — Product Agent: требования

**Входные данные:** бизнес-запрос от Product Owner

**Выход:** оформленный Product Brief с:
- User story
- Критерии приёмки
- Scope boundary (что входит / что не входит)
- Открытые вопросы

**Supervisor проверяет:**
- Все вопросы закрыты?
- Нет противоречий с TZ.md или DECISIONS.md?

**Подтверждение Product Owner → переход к шагу 2**

---

## Шаг 2 — Architect Agent: дизайн

**Входные данные:** Product Brief

**Выход:** Architectural Plan с:
- DB Changes (SQL DDL)
- API Contracts (endpoint list + request/response)
- Migration Strategy (тип, reversibility, backfill)
- Impact Assessment (затронутые компоненты)
- ADR если принято новое решение

**Architect обязан пройти чеклист:**
- [ ] Все новые колонки nullable или имеют DEFAULT?
- [ ] Backfill-план если нужен?
- [ ] Индексы покрывают WHERE-условия?
- [ ] Не нарушает layered architecture?
- [ ] Нет N+1 проблем?

**Supervisor проверяет:**
- Нет breaking changes без migration plan?
- Нет изменений существующих данных без backfill?

**Подтверждение Supervisor → переход к шагу 3**

---

## Шаг 3 — Backend Agent: DB migration

**Если есть DB changes:**

Backend Agent создаёт migration-файлы строго по DDL от Architect:
```
backend/migrations/YYYYMMDDHHMMSS_<name>.sql
```

Формат файла:
```sql
-- +goose Up
[DDL здесь]

-- +goose Down
[ROLLBACK DDL здесь]
```

**Проверка:**
```powershell
docker compose run --rm backend goose -dir /app/migrations up
docker compose run --rm backend goose -dir /app/migrations status
```

**Supervisor проверяет git diff** — только migration файлы изменены

---

## Шаг 4 — Backend Agent: реализация

**Порядок файлов:**
1. `internal/model/` — новые/расширенные структуры
2. `internal/repository/` — SQL-запросы
3. `internal/service/` — бизнес-логика
4. `internal/api/handler/` — HTTP handlers
5. `backend/cmd/api/main.go` — регистрация routes

**Правила:**
- Каждый новый handler прикрыт middleware
- Service возвращает доменные ошибки
- context.Context везде
- slog для логирования

**Проверка:**
```powershell
GO111MODULE=on go build ./...
GO111MODULE=on go vet ./...
```

**Supervisor проверяет git diff** — только backend файлы из списка Architect

---

## Шаг 5 — QA Agent: backend tests

**QA Agent пишет:**
- Unit tests для новой Service логики
- Integration tests если нужны (httptest)
- Проверяет coverage

**Минимум:**
```powershell
GO111MODULE=on go test -cover ./internal/...
```

**Если coverage < 70% для новой логики → STOP, написать тесты**

**Supervisor проверяет** — все тесты зелёные

---

## Шаг 6 — Frontend Agent: реализация

**Порядок файлов:**
1. `frontend/src/api/` — новые API-функции
2. `frontend/src/pages/` — новые страницы
3. `frontend/src/components/` — новые компоненты (если нужны)
4. `frontend/src/App.jsx` — новые routes
5. `frontend/src/components/Layout.jsx` — nav (если нужен)

**Правила:**
- Данные только через react-query
- Формы только через react-hook-form + zod
- Ошибки — user-friendly, не raw backend
- Токены — только через stores/auth.js

**Проверка:**
```powershell
npm run build
npm run lint
```

**Supervisor проверяет git diff** — только frontend файлы

---

## Шаг 7 — QA Agent: ручная проверка

**QA Agent проверяет:**
- Golden path: создание → просмотр → редактирование → удаление
- Auth: ролевой доступ (admin / doctor / owner)
- Edge cases: пустой список, 404, 403
- Мобильный viewport (если новый компонент)

**Итоговый QA Report:**
```
PASS / FAIL + детали
```

---

## Шаг 8 — Security Agent: аудит

**Security Agent проверяет по чеклисту:**
- Auth middleware на всех новых endpoints?
- Нет секретов в коде?
- Нет PII в логах?
- Нет raw errors в UI?

**Итоговый Security Report:**
```
PASS / FAIL + severity
```

---

## Шаг 9 — Supervisor: финальная проверка

```powershell
.\ai-team\scripts\run-checks.ps1
```

**Supervisor проверяет:**
- [ ] git diff — изменены только файлы из задачи
- [ ] git status — нет лишних файлов
- [ ] Все проверки зелёные (build, lint, test, vet)
- [ ] QA Report: PASS
- [ ] Security Report: PASS

---

## Шаг 10 — Коммит

Только после явного подтверждения Product Owner:

```powershell
git add <конкретные файлы>
git commit -m "feat(<scope>): <description>"
# НЕ push без отдельного разрешения
```

Формат commit message:
```
feat(patients): add patient list and detail pages
feat(branches): add branch management CRUD
fix(auth): redirect owner to admin panel after login
```

---

## Правило остановки

На любом шаге если обнаружено:
- Задача шире ожидаемого → STOP → показать объём → ждать разрешения
- Конфликт с DECISIONS.md → STOP → процитировать → ждать
- Риск потери данных → STOP → объяснить → ждать
- Falling tests → STOP → не коммитить → исправить

**Supervisor не даёт добро на следующий шаг если текущий не PASS.**
