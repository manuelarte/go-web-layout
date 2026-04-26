// Package usersv1 contains the gRPC server implementation for the users service.
package usersv1

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/manuelarte/go-web-layout/internal/logging"
	"github.com/manuelarte/go-web-layout/internal/observability"
	"github.com/manuelarte/go-web-layout/internal/users"
)

type Server struct {
	UnimplementedUsersServiceServer

	userRepository users.Repository
}

func NewServer(userRepository users.Repository) Server {
	return Server{
		userRepository: userRepository,
	}
}

// CreateUser creates a new user.
func (s Server) CreateUser(ctx context.Context, request *CreateUserRequest) (*CreateUserResponse, error) {
	ctx, span := observability.StartSpan(ctx, "Server.CreateUser")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "username",
			Value: attribute.StringValue(request.GetUsername()),
		},
	)

	user, err := users.NewUser(
		ctx,
		users.Username(request.GetUsername()),
		users.Password(request.GetPassword()),
		s.userRepository,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	logging.FromContext(ctx).InfoContext(
		ctx,
		"User created",
		slog.String("username", string(user.Username)),
		slog.String("userID", user.ID.String()),
	)

	return &CreateUserResponse{
		User: new(transformUser(user)),
	}, nil
}

// DeleteUser deletes a user.
func (s Server) DeleteUser(_ context.Context, _ *DeleteUserRequest) (*DeleteUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUser not implemented")
}

func transformUser(user users.User) User {
	return User{
		Id:        user.ID.String(),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
		Username:  string(user.Username),
	}
}
