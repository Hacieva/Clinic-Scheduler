# ТЗ: Clinic Scheduler — MVP система управления записью пациентов

**Статус:** Ready for development
**Дата:** 2026-05-18
**Версия:** 1.0 (MVP)

---

## Содержание
1. Обзор проекта
2. Ответы на вопросы
3. Полное описание функциональности
4. API контракты
5. DB Schema
6. Архитектура для многоканальности
7. Приоритет разработки

---

## 1. Обзор проекта

### Цель
Создать систему управления записью пациентов в частную клинику.

Система позволяет:
- **Администратору** управлять врачами, услугами, расписанием, смотреть все записи
- **Врачу** смотреть свое расписание и записи пациентов
- **Пациентам** записываться через Telegram-бота

### На будущее (не в MVP)
- WhatsApp бот
- Viber бот
- SMS-уведомления
- Email-уведомления
- Личный кабинет пациента
- Оплата

---

## 2. Ответы на оставшиеся вопросы

### 1.5. По пациентам и Telegram-боту

**Q: Какие данные пациент должен оставить при записи?**
A: 
- ФИО (обязательно)
- Телефон (обязательно)
- Дата рождения (нет, не нужна в MVP)
- Комментарий (опционально)

**Q: Подтверждение номера через "Поделиться контактом"?**
A: Да. Бот должен предложить кнопку "Поделиться контактом" вместо ввода текста.

**Q: Может ли пациент записать другого человека?**
A: Нет. В MVP пациент записывает только себя. На будущее можно добавить.

**Q: Может ли пациент отменить запись через Telegram?**
A: Нет в MVP. Отмену делает администратор.
На будущее: `/cancel` команда с подтверждением.

**Q: Может ли пациент перенести запись?**
A: Нет в MVP. Переносит администратор.

**Q: Напоминания?**
A: Нет в MVP. На будущее с интеграцией отправки сообщений.

**Q: Уведомление врачу о новой записи?**
A: Нет в MVP. На будущее можно через Telegram/Email.

**Q: Уведомление администратору?**
A: Нет в MVP.

**Q: Показывать цену перед подтверждением?**
A: Да. Итоговое сообщение должно содержать цену.

**Q: Показывать адрес клиники?**
A: Да. В итоговом сообщении показываем:
- Адрес филиала врача
- Кабинет врача

**Q: Согласие на обработку персональных данных?**
A: Да. Перед подтверждением бот показывает текст согласия и просит кнопкой подтвердить.

### 1.6. По записи на прием

**Q: Администратор может создать запись вручную?**
A: Да.

**Q: Администратор может отменить/перенести?**
A: Отменить — да. Перенести — нет в MVP (это новая запись + отмена старой).

**Q: Какие статусы записи?**
A: 
- `created` — запись создана через бота/админку
- `confirmed` — подтверждена администратором (опционально)
- `cancelled_by_admin` — отменена админом
- `cancelled_by_patient` — для будущего функционала
- `completed` — прием завершен
- `no_show` — не пришел

Для MVP активные: `created`, `confirmed`, `cancelled_by_admin`.

**Q: Хранить историю изменений?**
A: Да. Таблица `appointment_status_history`.

**Q: Что делать при одновременной записи?**
A: Защита на уровне:
1. Backend-логика с транзакциями
2. PostgreSQL EXCLUDE constraint на пересечение временных интервалов

**Q: "Держать" слот за пациентом при прохождении шагов?**
A: Нет в MVP. Слот резервируется только при подтверждении записи.
На будущее: временное резервирование на 5 минут.

**Q: Несколько услуг подряд?**
A: Нет. Одна запись = одна услуга одного врача.

**Q: Несколько активных записей у пациента?**
A: Да. Пациент может иметь несколько записей к разным врачам или в разные дни.

**Q: Врач видит причину обращения?**
A: Нет. Только служебный комментарий, который админ может добавить при создании.

### 1.7. По фронтенду

**Q: Десктоп или мобильный?**
A: Оба (responsive). React + Tailwind или Material-UI.

**Q: Какие страницы в MVP?**
A: 
**Админка:**
- `/login` — вход
- `/dashboard` — статистика (опционально для MVP)
- `/doctors` — список врачей
- `/doctors/:id` — карточка врача (услуги, расписание)
- `/directions` — список направлений
- `/services?doctor=:id` — услуги врача
- `/schedule?doctor=:id` — расписание
- `/appointments` — все записи
- `/profile` — профиль админа

**Кабинет врача:**
- `/login` — вход
- `/schedule` — мое расписание (день/неделя)

**Q: Календарный интерфейс?**
A: Да для врача (неделя, как Google Calendar). Для админа — список с фильтрами.

**Q: Фильтры?**
A: Да:
- По врачу
- По направлению
- По дате (from/to)
- По статусу

**Q: Экспорт/печать?**
A: Нет в MVP.

**Q: Темная тема?**
A: Нет в MVP.

### 1.8. По безопасности

**Q: Как создать учетную запись врача?**
A: Админ в UI создает врача, затем переходит в карточку врача и нажимает "Создать учетную запись".
При этом генерируется:
- email врача (можно задать или сгенерировать)
- временный пароль (показать один раз, врач должен сменить)

**Q: Сброс пароля?**
A: Да. На странице логина ссылка "Забыли пароль?".
Отправляем на email.

**Q: 2FA?**
A: Нет в MVP.

**Q: Сколько живет сессия?**
A: 7 дней для админа, 30 дней для врача.

**Q: Логирование действий админа?**
A: Да. Таблица `audit_logs`.
Логируем:
- Создание/редактирование врача
- Удаление записи
- Изменение расписания

Не логируем полные персональные данные, только IDs и действия.

**Q: Разграничение прав?**
A: Да.
- Админ может всё
- Врач не может редактировать услуги, не видит чужие записи, не может отменять запись

