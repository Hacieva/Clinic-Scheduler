package keyboard

import (
	"strings"
	"testing"

	"github.com/Hacieva/clinic-scheduler/bot/internal/client"
)

func TestDirections_BuildsOneRowPerDirection(t *testing.T) {
	dirs := []client.Direction{{ID: 1, Name: "Кардиология"}, {ID: 2, Name: "Неврология"}}
	rows := Directions(dirs)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0][0].Text != "Кардиология" {
		t.Errorf("want %q, got %q", "Кардиология", rows[0][0].Text)
	}
	if rows[0][0].CallbackData != "direction:1" {
		t.Errorf("want direction:1, got %q", rows[0][0].CallbackData)
	}
	if rows[1][0].CallbackData != "direction:2" {
		t.Errorf("want direction:2, got %q", rows[1][0].CallbackData)
	}
}

func TestDirections_Empty(t *testing.T) {
	if len(Directions(nil)) != 0 {
		t.Error("expected empty result for nil input")
	}
}

func TestDoctors_AbbreviatesName(t *testing.T) {
	mid := "Петрович"
	docs := []client.Doctor{{ID: 5, LastName: "Иванов", FirstName: "Алексей", MiddleName: &mid}}
	rows := Doctors(docs)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0][0].Text != "Иванов А.П." {
		t.Errorf("want %q, got %q", "Иванов А.П.", rows[0][0].Text)
	}
	if rows[0][0].CallbackData != "doctor:5" {
		t.Errorf("want doctor:5, got %q", rows[0][0].CallbackData)
	}
}

func TestDoctors_NoMiddleName(t *testing.T) {
	docs := []client.Doctor{{ID: 3, LastName: "Смирнов", FirstName: "Олег"}}
	rows := Doctors(docs)
	if rows[0][0].Text != "Смирнов О." {
		t.Errorf("want %q, got %q", "Смирнов О.", rows[0][0].Text)
	}
}

func TestServices_WithPrice(t *testing.T) {
	price := int64(50000) // 500 ₽ in kopecks
	svcs := []client.Service{{ID: 1, Name: "Консультация", DurationMinutes: 30, Price: &price}}
	rows := Services(svcs)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if !strings.Contains(rows[0][0].Text, "500 ₽") {
		t.Errorf("price not in label: %q", rows[0][0].Text)
	}
	if !strings.Contains(rows[0][0].Text, "30 мин") {
		t.Errorf("duration not in label: %q", rows[0][0].Text)
	}
	if rows[0][0].CallbackData != "service:1" {
		t.Errorf("want service:1, got %q", rows[0][0].CallbackData)
	}
}

func TestServices_WithoutPrice(t *testing.T) {
	svcs := []client.Service{{ID: 2, Name: "УЗИ", DurationMinutes: 45}}
	rows := Services(svcs)
	if strings.Contains(rows[0][0].Text, "₽") {
		t.Errorf("price should be absent, got: %q", rows[0][0].Text)
	}
}

func TestDates_SkipsEmptySlotDays(t *testing.T) {
	days := []client.AvailabilityDay{
		{Date: "2026-05-25", Slots: []string{"10:00", "11:00"}},
		{Date: "2026-05-26", Slots: nil},
		{Date: "2026-05-27", Slots: []string{"14:00"}},
	}
	rows := Dates(days)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows (skip empty day), got %d", len(rows))
	}
	if rows[0][0].CallbackData != "date:2026-05-25" {
		t.Errorf("want date:2026-05-25, got %q", rows[0][0].CallbackData)
	}
	if rows[1][0].CallbackData != "date:2026-05-27" {
		t.Errorf("want date:2026-05-27, got %q", rows[1][0].CallbackData)
	}
}

func TestTimes_ThreePerRow(t *testing.T) {
	slots := []string{"09:00", "09:30", "10:00", "10:30"}
	rows := Times(slots)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if len(rows[0]) != 3 {
		t.Errorf("want 3 buttons in row 0, got %d", len(rows[0]))
	}
	if len(rows[1]) != 1 {
		t.Errorf("want 1 button in row 1, got %d", len(rows[1]))
	}
	if rows[0][0].CallbackData != "time:09:00" {
		t.Errorf("want time:09:00, got %q", rows[0][0].CallbackData)
	}
	if rows[1][0].CallbackData != "time:10:30" {
		t.Errorf("want time:10:30, got %q", rows[1][0].CallbackData)
	}
}

func TestTimes_Empty(t *testing.T) {
	if len(Times(nil)) != 0 {
		t.Error("expected empty result for nil slots")
	}
}

func TestConfirm_HasTwoButtons(t *testing.T) {
	rows := Confirm()
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0][0].CallbackData != "confirm" {
		t.Errorf("want confirm, got %q", rows[0][0].CallbackData)
	}
	if rows[1][0].CallbackData != "cancel" {
		t.Errorf("want cancel, got %q", rows[1][0].CallbackData)
	}
}

func TestFormatDate_ValidDate(t *testing.T) {
	result := FormatDate("2026-05-25")
	if !strings.Contains(result, "25") {
		t.Errorf("day missing: %q", result)
	}
	if !strings.Contains(result, "мая") {
		t.Errorf("month missing: %q", result)
	}
	if !strings.Contains(result, "Пн") {
		t.Errorf("weekday missing: %q", result)
	}
}

func TestFormatDate_InvalidDate_ReturnsRaw(t *testing.T) {
	if FormatDate("bad-date") != "bad-date" {
		t.Error("expected raw string for invalid date")
	}
}
