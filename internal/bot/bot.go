package bot

import (
	"context"
	"linkshortener/internal/database"
	"linkshortener/internal/service"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
)

type TelegramBot struct {
	baseLink  string
	tgBot     *tele.Bot
	db        *database.Database
	analytic  *database.Analytics
	shortener *service.Shortener
}

func NewTelegramBot(baseLink, tgToken string, db *database.Database, analytics *database.Analytics, shortener *service.Shortener) (*TelegramBot, error) {
	pref := tele.Settings{
		Token:  tgToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		slog.Error("failed to initialize telegram bot", "error", err)
		return nil, err
	}

	b := &TelegramBot{
		baseLink:  baseLink,
		tgBot:     bot,
		db:        db,
		analytic:  analytics,
		shortener: shortener,
	}

	return b, nil
}

func (b *TelegramBot) Start(ctx context.Context) error {
	slog.Info("Telegram bot started", "bot_username", b.tgBot.Me.Username)

	b.tgBot.Handle("/start", b.handleStart)
	b.tgBot.Handle(tele.OnText, b.handleLink)
	b.tgBot.Handle("/my_links", b.handleMyLinks)
	b.tgBot.Handle("/all_analytics", b.handleAllAnalytics)

	commands := []tele.Command{
		{Text: "start", Description: "Запустити бота"},
		{Text: "my_links", Description: "Список моїх посилань та окрема статистика"},
		{Text: "all_analytics", Description: "Повна статистика переходів"},
	}

	if err := b.tgBot.SetCommands(commands); err != nil {
		slog.Error("failed to set commands", "error", err)
	}

	go func() {
		<-ctx.Done()
		slog.Info("Telegram bot shutting down")
		b.tgBot.Stop()
	}()

	b.tgBot.Start()
	return nil
}

func (b *TelegramBot) handleStart(c tele.Context) error {
	slog.Debug("command /start received", "user_id", c.Sender().ID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := b.db.CreateUser(ctx, c.Sender().ID)
	if err != nil {
		slog.Error("failed to create user", "user_id", c.Sender().ID, "error", err)
		return c.Send("Вибачте, виникла помилка при реєстрації. Спробуйте пізніше.")
	}

	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnMyLinks := menu.Text("🗂 Мої посилання")
	btnStats := menu.Text("📊 Загальна аналітика")

	menu.Reply(
		menu.Row(btnMyLinks),
		menu.Row(btnStats),
	)

	b.tgBot.Handle(&btnMyLinks, b.handleMyLinks)
	b.tgBot.Handle(&btnStats, b.handleAllAnalytics)

	welcomeMsg := "👋 <b>Привіт! Я твій персональний сервіс скорочення посилань.</b>\n\n" +
		"🚀 <b>Як користуватися:</b>\n" +
		"Просто надішліть мені будь-яке довге посилання, і я зроблю його коротким.\n\n" +
		"📈 <b>Можливості:</b>\n" +
		"• Зберігання історії посилань\n" +
		"• Детальна аналітика переходів (країна, місто, браузер)\n\n" +
		"<i>Скористайтеся кнопками меню нижче для управління:</i>"

	return c.Send(welcomeMsg, menu, &tele.SendOptions{ParseMode: tele.ModeHTML})
}

func (b *TelegramBot) handleLink(c tele.Context) error {
	newLink := c.Text()
	u, err := url.ParseRequestURI(newLink)
	if err != nil {
		slog.Error("failed to parse url", "url", newLink)
		return c.Send("Ваше посилання не валідне.")
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		slog.Warn("invalid url scheme or host", "url", newLink, "scheme", u.Scheme, "host", u.Host)
		return c.Send("Посилання повинно починатися з http:// або https:// і містити домен.")
	}
	shortLink, err := b.shortener.CreateNewShortLink(newLink, c.Sender().ID)
	if err != nil {
		slog.Error("failed to create short link", "error", err)
		return c.Send("Помилка при створенні посилання. Спробуйте ще раз")
	}
	return c.Send("Ось ваше нове скорочене посилання:\n" + b.baseLink + "/" + shortLink)
}

func (b *TelegramBot) handleMyLinks(c tele.Context) error {
	// TODO
	return nil
}
func (b *TelegramBot) handleAllAnalytics(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userId, err := b.db.GetUserIDByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		slog.Error("failed to get user id from db", "user_id", c.Sender().ID)
		return c.Send("Помилка бази даних.")
	}

	analytics, err := b.analytic.GetAllAnalytic(ctx, userId)
	if err != nil {
		slog.Error("failed to get analytics from db", "user_id", c.Sender().ID)
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
	sb.WriteString(getTopStats(shortCodeStats, 5))

	sb.WriteString("\n<b>🌍 Географія:</b>\n")
	sb.WriteString(getTopStats(countryStats, 3))

	sb.WriteString("\n<b>🏙 Міста:</b>\n")
	sb.WriteString(getTopStats(cityStats, 3))

	sb.WriteString("\n<b>🌐 Джерела (Referer):</b>\n")
	sb.WriteString(getTopStats(refererStats, 3))

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

	return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML})
}

func getTopStats(stats map[string]int, limit int) string {
	type entry struct {
		key   string
		count int
	}
	var sorted []entry
	for k, v := range stats {
		if k == "" {
			k = "Unknown"
		}
		sorted = append(sorted, entry{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	var sb strings.Builder
	for i := 0; i < len(sorted) && i < limit; i++ {
		sb.WriteString("  • ")
		sb.WriteString(sorted[i].key)
		sb.WriteString(": ")
		sb.WriteString(strconv.Itoa(sorted[i].count))
		sb.WriteByte('\n')
	}

	if sb.Len() == 0 {
		return "  (немає даних)\n"
	}
	return sb.String()
}