### 1.9. По технической части

**Q: Язык backend?**
A: **Go** (учитывая твой опыт и что это лучше для многоканальности).

**Q: Фронтенд?**
A: **React 18 + Vite + Zustand** (как в предыдущем проекте, но чистше).

**Q: Размещение?**
A: **Docker Compose** для MVP. На будущее Docker + Kubernetes.

**Q: Telegram-бот отдельный или часть backend?**
A: **Отдельный Go-сервис** в одном репозитории.
Это даст гибкость на будущее для других мессенджеров.

**Q: Polling или webhook?**
A: **Webhook** (безопаснее, быстрее).

**Q: Миграции?**
A: **goose** (проще чем golang-migrate).

**Q: Тесты?**
A: Минимум:
- Unit tests для availability, appointment creation
- Integration tests для БД
- API tests для critical endpoints

**Q: Swagger?**
A: Да. Генерируем из Go-кода через swag.

**Q: Готовая админ-панель?**
A: Нет. Пишем свою на React. Это быстрее и проще кастомизировать.

---

## 3. Полная архитектура и функциональность

### 3.1. Система ролей

```
User (базовая сущность)
├── admin (ты)
│   └── полный доступ ко всему
│
└── doctor
    ├── видит только свое расписание
    ├── видит только свои записи
    ├── видит ФИО пациента
    └── не видит телефон/Telegram
```

### 3.2. Сущности и связи

```
Direction (направление)
├── Кардиология
├── Терапия
└── ...

Doctor (врач)
├── ФИО
├── кабинет
├── адрес филиала
├── description
├── photo_url
├── is_active
│
├─── DoctorDirection (связь врача и направления)
│    ├── doctor_id
│    └── direction_id
│
├─── Service (услуга конкретного врача)
│    ├── name
│    ├── duration_minutes
│    ├── price
│    ├── direction_id
│    └── is_active
│
├─── DoctorWorkingHours (расписание на неделю)
│    ├── day_of_week (1-7)
│    ├── start_time
│    ├── end_time
│    └── is_active
│
└─── DoctorScheduleException (исключения)
     ├── date
     ├── type (day_off | custom_working_hours)
     ├── start_time
     └── end_time

Patient (пациент)
├── full_name
├── phone
├── telegram_user_id
├── telegram_username
└── created_at

Appointment (запись)
├── patient_id
├── doctor_id
├── service_id
├── direction_id
├── start_at
├── end_at
├── status (created | confirmed | cancelled_by_admin | completed | no_show)
├── source (telegram_bot | admin_panel)
└── created_at

AppointmentStatusHistory (история изменений)
├── appointment_id
├── old_status
├── new_status
├── changed_by_user_id
├── changed_at
└── comment
```

### 3.3. Основные сценарии

#### Сценарий 1: Администратор создает врача

```
1. Админ -> Врачи -> Добавить врача
2. Заполняет:
   - ФИО
   - Описание (опционально)
   - Кабинет
   - Адрес филиала
3. Выбирает направления (может несколько)
4. Сохраняет
5. Переходит в карточку врача
6. Создает учетную запись (email, временный пароль)
7. Добавляет услуги (название, длительность, цена, направление)
8. Настраивает расписание (дни недели, часы)
9. Может добавить исключения (отпуск, больничный)

Результат:
- Врач готов к записи пациентов
- Врач может войти в свой кабинет
- Пациент может выбрать этого врача в Telegram
```

#### Сценарий 2: Пациент записывается через Telegram

```
1. Пациент -> /start
2. Бот показывает направления (инлайн-кнопки):
   - Кардиология
   - Терапия
   - и т.д.
3. Пациент выбирает направление
4. Бот показывает врачей этого направления:
   - Иванов Иван Иванович
   - Петров Петр Петрович
5. Пациент выбирает врача
6. Бот показывает услуги этого врача по направлению:
   - Первичная консультация (60 мин, 3000 ₽)
   - Повторная консультация (30 мин, 2000 ₽)
7. Пациент выбирает услугу
8. Бот показывает доступные даты (вперед на 30 дней)
   - 20 мая
   - 21 мая
   - 22 мая
   - и т.д. (только дни когда врач работает)
9. Пациент выбирает дату
10. Бот показывает свободные окна:
    - 10:00-11:00
    - 11:00-12:00
    - 14:00-15:00
11. Пациент выбирает время
12. Бот запрашивает: "Укажите ваше ФИО"
13. Пациент: "Алексей Иванов"
14. Бот запрашивает: "Поделитесь номером телефона" (кнопка "Поделиться контактом")
15. Пациент нажимает кнопку, отправляет контакт
16. Бот показывает итого:
    ```
    ✓ Врач: Иванов Иван Иванович
    ✓ Направление: Кардиология
    ✓ Услуга: Первичная консультация
    ✓ Дата: 20.05.2026
    ✓ Время: 10:00-11:00
    ✓ Кабинет: 205
    ✓ Адрес: Москва, ул. Примерная, д. 1
    ✓ Стоимость: 3000 ₽
    
    Согласие на обработку персональных данных:
    Нажимая "Подтвердить", вы согласны с политикой обработки данных.
    
    [Отмена]  [Подтвердить]
    ```
17. Пациент нажимает "Подтвердить"
18. Backend создает запись в БД
19. Бот отправляет подтверждение:
    ```
    ✅ Вы успешно записаны!
    
    Врач: Иванов Иван Иванович
    Услуга: Первичная консультация
    Дата: 20 мая 2026
    Время: 10:00
    Кабинет: 205
    Адрес: Москва, ул. Примерная, д. 1
    
    Если нужно отменить, свяжитесь с клиникой:
    +7 (999) 999-99-99
    
    /help — помощь
    /my_appointments — мои записи
    ```

Результат:
- Запись создана в БД
- Слот становится занятым
- Врач видит запись в своем расписании
- Админ видит запись в списке всех записей
```

