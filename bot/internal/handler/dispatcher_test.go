package handler

import (
	"testing"

	"github.com/mymmrac/telego"

	"github.com/Hacieva/clinic-scheduler/bot/internal/flow"
)

func ptr[T any](v T) *T { return &v }

func TestToFlowUpdate_TextMessage(t *testing.T) {
	msg := &telego.Message{
		From: &telego.User{ID: 42, Username: "ivan"},
		Chat: telego.Chat{ID: 100},
		Text: "Привет",
	}
	u := toFlowUpdate(telego.Update{Message: msg})
	if u == nil {
		t.Fatal("expected non-nil flow.Update")
	}
	if u.UserID != 42 {
		t.Errorf("UserID: want 42, got %d", u.UserID)
	}
	if u.ChatID != 100 {
		t.Errorf("ChatID: want 100, got %d", u.ChatID)
	}
	if u.Text != "Привет" {
		t.Errorf("Text: want 'Привет', got %q", u.Text)
	}
	if u.Command != "" {
		t.Errorf("Command: want empty, got %q", u.Command)
	}
	if u.TelegramUsername == nil || *u.TelegramUsername != "ivan" {
		t.Errorf("TelegramUsername: want 'ivan', got %v", u.TelegramUsername)
	}
}

func TestToFlowUpdate_CommandMessage(t *testing.T) {
	msg := &telego.Message{
		From: &telego.User{ID: 1},
		Chat: telego.Chat{ID: 10},
		Text: "/start",
	}
	u := toFlowUpdate(telego.Update{Message: msg})
	if u == nil {
		t.Fatal("expected non-nil")
	}
	if u.Command != "start" {
		t.Errorf("Command: want 'start', got %q", u.Command)
	}
	if u.Text != "" {
		t.Errorf("Text should be empty for commands, got %q", u.Text)
	}
}

func TestToFlowUpdate_CommandWithBotName(t *testing.T) {
	msg := &telego.Message{
		From: &telego.User{ID: 1},
		Chat: telego.Chat{ID: 10},
		Text: "/start@mybot",
	}
	u := toFlowUpdate(telego.Update{Message: msg})
	if u == nil || u.Command != "start" {
		t.Errorf("Command: want 'start', got %q", u.Command)
	}
}

func TestToFlowUpdate_CommandUpperCase(t *testing.T) {
	msg := &telego.Message{
		From: &telego.User{ID: 1},
		Chat: telego.Chat{ID: 10},
		Text: "/Cancel",
	}
	u := toFlowUpdate(telego.Update{Message: msg})
	if u == nil || u.Command != "cancel" {
		t.Errorf("Command must be lowercased, got %q", u.Command)
	}
}

func TestToFlowUpdate_MessageNoFrom_ReturnsNil(t *testing.T) {
	msg := &telego.Message{
		Chat: telego.Chat{ID: 10},
		Text: "channel post",
	}
	u := toFlowUpdate(telego.Update{Message: msg})
	if u != nil {
		t.Errorf("expected nil for message without From, got %+v", u)
	}
}

func TestToFlowUpdate_CallbackQuery(t *testing.T) {
	msg := &telego.Message{Chat: telego.Chat{ID: 200}}
	cq := &telego.CallbackQuery{
		ID:      "cb123",
		From:    telego.User{ID: 55, Username: "anna"},
		Message: msg,
		Data:    "direction:3",
	}
	u := toFlowUpdate(telego.Update{CallbackQuery: cq})
	if u == nil {
		t.Fatal("expected non-nil")
	}
	if u.UserID != 55 {
		t.Errorf("UserID: want 55, got %d", u.UserID)
	}
	if u.ChatID != 200 {
		t.Errorf("ChatID: want 200, got %d", u.ChatID)
	}
	if u.CallbackData != "direction:3" {
		t.Errorf("CallbackData: want 'direction:3', got %q", u.CallbackData)
	}
	if u.CallbackID != "cb123" {
		t.Errorf("CallbackID: want 'cb123', got %q", u.CallbackID)
	}
	if u.TelegramUsername == nil || *u.TelegramUsername != "anna" {
		t.Errorf("TelegramUsername: want 'anna', got %v", u.TelegramUsername)
	}
}

func TestToFlowUpdate_UnknownUpdateType_ReturnsNil(t *testing.T) {
	u := toFlowUpdate(telego.Update{UpdateID: 99}) // neither Message nor CallbackQuery
	if u != nil {
		t.Errorf("expected nil for unknown update type, got %+v", u)
	}
}

func TestToFlowUpdate_CallbackNoMessage_ChatIDZero(t *testing.T) {
	cq := &telego.CallbackQuery{
		ID:   "cb1",
		From: telego.User{ID: 10},
		Data: "confirm",
	}
	u := toFlowUpdate(telego.Update{CallbackQuery: cq})
	if u == nil {
		t.Fatal("expected non-nil")
	}
	if u.ChatID != 0 {
		t.Errorf("ChatID should be 0 when Message is nil, got %d", u.ChatID)
	}
}

// Verify toFlowUpdate returns a flow.Update (compile-time check).
var _ *flow.Update = toFlowUpdate(telego.Update{})
