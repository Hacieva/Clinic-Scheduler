# Release Workflow

Пошаговый процесс выпуска релиза. Выполняется только после явного разрешения Product Owner.

---

## Перед релизом — чеклист готовности

Supervisor проверяет каждый пункт:

### Код
- [ ] Все задачи milestone закрыты (или явно перенесены)
- [ ] Нет незакоммиченных изменений (`git status` чистый)
- [ ] Ветка `develop` смержена в `main` (или PR открыт и одобрен)
- [ ] Нет открытых TODO или FIXME в новом коде

### Тесты
- [ ] `GO111MODULE=on go test ./internal/...` — все зелёные
- [ ] `GO111MODULE=on go vet ./...` — чисто
- [ ] `npm run build` — без ошибок
- [ ] `npm run lint` — без ошибок
- [ ] Coverage availability > 80%

### БД
- [ ] Все миграции применены в правильном порядке
- [ ] Down-миграции протестированы
- [ ] Backfill данных завершён (если нужен)

### Security
- [ ] Security Agent провёл финальный аудит
- [ ] Нет секретов в коде (grep по паттернам)
- [ ] `.env.example` актуален

### Docker
- [ ] `docker compose up --build` поднимает всё с нуля
- [ ] Все сервисы healthy
- [ ] `GET /health` → 200

---

## Шаг 1 — QA: финальная проверка

QA Agent проходит полный regression checklist:

```
Admin panel:
  ✅ Login (admin, owner, doctor)
  ✅ Schedule grid — отображение записей
  ✅ Appointments — создание, отмена, статусы
  ✅ Patients — список, карточка, поиск
  ✅ Doctors — CRUD, расписание
  ✅ Branches — CRUD (owner/admin)
  ✅ Directions — CRUD
  ✅ Logout

Doctor panel:
  ✅ Login как doctor
  ✅ Расписание — неделя, навигация
  ✅ Записи — только свои, без телефона пациента

Bot:
  ✅ /start → выбор направления → врач → услуга → дата → время → подтверждение
  ✅ /cancel → отмена записи
  ✅ Занятой слот недоступен
```

---

## Шаг 2 — Security: финальный аудит

Security Agent запускает финальный чеклист по всему изменённому коду с момента последнего релиза:

```powershell
# Поиск потенциальных секретов
Select-String -Path "backend/**/*.go" -Pattern "(password|secret|token)\s*=\s*`"" -Recurse
Select-String -Path "frontend/src/**/*.js*" -Pattern "localStorage.*token" -Recurse
```

---

## Шаг 3 — DevOps: подготовка образов

```powershell
# Проверить что образы собираются
docker compose build

# Проверить что всё поднимается с нуля
docker compose down -v
docker compose up -d
Start-Sleep -Seconds 15
docker compose ps
```

---

## Шаг 4 — Supervisor: тег релиза

После подтверждения Product Owner:

```powershell
# Версия по semver: MAJOR.MINOR.PATCH
# MVP = 1.0.0, первая доработка = 1.1.0, bugfix = 1.0.1

git tag -a v1.0.0 -m "Release v1.0.0: MVP"
# push тега — только после явного разрешения
# git push origin v1.0.0
```

---

## Шаг 5 — Changelog

Supervisor создаёт/обновляет `CHANGELOG.md`:

```markdown
## [v1.0.0] — 2026-05-20

### Added
- Multi-branch clinic support
- Patient CRM module
- Schedule grid (AppointmentGrid)
- Owner role with full access

### Fixed
- Login redirect for owner role
- Branch create/update permissions for admin
```

---

## Шаг 6 — Обновить ROADMAP

Product Agent отмечает выполненные этапы в `docs/ROADMAP.md`.

---

## Что НЕ делать при релизе

- ❌ `git push --force` на `main`
- ❌ Применять миграции на production без backup
- ❌ Деплоить в production без явного разрешения Product Owner
- ❌ Мержить незавершённые фичи "чтобы был прогресс"
- ❌ Пропускать Security аудит

---

## Откат релиза

Если после релиза обнаружена critical проблема:

```
1. Supervisor уведомляет Product Owner
2. Принять решение: hotfix или rollback
3. Hotfix: запустить bugfix_workflow с severity=critical
4. Rollback: docker compose rollback (если есть предыдущий образ)
   - Миграции: goose down до предыдущей версии (ОПАСНО — потеря данных)
   - ТОЛЬКО с явного разрешения Product Owner
```
