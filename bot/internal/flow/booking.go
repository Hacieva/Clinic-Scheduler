package flow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Hacieva/clinic-scheduler/bot/internal/client"
	"github.com/Hacieva/clinic-scheduler/bot/internal/keyboard"
	"github.com/Hacieva/clinic-scheduler/bot/internal/session"
)

const availabilityDaysAhead = 14

// Sender abstracts Telegram message delivery so the FSM stays library-independent.
// The handler layer (7.6) implements this using the chosen bot library.
type Sender interface {
	SendText(ctx context.Context, chatID int64, text string) error
	SendKeyboard(ctx context.Context, chatID int64, text string, buttons [][]keyboard.Button) error
	AnswerCallback(ctx context.Context, callbackID string) error
}

// APIClient abstracts backend calls so the FSM can be unit-tested without HTTP.
// *client.Client satisfies this interface.
type APIClient interface {
	GetDirections(ctx context.Context) ([]client.Direction, error)
	GetDoctors(ctx context.Context, directionID int64) ([]client.Doctor, error)
	GetServicesByDoctor(ctx context.Context, doctorID int64) ([]client.Service, error)
	GetAvailability(ctx context.Context, doctorID, serviceID int64, dateFrom, dateTo string) (*client.AvailabilityResponse, error)
	CreateAppointment(ctx context.Context, input client.CreateAppointmentInput) (*client.AppointmentResult, error)
	CancelAppointment(ctx context.Context, id int64) error
}

// Update is a normalised Telegram update.
// The handler layer populates this from the bot-library-specific type.
type Update struct {
	UserID           int64
	ChatID           int64
	Text             string  // non-empty for plain text messages
	Command          string  // non-empty for /commands (without the slash)
	CallbackData     string  // non-empty for inline button presses
	CallbackID       string  // must be answered to dismiss the button spinner
	TelegramUsername *string // may be nil
}

// Handler is the single entry point for all Telegram updates in the booking flow.
type Handler struct {
	sessions session.Store
	api      APIClient
	sender   Sender
}

func NewHandler(sessions session.Store, api APIClient, sender Sender) *Handler {
	return &Handler{sessions: sessions, api: api, sender: sender}
}

// Handle dispatches one Telegram update through the FSM.
// All state transitions are explicit and happen only inside the named state handlers.
func (h *Handler) Handle(ctx context.Context, u Update) {
	// — Global /cancel (command or callback) — clears session and returns to start —
	if u.Command == "cancel" || u.CallbackData == CallbackCancel {
		_ = h.sessions.Delete(ctx, u.UserID)
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID,
			"Запись отменена. Используйте /start для новой записи.")
		return
	}

	// — /help — informational, does not touch session —
	if u.Command == "help" {
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID, msgHelp)
		return
	}

	// — /start always resets the flow regardless of current state —
	if u.Command == "start" {
		h.enterStart(ctx, u)
		return
	}

	// — Load session —
	sess, err := h.sessions.Get(ctx, u.UserID)
	if err != nil {
		slog.Error("session.Get failed", "user_id", u.UserID, "err", err)
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
		return
	}
	if sess == nil {
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID, "Используйте /start для начала записи.")
		return
	}

	// — Dispatch by state —
	switch sess.State {
	case StateChooseDirection:
		h.handleChooseDirection(ctx, u, sess)
	case StateChooseDoctor:
		h.handleChooseDoctor(ctx, u, sess)
	case StateChooseService:
		h.handleChooseService(ctx, u, sess)
	case StateChooseDate:
		h.handleChooseDate(ctx, u, sess)
	case StateChooseTime:
		h.handleChooseTime(ctx, u, sess)
	case StateEnterName:
		h.handleEnterName(ctx, u, sess)
	case StateEnterPhone:
		h.handleEnterPhone(ctx, u, sess)
	case StateConfirm:
		h.handleConfirm(ctx, u, sess)
	default:
		// Unknown persisted state: reset to start
		slog.Warn("unknown FSM state, resetting", "state", sess.State, "user_id", u.UserID)
		h.enterStart(ctx, u)
	}
}

// — State entry: Start —

