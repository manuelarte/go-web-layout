package usersv1

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"buf.build/go/protovalidate"
	"github.com/google/uuid"
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

	"github.com/manuelarte/go-web-layout/internal/users"
)

func TestServer_CreateUser_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		request     *CreateUserRequest
		expectedErr string
	}{
		"username and password are empty": {
			request: &CreateUserRequest{
				Username: "",
				Password: "",
			},
			expectedErr: "username: value is required",
		},
		"username not sent": {
			request: &CreateUserRequest{
				Username: "",
				Password: "MyPassword",
			},
			expectedErr: "username: value is required",
		},
		"username is too long": {
			request: &CreateUserRequest{
				Username: strings.Repeat("a", 600),
				Password: "MyPassword",
			},
			expectedErr: "username: value length must be at most 32 characters",
		},
		"password not present": {
			request: &CreateUserRequest{
				Username: "MyUsername",
				Password: "",
			},
			expectedErr: "password: value is required",
		},
		"password too short": {
			request: &CreateUserRequest{
				Username: "MyUsername",
				Password: "a",
			},
			expectedErr: "password: value length must be at least 8 characters",
		},
		"password too long": {
			request: &CreateUserRequest{
				Username: "MyUsername",
				Password: strings.Repeat("a", 600),
			},
			expectedErr: "password: value length must be at most 64 characters",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctx := t.Context()
			ctrl := gomock.NewController(t)
			usersService := users.NewMockService(ctrl)
			listener := setup(t, ctx, NewServer(usersService))

			resolver.SetDefaultScheme("passthrough")

			conn, errClient := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, errClient)

			defer conn.Close()

			client := NewUsersServiceClient(conn)

			// Act
			_, err := client.CreateUser(ctx, test.request)

			// Assert
			require.ErrorContains(t, err, test.expectedErr)
		})
	}
}

func TestServer_CreateUser_Successful(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		request  *CreateUserRequest
		response func(userCreated users.User) *CreateUserResponse
	}{
		"username and password are valid": {
			request: &CreateUserRequest{
				Username: "John",
				Password: "12345678",
			},
			response: func(userCreated users.User) *CreateUserResponse {
				return &CreateUserResponse{
					User: &User{
						Id:        userCreated.ID.String(),
						CreatedAt: timestamppb.New(userCreated.CreatedAt),
						UpdatedAt: timestamppb.New(userCreated.UpdatedAt),
						Username:  userCreated.Username,
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
			usersService := users.NewMockService(ctrl)
			listener := setup(t, ctx, NewServer(usersService))

			resolver.SetDefaultScheme("passthrough")

			conn, errClient := grpc.NewClient("bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, errClient)

			defer conn.Close()

			client := NewUsersServiceClient(conn)

			// Assert mocks
			userCreated := users.User{
				ID:        uuid.UUID{},
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				Username:  test.request.GetUsername(),
			}
			usersService.EXPECT().Create(gomock.Any(), users.NewUser{
				Username: test.request.GetUsername(),
				Password: test.request.GetPassword(),
			}).Return(userCreated, nil)

			// Act
			resp, err := client.CreateUser(ctx, test.request)

			// Assert
			require.NoError(t, err)
			assertCreateUsersResponse(t, resp, test.response(userCreated))
		})
	}
}

func assertCreateUsersResponse(t *testing.T, resp, response *CreateUserResponse) {
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
	RegisterUsersServiceServer(grpcServer, server)

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
