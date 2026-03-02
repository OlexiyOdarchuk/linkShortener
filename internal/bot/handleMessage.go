package bot

import (
	"log/slog"
	"net/url"

	tele "gopkg.in/telebot.v4"
)

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