#### Сценарий 3: Врач смотрит свое расписание

```
1. Врач -> Вход (email + пароль)
2. Попадает в кабинет врача
3. Видит "Мое расписание"
4. По умолчанию открыта неделя
5. Может переключиться на день
6. Видит записи:
   - 10:00-11:00: Алексей Иванов, Первичная консультация
   - 11:00-11:30: Петр Петров, Повторная консультация
7. Может кликнуть на запись и увидеть:
   - ФИО пациента
   - Услугу
   - Направление
   - Время начала/окончания
   - Кабинет
8. Не видит: телефон, Telegram, email пациента

Результат:
- Врач знает свою загрузку
- Врач готов к приемам
```

#### Сценарий 4: Администратор отменяет запись

```
1. Админ -> Записи
2. Видит все записи с фильтрами
3. Находит запись (например, "Алексей Иванов, 20 мая, 10:00, Кардиология")
4. Нажимает "Отменить"
5. Система просит подтверждение
6. Админ подтверждает
7. Статус меняется на cancelled_by_admin
8. Создается запись в audit_logs
9. Создается запись в appointment_status_history

Результат:
- Запись отменена
- Слот снова доступен для записи
- История сохранена
```

### 3.4. Правила расписания

#### Регулярное расписание (weekly pattern)

```
Понедельник:  10:00-16:00
Вторник:      нет (врач не работает)
Среда:        10:00-13:00, 14:00-18:00 (два интервала - обед 13:00-14:00)
Четверг:      10:00-16:00
Пятница:      10:00-14:00
Суббота:      нет
Воскресенье:  нет
```

#### Исключения (exceptions)

```
Тип: day_off
Дата: 2026-05-20 (День независимости, например)
Результат: врач вообще не работает в этот день

Тип: custom_working_hours
Дата: 2026-05-21
Время: 09:00-12:00 (вместо обычного 10:00-16:00)
Результат: врач работает только эти часы
```

#### Алгоритм поиска свободных окон

```
FOR каждого дня в диапазоне {
  1. Проверяем исключение (exception)
     - Если day_off -> день недоступен
     - Если custom_working_hours -> используем это время
     - Иначе используем регулярное расписание (day_of_week)
  
  2. Если день доступен:
     - Берем рабочие часы (например, 10:00-16:00)
     - Берем выбранную услугу (длительность, например, 60 мин)
     - Берем минимальную сетку (30 мин)
  
  3. Генерируем потенциальные слоты с шагом 30 мин:
     - 10:00-11:00
     - 10:30-11:30
     - 11:00-12:00
     - 11:30-12:30
     - и т.д. до конца дня
  
  4. Для каждого слота проверяем:
     - Достаточно ли времени? (end_at <= рабочий_конец)
     - Нет ли пересечения с активными записями?
     - Записи со статусами created, confirmed блокируют слот
     - Отмененные записи не блокируют
  
  5. Добавляем в результат только свободные слоты
}

Результат: список доступных дат и времен для Telegram-бота
```

#### Защита от двойной записи

**PostgreSQL EXCLUDE constraint:**

```sql
ALTER TABLE appointments 
ADD CONSTRAINT no_overlapping_appointments 
EXCLUDE USING GIST (
  doctor_id WITH =,
  tstzrange(start_at, end_at) WITH &&
) 
WHERE (status IN ('created', 'confirmed'));
```

**Backend логика:**

```go
func CreateAppointment(ctx context.Context, apt *Appointment) error {
  // 1. Начинаем транзакцию
  tx := db.BeginTx(ctx, nil)
  
  // 2. Проверяем что слот свободен
  exists := tx.
    Where("doctor_id = ? AND status IN ('created', 'confirmed')", apt.DoctorID).
    Where("start_at < ? AND end_at > ?", apt.EndAt, apt.StartAt).
    Exists()
  
  if exists {
    return errors.New("slot already booked")
  }
  
  // 3. Создаем запись
  if err := tx.Create(apt).Error; err != nil {
    tx.Rollback()
    return err
  }
  
  // 4. Создаем историю
  if err := tx.Create(&AppointmentHistory{
    AppointmentID: apt.ID,
    OldStatus: nil,
    NewStatus: apt.Status,
    ChangedAt: time.Now(),
  }).Error; err != nil {
    tx.Rollback()
    return err
  }
  
  // 5. Коммитим
  return tx.Commit().Error
}
```

---

## 4. API контракты

### 4.1. Auth API

```
POST /api/auth/login
{
  "email": "admin@clinic.ru",
  "password": "password123"
}

Response:
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "email": "admin@clinic.ru",
    "role": "admin"
  }
}

---

POST /api/auth/logout
Header: Authorization: Bearer <token>

Response: 200 OK

---

GET /api/auth/me
Header: Authorization: Bearer <token>

Response:
{
  "id": 1,
  "email": "admin@clinic.ru",
  "role": "admin"
}

---

POST /api/auth/refresh
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}

Response:
{
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}

---

POST /api/auth/change-password
Header: Authorization: Bearer <token>
{
  "old_password": "password123",
  "new_password": "newpassword123"
}

Response: 200 OK

---

POST /api/auth/forgot-password
{
  "email": "admin@clinic.ru"
}

Response: 200 OK

---

POST /api/auth/reset-password
{
  "token": "reset_token_from_email",
  "new_password": "newpassword123"
}

Response: 200 OK
```

### 4.2. Directions API

