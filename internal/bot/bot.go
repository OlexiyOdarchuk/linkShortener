package bot

import (
	"context"
	"errors"
	"linkshortener/internal/database"
	"linkshortener/internal/service"
	"log/slog"
	"net/url"
	"time"

	tele "gopkg.in/telebot.v4"
)

type TelegramBot struct {
	tgBot     *tele.Bot
	db        *database.Database
	analytic  *database.Analytics
	shortener *service.Shortener
}

var ErrLinkNotValid = errors.New("link not valid")

func NewTelegramBot(tgToken string, db *database.Database, analytics *database.Analytics, shortener *service.Shortener) (*TelegramBot, error) {
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
	b.tgBot.Handle(tele.OnText, b.handleMessage)

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
		return c.Send("failed to create user in database, please, try again later.")
	}
	return c.Send("Привіт! Я допоможу тобі скоротити довге посилання. Просто надішліть його мені.")
}

func (b *TelegramBot) handleMessage(c tele.Context) error {
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
	return c.Send("Ось ваше нове скорочене посилання:\n" + shortLink)
}
