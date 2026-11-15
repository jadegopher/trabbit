package log

import (
	"context"
	"errors"
	"net/url"
)

// FailureContext treats superseded requests as diagnostics. URL errors omit the
// request URL because it can contain credentials or sensitive query parameters.
func FailureContext(logger Logger, ctx context.Context, message string, err error, attrs ...any) {
	detail := err
	var requestError *url.Error
	if errors.As(err, &requestError) {
		detail = requestError.Err
	}
	attrs = append(attrs, Err, detail)
	if errors.Is(err, context.Canceled) {
		logger.DebugContext(ctx, message, attrs...)
		return
	}
	logger.ErrorContext(ctx, message, attrs...)
}
