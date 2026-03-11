package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

// Headers repassados pelo API Gateway (Lambda Authorizer).
// O serviço não valida JWT; apenas lê a identidade injetada pelo Gateway.
const (
	HeaderUserID    = "X-User-Id"
	HeaderUserEmail = "X-User-Email"
)

type contextKey string

const (
	contextKeyUserID    contextKey = "user_id"
	contextKeyUserEmail contextKey = "user_email"
)

// RequireUserID rejeita com 401 se X-User-Id estiver ausente ou inválido.
// Injeta user_id e user_email no context para os handlers
func RequireUserID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromRequest(r)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing or invalid X-User-Id header"})
			return
		}
		email := r.Header.Get(HeaderUserEmail)
		ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
		ctx = context.WithValue(ctx, contextKeyUserEmail, email)
		next(w, r.WithContext(ctx))
	}
}

// UserIDFromContext retorna o user_id injetado pelo middleware (header X-User-Id do API Gateway).
func UserIDFromContext(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(contextKeyUserID).(int)
	return id, ok
}

// UserEmailFromContext retorna o email injetado pelo middleware (header X-User-Email).
func UserEmailFromContext(ctx context.Context) string {
	s, _ := ctx.Value(contextKeyUserEmail).(string)
	return s
}

func userIDFromRequest(r *http.Request) (int, bool) {
	s := r.Header.Get(HeaderUserID)
	if s == "" {
		return 0, false
	}
	id, err := strconv.Atoi(s)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
