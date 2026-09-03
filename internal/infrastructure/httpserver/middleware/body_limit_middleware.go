package middleware

import "net/http"

const defaultMaxBodyBytes int64 = 1_000_000 // 1 MB

type BodyLimitMiddleware struct {
	maxBytes int64
}

func NewBodyLimitMiddleware(maxBytes int64) *BodyLimitMiddleware {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBodyBytes
	}
	return &BodyLimitMiddleware{maxBytes: maxBytes}
}

func (m *BodyLimitMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, m.maxBytes)
		next.ServeHTTP(w, r)
	})
}
