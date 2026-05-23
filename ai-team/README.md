# AI Dev Team — Clinic Scheduler

Полноценный отдел AI-агентов для разработки и поддержки **clinic management web application**.

---

## Что это

Структурированная система AI-агентов, каждый из которых имеет чёткую зону ответственности, ограниченные права на изменения и обязательные проверки перед сдачей работы. Supervisor координирует агентов и не даёт системе выходить за рамки задачи.

Цель — безопасная, предсказуемая разработка без самодеятельности.

---

## Структура команды

```
Product Owner
      │
      ▼
 Supervisor
      │
 ┌────┴────────────────────────────────────┐
 │                                         │
 ▼                                         ▼
Product Agent                         Architect Agent
(requirements, PRD)                   (design, ADR, schema)
                                           │
                          ┌────────────────┼────────────────┐
                          ▼                ▼                ▼
                    Backend Agent    Frontend Agent    DevOps Agent
                    (Go API)         (React UI)        (Docker, CI)
                          │                │
                          └───────┬────────┘
                                  ▼
                             QA Agent
                             (tests, coverage)
                                  │
                                  ▼
                           Security Agent
                           (auth, access control, secrets)
```

---

## Стек проекта

| Компонент | Технология |
|---|---|
| Backend | Go 1.25, chi router, pgx v5, goose |
| Frontend | React 18, Vite 5, Tailwind CSS, Zustand |
| Bot | Go, Telegram Bot API |
| DB | PostgreSQL 16 |
| Infra | Docker Compose |

---

## Файловая структура команды

```
ai-team/
  README.md                      этот файл
  supervisor.md                  Supervisor Agent — главный координатор
  agents/
    architect.md                 Architect Agent
    backend.md                   Backend Agent
    frontend.md                  Frontend Agent
    qa.md                        QA Agent
    security.md                  Security Agent
    product.md                   Product Agent
    devops.md                    DevOps Agent
  workflows/
    feature_workflow.md          как реализовать новую фичу
    bugfix_workflow.md           как исправить баг
    migration_workflow.md        как добавить DB-миграцию
    release_workflow.md          как выпустить релиз
  scripts/
    run-checks.ps1               запуск всех проверок (tests, lint, build)
    project-status.ps1           текущее состояние проекта
```

---

## Как запустить

### Проверить состояние проекта

```powershell
.\ai-team\scripts\project-status.ps1
```

### Запустить все проверки перед коммитом

```powershell
.\ai-team\scripts\run-checks.ps1
```

---

## Правила системы

1. **Supervisor всегда читает задачу перед делегированием** — нет делегирования без понимания scope.
2. **Каждый агент работает только в своей зоне** — Backend Agent не трогает frontend, и наоборот.
3. **Никаких коммитов без подтверждения Product Owner** — агенты готовят изменения, не пушат.
4. **При любом риске — стоп** — агент сигнализирует Supervisor, не угадывает.
5. **Тесты обязательны** — QA Agent проверяет всё перед передачей Supervisor.
6. **Security Agent проверяет каждый PR** — не bypassed, не skippped.

---

## Ссылки на документацию проекта

- `CLAUDE.md` — правила для AI агентов (обязательное чтение)
- `docs/TZ.md` — полное техническое задание
- `docs/ROADMAP.md` — план этапов разработки
- `docs/DECISIONS.md` — лог архитектурных решений
- `ai-team/agents/` — описание каждого агента
- `ai-team/workflows/` — пошаговые процессы
