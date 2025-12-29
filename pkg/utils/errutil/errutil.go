package errutil

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// Handle logs the error with full context
// This ensures that errors are never silently swallowed
func Handle(ctx context.Context, err error, msg string) {
	if err == nil {
		return
	}

	logger := logging.From(ctx)

	// Log error with full stack trace and context
	if goErr, ok := err.(*goerr.Error); ok {
		// goerr.Error contains detailed context
		logger.Error(msg,
			"error", goErr.Error(),
			"values", goErr.Values(),
			"stacks", goErr.Stacks())
	} else {
		// Standard error
		logger.Error(msg, "error", err.Error())
	}
}
