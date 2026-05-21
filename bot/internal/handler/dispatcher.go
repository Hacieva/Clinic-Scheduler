package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/Hacieva/clinic-scheduler/bot/internal/flow"
)

// Dispatcher converts telego.Update to flow.Update and routes it to the FSM.
// It is the single entry point for all Telegram updates.
type Dispatcher struct {
	fsm *flow.Handler
}

func NewDispatcher(fsm *flow.Handler) *Dispatcher {
	return &Dispatcher{fsm: fsm}
}

// Handle dispatches one Telegram update.
// A deferred recover wrapper ensures a panic inside the FSM does not crash the bot.
// Unknown update types (no Message, no CallbackQuery) are silently dropped.
func (d *Dispatcher) Handle(ctx context.Context, update telego.Update) {
	defer func() {
		if r := recover(); r != nil {
			// update_id is safe; no user data is logged here.
			slog.Error("panic recovered in update handler",
				"update_id", update.UpdateID,
				"recover", fmt.Sprint(r),
			)
		}
	}()

	u := toFlowUpdate(update)
	if u == nil {
		// Unknown update type — no-op, session is not touched.
		return
	}
	d.fsm.Handle(ctx, *u)
}

// toFlowUpdate converts a telego.Update into a flow.Update.
// Returns nil for update types not handled by the booking FSM.
// Only Message and CallbackQuery are routed; all others are dropped.
func toFlowUpdate(update telego.Update) *flow.Update {
	if update.Message != nil {
		return fromMessage(update.Message)
	}
	if update.CallbackQuery != nil {
		return fromCallbackQuery(update.CallbackQuery)
	}
	return nil
}

func fromMessage(msg *telego.Message) *flow.Update {
	if msg.From == nil {
		// Channel posts have no From field; ignore them.
		return nil
	}
	fu := &flow.Update{
		UserID: msg.From.ID,
		ChatID: msg.Chat.ID,
	}
	if msg.From.Username != "" {
		u := msg.From.Username
		fu.TelegramUsername = &u
	}
	text := strings.TrimSpace(msg.Text)
	if strings.HasPrefix(text, "/") {
		parts := strings.Fields(text)
		cmd := strings.TrimPrefix(parts[0], "/")
		// Strip @botname suffix so /start@mybot == /start.
		if idx := strings.IndexByte(cmd, '@'); idx != -1 {
			cmd = cmd[:idx]
		}
		fu.Command = strings.ToLower(cmd)
	} else {
		fu.Text = text
	}
	return fu
}

func fromCallbackQuery(cq *telego.CallbackQuery) *flow.Update {
	fu := &flow.Update{
		UserID:       cq.From.ID,
		CallbackData: cq.Data,
		CallbackID:   cq.ID,
	}
	if cq.Message != nil {
		fu.ChatID = cq.Message.GetChat().ID
	}
	if cq.From.Username != "" {
		u := cq.From.Username
		fu.TelegramUsername = &u
	}
	return fu
}
