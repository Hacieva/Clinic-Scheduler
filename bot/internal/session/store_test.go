package session_test

import (
	"encoding/json"
	"testing"

	"github.com/Hacieva/clinic-scheduler/bot/internal/session"
)

func TestData_JSONRoundtrip(t *testing.T) {
	price := int64(300000)
	dirID := int64(1)
	docID := int64(5)
	svcID := int64(10)

	d := session.Data{
		State:         "confirm",
		DirectionID:   &dirID,
		DirectionName: "Кардиология",
		DoctorID:      &docID,
		DoctorName:    "Иванов И.И.",
		ServiceID:     &svcID,
		ServiceName:   "Первичная консультация",
		ServicePrice:  &price,
		Date:          "2026-05-25",
		Time:          "10:00",
		PatientName:   "Петров Пётр",
		PatientPhone:  "+79001234567",
	}

	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got session.Data
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.State != d.State {
		t.Errorf("State: want %q, got %q", d.State, got.State)
	}
	if got.DirectionID == nil || *got.DirectionID != *d.DirectionID {
		t.Errorf("DirectionID mismatch")
	}
	if got.ServicePrice == nil || *got.ServicePrice != *d.ServicePrice {
		t.Errorf("ServicePrice: want %d, got %v", *d.ServicePrice, got.ServicePrice)
	}
	if got.PatientName != d.PatientName {
		t.Errorf("PatientName: want %q, got %q", d.PatientName, got.PatientName)
	}
	if got.PatientPhone != d.PatientPhone {
		t.Errorf("PatientPhone: want %q, got %q", d.PatientPhone, got.PatientPhone)
	}
	if got.Date != d.Date {
		t.Errorf("Date: want %q, got %q", d.Date, got.Date)
	}
	if got.Time != d.Time {
		t.Errorf("Time: want %q, got %q", d.Time, got.Time)
	}
}

func TestData_NilPtrFieldsOmittedFromJSON(t *testing.T) {
	d := session.Data{State: "start"}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	for _, key := range []string{"direction_id", "doctor_id", "service_id", "service_price"} {
		if _, ok := m[key]; ok {
			t.Errorf("key %q should be omitted when nil, but was present in JSON", key)
		}
	}
}

func TestData_ServicePrice_Int64(t *testing.T) {
	// Verify kopecks can hold large values without overflow (int32 max = ~21M kopecks = 210k RUB)
	price := int64(1_000_000_00) // 1 000 000 RUB in kopecks
	d := session.Data{State: "confirm", ServicePrice: &price}
	raw, _ := json.Marshal(d)
	var got session.Data
	json.Unmarshal(raw, &got)
	if got.ServicePrice == nil || *got.ServicePrice != price {
		t.Errorf("ServicePrice int64 overflow: want %d, got %v", price, got.ServicePrice)
	}
}
