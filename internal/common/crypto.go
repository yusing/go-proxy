package common

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"

	"github.com/rs/zerolog/log"
)

func HashPassword(pwd string) []byte {
	h := sha512.New()
	h.Write([]byte(pwd))
	return h.Sum(nil)
}

func generateJWTKey(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		log.Panic().Err(err).Msg("failed to generate jwt key")
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

func decodeJWTKey(key string) []byte {
	bytes, err := base64.URLEncoding.DecodeString(key)
	if err != nil {
		log.Panic().Err(err).Msg("failed to decode jwt key")
	}
	return bytes
}
