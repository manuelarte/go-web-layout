package grpc

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"buf.build/go/protovalidate"
	protovalidatemiddleware "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/manuelarte/go-web-layout/internal/infrastructure/api/grpc/users/v1"
	"github.com/manuelarte/go-web-layout/internal/users"
)

func TestServer_CreateUser_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		request *usersv1.CreateUserRequest
		wantErr string
	}{
		"username and password are empty": {
			request: &usersv1.CreateUserRequest{
				Username: "",
				Password: "",
			},
			wantErr: "username: value is required",
		},
		"username not sent": {
			request: &usersv1.CreateUserRequest{
				Username: "",
				Password: "MyPassword",
			},
			wantErr: "username: value is required",
		},
		"username is too long": {
			request: &usersv1.CreateUserRequest{
				Username: strings.Repeat("a", 600),
				Password: "MyPassword",
			},
			wantErr: "username: must be at most 32 characters",
		},
		"password not present": {
			request: &usersv1.CreateUserRequest{
				Username: "MyUsername",
				Password: "",
			},
			wantErr: "password: value is required",
		},
		"password too short": {
			request: &usersv1.CreateUserRequest{
				Username: "MyUsername",
				Password: "a",
			},
			wantErr: "password: must be at least 8 characters",
		},
		"password too long": {
			request: &usersv1.CreateUserRequest{
				Username: "MyUsername",
				Password: strings.Repeat("a", 600),
			},
			wantErr: "password: must be at most 64 characters",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctx := t.Context()
			ctrl := gomock.NewController(t)
			usersService := users.NewMockRepository(ctrl)
			listener := setup(t, ctx, NewServer(usersService))

			resolver.SetDefaultScheme("passthrough")

			conn, errClient := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, errClient)

			defer conn.Close()

			client := usersv1.NewUsersServiceClient(conn)

			// Act
			_, err := client.CreateUser(ctx, test.request)

			// Assert
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestServer_CreateUser_Successful(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		request  *usersv1.CreateUserRequest
		response func(userCreated users.User) *usersv1.CreateUserResponse
	}{
		"username and password are valid": {
			request: &usersv1.CreateUserRequest{
				Username: "John",
				Password: "12345678",
			},
			response: func(userCreated users.User) *usersv1.CreateUserResponse {
				return &usersv1.CreateUserResponse{
					User: &usersv1.User{
						Id:        userCreated.ID.String(),
						CreatedAt: timestamppb.New(userCreated.CreatedAt),
						UpdatedAt: timestamppb.New(userCreated.UpdatedAt),
						Username:  string(userCreated.Username),
					},
				}
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctx := t.Context()
			ctrl := gomock.NewController(t)
			usersService := users.NewMockRepository(ctrl)
			listener := setup(t, ctx, NewServer(usersService))

			resolver.SetDefaultScheme("passthrough")

			conn, errClient := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, errClient)

			defer conn.Close()

			client := usersv1.NewUsersServiceClient(conn)

			// Assert mocks
			userCreated := users.User{
				ID:        users.UserID{},
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				Username:  users.Username(test.request.GetUsername()),
			}
			usersService.EXPECT().Create(
				gomock.Any(),
				gomock.Eq(users.Username(test.request.GetUsername())),
				gomock.Eq(users.Password(test.request.GetPassword())),
			).Return(userCreated, nil)

			// Act
			resp, err := client.CreateUser(ctx, test.request)

			// Assert
			require.NoError(t, err)
			assertCreateUsersResponse(t, resp, test.response(userCreated))
		})
	}
}

func assertCreateUsersResponse(t *testing.T, resp, response *usersv1.CreateUserResponse) {
	t.Helper()

	assert.Equal(t, resp.GetUser().GetId(), response.GetUser().GetId())
	assert.Equal(t, resp.GetUser().GetCreatedAt(), response.GetUser().GetCreatedAt())
	assert.Equal(t, resp.GetUser().GetUpdatedAt(), response.GetUser().GetUpdatedAt())
	assert.Equal(t, resp.GetUser().GetUsername(), response.GetUser().GetUsername())
}

func setup(t *testing.T, ctx context.Context, server Server) *bufconn.Listener {
	t.Helper()

	validator, errValidator := protovalidate.New()
	require.NoError(t, errValidator)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(protovalidatemiddleware.UnaryServerInterceptor(validator)),
	)
	usersv1.RegisterUsersServiceServer(grpcServer, server)

	errorGroup, ctx := errgroup.WithContext(ctx)

	const bufSize = 1024 * 1024

	lis := bufconn.Listen(bufSize)

	errorGroup.Go(func() error {
		errGrpc := grpcServer.Serve(lis)
		if errGrpc != nil {
			t.Fatalf("failed to serve grpc service: %q", errGrpc)
		}

		return nil
	})

	errorGroup.Go(func() error {
		<-ctx.Done()
		grpcServer.GracefulStop()

		return ctx.Err()
	})

	return lis
}
