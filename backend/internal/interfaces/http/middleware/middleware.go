package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/voidwg/control/internal/application/port"
	"github.com/voidwg/control/internal/domain"
	"github.com/voidwg/control/internal/infrastructure/metrics"
)

type ctxKey string

const (
	ctxUserID    ctxKey = "user_id"
	ctxRole      ctxKey = "role"
	ctxRequestID ctxKey = "rid"
)

// RequestID — генерирует X-Request-Id если его нет.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", rid)
		ctx := context.WithValue(r.Context(), ctxRequestID, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger — пишет запрос в zerolog.
func Logger(log *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(ww, r)
			log.Info().
				Str("rid", RequestIDFromCtx(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.status).
				Dur("dur", time.Since(start)).
				Msg("http")
		})
	}
}

// Metrics — Prometheus middleware.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)
		metrics.HTTPRequests.WithLabelValues(r.Method, r.URL.Path, http.StatusText(ww.status)).Inc()
		metrics.HTTPDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
	})
}

// Recover — ловит паники и возвращает 500.
func Recover(log *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error().Interface("panic", rec).Msg("panic recovered")
					http.Error(w, "internal", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Auth — проверка JWT и инжекция UserID/Role в контекст.
func Auth(tokens port.TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims, err := tokens.Verify(strings.TrimPrefix(h, "Bearer "))
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole — RBAC.
func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	allowed := map[domain.Role]struct{}{}
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ctxRole).(domain.Role)
			if _, ok := allowed[role]; !ok {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// helpers

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func UserIDFromCtx(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ctxUserID).(uuid.UUID)
	return v
}

func RoleFromCtx(ctx context.Context) domain.Role {
	v, _ := ctx.Value(ctxRole).(domain.Role)
	return v
}

func RequestIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxRequestID).(string)
	return v
}
