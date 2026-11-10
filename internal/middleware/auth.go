package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"blog-api/internal/auth"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UsernameKey contextKey = "username"
)

// проверка JWT в заголовке Authorization
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "authorization header is required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid authorization format, use 'Bearer <token>'"})
			return
		}

		token := parts[1]
		userID, username, err := auth.ValidateToken(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired token"})
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, userID)
		ctx = context.WithValue(ctx, UsernameKey, username)

		next(w, r.WithContext(ctx))
	}
}

// хелпер для извлечения пользователя из контекста
func GetUserFromContext(ctx context.Context) (int, string, bool) {
	userID, ok1 := ctx.Value(UserIDKey).(int)
	username, ok2 := ctx.Value(UsernameKey).(string)
	if !ok1 || !ok2 {
		return 0, "", false
	}
	return userID, username, true
}