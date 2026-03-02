package bot

import (
	"context"
	"linkshortener/internal/database"
	"linkshortener/internal/service"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v4"
)

type TelegramBot struct {
	baseLink   string
	tgBot      *tele.Bot
	userStates map[int64]UserState
	db         *database.Database
	analytic   *database.Analytics
	shortener  *service.Shortener
	mu         *sync.RWMutex
}

const (
	StateWaitingLink   = "waiting_link"
	StateWaitingCustom = "waiting_custom_name"
	StateEditing       = "editing"
)

type UserState struct {
	Action string
	Data   string
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
		baseLink:   baseLink,
		tgBot:      bot,
		userStates: make(map[int64]UserState),
		db:         db,
		analytic:   analytics,
		shortener:  shortener,
		mu:         &sync.RWMutex{},
	}

	return b, nil
}

func (b *TelegramBot) Start(ctx context.Context) error {
	slog.Info("Telegram bot started", "bot_username", b.tgBot.Me.Username)

	b.tgBot.Handle("/start", b.handleStart)
	b.tgBot.Handle("/create_custom", b.handleCustomLink)
	b.tgBot.Handle("/my_links", b.handleMyLinks)
	b.tgBot.Handle("/all_analytics", b.handleAllAnalytics)
	b.tgBot.Handle("/cancel", b.handleCancel)
	b.tgBot.Handle(tele.OnText, b.handleLink)
	b.tgBot.Handle(tele.OnCallback, b.handleCallback)

	commands := []tele.Command{
		{Text: "start", Description: "Запустити бота"},
		{Text: "create_custom", Description: "Створити нове посилання з власним скороченням"},
		{Text: "my_links", Description: "Список моїх посилань та окрема статистика"},
		{Text: "all_analytics", Description: "Повна статистика переходів"},
		{Text: "cancel", Description: "Відмінити нинішню дію"},
	}

	if err := b.tgBot.SetCommands(commands); err != nil {
		slog.Error("failed to set commands", "error", err)
	}

	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnMyLinks := menu.Text("🗂 Мої посилання")
	btnStats := menu.Text("📊 Загальна аналітика")

	b.tgBot.Handle(&btnMyLinks, b.handleMyLinks)
	b.tgBot.Handle(&btnStats, b.handleAllAnalytics)

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
