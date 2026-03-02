package bot

import (
	"context"
	"linkshortener/internal/database"
	"linkshortener/internal/service"
	"log/slog"
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
	b.tgBot.Handle("/my_links", b.handleMyLinks)
	b.tgBot.Handle("/all_analytics", b.handleAllAnalytics)
	b.tgBot.Handle(tele.OnText, b.handleLink)

	commands := []tele.Command{
		{Text: "start", Description: "Запустити бота"},
		{Text: "my_links", Description: "Список моїх посилань та окрема статистика"},
		{Text: "all_analytics", Description: "Повна статистика переходів"},
	}

	if err := b.tgBot.SetCommands(commands); err != nil {
		slog.Error("failed to set commands", "error", err)
	}

	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnMyLinks := menu.Text("🗂 Мої посилання")
	btnStats := menu.Text("📊 Загальна аналітика")

	b.tgBot.Handle(&btnMyLinks, b.handleMyLinks)
	b.tgBot.Handle(&btnStats, b.handleAllAnalytics)
	b.tgBot.Handle(tele.OnCallback, b.handleCallback)

	go func() {
		<-ctx.Done()
		slog.Info("Telegram bot shutting down")
		b.tgBot.Stop()
	}()

	b.tgBot.Start()
	return nil
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
