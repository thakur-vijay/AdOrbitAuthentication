package grpcserver

import (
	"net"

	"github.com/adorbit/authentication/internal/sessions"
	authv1 "github.com/thakur-vijay/adorbit-go/proto/authv1"
	"google.golang.org/grpc"
)

func Start(address string, sessionService *sessions.Service) (*grpc.Server, net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}

	server := grpc.NewServer()

	authv1.RegisterAuthServiceServer(
		server,
		NewAuthServer(sessionService),
	)

	go func() {
		_ = server.Serve(listener)
	}()

	return server, listener, nil
}
