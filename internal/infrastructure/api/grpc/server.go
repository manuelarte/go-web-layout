// Package grpc contains the gRPC server implementation for the services.
package grpc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/manuelarte/go-web-layout/internal/infrastructure/api/grpc/users/v1"
	"github.com/manuelarte/go-web-layout/internal/logging/wideevents"
	"github.com/manuelarte/go-web-layout/internal/observability"
	"github.com/manuelarte/go-web-layout/internal/users"
)

type Server struct {
	usersv1.UnimplementedUsersServiceServer

	userRepository users.Repository
}

func NewServer(userRepository users.Repository) Server {
	return Server{
		userRepository: userRepository,
	}
}

// CreateUser creates a new user.
func (s Server) CreateUser(
	ctx context.Context,
	request *usersv1.CreateUserRequest,
) (*usersv1.CreateUserResponse, error) {
	ctx, span := observability.StartSpan(ctx, "Server.CreateUser")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "username",
			Value: attribute.StringValue(request.GetUsername()),
		},
	)
	wideevents.AddUsername(ctx, request.GetUsername())

	user, err := users.NewUser(
		ctx,
		users.Username(request.GetUsername()),
		users.Password(request.GetPassword()),
		s.userRepository,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	wideevents.AddUserID(ctx, user.ID.String())

	return &usersv1.CreateUserResponse{
		User: new(transformUser(user)),
	}, nil
}

// DeleteUser deletes a user.
func (s Server) DeleteUser(_ context.Context, _ *usersv1.DeleteUserRequest) (*usersv1.DeleteUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUser not implemented")
}

func transformUser(user users.User) usersv1.User {
	return usersv1.User{
		Id:        user.ID.String(),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
		Username:  string(user.Username),
	}
}
