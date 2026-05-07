package wideevents

import (
	"context"
	"log/slog"
	"sync"

	"google.golang.org/grpc"

	"github.com/manuelarte/go-web-layout/internal/config/logging"
)

type (
	createUserLogKey struct{}

	createUserLogEvent struct {
		mu       sync.RWMutex
		Username string `json:"username"`
		UserID   string `json:"userId"`
		Error    createUserError
	}

	createUserError struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
)

func AddCreateUserLogEvent(ctx context.Context) context.Context {
	event := &createUserLogEvent{}
	return context.WithValue(ctx, createUserLogKey{}, event)
}

func (we *createUserLogEvent) isSuccessful() bool {
	return !we.isError()
}

func (we *createUserLogEvent) isError() bool {
	we.mu.RLock()
	defer we.mu.RUnlock()

	return we.Error.Type != ""
}

func (we *createUserLogEvent) mapToArgs() []any {
	we.mu.RLock()
	defer we.mu.RUnlock()

	return []any{
		slog.String("username", we.Username),
		slog.String("userId", we.UserID),
		slog.Group("error", slog.String("type", we.Error.Type), slog.String("message", we.Error.Message)),
	}
}

// AddCreateUserWideEvent returns a gRPC unary server interceptor that injects
// the create user wide event into the context.
func AddCreateUserWideEvent(injectWideEventFn func(ctx context.Context, req any) (context.Context, bool)) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		event := &createUserLogEvent{}

		ctx, ok := injectWideEventFn(ctx, req)
		toReturnAny, toReturnErr := handler(ctx, req)
		if !ok {
			return handler(ctx, req)
		}

		if event.isSuccessful() {
			logging.FromContext(ctx).InfoContext(
				ctx,
				"User created",
				event.mapToArgs()...,
			)
		} else {
			logging.FromContext(ctx).ErrorContext(
				ctx,
				"Error creating user",
				event.mapToArgs()...,
			)
		}

		return toReturnAny, toReturnErr
	}
}

func AddUsername(ctx context.Context, username string) {
	event, ok := ctx.Value(createUserLogKey{}).(*createUserLogEvent)
	if !ok {
		return
	}

	event.mu.Lock()
	defer event.mu.Unlock()

	event.Username = username
}

func AddUserID(ctx context.Context, userID string) {
	event, ok := ctx.Value(createUserLogKey{}).(*createUserLogEvent)
	if !ok {
		return
	}

	event.mu.Lock()
	defer event.mu.Unlock()

	event.UserID = userID
}

func AddError(ctx context.Context, errType string, err error) {
	event, ok := ctx.Value(createUserLogKey{}).(*createUserLogEvent)
	if !ok {
		return
	}

	event.mu.Lock()
	defer event.mu.Unlock()

	event.Error = createUserError{
		Type:    errType,
		Message: err.Error(),
	}
}
