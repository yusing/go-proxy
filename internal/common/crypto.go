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
	return base64.StdEncoding.EncodeToString(bytes)
}

func decodeJWTKey(key string) []byte {
	if key == "" {
		return nil
	}
	bytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		log.Panic().Err(err).Msg("failed to decode jwt key")
	}
	return bytes
}
