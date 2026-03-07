package bot

import (
	"bytes"
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	tele "gopkg.in/telebot.v4"
)

func (b *TelegramBot) handleCallback(c tele.Context) error {
	data := strings.TrimPrefix(c.Callback().Data, "\f")
	parts := strings.Split(data, "|")
	slog.Info("callback message", "data", data, "parts", parts)

	if len(parts) < 1 {
		return c.Respond()
	}

	unique := parts[0]

	switch unique {
	case "list_page":
		if len(parts) < 2 {
			return c.Respond()
		}
		page, _ := strconv.Atoi(parts[1])
		slog.Info("list_page", "page", page, "telegram_id", c.Sender().ID)
		return b.sendLinksPage(c, page)

	case "stats":
		if len(parts) < 2 {
			return c.Respond()
		}
		shortCode := parts[1]
		slog.Info("stats", "short_code", shortCode, "telegram_id", c.Sender().ID)
		return b.handleLinkCallback(c, shortCode)

	case "ignore":
		slog.Info("Ignoring " + unique)
		return c.Respond()

	case "delete":
		if len(parts) < 2 {
			return c.Respond()
		}
		shortCode := parts[1]
		slog.Info("try_delete", "short_code", shortCode, "telegram_id", c.Sender().ID)
		return b.handleConfirmDelete(c, shortCode)

	case "confirm":
		if len(parts) < 2 {
			return c.Respond()
		}
		shortCode := parts[1]
		slog.Info("full_delete", "short_code", shortCode, "telegram_id", c.Sender().ID)
		return b.handleDelete(c, shortCode)

	case "cancel":
		if len(parts) < 2 {
			return c.Respond()
		}
		shortCode := parts[1]
		slog.Info("cancel", "short_code", shortCode, "telegram_id", c.Sender().ID)
		return b.handleCancel(c)

	case "update":
		if len(parts) < 2 {
			return c.Respond()
		}
		shortCode := parts[1]
		slog.Info("update", "short_code", shortCode, "telegram_id", c.Sender().ID)
		_ = c.Respond()
		b.mu.Lock()
		b.userStates[c.Sender().ID] = UserState{
			Action: StateEditing,
			Data:   shortCode,
		}
		b.mu.Unlock()
		return c.Send("📝 Будь ласка, надішліть нове оригінальне посилання для <code>"+shortCode+"</code>:", &tele.SendOptions{ParseMode: tele.ModeHTML})
	case "qr":
		if len(parts) < 2 {
			return c.Respond()
		}
		shortCode := parts[1]
		slog.Info("qr", "short_code", shortCode, "telegram_id", c.Sender().ID)
		_ = c.Respond()
		qrc, err := b.getQrCode(shortCode)
		if err != nil {
			return c.Send("Помилка отримання qr-code")
		}
		return c.Send(qrc)
	}
	return c.Respond()
}

func (b *TelegramBot) sendLinksPage(c tele.Context, page int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userId, err := b.db.GetUserIDByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		return c.Send("Помилка БД.")
	}

	links, err := b.db.GetAllLinksByUser(ctx, userId)
	if err != nil || len(links) == 0 {
		return c.Send("У вас немає посилань.")
	}

	const pageSize = 10
	totalLinks := len(links)
	totalPages := (totalLinks + pageSize - 1) / pageSize

	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * pageSize
	end := start + pageSize
	if end > totalLinks {
		end = totalLinks
	}
	pageLinks := links[start:end]

	menu := &tele.ReplyMarkup{}
	var rows []tele.Row

	for _, link := range pageLinks {
		btnText := link.ShortCode
		if len(link.OriginalLink) > 20 {
			btnText += " (" + link.OriginalLink[:17] + "...)"
		} else {
			btnText += " (" + link.OriginalLink + ")"
		}
		rows = append(rows, menu.Row(menu.Data(btnText, "stats", link.ShortCode)))
	}

	navRow := tele.Row{}
	if page > 0 {
		navRow = append(navRow, menu.Data("⬅️ Назад", "list_page", strconv.Itoa(page-1)))
	}

	navRow = append(navRow, menu.Data(strconv.Itoa(page+1)+"/"+strconv.Itoa(totalPages), "ignore"))

	if end < totalLinks {
		navRow = append(navRow, menu.Data("Вперед ➡️", "list_page", strconv.Itoa(page+1)))
	}
	rows = append(rows, navRow)

	menu.Inline(rows...)

	if c.Callback() != nil {
		return c.Edit("Ось ваші посилання (сторінка "+strconv.Itoa(page+1)+"):", menu)
	}
	return c.Send("Ось ваші посилання:", menu)
}

