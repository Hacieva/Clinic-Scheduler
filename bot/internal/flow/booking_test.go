package flow

import (
	"context"
	"strings"
	"testing"

	"github.com/Hacieva/clinic-scheduler/bot/internal/client"
	"github.com/Hacieva/clinic-scheduler/bot/internal/keyboard"
	"github.com/Hacieva/clinic-scheduler/bot/internal/session"
)

// — test doubles —

type mockSender struct {
	texts     []string
	keyboards []kbCapture
}

type kbCapture struct {
	text    string
	buttons [][]keyboard.Button
}

func (m *mockSender) SendText(_ context.Context, _ int64, text string) error {
	m.texts = append(m.texts, text)
	return nil
}
func (m *mockSender) SendKeyboard(_ context.Context, _ int64, text string, buttons [][]keyboard.Button) error {
	m.keyboards = append(m.keyboards, kbCapture{text, buttons})
	return nil
}
func (m *mockSender) AnswerCallback(_ context.Context, _ string) error { return nil }

type mockSession struct {
	data *session.Data
	err  error
}

func (m *mockSession) Get(_ context.Context, _ int64) (*session.Data, error) {
	return m.data, m.err
}
func (m *mockSession) Replace(_ context.Context, _ int64, data *session.Data) error {
	if m.err != nil {
		return m.err
	}
	m.data = data
	return nil
}
func (m *mockSession) Delete(_ context.Context, _ int64) error {
	m.data = nil
	return m.err
}

type mockAPI struct {
	directions   []client.Direction
	doctors      []client.Doctor
	services     []client.Service
	availability *client.AvailabilityResponse
	appointment  *client.AppointmentResult
	err          error
}

func (m *mockAPI) GetDirections(_ context.Context) ([]client.Direction, error) {
	return m.directions, m.err
}
func (m *mockAPI) GetDoctors(_ context.Context, _ int64) ([]client.Doctor, error) {
	return m.doctors, m.err
}
func (m *mockAPI) GetServicesByDoctor(_ context.Context, _ int64) ([]client.Service, error) {
	return m.services, m.err
}
func (m *mockAPI) GetAvailability(_ context.Context, _, _ int64, _, _ string) (*client.AvailabilityResponse, error) {
	return m.availability, m.err
}
func (m *mockAPI) CreateAppointment(_ context.Context, _ client.CreateAppointmentInput) (*client.AppointmentResult, error) {
	return m.appointment, m.err
}
func (m *mockAPI) CancelAppointment(_ context.Context, _ int64) error { return m.err }

// slotTakenAPI returns ErrSlotTaken only for CreateAppointment; GetAvailability succeeds.
type slotTakenAPI struct {
	avail *client.AvailabilityResponse
}

func (m *slotTakenAPI) GetDirections(_ context.Context) ([]client.Direction, error)  { return nil, nil }
func (m *slotTakenAPI) GetDoctors(_ context.Context, _ int64) ([]client.Doctor, error) {
	return nil, nil
}
func (m *slotTakenAPI) GetServicesByDoctor(_ context.Context, _ int64) ([]client.Service, error) {
	return nil, nil
}
func (m *slotTakenAPI) GetAvailability(_ context.Context, _, _ int64, _, _ string) (*client.AvailabilityResponse, error) {
	return m.avail, nil
}
func (m *slotTakenAPI) CreateAppointment(_ context.Context, _ client.CreateAppointmentInput) (*client.AppointmentResult, error) {
	return nil, client.ErrSlotTaken
}
func (m *slotTakenAPI) CancelAppointment(_ context.Context, _ int64) error { return nil }

// — helpers —

func newHandler(sess *mockSession, api APIClient) (*Handler, *mockSender) {
	s := &mockSender{}
	return NewHandler(sess, api, s), s
}

func cmdUpdate(command string) Update {
	return Update{UserID: 1, ChatID: 100, Command: command}
}

func cbUpdate(data string) Update {
	return Update{UserID: 1, ChatID: 100, CallbackData: data, CallbackID: "cb1"}
}

func textUpdate(text string) Update {
	return Update{UserID: 1, ChatID: 100, Text: text}
}

// — tests —

