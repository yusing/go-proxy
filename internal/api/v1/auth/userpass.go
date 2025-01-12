package auth

import (
	"bytes"
	"encoding/json"
	"net/http"

	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
)

var (
	ErrInvalidUsername = E.New("invalid username")
	ErrInvalidPassword = E.New("invalid password")
)

func validatePassword(cred *Credentials) error {
	if cred.Username != common.APIUser {
		return ErrInvalidUsername.Subject(cred.Username)
	}
	if !bytes.Equal(common.HashPassword(cred.Password), common.APIPasswordHash) {
		return ErrInvalidPassword.Subject(cred.Password)
	}
	return nil
}

// UserPassLoginHandler handles user login.
func UserPassLoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		U.HandleErr(w, r, err, http.StatusBadRequest)
		return
	}
	if err := validatePassword(&creds); err != nil {
		U.HandleErr(w, r, err, http.StatusUnauthorized)
		return
	}
	if err := setAuthenticatedCookie(w, creds.Username); err != nil {
		U.HandleErr(w, r, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
