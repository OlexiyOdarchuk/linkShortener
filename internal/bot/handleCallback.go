package bot

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

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
		return b.handleLinkStats(c, shortCode)

	case "ignore":
		slog.Info("Ignoring " + unique)
		return c.Respond()
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

func (b *TelegramBot) handleLinkStats(c tele.Context, shortCode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.Respond(&tele.CallbackResponse{Text: "Завантажую статистику для " + shortCode})
	userId, err := b.db.GetUserIDByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		slog.Error("failed to get user id from db", "user_id", userId)
		return c.Send("Помилка звернення до бази даних.")
	}
	analytics, err := b.analytic.GetAnalyticByCode(ctx, shortCode, userId)
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
	sb.WriteString("<b>📊 Ваша загальна аналітика</b>\n")
	sb.WriteString("Всього переходів: <code>")
	sb.WriteString(strconv.Itoa(len(analytics)))
	sb.WriteString("</code>\n\n")

	sb.WriteString("\n<b>🌍 Географія:</b>\n")
	sb.WriteString(getTopStats(countryStats, 8))

	sb.WriteString("\n<b>🏙 Міста:</b>\n")
	sb.WriteString(getTopStats(cityStats, 8))

	sb.WriteString("\n<b>🌐 Джерела (Referer):</b>\n")
	sb.WriteString(getTopStats(refererStats, 8))

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
	slog.Info("show shortCode analytics info", "user_id", userId)
	return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML})
}
