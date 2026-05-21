package keyboard

import (
	"fmt"
	"time"

	"github.com/Hacieva/clinic-scheduler/bot/internal/client"
)

// Button is a single Telegram inline keyboard button.
// The handler layer converts [][]Button to the bot-library-specific markup type.
type Button struct {
	Text         string
	CallbackData string
}

// Directions builds one button per direction, one per row.
func Directions(dirs []client.Direction) [][]Button {
	rows := make([][]Button, 0, len(dirs))
	for _, d := range dirs {
		rows = append(rows, []Button{{
			Text:         d.Name,
			CallbackData: fmt.Sprintf("direction:%d", d.ID),
		}})
	}
	return rows
}

// Doctors builds one button per doctor (Фамилия И.О.), one per row.
func Doctors(docs []client.Doctor) [][]Button {
	rows := make([][]Button, 0, len(docs))
	for _, d := range docs {
		name := Abbreviate(d.LastName, d.FirstName, d.MiddleName)
		rows = append(rows, []Button{{
			Text:         name,
			CallbackData: fmt.Sprintf("doctor:%d", d.ID),
		}})
	}
	return rows
}

// Services builds one button per service with duration and optional price.
func Services(svcs []client.Service) [][]Button {
	rows := make([][]Button, 0, len(svcs))
	for _, s := range svcs {
		label := fmt.Sprintf("%s (%d мин)", s.Name, s.DurationMinutes)
		if s.Price != nil && *s.Price > 0 {
			label += fmt.Sprintf(" — %d ₽", *s.Price/100)
		}
		rows = append(rows, []Button{{
			Text:         label,
			CallbackData: fmt.Sprintf("service:%d", s.ID),
		}})
	}
	return rows
}

// Dates builds one button per day that has at least one slot.
func Dates(days []client.AvailabilityDay) [][]Button {
	rows := make([][]Button, 0, len(days))
	for _, day := range days {
		if len(day.Slots) == 0 {
			continue
		}
		rows = append(rows, []Button{{
			Text:         FormatDate(day.Date),
			CallbackData: fmt.Sprintf("date:%s", day.Date),
		}})
	}
	return rows
}

// Times arranges time slots in rows of three.
func Times(slots []string) [][]Button {
	var rows [][]Button
	var row []Button
	for i, slot := range slots {
		row = append(row, Button{
			Text:         slot,
			CallbackData: fmt.Sprintf("time:%s", slot),
		})
		if (i+1)%3 == 0 || i == len(slots)-1 {
			rows = append(rows, row)
			row = nil
		}
	}
	return rows
}

// Confirm returns a two-button keyboard: confirm + cancel.
func Confirm() [][]Button {
	return [][]Button{
		{{Text: "✅ Подтвердить", CallbackData: "confirm"}},
		{{Text: "❌ Отменить", CallbackData: "cancel"}},
	}
}

// FormatDate parses YYYY-MM-DD and returns a Russian label like "25 мая (Пн)".
// Returns the raw string unchanged if parsing fails.
func FormatDate(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	months := [...]string{"", "янв", "фев", "мар", "апр", "мая", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек"}
	days := [...]string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	return fmt.Sprintf("%d %s (%s)", t.Day(), months[t.Month()], days[t.Weekday()])
}

// Abbreviate returns "Фамилия И.О." from separate name parts.
func Abbreviate(last, first string, middle *string) string {
	name := last
	if len([]rune(first)) > 0 {
		name += " " + string([]rune(first)[:1]) + "."
	}
	if middle != nil && len([]rune(*middle)) > 0 {
		name += string([]rune(*middle)[:1]) + "."
	}
	return name
}