```
GET /api/directions?active=true
Header: Authorization: Bearer <token>

Response:
[
  {
    "id": 1,
    "name": "Кардиология",
    "description": "Заболевания сердца",
    "is_active": true
  },
  {
    "id": 2,
    "name": "Терапия",
    "description": "Общая терапия",
    "is_active": true
  }
]

---

POST /api/directions
Header: Authorization: Bearer <token>
{
  "name": "Неврология",
  "description": "Заболевания нервной системы"
}

Response: 201 Created
{
  "id": 3,
  "name": "Неврология",
  "description": "Заболевания нервной системы",
  "is_active": true,
  "created_at": "2026-05-18T10:00:00Z"
}

---

PATCH /api/directions/:id
Header: Authorization: Bearer <token>
{
  "name": "Неврология (обновлено)",
  "is_active": false
}

Response: 200 OK
{
  "id": 3,
  "name": "Неврология (обновлено)",
  "is_active": false
}

---

DELETE /api/directions/:id
Header: Authorization: Bearer <token>

Response: 204 No Content
```

### 4.3. Doctors API

```
GET /api/doctors?active=true
Header: Authorization: Bearer <token>

Response:
[
  {
    "id": 1,
    "user_id": 2,
    "first_name": "Иван",
    "last_name": "Иванов",
    "middle_name": "Иванович",
    "cabinet": "205",
    "branch_address": "Москва, ул. Примерная, д. 1",
    "description": "Опыт работы 10 лет",
    "photo_url": "https://...",
    "is_active": true,
    "directions": [1, 2],
    "created_at": "2026-05-18T10:00:00Z"
  }
]

---

POST /api/doctors
Header: Authorization: Bearer <token>
{
  "first_name": "Иван",
  "last_name": "Иванов",
  "middle_name": "Иванович",
  "cabinet": "205",
  "branch_address": "Москва, ул. Примерная, д. 1",
  "description": "Опыт работы 10 лет",
  "photo_url": "https://...",
  "direction_ids": [1, 2]
}

Response: 201 Created
{
  "id": 1,
  "first_name": "Иван",
  "last_name": "Иванов",
  "middle_name": "Иванович",
  "cabinet": "205",
  "branch_address": "Москва, ул. Примерная, д. 1",
  "description": "Опыт работы 10 лет",
  "photo_url": "https://...",
  "is_active": true,
  "directions": [1, 2],
  "created_at": "2026-05-18T10:00:00Z"
}

---

GET /api/doctors/:id
Header: Authorization: Bearer <token>

Response:
{
  "id": 1,
  "user_id": 2,
  "first_name": "Иван",
  "last_name": "Иванов",
  "middle_name": "Иванович",
  "cabinet": "205",
  "branch_address": "Москва, ул. Примерная, д. 1",
  "description": "Опыт работы 10 лет",
  "photo_url": "https://...",
  "is_active": true,
  "directions": [
    {
      "id": 1,
      "name": "Кардиология"
    },
    {
      "id": 2,
      "name": "Терапия"
    }
  ],
  "services": [
    {
      "id": 10,
      "name": "Первичная консультация",
      "description": "...",
      "duration_minutes": 60,
      "price": 3000,
      "direction_id": 1,
      "is_active": true
    }
  ],
  "working_hours": [
    {
      "id": 1,
      "day_of_week": 1,
      "start_time": "10:00",
      "end_time": "16:00"
    }
  ],
  "schedule_exceptions": [
    {
      "id": 1,
      "date": "2026-05-20",
      "type": "day_off",
      "comment": "Отпуск"
    }
  ],
  "created_at": "2026-05-18T10:00:00Z"
}

---

PATCH /api/doctors/:id
Header: Authorization: Bearer <token>
{
  "cabinet": "206",
  "description": "Опыт работы 15 лет",
  "is_active": true,
  "direction_ids": [1, 2, 3]
}

Response: 200 OK
{
  "id": 1,
  "cabinet": "206",
  "description": "Опыт работы 15 лет",
  "is_active": true,
  "directions": [1, 2, 3]
}

---

DELETE /api/doctors/:id
Header: Authorization: Bearer <token>

Response: 204 No Content
(Soft delete: is_active = false)

---

POST /api/doctors/:id/account
Header: Authorization: Bearer <token>
{
  "email": "ivanov@clinic.ru",
  "password": "temp_password_123"
}

Response: 201 Created
{
  "user_id": 2,
  "email": "ivanov@clinic.ru",
  "temporary_password": "temp_password_123",
  "message": "Учетная запись создана. Врач должен сменить пароль при первом входе."
}
```

### 4.4. Services API (для врача)

```
GET /api/doctors/:doctorId/services
Header: Authorization: Bearer <token>

Response:
[
  {
    "id": 10,
    "name": "Первичная консультация",
    "description": "Первый прием у кардиолога",
    "duration_minutes": 60,
    "price": 3000,
    "direction_id": 1,
    "direction_name": "Кардиология",
    "is_active": true,
    "created_at": "2026-05-18T10:00:00Z"
  },
  {
    "id": 11,
    "name": "Повторная консультация",
    "description": "Повторный прием",
    "duration_minutes": 30,
    "price": 2000,
    "direction_id": 1,
    "direction_name": "Кардиология",
    "is_active": true,
    "created_at": "2026-05-18T10:00:00Z"
  }
]

---

POST /api/doctors/:doctorId/services
Header: Authorization: Bearer <token>
{
  "name": "УЗИ сердца",
  "description": "Ультразвуковое исследование сердца",
  "duration_minutes": 30,
  "price": 2500,
  "direction_id": 1
}

Response: 201 Created
{
  "id": 12,
  "name": "УЗИ сердца",
  "description": "Ультразвуковое исследование сердца",
  "duration_minutes": 30,
  "price": 2500,
  "direction_id": 1,
  "direction_name": "Кардиология",
  "is_active": true,
  "created_at": "2026-05-18T10:00:00Z"
}

---

PATCH /api/doctors/:doctorId/services/:serviceId
Header: Authorization: Bearer <token>
{
  "price": 3000,
  "duration_minutes": 45
}

Response: 200 OK

---

DELETE /api/doctors/:doctorId/services/:serviceId
Header: Authorization: Bearer <token>

Response: 204 No Content
(Soft delete)
```

