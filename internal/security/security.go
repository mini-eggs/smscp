package security

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

type security struct {
	secret string
}

func SecurityDefault(secret string) security {
	return security{secret}
}

func (sec security) HashCreate(pass string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pass), 4)
	return string(bytes), err
}

func (sec security) HashCompare(pass, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
}

func (sec security) TokenCreate(val jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, val)
	return token.SignedString([]byte(sec.secret))
}

func (sec security) TokenFrom(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(sec.secret), nil
	})
	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return token.Claims.(jwt.MapClaims), nil
	} else {
		return jwt.MapClaims{}, err
	}
}
