package usersclient

import (
	"context"

	usersv1 "github.com/adorbit/authentication/gen/users/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client usersv1.UsersServiceClient
}

func Dial(address string) (*Client, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	return &Client{
		client: usersv1.NewUsersServiceClient(conn),
	}, conn, nil
}

func (c *Client) CreateUser(
	ctx context.Context,
	id string,
	fullName string,
	email string,
	role string,
) (string, error) {
	resp, err := c.client.CreateUser(ctx, &usersv1.CreateUserRequest{
		Id:       id,
		FullName: fullName,
		Email:    email,
		Role:     role,
	})
	if err != nil {
		return "", err
	}

	return resp.GetId(), nil
}
