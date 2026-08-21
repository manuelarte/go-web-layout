// Package grpc contains the gRPC server implementation for the services.
package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/manuelarte/logevent/mw"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/manuelarte/go-web-layout/internal/config/observability"
	"github.com/manuelarte/go-web-layout/internal/infrastructure/api/grpc/users/v1"
	"github.com/manuelarte/go-web-layout/internal/services"
	"github.com/manuelarte/go-web-layout/internal/users"
)

type Server struct {
	usersv1.UnimplementedUsersServiceServer

	createUserService services.CreateUser
}

func NewServer(createUserService services.CreateUser) Server {
	return Server{
		createUserService: createUserService,
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

	_ = mw.UpdateLogEvent(ctx, func(event *users.CreateUserLogEvent) {
		event.Username = request.GetUsername()
	})

	user, err := s.createUserService.CreateUser(
		ctx,
		users.Username(request.GetUsername()),
		users.Password(request.GetPassword()),
	)
	if err != nil {
		_ = mw.UpdateLogEvent(ctx, func(event *users.CreateUserLogEvent) {
			event.Error = &users.CreateUserErrorLogEvent{
				Type: "db",
				Err:  err,
			}
		})

		return nil, fmt.Errorf("error creating user: %w", err)
	}

	_ = mw.UpdateLogEvent(ctx, func(event *users.CreateUserLogEvent) {
		event.UserID = user.ID().String()
	})

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
		Id:        user.ID().String(),
		CreatedAt: timestamppb.New(user.CreatedAt()),
		UpdatedAt: timestamppb.New(user.UpdatedAt()),
		Username:  string(user.Username()),
	}
}

// ServiceWithSelectiveInterceptor wraps a UsersServiceServer and applies an interceptor
// only to specific methods (currently CreateUser).
type ServiceWithSelectiveInterceptor struct {
	usersv1.UnimplementedUsersServiceServer
	actual      usersv1.UsersServiceServer
	interceptor grpc.UnaryServerInterceptor
	logger      *slog.Logger
}

// NewServiceWithSelectiveInterceptor creates a new UsersServiceServer with an interceptor
// applied selectively to specific methods.
func NewServiceWithSelectiveInterceptor(
	actual usersv1.UsersServiceServer,
	interceptor grpc.UnaryServerInterceptor,
	logger *slog.Logger,
) *ServiceWithSelectiveInterceptor {
	return &ServiceWithSelectiveInterceptor{
		actual:      actual,
		interceptor: interceptor,
		logger:      logger,
	}
}

// CreateUser applies the interceptor before delegating to the actual service.
func (s *ServiceWithSelectiveInterceptor) CreateUser(
	ctx context.Context,
	request *usersv1.CreateUserRequest,
) (*usersv1.CreateUserResponse, error) {
	handler := func(ctx context.Context, req any) (any, error) {
		return s.actual.CreateUser(ctx, req.(*usersv1.CreateUserRequest))
	}

	info := &grpc.UnaryServerInfo{
		Server:     s,
		FullMethod: usersv1.UsersService_CreateUser_FullMethodName,
	}

	resp, err := s.interceptor(ctx, request, info, handler)
	if err != nil {
		return nil, err
	}

	return resp.(*usersv1.CreateUserResponse), nil
}

// DeleteUser passes through to the actual service without interceptor.
func (s *ServiceWithSelectiveInterceptor) DeleteUser(
	ctx context.Context,
	request *usersv1.DeleteUserRequest,
) (*usersv1.DeleteUserResponse, error) {
	return s.actual.DeleteUser(ctx, request)
}
