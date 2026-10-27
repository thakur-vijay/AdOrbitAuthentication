package grpcserver

import (
	"context"

	"github.com/adorbit/authentication/internal/sessions"
	authv1 "github.com/thakur-vijay/adorbit-go/proto/authv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	sessionService *sessions.Service
}

func NewAuthServer(sessionService *sessions.Service) *AuthServer {
	return &AuthServer{
		sessionService: sessionService,
	}
}

func (s *AuthServer) ValidateAccessToken(
	ctx context.Context,
	req *authv1.ValidateAccessTokenRequest,
) (*authv1.ValidateAccessTokenResponse, error) {
	if req.GetAccessToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "access token is required")
	}

	session, err := s.sessionService.ValidateAccessToken(ctx, req.GetAccessToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication failed")
	}

	return &authv1.ValidateAccessTokenResponse{
		UserId:    session.UserID,
		SessionId: session.ID,
	}, nil
}
