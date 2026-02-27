package service

import "linkshortener/internal/types"

func ShortLink(link string) types.LinkPair {
	// TODO
	pair := types.LinkPair{
		ShortLink:    "",
		OriginalLink: link,
	}
	return pair
}
