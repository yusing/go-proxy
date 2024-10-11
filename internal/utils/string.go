package utils

import (
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TODO: support other languages.
var titleCaser = cases.Title(language.AmericanEnglish)

func CommaSeperatedList(s string) []string {
	res := strings.Split(s, ",")
	for i, part := range res {
		res[i] = strings.TrimSpace(part)
	}
	return res
}

func Title(s string) string {
	return titleCaser.String(s)
}

func ExtractPort(fullURL string) (int, error) {
	url, err := url.Parse(fullURL)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(url.Port())
}

func PortString(port uint16) string {
	return strconv.FormatUint(uint64(port), 10)
}