func (b *TelegramBot) handleLinkCallback(c tele.Context, shortCode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.Respond(&tele.CallbackResponse{Text: "Завантажую статистику для " + shortCode})
	userId, err := b.db.GetUserIDByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		slog.Error("failed to get user id from db", "user_id", userId)
		return c.Send("Помилка звернення до бази даних.")
	}
	analytics, err := b.db.GetAnalyticByCode(ctx, shortCode, userId)
	if err != nil {
		slog.Error("failed to get analytic from db", "user_id", userId)
		return c.Send("Помилка отримання аналітики.")
	}

	cityStats := make(map[string]int)
	countryStats := make(map[string]int)
	refererStats := make(map[string]int)
	clickedAtStats := make(map[int]int)

	for _, v := range analytics {
		cityStats[v.City]++
		countryStats[v.Country]++
		refererStats[v.Referer]++
		clickedAtStats[v.ClickedAt.Hour()]++
	}

	var sb strings.Builder
	sb.WriteString("<b>📊 Ваша аналітика по " + shortCode + "</b>\n")
	sb.WriteString("Всього переходів: <code>")
	sb.WriteString(strconv.Itoa(len(analytics)))
	sb.WriteString("</code>\n\n")

	sb.WriteString("\n<b>🌍 Географія:</b>\n")
	sb.WriteString(b.getTopStats(countryStats, 8))

	sb.WriteString("\n<b>🏙 Міста:</b>\n")
	sb.WriteString(b.getTopStats(cityStats, 8))

	sb.WriteString("\n<b>🌐 Джерела (Referer):</b>\n")
	sb.WriteString(b.getTopStats(refererStats, 8))

	peakHour, peakCount := -1, 0
	for hour, count := range clickedAtStats {
		if count > peakCount {
			peakCount, peakHour = count, hour
		}
	}

	if peakHour != -1 {
		sb.WriteString("\n<b>⏰ Пікова година (GMT):</b> ")
		if peakHour < 10 {
			sb.WriteByte('0')
		}
		sb.WriteString(strconv.Itoa(peakHour))
		sb.WriteString(":00 (")
		sb.WriteString(strconv.Itoa(peakCount))
		sb.WriteString(" кліків)\n")
	}
	menu := &tele.ReplyMarkup{}
	updateBtn := menu.Data("✍️ Оновити оригінальне посилання", "update", shortCode)
	deleteBtn := menu.Data("🗑️ Видалити це посилання", "delete", shortCode)
	qrBtn := menu.Data("🖼 Отримати QR-код", "qr", shortCode)
	menu.Inline(
		menu.Row(updateBtn),
		menu.Row(deleteBtn),
		menu.Row(qrBtn),
	)
	slog.Info("show shortCode analytics info", "user_id", userId)
	return c.Send(sb.String(), menu, &tele.SendOptions{ParseMode: tele.ModeHTML, ReplyMarkup: menu})
}

func (b *TelegramBot) handleConfirmDelete(c tele.Context, shortCode string) error {
	menu := &tele.ReplyMarkup{}
	btnConfirm := menu.Data("✅ Підтвердити", "confirm", shortCode)
	btnCancel := menu.Data("❌ Відмінити", "cancel", shortCode)
	menu.Inline(menu.Row(btnConfirm), menu.Row(btnCancel))
	_ = c.Respond()
	return c.Send("Ви впевнені, що хочете видалити посилання "+shortCode+"?", menu)
}

func (b *TelegramBot) handleDelete(c tele.Context, shortCode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	userId, err := b.db.GetUserIDByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		slog.Error("failed to get user id from db", "telegram_id", c.Sender().ID, "error", err)
		return err
	}
	if err := b.db.DeleteLink(ctx, userId, shortCode); err != nil {
		return err
	}
	_ = c.RespondAlert("Успішно видалено")
	return c.Edit("Успішно видалено посилання " + shortCode)
}

func (b *TelegramBot) handleCancel(c tele.Context) error {
	userID := c.Sender().ID

	b.mu.Lock()
	delete(b.userStates, userID)
	b.mu.Unlock()

	if c.Callback() != nil {
		return c.RespondAlert("Відміна")
	}

	return c.Send("Відміна")
}

func (b *TelegramBot) getQrCode(shortCode string) (*tele.Photo, error) {
	fullLink := b.baseLink + "/" + shortCode
	qrc, err := qrcode.Encode(fullLink, qrcode.Medium, 256)
	if err != nil {
		slog.Error("failed to generate qrcode", "shortCode", shortCode, "error", err)
		return nil, err
	}
	photo := &tele.Photo{
		File:    tele.FromReader(bytes.NewReader(qrc)),
		Caption: "🖼 Ось ваше посилання і QR-код для нього: " + fullLink,
	}
	return photo, nil
}
