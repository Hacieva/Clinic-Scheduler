# Migration Workflow

Пошаговый процесс добавления DB-миграции. Самый опасный тип изменений — требует максимальной осторожности.

---

## Принципы

1. **Миграции необратимы в production.** Каждый файл — навсегда.
2. **Additive first.** Предпочитать ADD COLUMN, CREATE TABLE вместо ALTER TYPE, RENAME.
3. **Goose Down обязателен.** Даже если никогда не используется — нужен для тестирования.
4. **Никаких данных в Up-миграции** кроме seed-данных (DEFAULT значения — ok).
5. **Одна миграция = одно логическое изменение.** Не валить всё в один файл.

---

## Шаг 0 — Architect: проектирование

Architect Agent проектирует DDL и отвечает на вопросы:

- [ ] Все новые колонки nullable или имеют DEFAULT?
- [ ] Есть backfill-план для существующих строк?
- [ ] Индексы покрывают ожидаемые WHERE-условия?
- [ ] UNIQUE/EXCLUDE constraints не сломают существующие данные?
- [ ] Миграция обратима (есть goose Down)?
- [ ] Порядок: CREATE TABLE перед ALTER TABLE который ссылается на неё?

---

## Шаг 1 — Именование файла

```
backend/migrations/YYYYMMDDHHMMSS_<name>.sql
```

Правила:
- Timestamp — UTC, формат `20260520100000`
- Name — snake_case, описывает изменение: `add_branches`, `extend_patients`, `add_branch_id_to_doctors`
- Один файл = одна логическая группа (можно несколько ALTER в одном файле если они атомарны)

---

## Шаг 2 — Шаблон файла

```sql
-- +goose Up

-- [описание что делает эта миграция]

CREATE TABLE example (
  id         BIGSERIAL    PRIMARY KEY,
  name       VARCHAR(200) NOT NULL,
  is_active  BOOLEAN      NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE IF EXISTS example;
```

---

## Шаг 3 — Порядок операций в Up

```sql
-- 1. Сначала CREATE TABLE (если нужен для FK)
CREATE TABLE branches (...);

-- 2. Потом ALTER TABLE существующих таблиц
ALTER TABLE doctors ADD COLUMN branch_id BIGINT REFERENCES branches(id);

-- 3. Потом backfill (если нужен)
UPDATE doctors SET branch_id = 1;

-- 4. Потом индексы
CREATE INDEX ON doctors(branch_id);

-- 5. НЕ делать NOT NULL до backfill:
-- НЕЛЬЗЯ сразу: ALTER TABLE doctors ALTER COLUMN branch_id SET NOT NULL;
-- Это отдельная миграция после проверки что backfill прошёл
```

---

## Шаг 4 — Порядок операций в Down

Обратный порядок Up:

```sql
-- +goose Down

-- 4. Сначала индексы (если создавали явно)
DROP INDEX IF EXISTS doctors_branch_id_idx;

-- 3. Потом ALTER TABLE
ALTER TABLE doctors DROP COLUMN IF EXISTS branch_id;

-- 1. Потом DROP TABLE (в обратном порядке от CREATE)
DROP TABLE IF EXISTS branches;
```

---

## Шаг 5 — Тестирование миграции

**Обязательно перед сдачей:**

```powershell
# Применить Up
docker compose run --rm backend goose -dir /app/migrations up

# Проверить статус
docker compose run --rm backend goose -dir /app/migrations status

# Откатить Down
docker compose run --rm backend goose -dir /app/migrations down

# Применить снова Up (проверить идемпотентность)
docker compose run --rm backend goose -dir /app/migrations up
```

Все три операции должны завершиться без ошибок.

---

## Шаг 6 — Проверка backfill

Если миграция содержит UPDATE:

```powershell
# После миграции проверить что не осталось NULL где не должно
docker compose exec db psql -U user -d clinic -c "SELECT COUNT(*) FROM doctors WHERE branch_id IS NULL"
# Ожидается: 0
```

---

## Шаг 7 — Supervisor: проверка

Supervisor проверяет:
- [ ] git diff — только новые migration файлы
- [ ] Нет изменений в существующих migration файлах (НИКОГДА нельзя редактировать!)
- [ ] Down-миграция существует и работает
- [ ] Backfill выполнен (если нужен)

---

## Шаг 8 — Коммит

```powershell
git add backend/migrations/
git commit -m "feat(db): <описание изменения>"
```

Примеры:
```
feat(db): add branches table and branch_id to doctors
feat(db): extend patients with date_of_birth and email
feat(db): expand users role check to include owner
```

---

## Типы миграций по риску

### Безопасные (зелёный)
```sql
CREATE TABLE new_table (...)
ALTER TABLE t ADD COLUMN col TYPE DEFAULT value  -- nullable или с DEFAULT
CREATE INDEX CONCURRENTLY ON t(col)
```

### Требуют проверки (жёлтый)
```sql
ALTER TABLE t ADD COLUMN col TYPE NOT NULL DEFAULT value  -- блокирует таблицу
UPDATE t SET col = ...  -- backfill большой таблицы
ALTER TABLE t ADD CONSTRAINT ...  -- может упасть если данные не соответствуют
```

### Опасные — только с явным подтверждением (красный)
```sql
DROP TABLE t
DROP COLUMN t.col
ALTER TABLE t ALTER COLUMN col TYPE new_type  -- если данные несовместимы
ALTER TABLE t RENAME COLUMN old TO new  -- breaking для существующего кода
```

---

## Что НЕЛЬЗЯ делать с миграциями

- ❌ Редактировать существующие файлы миграций (они применены на prod!)
- ❌ Удалять migration файлы
- ❌ Делать DROP TABLE без явного подтверждения Product Owner
- ❌ Использовать AutoMigrate или любой ORM-механизм
- ❌ Хранить секреты в seed-данных (пароли — только bcrypt hash)
- ❌ Писать UPDATE/backfill без WHERE (полный table scan без индекса на большой таблице)