func (h *Handler) enterStart(ctx context.Context, u Update) {
	dirs, err := h.api.GetDirections(ctx)
	if err != nil {
		h.answerIfNeeded(ctx, u)
		h.handleAPIError(ctx, u, err)
		return
	}
	if len(dirs) == 0 {
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID,
			"Нет доступных направлений. Обратитесь к администратору.")
		return
	}
	if err := h.sessions.Replace(ctx, u.UserID, &session.Data{State: StateChooseDirection}); err != nil {
		slog.Error("session.Replace failed", "user_id", u.UserID, "err", err)
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
		return
	}
	h.answerIfNeeded(ctx, u)
	_ = h.sender.SendKeyboard(ctx, u.ChatID, "Выберите направление:", keyboard.Directions(dirs))
}

// — State handler: ChooseDirection —

func (h *Handler) handleChooseDirection(ctx context.Context, u Update, sess *session.Data) {
	dirID, ok := parseIDCallback(u.CallbackData, CallbackPrefixDirection)
	if !ok {
		h.noOp(ctx, u)
		return
	}
	// Re-fetch directions to resolve the display name for the confirmation screen.
	dirs, err := h.api.GetDirections(ctx)
	if err != nil {
		h.answerIfNeeded(ctx, u)
		h.handleAPIError(ctx, u, err)
		return
	}
	dirName := nameFromDirections(dirs, dirID)

	docs, err := h.api.GetDoctors(ctx, dirID)
	if err != nil {
		h.answerIfNeeded(ctx, u)
		h.handleAPIError(ctx, u, err)
		return
	}
	if len(docs) == 0 {
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID,
			"Нет доступных врачей для этого направления. Выберите другое направление.")
		return
	}

	sess.State = StateChooseDoctor
	sess.DirectionID = &dirID
	sess.DirectionName = dirName
	clearFromDoctor(sess)

	if err := h.sessions.Replace(ctx, u.UserID, sess); err != nil {
		slog.Error("session.Replace failed", "user_id", u.UserID, "err", err)
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
		return
	}
	h.answerIfNeeded(ctx, u)
	_ = h.sender.SendKeyboard(ctx, u.ChatID, "Выберите врача:", keyboard.Doctors(docs))
}

// — State handler: ChooseDoctor —

func (h *Handler) handleChooseDoctor(ctx context.Context, u Update, sess *session.Data) {
	docID, ok := parseIDCallback(u.CallbackData, CallbackPrefixDoctor)
	if !ok {
		h.noOp(ctx, u)
		return
	}
	var dirID int64
	if sess.DirectionID != nil {
		dirID = *sess.DirectionID
	}
	// Re-fetch doctors to resolve the display name.
	docs, err := h.api.GetDoctors(ctx, dirID)
	if err != nil {
		h.answerIfNeeded(ctx, u)
		h.handleAPIError(ctx, u, err)
		return
	}
	docName := nameFromDoctors(docs, docID)

	svcs, err := h.api.GetServicesByDoctor(ctx, docID)
	if err != nil {
		h.answerIfNeeded(ctx, u)
		h.handleAPIError(ctx, u, err)
		return
	}
	if len(svcs) == 0 {
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID,
			"У этого врача нет доступных услуг. Выберите другого врача.")
		return
	}

	sess.State = StateChooseService
	sess.DoctorID = &docID
	sess.DoctorName = docName
	clearFromService(sess)

	if err := h.sessions.Replace(ctx, u.UserID, sess); err != nil {
		slog.Error("session.Replace failed", "user_id", u.UserID, "err", err)
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
		return
	}
	h.answerIfNeeded(ctx, u)
	_ = h.sender.SendKeyboard(ctx, u.ChatID, "Выберите услугу:", keyboard.Services(svcs))
}

// — State handler: ChooseService —

