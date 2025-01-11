package rules

import "golang.org/x/crypto/bcrypt"

type (
	HashedCrendentials struct {
		Username   string
		CheckMatch func(inputPwd []byte) bool
	}
	Credentials struct {
		Username string
		Password []byte
	}
)

func BCryptCrendentials(username string, hashedPassword []byte) *HashedCrendentials {
	return &HashedCrendentials{username, func(inputPwd []byte) bool {
		return bcrypt.CompareHashAndPassword(hashedPassword, inputPwd) == nil
	}}
}

func (hc *HashedCrendentials) Match(cred *Credentials) bool {
	if cred == nil {
		return false
	}
	return hc.Username == cred.Username && hc.CheckMatch(cred.Password)
}
