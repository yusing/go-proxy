package utils

import (
	"net/url"
	"strconv"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

func CommaSeperatedList(s string) []string {
	res := strings.Split(s, ",")
	for i, part := range res {
		res[i] = strings.TrimSpace(part)
	}
	return res
}

func IntParser(value string) (int, E.NestedError) {
	return E.Check(strconv.Atoi(value))
}

func ExtractPort(fullURL string) (int, error) {
	url, err := url.Parse(fullURL)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(url.Port())
}