func (h *Handler) handleChooseService(ctx context.Context, u Update, sess *session.Data) {
	svcID, ok := parseIDCallback(u.CallbackData, CallbackPrefixService)
	if !ok {
		h.noOp(ctx, u)
		return
	}
	var docID int64
	if sess.DoctorID != nil {
		docID = *sess.DoctorID
	}
	// Re-fetch services to resolve name and price.
	svcs, err := h.api.GetServicesByDoctor(ctx, docID)
	if err != nil {
		h.answerIfNeeded(ctx, u)
		h.handleAPIError(ctx, u, err)
		return
	}
	svcName, svcPrice := nameAndPriceFromServices(svcs, svcID)

	dateFrom, dateTo := availabilityRange()
	avail, err := h.api.GetAvailability(ctx, docID, svcID, dateFrom, dateTo)
	if err != nil {
		h.answerIfNeeded(ctx, u)
		h.handleAPIError(ctx, u, err)
		return
	}
	if avail == nil || len(avail.Availability) == 0 {
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID,
			"Нет доступных дней в ближайшие 2 недели. Выберите другую услугу или врача.")
		return
	}
	datesKb := keyboard.Dates(avail.Availability)
	if len(datesKb) == 0 {
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID,
			"Нет доступных дней в ближайшие 2 недели. Выберите другую услугу или врача.")
		return
	}

	sess.State = StateChooseDate
	sess.ServiceID = &svcID
	sess.ServiceName = svcName
	sess.ServicePrice = svcPrice
	sess.Date = ""
	sess.Time = ""
	sess.PatientName = ""
	sess.PatientPhone = ""

	if err := h.sessions.Replace(ctx, u.UserID, sess); err != nil {
		slog.Error("session.Replace failed", "user_id", u.UserID, "err", err)
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
		return
	}
	h.answerIfNeeded(ctx, u)
	_ = h.sender.SendKeyboard(ctx, u.ChatID, "Выберите дату:", datesKb)
}

// — State handler: ChooseDate —

func (h *Handler) handleChooseDate(ctx context.Context, u Update, sess *session.Data) {
	date, ok := parseStringCallback(u.CallbackData, CallbackPrefixDate)
	if !ok {
		h.noOp(ctx, u)
		return
	}
	var docID, svcID int64
	if sess.DoctorID != nil {
		docID = *sess.DoctorID
	}
	if sess.ServiceID != nil {
		svcID = *sess.ServiceID
	}
	// Fetch slots for the selected date only.
	avail, err := h.api.GetAvailability(ctx, docID, svcID, date, date)
	if err != nil {
		h.answerIfNeeded(ctx, u)
		h.handleAPIError(ctx, u, err)
		return
	}
	var slots []string
	if avail != nil {
		for _, day := range avail.Availability {
			if day.Date == date {
				slots = day.Slots
				break
			}
		}
	}
	if len(slots) == 0 {
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID,
			"На выбранную дату нет свободных слотов. Пожалуйста, выберите другую дату.")
		return
	}

	sess.State = StateChooseTime
	sess.Date = date
	sess.Time = ""
	sess.PatientName = ""
	sess.PatientPhone = ""

	if err := h.sessions.Replace(ctx, u.UserID, sess); err != nil {
		slog.Error("session.Replace failed", "user_id", u.UserID, "err", err)
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
		return
	}
	h.answerIfNeeded(ctx, u)
	_ = h.sender.SendKeyboard(ctx, u.ChatID,
		fmt.Sprintf("Выберите время на %s:", keyboard.FormatDate(date)),
		keyboard.Times(slots))
}

// — State handler: ChooseTime —

func (h *Handler) handleChooseTime(ctx context.Context, u Update, sess *session.Data) {
	t, ok := parseStringCallback(u.CallbackData, CallbackPrefixTime)
	if !ok {
		h.noOp(ctx, u)
		return
	}

	sess.State = StateEnterName
	sess.Time = t

	if err := h.sessions.Replace(ctx, u.UserID, sess); err != nil {
		slog.Error("session.Replace failed", "user_id", u.UserID, "err", err)
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
		return
	}
	h.answerIfNeeded(ctx, u)
	_ = h.sender.SendText(ctx, u.ChatID, "Введите ваше имя:")
}

// — State handler: EnterName (text input) —

func (h *Handler) handleEnterName(ctx context.Context, u Update, sess *session.Data) {
	if u.Text == "" {
		// Callback or command in a text-input state → no-op.
		h.noOp(ctx, u)
		return
	}
	name := strings.TrimSpace(u.Text)
	if name == "" {
		_ = h.sender.SendText(ctx, u.ChatID, "Имя не может быть пустым. Введите ваше имя:")
		return
	}

	sess.State = StateEnterPhone
	sess.PatientName = name

	if err := h.sessions.Replace(ctx, u.UserID, sess); err != nil {
		slog.Error("session.Replace failed", "user_id", u.UserID, "err", err)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
		return
	}
	_ = h.sender.SendText(ctx, u.ChatID, "Введите ваш номер телефона:")
}