func TestHandle_Start_ShowsDirectionKeyboard(t *testing.T) {
	sess := &mockSession{}
	api := &mockAPI{directions: []client.Direction{{ID: 1, Name: "Кардиология"}}}
	h, s := newHandler(sess, api)

	h.Handle(context.Background(), cmdUpdate("start"))

	if len(s.keyboards) != 1 {
		t.Fatalf("want 1 keyboard, got %d", len(s.keyboards))
	}
	if !strings.Contains(s.keyboards[0].text, "направление") {
		t.Errorf("keyboard prompt missing 'направление': %q", s.keyboards[0].text)
	}
	if sess.data == nil || sess.data.State != StateChooseDirection {
		t.Errorf("want state %q, got %v", StateChooseDirection, sess.data)
	}
}

func TestHandle_Start_NoDirections_SendsText(t *testing.T) {
	sess := &mockSession{}
	api := &mockAPI{directions: []client.Direction{}}
	h, s := newHandler(sess, api)

	h.Handle(context.Background(), cmdUpdate("start"))

	if len(s.keyboards) != 0 {
		t.Errorf("expected no keyboard, got %d", len(s.keyboards))
	}
	if len(s.texts) == 0 || !strings.Contains(s.texts[0], "администратор") {
		t.Errorf("expected admin message, got %v", s.texts)
	}
}

func TestHandle_ChooseDirection_TransitionsToChooseDoctor(t *testing.T) {
	sess := &mockSession{data: &session.Data{State: StateChooseDirection}}
	api := &mockAPI{
		directions: []client.Direction{{ID: 1, Name: "Кардиология"}},
		doctors:    []client.Doctor{{ID: 2, LastName: "Иванов", FirstName: "А"}},
	}
	h, s := newHandler(sess, api)

	h.Handle(context.Background(), cbUpdate("direction:1"))

	if len(s.keyboards) != 1 {
		t.Fatalf("want 1 keyboard, got %d", len(s.keyboards))
	}
	if !strings.Contains(s.keyboards[0].text, "врача") {
		t.Errorf("want doctor prompt, got %q", s.keyboards[0].text)
	}
	if sess.data.State != StateChooseDoctor {
		t.Errorf("want state %q, got %q", StateChooseDoctor, sess.data.State)
	}
	if sess.data.DirectionID == nil || *sess.data.DirectionID != 1 {
		t.Errorf("want DirectionID=1, got %v", sess.data.DirectionID)
	}
	if sess.data.DirectionName != "Кардиология" {
		t.Errorf("want DirectionName=Кардиология, got %q", sess.data.DirectionName)
	}
}

func TestHandle_UnknownCallbackInChooseDirection_NoStateChange(t *testing.T) {
	original := &session.Data{State: StateChooseDirection}
	sess := &mockSession{data: original}
	h, s := newHandler(sess, &mockAPI{})

	h.Handle(context.Background(), cbUpdate("doctor:99")) // wrong prefix for this state

	if len(s.texts) != 0 || len(s.keyboards) != 0 {
		t.Errorf("no-op expected: texts=%v keyboards=%v", s.texts, s.keyboards)
	}
	if sess.data.State != StateChooseDirection {
		t.Errorf("state must not change on unknown callback, got %q", sess.data.State)
	}
}

func TestHandle_ChooseDoctor_TransitionsToChooseService(t *testing.T) {
	dirID := int64(1)
	sess := &mockSession{data: &session.Data{State: StateChooseDoctor, DirectionID: &dirID}}
	api := &mockAPI{
		doctors:  []client.Doctor{{ID: 2, LastName: "Иванов", FirstName: "А"}},
		services: []client.Service{{ID: 3, Name: "Консультация", DurationMinutes: 30}},
	}
	h, s := newHandler(sess, api)

	h.Handle(context.Background(), cbUpdate("doctor:2"))

	if len(s.keyboards) != 1 {
		t.Fatalf("want 1 keyboard, got %d", len(s.keyboards))
	}
	if sess.data.State != StateChooseService {
		t.Errorf("want state %q, got %q", StateChooseService, sess.data.State)
	}
	if sess.data.DoctorID == nil || *sess.data.DoctorID != 2 {
		t.Errorf("want DoctorID=2, got %v", sess.data.DoctorID)
	}
}

func TestHandle_ChooseService_TransitionsToChooseDate(t *testing.T) {
	dirID, docID := int64(1), int64(2)
	sess := &mockSession{data: &session.Data{
		State: StateChooseService, DirectionID: &dirID, DoctorID: &docID,
	}}
	api := &mockAPI{
		services: []client.Service{{ID: 3, Name: "Консультация", DurationMinutes: 30}},
		availability: &client.AvailabilityResponse{
			Availability: []client.AvailabilityDay{{Date: "2026-06-01", Slots: []string{"10:00"}}},
		},
	}
	h, s := newHandler(sess, api)

	h.Handle(context.Background(), cbUpdate("service:3"))

	if len(s.keyboards) != 1 {
		t.Fatalf("want 1 keyboard, got %d", len(s.keyboards))
	}
	if sess.data.State != StateChooseDate {
		t.Errorf("want state %q, got %q", StateChooseDate, sess.data.State)
	}
	if sess.data.ServiceID == nil || *sess.data.ServiceID != 3 {
		t.Errorf("want ServiceID=3, got %v", sess.data.ServiceID)
	}
}

