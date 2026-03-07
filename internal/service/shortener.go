package service

import (
	"context"
	"database/sql"
	"errors"
	customerrs "linkshortener/internal/customErrs"
	"linkshortener/internal/types"
	"regexp"
	"slices"
	"strings"
)

const alphabet = "q1werty8uiop3asdfg4hjkl9zxcvb_n2mMNBVC5XZLKJ6HGFDQ-0ASWERTYU7IOP!"

//go:generate mockgen -source=shortener.go -destination=mock_shortener_db_test.go -package=service
type ShortenerDB interface {
	DeleteLinkById(ctx context.Context, userId, linkId int64) error
	SetShortCode(ctx context.Context, id int64, shortCode string) error
	CreateLink(ctx context.Context, userID int64, originalLink string) (int64, error)
	GetLinkCacheByCode(ctx context.Context, shortCode string) (*types.LinkCache, error)
}
type Shortener struct {
	database ShortenerDB
}

func NewShortener(database ShortenerDB) *Shortener {
	return &Shortener{database: database}
}

func (s *Shortener) CreateNewShortLink(ctx context.Context, originalLink string, userId int64) (string, error) {
	linkId, err := s.database.CreateLink(ctx, userId, originalLink)
	if err != nil {
		return "", err
	}
	shortCode := s.base65Encode(linkId)

	for {
		hasLink, err := s.database.GetLinkCacheByCode(ctx, shortCode)
		if hasLink == nil && errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return "", err
		}
		err = s.database.DeleteLinkById(ctx, userId, linkId)
		if err != nil {
			return "", err
		}
		linkId, err = s.database.CreateLink(ctx, userId, originalLink)
		if err != nil {
			return "", err
		}
		shortCode = s.base65Encode(linkId)
	}

	if err := s.database.SetShortCode(ctx, linkId, shortCode); err != nil {
		return "", err
	}
	return shortCode, nil
}

func (s *Shortener) CreateNewCustomShortLink(ctx context.Context, originalLink, shortCode string, userId int64) error {
	hasCode, err := s.database.GetLinkCacheByCode(ctx, shortCode)
	if hasCode != nil && err == nil {
		return customerrs.ErrCodeIsBusy
	}

	linkId, err := s.database.CreateLink(ctx, userId, originalLink)
	if err != nil {
		return err
	}

	if err := s.database.SetShortCode(ctx, linkId, shortCode); err != nil {
		return err
	}

	return nil
}

func (s *Shortener) base65Encode(linkId int64) string {
	if linkId == 0 {
		return string(alphabet[0])
	}

	res := make([]byte, 0, 12)

	for linkId > 0 {
		res = append(res, alphabet[linkId%62])
		linkId /= 62
	}
	slices.Reverse(res)
	return string(res)
}

func (s *Shortener) base65Decode(shortCode string) (int64, error) {
	var res int64

	for _, char := range shortCode {
		index := strings.IndexRune(alphabet, char)

		if index == -1 {
			return 0, customerrs.ErrInvalidCharacter
		}

		res = res*62 + int64(index)
	}

	return res, nil
}

func (s *Shortener) IsValidShortCode(code string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_!~]{1,20}$`, code)
	return matched
}