// — State handler: EnterPhone (text input) —

func (h *Handler) handleEnterPhone(ctx context.Context, u Update, sess *session.Data) {
	if u.Text == "" {
		h.noOp(ctx, u)
		return
	}
	phone := strings.TrimSpace(u.Text)
	if phone == "" {
		_ = h.sender.SendText(ctx, u.ChatID, "Телефон не может быть пустым. Введите номер телефона:")
		return
	}

	sess.State = StateConfirm
	sess.PatientPhone = phone

	if err := h.sessions.Replace(ctx, u.UserID, sess); err != nil {
		slog.Error("session.Replace failed", "user_id", u.UserID, "err", err)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
		return
	}
	_ = h.sender.SendKeyboard(ctx, u.ChatID, buildConfirmText(sess), keyboard.Confirm())
}

// — State handler: Confirm —

func (h *Handler) handleConfirm(ctx context.Context, u Update, sess *session.Data) {
	if u.CallbackData != CallbackConfirm {
		h.noOp(ctx, u)
		return
	}
	if sess.DoctorID == nil || sess.ServiceID == nil || sess.Date == "" || sess.Time == "" {
		h.answerIfNeeded(ctx, u)
		_ = h.sender.SendText(ctx, u.ChatID,
			"Сессия повреждена. Начните заново: /start")
		_ = h.sessions.Delete(ctx, u.UserID)
		return
	}

	// RFC3339 with UTC — timezone is resolved by backend based on clinic config.
	startAt := sess.Date + "T" + sess.Time + ":00Z"
	uid := u.UserID
	result, err := h.api.CreateAppointment(ctx, client.CreateAppointmentInput{
		PatientTelegramID:       &uid,
		PatientTelegramUsername: u.TelegramUsername,
		PatientName:             sess.PatientName,
		PatientPhone:            sess.PatientPhone,
		DoctorID:                *sess.DoctorID,
		ServiceID:               *sess.ServiceID,
		StartAt:                 startAt,
	})

	if err != nil {
		h.answerIfNeeded(ctx, u)

		if errors.Is(err, client.ErrSlotTaken) {
			// Slot taken: clear time, transition back to StateChooseTime.
			// Session is NOT fully reset — user keeps direction/doctor/service/date.
			sess.Time = ""
			sess.State = StateChooseTime
			if replErr := h.sessions.Replace(ctx, u.UserID, sess); replErr != nil {
				slog.Error("session.Replace after slot taken", "user_id", u.UserID, "err", replErr)
			}
			_ = h.sender.SendText(ctx, u.ChatID,
				"К сожалению, это время уже занято. Пожалуйста, выберите другое время.")

			// Re-fetch time slots for the same date.
			avail, avErr := h.api.GetAvailability(ctx,
				*sess.DoctorID, *sess.ServiceID, sess.Date, sess.Date)
			if avErr != nil {
				h.handleAPIError(ctx, u, avErr)
				return
			}
			var slots []string
			if avail != nil {
				for _, day := range avail.Availability {
					if day.Date == sess.Date {
						slots = day.Slots
						break
					}
				}
			}
			if len(slots) == 0 {
				_ = h.sender.SendText(ctx, u.ChatID,
					"На этот день больше нет свободных слотов. Начните заново: /start")
				return
			}
			_ = h.sender.SendKeyboard(ctx, u.ChatID,
				fmt.Sprintf("Выберите время на %s:", keyboard.FormatDate(sess.Date)),
				keyboard.Times(slots))
			return
		}

		h.handleAPIError(ctx, u, err)
		return
	}

	// Success — clear session.
	_ = h.sessions.Delete(ctx, u.UserID)
	_ = h.sender.SendText(ctx, u.ChatID, fmt.Sprintf(
		"✅ Запись создана!\n\nВрач: %s\nУслуга: %s\nДата: %s\nВремя: %s\n\nНомер записи: #%d",
		sess.DoctorName, sess.ServiceName,
		keyboard.FormatDate(sess.Date), sess.Time,
		result.ID,
	))
}

