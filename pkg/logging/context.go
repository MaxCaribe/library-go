package logging

import "context"

const RequestIDKey = "requestID"

func GetRequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(RequestIDKey).(string)
	return id, ok
}