func TestHandle_ChooseDate_TransitionsToChooseTime(t *testing.T) {
	docID, svcID := int64(2), int64(3)
	sess := &mockSession{data: &session.Data{
		State: StateChooseDate, DoctorID: &docID, ServiceID: &svcID,
	}}
	api := &mockAPI{
		availability: &client.AvailabilityResponse{
			Availability: []client.AvailabilityDay{
				{Date: "2026-06-01", Slots: []string{"10:00", "10:30"}},
			},
		},
	}
	h, s := newHandler(sess, api)

	h.Handle(context.Background(), cbUpdate("date:2026-06-01"))

	if len(s.keyboards) != 1 {
		t.Fatalf("want 1 keyboard, got %d", len(s.keyboards))
	}
	if sess.data.State != StateChooseTime {
		t.Errorf("want state %q, got %q", StateChooseTime, sess.data.State)
	}
	if sess.data.Date != "2026-06-01" {
		t.Errorf("want date=2026-06-01, got %q", sess.data.Date)
	}
}

func TestHandle_ChooseTime_TransitionsToEnterName(t *testing.T) {
	docID, svcID := int64(2), int64(3)
	sess := &mockSession{data: &session.Data{
		State: StateChooseTime, DoctorID: &docID, ServiceID: &svcID, Date: "2026-06-01",
	}}
	h, s := newHandler(sess, &mockAPI{})

	h.Handle(context.Background(), cbUpdate("time:10:00"))

	if len(s.texts) != 1 || !strings.Contains(s.texts[0], "имя") {
		t.Errorf("expected name prompt, got %v", s.texts)
	}
	if sess.data.State != StateEnterName {
		t.Errorf("want state %q, got %q", StateEnterName, sess.data.State)
	}
	if sess.data.Time != "10:00" {
		t.Errorf("want time=10:00, got %q", sess.data.Time)
	}
}

func TestHandle_EnterName_TransitionsToEnterPhone(t *testing.T) {
	sess := &mockSession{data: &session.Data{State: StateEnterName}}
	h, s := newHandler(sess, &mockAPI{})

	h.Handle(context.Background(), textUpdate("Иван Иванов"))

	if len(s.texts) != 1 || !strings.Contains(s.texts[0], "телефон") {
		t.Errorf("expected phone prompt, got %v", s.texts)
	}
	if sess.data.State != StateEnterPhone {
		t.Errorf("want state %q, got %q", StateEnterPhone, sess.data.State)
	}
	if sess.data.PatientName != "Иван Иванов" {
		t.Errorf("want PatientName='Иван Иванов', got %q", sess.data.PatientName)
	}
}

func TestHandle_EnterPhone_TransitionsToConfirm(t *testing.T) {
	docID, svcID := int64(2), int64(3)
	sess := &mockSession{data: &session.Data{
		State: StateEnterPhone, DoctorID: &docID, ServiceID: &svcID,
		Date: "2026-06-01", Time: "10:00", PatientName: "Иван Иванов",
	}}
	h, s := newHandler(sess, &mockAPI{})

	h.Handle(context.Background(), textUpdate("+79001234567"))

	if len(s.keyboards) != 1 {
		t.Fatalf("want 1 keyboard (confirm), got %d", len(s.keyboards))
	}
	if sess.data.State != StateConfirm {
		t.Errorf("want state %q, got %q", StateConfirm, sess.data.State)
	}
	if sess.data.PatientPhone != "+79001234567" {
		t.Errorf("want PatientPhone='+79001234567', got %q", sess.data.PatientPhone)
	}
}

