package security

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type Security struct {
	secret string
}

func Default(secret string) Security {
	return Security{secret}
}

func (sec Security) HashCreate(pass string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pass), 4)
	return string(bytes), err
}

func (sec Security) HashCompare(pass, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
}

func (sec Security) TokenCreate(val jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, val)
	return token.SignedString([]byte(sec.secret))
}

func (sec Security) TokenFrom(tokenString string) (_ret jwt.MapClaims, recoverErr error) {
	defer func() {
		if recover() != nil {
			recoverErr = errors.New("bogus hash; hash has no value")
		}
	}()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(sec.secret), nil
	})
	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return token.Claims.(jwt.MapClaims), nil
	}
	return jwt.MapClaims{}, err
}
