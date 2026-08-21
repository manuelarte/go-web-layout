package users

import (
	"context"
	"log/slog"

	"github.com/manuelarte/logevent"
)

var _ logevent.LogEvent[*slog.Logger] = new(CreateUserLogEvent)

type (
	CreateUserLogEvent struct {
		UserID   string
		Username string
		Error    *CreateUserErrorLogEvent
	}

	CreateUserErrorLogEvent struct {
		Type string
		Err  error
	}
)

func (c CreateUserLogEvent) Log(ctx context.Context, li *slog.Logger) {
	if c.Error != nil {
		li.ErrorContext(ctx, "Error creating user", slog.Any("err", c.Error.Err), slog.String("errorType", c.Error.Type))

		return
	}

	li.InfoContext(ctx, "User created", slog.String("userID", c.UserID))
}
