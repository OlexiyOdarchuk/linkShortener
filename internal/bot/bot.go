package bot

import (
	"linkshortener/internal/cache"
	"linkshortener/internal/database"
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v4"
)

type TelegramBot struct {
	tgBot    *tele.Bot
	db       *database.Database
	cache    *cache.Cache
	analytic *database.Analytics
}

func NewTelegramBot(tgToken string, db *database.Database, cacheDB *cache.Cache, analytics *database.Analytics) (*TelegramBot, error) {
	pref := tele.Settings{
		Token:  tgToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		slog.Error("failed to initialize telegram bot", "error", err)
		return nil, err
	}

	return &TelegramBot{
		tgBot:    bot,
		db:       db,
		cache:    cacheDB,
		analytic: analytics,
	}, nil
}

func (b *TelegramBot) Start() {
	slog.Info("Telegram bot started", "bot_username", b.tgBot.Me.Username)

	b.tgBot.Handle("/start", b.handleStart)
	b.tgBot.Handle(tele.OnText, b.handleMessage)

	b.tgBot.Start()
}

func (b *TelegramBot) handleStart(c tele.Context) error {
	slog.Debug("command /start received", "user_id", c.Sender().ID)
	return c.Send("Привіт! Я допоможу тобі скоротити довге посилання. Просто надішліть його мені.")
}

func (b *TelegramBot) handleMessage(c tele.Context) error {
	// TODO
	newLink := c.Text()
	return c.Send("Отримав посилання: " + c.Text() + "\nОсь твоє нове скорочене посилання: " + newLink)
}
