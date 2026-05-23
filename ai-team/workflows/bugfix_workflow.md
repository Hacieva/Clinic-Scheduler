# Bugfix Workflow

Пошаговый процесс исправления бага. Быстрее feature workflow, но не менее строгий.

---

## Шаг 0 — Supervisor: приём бага

```
Получить:
  - Описание симптома (что происходит)
  - Шаги воспроизведения
  - Ожидаемое vs фактическое поведение
  - Severity: critical / high / medium / low

Supervisor НЕ начинает работу без:
  - Чёткого описания симптома
  - Способа воспроизведения
```

**Severity → действие:**

| Severity | Критерий | Действие |
|---|---|---|
| critical | Система недоступна, потеря данных, security breach | Немедленно, обойти roadmap |
| high | Основной workflow сломан | Текущий sprint |
| medium | Неудобство, есть workaround | Следующий sprint |
| low | Косметика, редкий edge case | Backlog |

---

## Шаг 1 — Диагностика

**Backend Agent или Frontend Agent (в зависимости от слоя):**

```
1. Воспроизвести баг локально
2. Определить слой: Handler / Service / Repository / Frontend / DB
3. Найти root cause — не симптом
4. Описать: что именно сломано и почему
```

**Supervisor проверяет:** root cause понятен и задокументирован

---

## Шаг 2 — Минимальный fix

**Правила:**
- Исправить только то что сломано — не рефакторить попутно
- Не добавлять features в bugfix
- Если fix требует > 3 файлов → показать план Supervisor

**Backend fix:**
```powershell
GO111MODULE=on go build ./...
GO111MODULE=on go vet ./...
```

**Frontend fix:**
```powershell
npm run build
npm run lint
```

---

## Шаг 3 — Regression test

**QA Agent пишет тест который воспроизводит баг (и теперь проходит):**

```go
func TestFix_IssueXXX(t *testing.T) {
    // воспроизводит условия бага
    // проверяет что исправлено
}
```

Если баг в frontend — описать ручной regression checklist.

**Все тесты зелёные:**
```powershell
GO111MODULE=on go test ./internal/...
```

---

## Шаг 4 — Security check (если баг security-related)

Если баг связан с:
- Auth / authorization
- Утечкой данных
- Секретами в логах
- IDOR / injection

→ Security Agent проводит минимальный аудит затронутого компонента

---

## Шаг 5 — Supervisor: финальная проверка

```powershell
.\ai-team\scripts\run-checks.ps1
```

Supervisor проверяет:
- [ ] git diff — исправлены только файлы в zone of fix
- [ ] Regression test добавлен
- [ ] Все проверки зелёные
- [ ] Нет попутного рефакторинга

---

## Шаг 6 — Коммит

Только после подтверждения Product Owner:

```powershell
git add <конкретные файлы>
git commit -m "fix(<scope>): <что исправлено>"
```

Примеры:
```
fix(frontend): redirect owner and admin to schedule grid after login
fix(backend): allow admin to create and update branches
fix(availability): exclude cancelled appointments from slot blocking
```

---

## Быстрый путь (critical bug)

Для critical severity Supervisor может пропустить шаги 1 и 3 (диагностика и regression test) и добавить их после fix. Но Security check — никогда не пропускается.

---

## Что НЕЛЬЗЯ при bugfix

- ❌ Рефакторить код вокруг бага
- ❌ "Пока здесь — улучшу вот это"
- ❌ Добавлять features
- ❌ Менять API контракты
- ❌ Коммитить без regression test (кроме critical fast path)
