package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode private key PEM")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		privateKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
		return privateKey, nil
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode public key PEM")
	}

	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		publicKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return publicKey, nil
	}

	certificate, err := x509.ParseCertificate(block.Bytes)
	if err == nil {
		publicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("certificate public key is not RSA")
		}
		return publicKey, nil
	}

	return nil, errors.New("failed to parse RSA public key")
}
