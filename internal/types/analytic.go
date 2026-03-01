package types

import "time"

type ClickData struct {
	UserId    int64  `json:"user_id" db:"user_id"`
	ShortCode string `json:"short_code" db:"short_code"`
	IP        string `json:"ip" db:"ip"`
	UserAgent string `json:"user_agent" db:"user_agent"`
	Referer   string `json:"referer" db:"referer"`
}

type Analytic struct {
	UserId    int64     `json:"user_id" db:"user_id"`
	ShortCode string    `json:"short_code" db:"short_code"`
	Country   string    `json:"country" db:"country"`
	City      string    `json:"city" db:"city"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	Referer   string    `json:"referer" db:"referer"`
	ClickedAt time.Time `json:"clicked_at" db:"clicked_at"`
}