// — Helpers —

func (h *Handler) noOp(ctx context.Context, u Update) {
	h.answerIfNeeded(ctx, u)
}

func (h *Handler) answerIfNeeded(ctx context.Context, u Update) {
	if u.CallbackID != "" {
		_ = h.sender.AnswerCallback(ctx, u.CallbackID)
	}
}

func (h *Handler) handleAPIError(ctx context.Context, u Update, err error) {
	switch {
	case errors.Is(err, client.ErrUnauthorized):
		slog.Error("bot auth rejected by backend", "user_id", u.UserID)
		_ = h.sender.SendText(ctx, u.ChatID, msgTryAgain)
	case errors.Is(err, client.ErrNotFound):
		_ = h.sender.SendText(ctx, u.ChatID,
			"Запрошенный ресурс не найден. Начните заново: /start")
	default:
		_ = h.sender.SendText(ctx, u.ChatID,
			"Сервис временно недоступен. Попробуйте позже.")
	}
}

// parseIDCallback extracts a positive int64 from "prefix:N" callback data.
func parseIDCallback(data, prefix string) (int64, bool) {
	if !strings.HasPrefix(data, prefix+":") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(data, prefix+":"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// parseStringCallback extracts a non-empty string from "prefix:value" callback data.
func parseStringCallback(data, prefix string) (string, bool) {
	if !strings.HasPrefix(data, prefix+":") {
		return "", false
	}
	val := strings.TrimPrefix(data, prefix+":")
	if val == "" {
		return "", false
	}
	return val, true
}

func buildConfirmText(sess *session.Data) string {
	price := ""
	if sess.ServicePrice != nil && *sess.ServicePrice > 0 {
		price = fmt.Sprintf("\nСтоимость: %d ₽", *sess.ServicePrice/100)
	}
	return fmt.Sprintf(
		"📋 Ваша запись:\n\nНаправление: %s\nВрач: %s\nУслуга: %s%s\nДата: %s\nВремя: %s\nИмя: %s\nТелефон: %s\n\nПодтвердить запись?",
		sess.DirectionName, sess.DoctorName, sess.ServiceName, price,
		keyboard.FormatDate(sess.Date), sess.Time,
		sess.PatientName, sess.PatientPhone,
	)
}

// availabilityRange returns today and today+availabilityDaysAhead in YYYY-MM-DD.
func availabilityRange() (string, string) {
	now := time.Now()
	from := now.Format("2006-01-02")
	to := now.AddDate(0, 0, availabilityDaysAhead).Format("2006-01-02")
	return from, to
}

// clearFromDoctor zeroes all fields that depend on doctor selection.
func clearFromDoctor(sess *session.Data) {
	sess.DoctorID = nil
	sess.DoctorName = ""
	sess.ServiceID = nil
	sess.ServiceName = ""
	sess.ServicePrice = nil
	sess.Date = ""
	sess.Time = ""
	sess.PatientName = ""
	sess.PatientPhone = ""
}

// clearFromService zeroes all fields that depend on service selection.
func clearFromService(sess *session.Data) {
	sess.ServiceID = nil
	sess.ServiceName = ""
	sess.ServicePrice = nil
	sess.Date = ""
	sess.Time = ""
	sess.PatientName = ""
	sess.PatientPhone = ""
}

func nameFromDirections(dirs []client.Direction, id int64) string {
	for _, d := range dirs {
		if d.ID == id {
			return d.Name
		}
	}
	return ""
}

func nameFromDoctors(docs []client.Doctor, id int64) string {
	for _, d := range docs {
		if d.ID == id {
			return keyboard.Abbreviate(d.LastName, d.FirstName, d.MiddleName)
		}
	}
	return ""
}

func nameAndPriceFromServices(svcs []client.Service, id int64) (string, *int64) {
	for _, s := range svcs {
		if s.ID == id {
			return s.Name, s.Price
		}
	}
	return "", nil
}

const msgTryAgain = "Что-то пошло не так. Попробуйте позже или начните заново: /start"

const msgHelp = "Этот бот помогает записаться на приём к врачу.\n\nКоманды:\n/start — начать запись\n/cancel — отменить текущую запись\n/help — эта справка"
