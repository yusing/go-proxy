package utils

import (
	"net/url"
	"strconv"
	"strings"
)

func CommaSeperatedList(s string) []string {
	res := strings.Split(s, ",")
	for i, part := range res {
		res[i] = strings.TrimSpace(part)
	}
	return res
}

func ExtractPort(fullURL string) (int, error) {
	url, err := url.Parse(fullURL)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(url.Port())
}
