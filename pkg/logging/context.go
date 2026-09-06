package logging

import "context"

type contextKey string

const RequestIDKey contextKey = "requestID"

func GetRequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(RequestIDKey).(string)
	return id, ok
}