### 4.5. Schedule API

```
GET /api/doctors/:doctorId/working-hours
Header: Authorization: Bearer <token>

Response:
[
  {
    "id": 1,
    "day_of_week": 1,
    "start_time": "10:00",
    "end_time": "16:00",
    "is_active": true
  },
  {
    "id": 2,
    "day_of_week": 3,
    "start_time": "10:00",
    "end_time": "13:00",
    "is_active": true
  },
  {
    "id": 3,
    "day_of_week": 3,
    "start_time": "14:00",
    "end_time": "18:00",
    "is_active": true
  }
]

---

PUT /api/doctors/:doctorId/working-hours
Header: Authorization: Bearer <token>
{
  "schedule": [
    {
      "day_of_week": 1,
      "start_time": "10:00",
      "end_time": "16:00"
    },
    {
      "day_of_week": 2,
      "start_time": "10:00",
      "end_time": "16:00"
    },
    {
      "day_of_week": 3,
      "start_time": "10:00",
      "end_time": "13:00"
    },
    {
      "day_of_week": 3,
      "start_time": "14:00",
      "end_time": "18:00"
    },
    {
      "day_of_week": 4,
      "start_time": "10:00",
      "end_time": "16:00"
    },
    {
      "day_of_week": 5,
      "start_time": "10:00",
      "end_time": "14:00"
    }
  ]
}

Response: 200 OK

---

GET /api/doctors/:doctorId/schedule-exceptions
Header: Authorization: Bearer <token>

Response:
[
  {
    "id": 1,
    "date": "2026-05-20",
    "type": "day_off",
    "start_time": null,
    "end_time": null,
    "comment": "Отпуск"
  },
  {
    "id": 2,
    "date": "2026-05-21",
    "type": "custom_working_hours",
    "start_time": "12:00",
    "end_time": "15:00",
    "comment": "Внеплановое сокращение часов"
  }
]

---

POST /api/doctors/:doctorId/schedule-exceptions
Header: Authorization: Bearer <token>
{
  "date": "2026-05-20",
  "type": "day_off",
  "comment": "Отпуск"
}

Response: 201 Created
{
  "id": 1,
  "date": "2026-05-20",
  "type": "day_off",
  "comment": "Отпуск"
}

---

DELETE /api/doctors/:doctorId/schedule-exceptions/:exceptionId
Header: Authorization: Bearer <token>

Response: 204 No Content
```

### 4.6. Availability API (для бота и фронтенда)

```
GET /api/availability?doctor_id=1&service_id=10&date_from=2026-05-20&date_to=2026-05-30
Header: Authorization: Bearer <token> (для фронтенда)
Header: X-Bot-Token: <bot_token> (для Telegram бота)

Response:
{
  "doctor_id": 1,
  "service_id": 10,
  "service_duration_minutes": 60,
  "availability": [
    {
      "date": "2026-05-20",
      "day_name": "вторник",
      "slots": [
        {
          "start": "10:00",
          "end": "11:00"
        },
        {
          "start": "11:00",
          "end": "12:00"
        },
        {
          "start": "14:00",
          "end": "15:00"
        }
      ]
    },
    {
      "date": "2026-05-21",
      "day_name": "среда",
      "slots": [
        {
          "start": "10:00",
          "end": "11:00"
        },
        {
          "start": "13:00",
          "end": "14:00"
        }
      ]
    }
  ]
}
```

### 4.7. Appointments API

```
GET /api/appointments?doctor_id=1&status=created,confirmed&date_from=2026-05-20&date_to=2026-05-30
Header: Authorization: Bearer <token>

Response:
[
  {
    "id": 1,
    "patient_id": 1,
    "patient_name": "Алексей Иванов",
    "doctor_id": 1,
    "doctor_name": "Иванов Иван Иванович",
    "service_id": 10,
    "service_name": "Первичная консультация",
    "direction_id": 1,
    "direction_name": "Кардиология",
    "start_at": "2026-05-20T10:00:00Z",
    "end_at": "2026-05-20T11:00:00Z",
    "status": "created",
    "source": "telegram_bot",
    "created_at": "2026-05-18T10:00:00Z",
    "phone": "***-****" (скрыто для врача)
  }
]

---

GET /api/appointments/:id
Header: Authorization: Bearer <token>

Response:
{
  "id": 1,
  "patient_id": 1,
  "patient_name": "Алексей Иванов",
  "patient_phone": "+79999999999",
  "patient_telegram": "@alexei_ivanov",
  "doctor_id": 1,
  "doctor_name": "Иванов Иван Иванович",
  "service_id": 10,
  "service_name": "Первичная консультация",
  "direction_id": 1,
  "direction_name": "Кардиология",
  "cabinet": "205",
  "branch_address": "Москва, ул. Примерная, д. 1",
  "start_at": "2026-05-20T10:00:00Z",
  "end_at": "2026-05-20T11:00:00Z",
  "price": 3000,
  "status": "created",
  "source": "telegram_bot",
  "comment": "",
  "created_at": "2026-05-18T10:00:00Z"
}

---

POST /api/appointments
Header: Authorization: Bearer <token>
{
  "patient_id": 1,
  "doctor_id": 1,
  "service_id": 10,
  "direction_id": 1,
  "start_at": "2026-05-20T10:00:00Z",
  "end_at": "2026-05-20T11:00:00Z",
  "status": "created",
  "source": "admin_panel",
  "comment": ""
}

Response: 201 Created
{
  "id": 1,
  "patient_id": 1,
  "doctor_id": 1,
  "service_id": 10,
  "direction_id": 1,
  "start_at": "2026-05-20T10:00:00Z",
  "end_at": "2026-05-20T11:00:00Z",
  "status": "created",
  "source": "admin_panel",
  "created_at": "2026-05-18T10:00:00Z"
}

---

POST /api/appointments/:id/cancel
Header: Authorization: Bearer <token>
{
  "reason": "Пациент попросил отменить"
}

Response: 200 OK
{
  "id": 1,
  "status": "cancelled_by_admin",
  "updated_at": "2026-05-18T10:30:00Z"
}

---

POST /api/appointments/:id/complete
Header: Authorization: Bearer <token>

Response: 200 OK
{
  "id": 1,
  "status": "completed",
  "updated_at": "2026-05-18T12:00:00Z"
}
```

