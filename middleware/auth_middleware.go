package middleware

import (
	"encoding/json"
	"net/http"
	"os"
	"snippet-service/data"
	"strings"

	"github.com/golang-jwt/jwt"
)

// Secret key for signing JWT tokens
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

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
			writeJSONError(w, "Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		// Extract the token from the header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			writeJSONError(w, "Invalid Authorization Header", http.StatusUnauthorized)
			return
		}

		// Validate the token
		token, err := ValidateJWT(tokenString)
		if err != nil || !token.Valid {
			writeJSONError(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		// If token is valid, proceed to the next handler
		next.ServeHTTP(w, r)
	})
}

func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := data.JsonResponse{
		Error:   true,
		Message: message,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Could not process response", http.StatusInternalServerError)
		return
	}

	w.Write(jsonResp)
}
