package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Hacieva/clinic-scheduler/bot/internal/client"
)

const testSecret = "test-bot-secret"

func newTestClient(srv *httptest.Server) *client.Client {
	return client.New(srv.URL, testSecret)
}

// tokenHandler returns a handler that checks X-Bot-Token and responds with the given status + body.
func tokenHandler(status int, body any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Bot-Token") != testSecret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(status)
		if body != nil {
			json.NewEncoder(w).Encode(body)
		}
	})
}

func errorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("want error %v, got %v", target, err)
	}
}

// — GetDirections —

func TestGetDirections_Success(t *testing.T) {
	want := []client.Direction{{ID: 1, Name: "Кардиология"}, {ID: 2, Name: "Неврология"}}
	srv := httptest.NewServer(tokenHandler(http.StatusOK, want))
	defer srv.Close()

	got, err := newTestClient(srv).GetDirections(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[1].Name != "Неврология" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestGetDirections_SendsBotToken(t *testing.T) {
	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Bot-Token")
		json.NewEncoder(w).Encode([]client.Direction{})
	}))
	defer srv.Close()

	client.New(srv.URL, testSecret).GetDirections(context.Background())
	if gotToken != testSecret {
		t.Errorf("X-Bot-Token: want %q, got %q", testSecret, gotToken)
	}
}

// — GetDoctors —

func TestGetDoctors_PassesDirectionIDParam(t *testing.T) {
	var gotParams url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParams = r.URL.Query()
		json.NewEncoder(w).Encode([]client.Doctor{})
	}))
	defer srv.Close()

	client.New(srv.URL, testSecret).GetDoctors(context.Background(), 42)
	if gotParams.Get("direction_id") != "42" {
		t.Errorf("direction_id param: want %q, got %q", "42", gotParams.Get("direction_id"))
	}
}

// — GetServicesByDoctor —

func TestGetServicesByDoctor_UsesCorrectPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode([]client.Service{})
	}))
	defer srv.Close()

	client.New(srv.URL, testSecret).GetServicesByDoctor(context.Background(), 7)
	const want = "/api/v1/bot/doctors/7/services"
	if gotPath != want {
		t.Errorf("path: want %q, got %q", want, gotPath)
	}
}

// — GetAvailability —

func TestGetAvailability_Success(t *testing.T) {
	want := client.AvailabilityResponse{
		DoctorID:               1,
		ServiceID:              2,
		ServiceDurationMinutes: 30,
		Availability: []client.AvailabilityDay{
			{Date: "2026-05-25", Slots: []string{"10:00", "10:30"}},
		},
	}
	srv := httptest.NewServer(tokenHandler(http.StatusOK, want))
	defer srv.Close()

	got, err := newTestClient(srv).GetAvailability(context.Background(), 1, 2, "2026-05-25", "2026-05-25")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Availability) != 1 || len(got.Availability[0].Slots) != 2 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestGetAvailability_PassesAllParams(t *testing.T) {
	var gotParams url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParams = r.URL.Query()
		json.NewEncoder(w).Encode(client.AvailabilityResponse{})
	}))
	defer srv.Close()

	client.New(srv.URL, testSecret).GetAvailability(context.Background(), 3, 7, "2026-05-20", "2026-05-27")
	checks := map[string]string{
		"doctor_id":  "3",
		"service_id": "7",
		"date_from":  "2026-05-20",
		"date_to":    "2026-05-27",
	}
	for k, v := range checks {
		if gotParams.Get(k) != v {
			t.Errorf("param %q: want %q, got %q", k, v, gotParams.Get(k))
		}
	}
}

// — CreateAppointment —

func TestCreateAppointment_Success(t *testing.T) {
	want := client.AppointmentResult{ID: 101, StartAt: "2026-05-25T10:00:00Z", EndAt: "2026-05-25T10:30:00Z"}
	srv := httptest.NewServer(tokenHandler(http.StatusCreated, want))
	defer srv.Close()

	got, err := newTestClient(srv).CreateAppointment(context.Background(), client.CreateAppointmentInput{
		PatientName: "Иванов Иван", PatientPhone: "+79001234567",
		DoctorID: 1, ServiceID: 2, StartAt: "2026-05-25T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 101 {
		t.Errorf("ID: want 101, got %d", got.ID)
	}
}

func TestCreateAppointment_SendsJSONBody(t *testing.T) {
	var decoded client.CreateAppointmentInput
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&decoded)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(client.AppointmentResult{ID: 1})
	}))
	defer srv.Close()

	telegramID := int64(777)
	client.New(srv.URL, testSecret).CreateAppointment(context.Background(), client.CreateAppointmentInput{
		PatientTelegramID: &telegramID,
		PatientName:       "Петров Пётр",
		PatientPhone:      "+79001111111",
		DoctorID:          5,
		ServiceID:         10,
		StartAt:           "2026-05-25T10:00:00Z",
	})
	if decoded.DoctorID != 5 || decoded.ServiceID != 10 {
		t.Errorf("body not sent correctly: %+v", decoded)
	}
	if decoded.PatientTelegramID == nil || *decoded.PatientTelegramID != telegramID {
		t.Errorf("PatientTelegramID not sent: %v", decoded.PatientTelegramID)
	}
}

// — CancelAppointment —

func TestCancelAppointment_Success(t *testing.T) {
	srv := httptest.NewServer(tokenHandler(http.StatusNoContent, nil))
	defer srv.Close()

	if err := newTestClient(srv).CancelAppointment(context.Background(), 99); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancelAppointment_UsesCorrectPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client.New(srv.URL, testSecret).CancelAppointment(context.Background(), 55)
	const want = "/api/v1/bot/appointments/55/cancel"
	if gotPath != want {
		t.Errorf("path: want %q, got %q", want, gotPath)
	}
}

// — Error mapping —

func TestStatus409_ErrSlotTaken(t *testing.T) {
	srv := httptest.NewServer(tokenHandler(http.StatusConflict, nil))
	defer srv.Close()

	_, err := newTestClient(srv).CreateAppointment(context.Background(), client.CreateAppointmentInput{})
	errorIs(t, err, client.ErrSlotTaken)
}

func TestStatus401_ErrUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad token", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := newTestClient(srv).GetDirections(context.Background())
	errorIs(t, err, client.ErrUnauthorized)
}

func TestStatus403_ErrUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := newTestClient(srv).GetDirections(context.Background())
	errorIs(t, err, client.ErrUnauthorized)
}

func TestStatus500_ErrTemporary(t *testing.T) {
	srv := httptest.NewServer(tokenHandler(http.StatusInternalServerError, nil))
	defer srv.Close()

	_, err := newTestClient(srv).GetDirections(context.Background())
	errorIs(t, err, client.ErrTemporary)
}

func TestStatus404_ErrNotFound(t *testing.T) {
	srv := httptest.NewServer(tokenHandler(http.StatusNotFound, nil))
	defer srv.Close()

	_, err := newTestClient(srv).GetServicesByDoctor(context.Background(), 1)
	errorIs(t, err, client.ErrNotFound)
}

func TestNetworkError_ErrTemporary(t *testing.T) {
	// Bind a port then close it so connections are refused immediately.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	_, err = client.New("http://"+addr, testSecret).GetDirections(context.Background())
	errorIs(t, err, client.ErrTemporary)
}

func TestMalformedJSON_ErrTemporary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{not valid json"))
	}))
	defer srv.Close()

	_, err := newTestClient(srv).GetDirections(context.Background())
	errorIs(t, err, client.ErrTemporary)
}