### 4.8. Telegram Bot API (internal)

```
Бот не должен иметь прямой доступ в БД.
Бот ходит в backend через эти endpoints:

GET /api/bot/directions
Header: X-Bot-Token: <secure_token>

Response:
[
  {
    "id": 1,
    "name": "Кардиология"
  },
  {
    "id": 2,
    "name": "Терапия"
  }
]

---

GET /api/bot/directions/:directionId/doctors
Header: X-Bot-Token: <secure_token>

Response:
[
  {
    "id": 1,
    "name": "Иванов Иван Иванович"
  },
  {
    "id": 2,
    "name": "Петров Петр Петрович"
  }
]

---

GET /api/bot/doctors/:doctorId/services?direction_id=1
Header: X-Bot-Token: <secure_token>

Response:
[
  {
    "id": 10,
    "name": "Первичная консультация",
    "duration_minutes": 60,
    "price": 3000
  },
  {
    "id": 11,
    "name": "Повторная консультация",
    "duration_minutes": 30,
    "price": 2000
  }
]

---

GET /api/bot/availability?doctor_id=1&service_id=10&date_from=2026-05-20&date_to=2026-05-30
Header: X-Bot-Token: <secure_token>

Response: (см. availability выше)

---

POST /api/bot/appointments
Header: X-Bot-Token: <secure_token>
{
  "telegram_user_id": 123456789,
  "telegram_username": "alexei_ivanov",
  "patient_name": "Алексей Иванов",
  "patient_phone": "+79999999999",
  "doctor_id": 1,
  "service_id": 10,
  "direction_id": 1,
  "start_at": "2026-05-20T10:00:00Z"
}

Response: 201 Created
{
  "id": 1,
  "patient_id": 1,
  "appointment_id": 1,
  "status": "created",
  "doctor_name": "Иванов Иван Иванович",
  "service_name": "Первичная консультация",
  "start_at": "2026-05-20T10:00:00Z",
  "end_at": "2026-05-20T11:00:00Z",
  "cabinet": "205",
  "branch_address": "Москва, ул. Примерная, д. 1",
  "price": 3000
}
```

### 4.9. Doctor API (для врача смотреть свое расписание)

```
GET /api/doctor/appointments?date_from=2026-05-20&date_to=2026-05-30
Header: Authorization: Bearer <token>
(Doctor_id берется из токена)

Response:
[
  {
    "id": 1,
    "patient_name": "Алексей Иванов",
    "service_name": "Первичная консультация",
    "direction_name": "Кардиология",
    "start_at": "2026-05-20T10:00:00Z",
    "end_at": "2026-05-20T11:00:00Z",
    "cabinet": "205",
    "status": "created"
  }
]

---

GET /api/doctor/schedule?date_from=2026-05-20&date_to=2026-05-30
Header: Authorization: Bearer <token>

Response:
{
  "doctor_id": 1,
  "doctor_name": "Иванов Иван Иванович",
  "doctor_cabinet": "205",
  "doctor_branch_address": "Москва, ул. Примерная, д. 1",
  "week_schedule": {
    "2026-05-20": {
      "date": "2026-05-20",
      "day_name": "вторник",
      "is_working": true,
      "working_hours": [
        {
          "start": "10:00",
          "end": "16:00"
        }
      ],
      "appointments": [
        {
          "id": 1,
          "start_at": "10:00",
          "end_at": "11:00",
          "patient_name": "Алексей Иванов",
          "service_name": "Первичная консультация"
        }
      ]
    }
  }
}
```

---

## 5. PostgreSQL Schema

