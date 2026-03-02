package bot

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
)

func (b *TelegramBot) handleStart(c tele.Context) error {
	slog.Debug("command /start received", "telegram_id", c.Sender().ID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := b.db.CreateUser(ctx, c.Sender().ID)
	if err != nil {
		slog.Error("failed to create user", "telegram_id", c.Sender().ID, "error", err)
		return c.Send("Вибачте, виникла помилка при реєстрації. Спробуйте пізніше.")
	}

	//err = c.Send("Видалення старих кнопок...", &tele.ReplyMarkup{RemoveKeyboard: true})
	//if err != nil {
	//	slog.Error("failed to send remove menu", "telegram_id", c.Sender().ID, "error", err)
	//}

	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(menu.Text("🗂 Мої посилання")),
		menu.Row(menu.Text("📊 Загальна аналітика")),
	)

	var sb strings.Builder
	sb.WriteString("👋 <b>Привіт! Я твій персональний сервіс скорочення посилань.</b>\n\n")
	sb.WriteString("🚀 <b>Як користуватися:</b>\n")
	sb.WriteString("Просто надішліть мені будь-яке довге посилання, і я зроблю його коротким.\n\n")
	sb.WriteString("📈 <b>Можливості:</b>\n")
	sb.WriteString("• Зберігання історії посилань\n")
	sb.WriteString("• Детальна аналітика переходів (країна, місто, реферер)\n\n")
	sb.WriteString("<i>Скористайтеся кнопками меню нижче для управління:</i>")

	return c.Send(sb.String(), menu, &tele.SendOptions{ParseMode: tele.ModeHTML})
}

func (b *TelegramBot) handleAllAnalytics(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userId, err := b.db.GetUserIDByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		slog.Error("failed to get user id from db", "user_id", userId)
		return c.Send("Помилка бази даних.")
	}

	analytics, err := b.analytic.GetAllAnalytic(ctx, userId)
	if err != nil {
		slog.Error("failed to get analytics from db", "user_id", userId)
		return c.Send("Помилка отримання аналітики.")
	}

	shortCodeStats := make(map[string]int)
	cityStats := make(map[string]int)
	countryStats := make(map[string]int)
	refererStats := make(map[string]int)
	clickedAtStats := make(map[int]int)

	for _, v := range analytics {
		shortCodeStats[v.ShortCode]++
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

	sb.WriteString("<b>🔗 Популярні коди:</b>\n")
	sb.WriteString(getTopStats(shortCodeStats, 8))

	sb.WriteString("\n<b>🌍 Географія:</b>\n")
	sb.WriteString(getTopStats(countryStats, 5))

	sb.WriteString("\n<b>🏙 Міста:</b>\n")
	sb.WriteString(getTopStats(cityStats, 5))

	sb.WriteString("\n<b>🌐 Джерела (Referer):</b>\n")
	sb.WriteString(getTopStats(refererStats, 5))

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
	slog.Info("show all analytics info", "user_id", userId)
	return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML})
}

func (b *TelegramBot) handleMyLinks(c tele.Context) error {
	slog.Info("command /my_links received", "telegram_id", c.Sender().ID)
	return b.sendLinksPage(c, 0)
}
