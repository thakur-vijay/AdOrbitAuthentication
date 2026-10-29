package app

import (
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"net/http"

	"github.com/adorbit/authentication/internal/api"
	"github.com/adorbit/authentication/internal/auth"
	"github.com/adorbit/authentication/internal/jwt"
	"github.com/adorbit/authentication/internal/messaging"
	"github.com/thakur-vijay/adorbit-go/middleware"
)

func parsePrivateKey(value string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, errors.New("invalid private key PEM")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		privateKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
		return privateKey, nil
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parsePublicKey(value string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, errors.New("invalid public key PEM")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	publicKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}

	return publicKey, nil
}

func BuildRouter(
	db *sql.DB,
	privateKeyPEM string,
	publicKeyPEM string,
	rabbitMQ *messaging.RabbitMQ,
) (http.Handler, error) {
	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	publicKey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return nil, err
	}

	jwtRepository := jwt.NewRepository(db)
	jwtService := jwt.NewService(
		privateKey,
		publicKey,
		jwtRepository,
		"adorbit-authentication",
	)

	jwtMiddleware, err := middleware.NewJWTMiddleware(
		publicKeyPEM,
		"adorbit-authentication",
	)
	if err != nil {
		return nil, err
	}
	authRepository := auth.NewRepository(db)
	authService := auth.NewService(jwtService, authRepository, rabbitMQ)
	authHandler := auth.NewHandler(authService)

	router := api.NewRouter(api.Dependencies{
		AuthHandler:   authHandler,
		JWTMiddleware: jwtMiddleware,
	})

	return router, nil
}