```sql
-- Users table
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'doctor')),
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Directions table
CREATE TABLE directions (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Doctors table
CREATE TABLE doctors (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  first_name VARCHAR(255) NOT NULL,
  last_name VARCHAR(255) NOT NULL,
  middle_name VARCHAR(255),
  cabinet VARCHAR(50),
  branch_address TEXT,
  description TEXT,
  photo_url VARCHAR(500),
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Doctor directions (many-to-many)
CREATE TABLE doctor_directions (
  id BIGSERIAL PRIMARY KEY,
  doctor_id BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
  direction_id BIGINT NOT NULL REFERENCES directions(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(doctor_id, direction_id)
);

-- Services table
CREATE TABLE services (
  id BIGSERIAL PRIMARY KEY,
  doctor_id BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
  direction_id BIGINT NOT NULL REFERENCES directions(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  duration_minutes INT NOT NULL CHECK (duration_minutes > 0),
  price DECIMAL(10, 2),
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Doctor working hours
CREATE TABLE doctor_working_hours (
  id BIGSERIAL PRIMARY KEY,
  doctor_id BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
  day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
  start_time TIME NOT NULL,
  end_time TIME NOT NULL,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(doctor_id, day_of_week, start_time, end_time)
);

-- Doctor schedule exceptions
CREATE TABLE doctor_schedule_exceptions (
  id BIGSERIAL PRIMARY KEY,
  doctor_id BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  type VARCHAR(50) NOT NULL CHECK (type IN ('day_off', 'custom_working_hours')),
  start_time TIME,
  end_time TIME,
  comment TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(doctor_id, date)
);

-- Patients table
CREATE TABLE patients (
  id BIGSERIAL PRIMARY KEY,
  telegram_user_id BIGINT UNIQUE,
  telegram_username VARCHAR(255),
  full_name VARCHAR(255) NOT NULL,
  phone VARCHAR(20) NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Appointments table
CREATE TABLE appointments (
  id BIGSERIAL PRIMARY KEY,
  patient_id BIGINT NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
  doctor_id BIGINT NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
  service_id BIGINT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
  direction_id BIGINT NOT NULL REFERENCES directions(id) ON DELETE CASCADE,
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  status VARCHAR(50) NOT NULL DEFAULT 'created' CHECK (status IN (
    'created', 'confirmed', 'cancelled_by_patient', 'cancelled_by_admin', 'completed', 'no_show'
  )),
  source VARCHAR(50) NOT NULL CHECK (source IN ('telegram_bot', 'admin_panel')),
  patient_comment TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  
  -- Защита от пересечения записей (EXCLUDE constraint)
  EXCLUDE USING GIST (
    doctor_id WITH =,
    tstzrange(start_at, end_at) WITH &&
  ) WHERE (status IN ('created', 'confirmed'))
);

-- Appointment status history
CREATE TABLE appointment_status_history (
  id BIGSERIAL PRIMARY KEY,
  appointment_id BIGINT NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
  old_status VARCHAR(50),
  new_status VARCHAR(50) NOT NULL,
  changed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  changed_at TIMESTAMPTZ DEFAULT NOW(),
  comment TEXT
);

-- Audit logs
CREATE TABLE audit_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action VARCHAR(255) NOT NULL,
  entity_type VARCHAR(50) NOT NULL,
  entity_id BIGINT,
  old_values JSONB,
  new_values JSONB,
  ip_address VARCHAR(50),
  user_agent TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_doctors_user_id ON doctors(user_id);
CREATE INDEX idx_doctors_is_active ON doctors(is_active);
CREATE INDEX idx_services_doctor_id ON services(doctor_id);
CREATE INDEX idx_services_direction_id ON services(direction_id);
CREATE INDEX idx_doctor_working_hours_doctor_id ON doctor_working_hours(doctor_id);
CREATE INDEX idx_doctor_schedule_exceptions_doctor_id ON doctor_schedule_exceptions(doctor_id);
CREATE INDEX idx_appointments_doctor_id ON appointments(doctor_id);
CREATE INDEX idx_appointments_patient_id ON appointments(patient_id);
CREATE INDEX idx_appointments_start_at ON appointments(start_at);
CREATE INDEX idx_appointments_status ON appointments(status);
CREATE INDEX idx_patients_telegram_user_id ON patients(telegram_user_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
```

---

## 6. Архитектура для многоканальности (Telegram + будущее)

### 6.1. Правильная архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                       Telegram Bot                          │
│  (Go service, work directly with backend API)              │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│                    Backend API (Go)                         │
│  ┌──────────────┐                                           │
│  │ Auth Service │─────────────────────────────────┐         │
│  └──────────────┘                                 │         │
│  ┌──────────────┐                                 │         │
│  │ Doctor Svc   │─────────────────────────────────┼─────┐   │
│  └──────────────┘                                 │     │   │
│  ┌──────────────┐                                 │     │   │
│  │Direction Svc │─────────────────────────────────┼─────┼─┐ │
│  └──────────────┘                                 │     │ │ │
│  ┌──────────────┐                                 │     │ │ │
│  │ Schedule Svc │─────────────────────────────────┼─────┼─┼─┤
│  └──────────────┘                                 │     │ │ │
│  ┌──────────────────────────────────────────────┐│     │ │ │
│  │ Appointment Service                         ││     │ │ │
│  │  - CreateAppointment (with protection)      ││     │ │ │
│  │  - CalculateAvailability                    ││     │ │ │
│  │  - CheckForConflicts                        ││     │ │ │
│  └──────────────────────────────────────────────┘│     │ │ │
│                                                   ▼     ▼ ▼ ▼
│                         Repository Layer (Database Access)   │
└─────────────────────────────────────────────────────────────┘
                 │
                 ▼
         ┌───────────────────┐
         │   PostgreSQL      │
         │  (Single source   │
         │   of truth)       │
         └───────────────────┘
```

### 6.2. На будущее: добавление WhatsApp/Viber

```
┌─────────────────────────────────────────────────────────────┐
│          Telegram Bot          │    WhatsApp Bot           │
│  (Go service)                  │    (Go service)           │
└────────────┬────────────────────┼─────────────┬─────────────┘
             │                    │             │
             └────────┬───────────┴─────────────┘
                      │
                      ▼
            ┌──────────────────┐
            │  Backend API     │
            │  (single source  │
            │  of truth for    │
            │  all channels)   │
            └──────┬───────────┘
                   │
                   ▼
            ┌──────────────────┐
            │  PostgreSQL      │
            └──────────────────┘
```

### 6.3. Ключевые моменты многоканальности

1. **Backend-first подход**
   - Все мессенджеры ходят только в Backend API
   - Backend содержит всю бизнес-логику
   - Легко добавить новый канал

2. **Stateless боты**
   - Боты только обрабатывают UI/UX диалога
   - Все данные хранятся в PostgreSQL через Backend
   - Если бот упадет, данные не потеряются

3. **API токены для ботов**
   - Каждый бот получает X-Bot-Token
   - Backend может разграничить доступ (telegram_bot не будет видеть то же что admin)
   - Легко отключить бота

4. **Независимые деплои**
   - Telegram бот деплоится отдельно
   - Если обновить Telegram бота, Backend не затронется
   - Если обновить Backend, Telegram бот продолжит работать

---

## 7. Приоритет разработки

### Этап 1: Database + Backend-core (неделя 1-2)

```
1. Создать миграции (goose)
2. Написать models
3. Реализовать repository pattern
4. Написать auth (JWT)
5. Написать основные сервисы:
   - DirectionService (CRUD)
   - DoctorService (CRUD)
   - ServiceService (CRUD)
   - ScheduleService (CRUD)
