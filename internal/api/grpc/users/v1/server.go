// Package usersv1 contains the gRPC server implementation for the users service.
package usersv1

import (
	"context"
	"fmt"

	"github.com/manuelarte/ptrutils"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/manuelarte/go-web-layout/internal/users"
)

type Server struct {
	UnimplementedUsersServiceServer

	userService users.Service
}

func NewServer(userService users.Service) Server {
	return Server{
		userService: userService,
	}
}

// CreateUser creates a new user.
func (s Server) CreateUser(ctx context.Context, request *CreateUserRequest) (*CreateUserResponse, error) {
	user, err := s.userService.Create(ctx, users.NewUser{
		Username: request.GetUsername(),
		Password: request.GetPassword(),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	log.Info().Msgf("User created: %q", user.ID)

	return &CreateUserResponse{
		User: ptrutils.Ptr(transformUser(user)),
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
		Username:  user.Username,
	}
}
