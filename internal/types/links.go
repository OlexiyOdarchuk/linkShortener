package types

type LinkPair struct {
	ShortLink    string `json:"short_link" db:"short_link"`
	OriginalLink string `json:"original_link" db:"original_link"`
}
