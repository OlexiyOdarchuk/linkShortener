package bot

import (
	"context"
	"errors"
	customerrs "linkshortener/internal/customErrs"
	"log/slog"
	"net/url"
	"time"

	tele "gopkg.in/telebot.v4"
)

func (b *TelegramBot) handleNewLink(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userId, err := b.db.GetUserIDByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		slog.Error("failed to get user id from db", "telegram_id", c.Sender().ID, "error", err)
		return c.Send("Помилка звернення до бази даних.")
	}

	newLink := c.Text()
	u, err := url.ParseRequestURI(newLink)
	if err != nil {
		slog.Error("failed to parse url", "url", newLink)
		return c.Send("❌ Ваше посилання не валідне. Спробуйте ще або напишіть /cancel")
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		slog.Warn("invalid url scheme or host", "url", newLink, "scheme", u.Scheme, "host", u.Host)
		return c.Send("❌ Посилання повинно починатися з http:// або https:// і містити домен.")
	}

	shortCode, err := b.shortener.CreateNewShortLink(ctx, newLink, userId)
	if err != nil {
		slog.Error("failed to create short link", "error", err)
		return c.Send("❌ Помилка при створенні посилання. Спробуйте ще раз")
	}
	qrc, err := b.getQrCode(shortCode)
	if err != nil {
		slog.Error("failed to get qrcode", "error", err)
		return c.Send("✅ Ось ваше нове скорочене посилання:\n" + b.baseLink + "/" + shortCode)
	}
	return c.Send(qrc)
}

func (b *TelegramBot) handleLink(c tele.Context) error {
	userTelegramID := c.Sender().ID
	text := c.Text()

	b.mu.RLock()
	state, ok := b.userStates[userTelegramID]
	b.mu.RUnlock()

	if ok {
		switch state.Action {
		case StateWaitingLink:
			if _, err := url.ParseRequestURI(text); err != nil {
				return c.Send("❌ Невалідний URL. Спробуйте ще раз.")
			}

			b.mu.Lock()
			b.userStates[userTelegramID] = UserState{Action: StateWaitingCustom, Data: text}
			b.mu.Unlock()

			return c.Send("✍️ Тепер напишіть бажане ім'я для посилання (наприклад, <code>my-blog</code>):", &tele.SendOptions{ParseMode: tele.ModeHTML})

		case StateWaitingCustom:
			longURL := state.Data
			customCode := text

			if !b.shortener.IsValidShortCode(customCode) {
				return c.Send("❌ Код може містити лише латинські літери, цифри та дефіс (3–20 символів).")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			userId, err := b.db.GetUserIDByTelegramID(ctx, userTelegramID)
			if err != nil {
				slog.Error("failed to get user id from db", "telegram_id", userTelegramID, "error", err)
				return c.Send("Помилка звернення до бази даних.")
			}

			err = b.shortener.CreateNewCustomShortLink(ctx, longURL, customCode, userId)
			if errors.Is(err, customerrs.ErrCodeIsBusy) {
				return c.Send("❌ Це ім'я вже зайняте. Спробуйте інше:")
			}
			if err != nil {
				slog.Error("failed to create custom short link", "error", err)
				return c.Send("❌ Помилка при створенні посилання. Спробуйте ще раз.")
			}

			b.mu.Lock()
			delete(b.userStates, userTelegramID)
			b.mu.Unlock()
			qrc, err := b.getQrCode(customCode)
			if err != nil {
				slog.Error("failed to get qrcode", "error", err)
				return c.Send("✅ Готово! Ваше посилання:\n" + b.baseLink + "/" + customCode)
			}
			return c.Send(qrc)

		case StateEditing:
			newLink := text
			u, err := url.ParseRequestURI(newLink)
			if err != nil {
				slog.Error("failed to parse url for editing", "url", newLink)
				return c.Send("❌ Ваше посилання не валідне. Спробуйте ще або напишіть /cancel")
			}
			if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				slog.Warn("invalid url scheme or host on edit", "url", newLink, "scheme", u.Scheme, "host", u.Host)
				return c.Send("❌ Посилання повинно починатися з http:// або https:// і містити домен.")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			userId, err := b.db.GetUserIDByTelegramID(ctx, userTelegramID)
			if err != nil {
				slog.Error("failed to get user id from db", "telegram_id", userTelegramID, "error", err)
				return c.Send("Помилка звернення до бази даних.")
			}

			if err := b.db.UpdateLink(ctx, userId, state.Data, newLink); err != nil {
				slog.Error("failed to update link", "error", err)
				return c.Send("⚠️ Не вдалося оновити посилання в базі.")
			}

			b.mu.Lock()
			delete(b.userStates, userTelegramID)
			b.mu.Unlock()

			return c.Send("✅ Посилання для <code>"+state.Data+"</code> успішно оновлено на нове!", &tele.SendOptions{ParseMode: tele.ModeHTML})
		}
	}

	return b.handleNewLink(c)
}
