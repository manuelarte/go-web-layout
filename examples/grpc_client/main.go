// Package main is a gRPC client example to test the gRPC server.
package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	usersv1 "github.com/manuelarte/go-web-layout/internal/api/grpc/users/v1"
)

func main() {
	grpcClient, err := grpc.NewClient(":3002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer grpcClient.Close()

	client := usersv1.NewUsersServiceClient(grpcClient)

	request := usersv1.CreateUserRequest{
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
