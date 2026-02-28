package types

import "time"

type LinkData struct {
	Id           int64     `json:"id" db:"id"`
	UserId       int64     `json:"user_id" db:"user_id"`
	OriginalLink string    `json:"original_link" db:"original_link"`
	ShortCode    string    `json:"short_code" db:"short_code"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type LinkCache struct {
	OriginalLink string `json:"original_link" db:"original_link"`
	UserID       int64  `json:"user_id" db:"user_id"`
}
