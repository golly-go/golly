package passport

import (
	"fmt"
	"reflect"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/slimloans/golly"
)

func JWTMiddleware(passportObject Identity) func(next golly.HandlerFunc) golly.HandlerFunc {
	var passportType = reflect.TypeOf(passportObject)

	return func(next golly.HandlerFunc) golly.HandlerFunc {
		passport := reflect.New(passportType).Elem()

		return func(c golly.WebContext) {
			token := DecodeAuthorizationHeader(c.Request().Header.Get("Authorization"))
			if token != "" {
				ident, err := DecodeToken(token, passport.Interface().(Identity))
				if err == nil {
					ident.ToContext(c.Context)
				}
			}
			next(c)
		}
	}
}

// DecodeToken - decodes JWT token into an identity
func DecodeToken(token string, passport Identity) (Identity, error) {
	_, err := jwt.ParseWithClaims(token, passport, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return golly.Secret(), nil
	})
	return passport, err
}

// DecodeAuthorizationHeader removes the "Bearer"
func DecodeAuthorizationHeader(header string) string {
	token := usersHeaderMatcher.FindStringSubmatch(header)
	if len(token) > 1 {
		return token[1]
	}
	return ""
}