func TestHandle_Confirm_Success_DeletesSession(t *testing.T) {
	docID, svcID := int64(2), int64(3)
	sess := &mockSession{data: &session.Data{
		State: StateConfirm, DoctorID: &docID, ServiceID: &svcID,
		Date: "2026-06-01", Time: "10:00",
		PatientName: "Иван Иванов", PatientPhone: "+79001234567",
	}}
	api := &mockAPI{appointment: &client.AppointmentResult{ID: 42}}
	h, s := newHandler(sess, api)

	h.Handle(context.Background(), cbUpdate("confirm"))

	if sess.data != nil {
		t.Errorf("session must be deleted after successful booking, got %+v", sess.data)
	}
	if len(s.texts) != 1 || !strings.Contains(s.texts[0], "42") {
		t.Errorf("expected success message with ID 42, got %v", s.texts)
	}
}

func TestHandle_Confirm_SlotTaken_BackToChooseTime(t *testing.T) {
	docID, svcID := int64(2), int64(3)
	sess := &mockSession{data: &session.Data{
		State: StateConfirm, DoctorID: &docID, ServiceID: &svcID,
		Date: "2026-06-01", Time: "10:00",
		PatientName: "Иван Иванов", PatientPhone: "+79001234567",
	}}
	api := &slotTakenAPI{avail: &client.AvailabilityResponse{
		Availability: []client.AvailabilityDay{
			{Date: "2026-06-01", Slots: []string{"11:00"}},
		},
	}}
	h, s := newHandler(sess, api)

	h.Handle(context.Background(), cbUpdate("confirm"))

	if sess.data == nil || sess.data.State != StateChooseTime {
		t.Errorf("want state %q, got %v", StateChooseTime, sess.data)
	}
	if sess.data.Time != "" {
		t.Errorf("time must be cleared after slot taken, got %q", sess.data.Time)
	}
	if len(s.texts) == 0 || !strings.Contains(s.texts[0], "занято") {
		t.Errorf("expected slot-taken message, got %v", s.texts)
	}
	if len(s.keyboards) != 1 {
		t.Errorf("expected time selection keyboard after slot taken, got %d", len(s.keyboards))
	}
}

func TestHandle_Cancel_Command_DeletesSession(t *testing.T) {
	sess := &mockSession{data: &session.Data{State: StateChooseDoctor}}
	h, s := newHandler(sess, &mockAPI{})

	h.Handle(context.Background(), cmdUpdate("cancel"))

	if sess.data != nil {
		t.Errorf("session must be deleted on /cancel, got %+v", sess.data)
	}
	if len(s.texts) != 1 || !strings.Contains(s.texts[0], "/start") {
		t.Errorf("expected cancel message with /start, got %v", s.texts)
	}
}

func TestHandle_CancelCallback_DeletesSession(t *testing.T) {
	sess := &mockSession{data: &session.Data{State: StateConfirm}}
	h, s := newHandler(sess, &mockAPI{})

	h.Handle(context.Background(), cbUpdate("cancel"))

	if sess.data != nil {
		t.Errorf("session must be deleted on cancel callback")
	}
	if len(s.texts) == 0 || !strings.Contains(s.texts[0], "/start") {
		t.Errorf("expected cancel message, got %v", s.texts)
	}
}

func TestHandle_NoSession_PromptsStart(t *testing.T) {
	sess := &mockSession{data: nil}
	h, s := newHandler(sess, &mockAPI{})

	h.Handle(context.Background(), textUpdate("привет"))

	if len(s.texts) != 1 || !strings.Contains(s.texts[0], "/start") {
		t.Errorf("expected /start prompt, got %v", s.texts)
	}
}

func TestHandle_APITemporaryError_SendsFriendlyMessage(t *testing.T) {
	sess := &mockSession{}
	api := &mockAPI{err: client.ErrTemporary}
	h, s := newHandler(sess, api)

	h.Handle(context.Background(), cmdUpdate("start"))

	if len(s.texts) != 1 || !strings.Contains(s.texts[0], "недоступен") {
		t.Errorf("expected temporary error message, got %v", s.texts)
	}
}

func TestHandle_UnknownCallbackInChooseTime_NoOp(t *testing.T) {
	docID, svcID := int64(2), int64(3)
	original := &session.Data{
		State: StateChooseTime, DoctorID: &docID, ServiceID: &svcID, Date: "2026-06-01",
	}
	sess := &mockSession{data: original}
	h, s := newHandler(sess, &mockAPI{})

	h.Handle(context.Background(), cbUpdate("direction:1")) // wrong prefix

	if len(s.texts) != 0 || len(s.keyboards) != 0 {
		t.Errorf("no-op expected: texts=%v keyboards=%v", s.texts, s.keyboards)
	}
	if sess.data.State != StateChooseTime {
		t.Errorf("state must not change on no-op, got %q", sess.data.State)
	}
}
