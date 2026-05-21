package handler

import (
	"context"
	"log/slog"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/Hacieva/clinic-scheduler/bot/internal/keyboard"
)

// TelegramSender implements flow.Sender using the telego Bot client.
// Telegram API errors are logged but never returned to the FSM — the session
// state is determined solely by business logic, not by delivery success.
type TelegramSender struct {
	bot *telego.Bot
}

func NewTelegramSender(bot *telego.Bot) *TelegramSender {
	return &TelegramSender{bot: bot}
}

func (s *TelegramSender) SendText(ctx context.Context, chatID int64, text string) error {
	_, err := s.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: tu.ID(chatID),
		Text:   text,
	})
	if err != nil {
		// Log chat_id only — message text may contain patient name/phone (PII).
		slog.Error("telegram: SendMessage failed", "chat_id", chatID, "err", err)
	}
	return err
}

func (s *TelegramSender) SendKeyboard(ctx context.Context, chatID int64, text string, buttons [][]keyboard.Button) error {
	_, err := s.bot.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:      tu.ID(chatID),
		Text:        text,
		ReplyMarkup: buildInlineKeyboard(buttons),
	})
	if err != nil {
		slog.Error("telegram: SendMessage (keyboard) failed", "chat_id", chatID, "err", err)
	}
	return err
}

func (s *TelegramSender) AnswerCallback(ctx context.Context, callbackID string) error {
	err := s.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
	})
	if err != nil {
		// callback_id is safe to log; it contains no user data.
		slog.Error("telegram: AnswerCallbackQuery failed", "err", err)
	}
	return err
}

func buildInlineKeyboard(buttons [][]keyboard.Button) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, len(buttons))
	for i, row := range buttons {
		rows[i] = make([]telego.InlineKeyboardButton, len(row))
		for j, btn := range row {
			rows[i][j] = tu.InlineKeyboardButton(btn.Text).WithCallbackData(btn.CallbackData)
		}
	}
	return tu.InlineKeyboard(rows...)
}
