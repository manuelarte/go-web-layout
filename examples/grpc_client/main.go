// Package main is a gRPC client example to test the gRPC server.
package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	usersv2 "github.com/manuelarte/go-web-layout/internal/infrastructure/api/grpc/users/v1"
)

func main() {
	grpcClient, err := grpc.NewClient(":3002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer grpcClient.Close()

	client := usersv2.NewUsersServiceClient(grpcClient)

	request := usersv2.CreateUserRequest{
		Username: "other",
		Password: "otherLongPassword",
	}

	resp, err := client.CreateUser(context.Background(), &request)
	if err != nil {
		panic(err)
	}

	//nolint:forbidigo // example
	fmt.Printf("User created: %v\n", resp.GetUser())
}
