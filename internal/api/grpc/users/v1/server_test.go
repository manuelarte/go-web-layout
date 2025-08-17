package usersv1

import (
	"context"
	"net"
	"strings"
	"testing"

	"buf.build/go/protovalidate"
	protovalidatemiddleware "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"

	"github.com/manuelarte/go-web-layout/internal/users"
)

func TestServer_CreateUser_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		request  *CreateUserRequest
		expected string
	}{
		"username and password are empty": {
			request: &CreateUserRequest{
				Username: "",
				Password: "",
			},
			expected: "username: value is required",
		},
		"username not sent": {
			request: &CreateUserRequest{
				Username: "",
				Password: "MyPassword",
			},
			expected: "username: value is required",
		},
		"username is too long": {
			request: &CreateUserRequest{
				Username: strings.Repeat("a", 600),
				Password: "MyPassword",
			},
			expected: "username: value length must be at most 32 characters",
		},
		"password not present": {
			request: &CreateUserRequest{
				Username: "MyUsername",
				Password: "",
			},
			expected: "password: value is required",
		},
		"password too long": {
			request: &CreateUserRequest{
				Username: "MyUsername",
				Password: strings.Repeat("a", 600),
			},
			expected: "password: value length must be at most 64 characters",
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
			require.ErrorContains(t, err, test.expected)
		})
	}
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
