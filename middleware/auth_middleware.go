package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
)

// Secret key for signing JWT tokens
var jwtSecret = []byte("REDACTED_JWT_SECRET")

// ValidateJWT parses and validates the JWT token
func ValidateJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, http.ErrAbortHandler
		}
		return jwtSecret, nil
	})
	return token, err
}

// JWTMiddleware is a middleware function for validating JWT tokens
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		// Extract the token from the header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			http.Error(w, "Invalid Authorization Header", http.StatusUnauthorized)
			return
		}

		// Validate the token
		token, err := ValidateJWT(tokenString)
		if err != nil || !token.Valid {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		// If token is valid, proceed to the next handler
		next.ServeHTTP(w, r)
	})
}