6. Написать критическую логику:
   - AvailabilityCalculator (расчет свободных окон)
   - AppointmentService with transactions
   - EXCLUDE constraint для защиты
```

**Критерии приемки:**
- Все сущности в БД созданы
- Все CRUD endpoints работают
- Транзакции работают
- Защита от пересечения записей на уровне БД+код

### Этап 2: Telegram Bot (неделя 2-3)

```
1. Создать Go service для бота
2. Реализовать основные команды:
   - /start (выбор направления)
   - Выбор врача
   - Выбор услуги
   - Выбор даты
   - Выбор времени
   - Ввод ФИО/телефона
   - Подтверждение
3. Интегрировать с Backend API
4. Обработка ошибок
```

**Критерии приемки:**
- Пациент может записаться через бота
- Запись сохраняется в БД
- Врач видит запись в расписании
- Админ видит запись в списке

### Этап 3: Admin Frontend (неделя 3-4)

```
1. React + Vite + Zustand
2. Страницы:
   - Login
   - Doctors (CRUD)
   - Directions (CRUD)
   - Services (CRUD)
   - Schedule (edit)
   - Appointments (list, filter, cancel)
3. Forms с валидацией
4. Error handling
```

**Критерии приемки:**
- Админ может управлять всем через UI
- Всё синхронизируется с БД
- Интеграция с Backend API работает

### Этап 4: Doctor Frontend (неделя 4)

```
1. Doctor panel с расписанием на день/неделю
2. Просмотр записей
3. Просмотр деталей пациента (только ФИО)
```

**Критерии приемки:**
- Врач видит свое расписание
- Врач видит свои записи
- Всё читается из Backend API

---

## 8. Отличия от предыдущего проекта (clinic-app)

### Что мы исправляем

| Проблема (clinic-app) | Решение (Clinic Scheduler) |
|---|---|
| Race condition в RequireRole | Правильный token-based auth в React + проверка изначально |
| Двойной guard + мерцание | Декларативная авторизация через Layout routes |
| Duplicate code в auth | Centralized auth service в backend |
| init_db() дублирует Alembic | Только миграции, никаких create_all |
| Lazy imports | Все imports наверху файлов |
| CHECK constraint только в коде | CHECK constraint в миграции (goose) |
| Soft delete без нужды | Мягкое удаление только где нужно |
| Price всегда null | Price сразу копируется при создании записи |
| React Query keys без параметров | Правильные ключи с параметрами из день 1 |
| Timezone issues (UTC показывается) | Timezone-aware datetimes с первого дня |
| Broken pagination | Limit/offset в API с сразу |
| Нет tests | Unit + integration tests с MVP |
| Нет Swagger | Swagger генерируется из Go |
| Python/FastAPI | Go (лучше для многоканальности + performance) |

### Что берём из предыдущего

- ErrorBoundary паттерн (скопируем в React)
- formatDate через date-fns
- Структура компонентов
- Zustand для состояния (но правильнее)
- RequireRole логика (но исправим)

---

## 9. Финальный чеклист перед стартом

- [ ] Все вопросы ТЗ раскрыты (есть в этом документе)
- [ ] Нет противоречий в требованиях
- [ ] DB schema согласован
- [ ] API контракты согласованы
- [ ] Приоритет разработки ясен
- [ ] Ясно что входит в MVP, что не входит
- [ ] Все сценарии (admin, doctor, patient) описаны
- [ ] Защита от двойной записи спроектирована
- [ ] Многоканальность заложена в архитектуру
- [ ] Известны инструменты (goose, React, Zustand и т.д.)

---

## 10. Дополнительные рекомендации

### 10.1. Для первого спринта

Сосредоточиться на **ядре системы**, а не на UI:

1. **БД + миграции** (день 1)
2. **Auth** (день 2)
3. **Availability calculator** (день 3) — это сердце системы
4. **Appointment creation with protection** (день 4)
5. **Telegram bot** (день 5-6)

**Только потом** думать о админке и кабинете врача.

### 10.2. Тестирование

```go
// Тестировать ВСЕ бизнес-правила:
func TestAvailabilityCalculator_WorkingHours(t *testing.T)
func TestAvailabilityCalculator_Exceptions(t *testing.T)
func TestAvailabilityCalculator_ExistingAppointments(t *testing.T)
func TestAppointmentCreation_NoOverlap(t *testing.T)
func TestAppointmentCreation_DoctorNotActive(t *testing.T)
func TestAppointmentCreation_ServiceNotActive(t *testing.T)
```

### 10.3. Документация

Всегда документируй:
- Как запустить локально (docker-compose)
- Как запустить миграции
- Как создать тестовых данных
- API в Swagger

### 10.4. Git strategy

```
main (production)
├── develop (staging)
    ├── feature/telegram-bot
    ├── feature/admin-panel
    ├── feature/doctor-panel
    ├── bugfix/availability-calc
```

---

## Резюме

Это полное ТЗ для MVP системы управления записью пациентов в клинику.

**Ключевые моменты:**
1. Backend-first подход (все мессенджеры идут в Backend API)
2. PostgreSQL как single source of truth
3. Защита от двойной записи на 2 уровнях (backend + БД)
4. Правильная архитектура для добавления других мессенджеров
5. Мягкое удаление где нужно
6. Правильные индексы и constraints с дня 1
7. Тесты с самого начала
8. Никакой magic, все explicit

**Готово к разработке!**
