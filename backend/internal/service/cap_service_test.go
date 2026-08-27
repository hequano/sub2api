//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type capVerifierStub struct {
	response *CapVerifyResponse
	err      error
	called   int
	endpoint string
	siteKey  string
	secret   string
	token    string
	remoteIP string
}

func (v *capVerifierStub) VerifyToken(ctx context.Context, apiEndpoint, siteKey, secretKey, token, remoteIP string) (*CapVerifyResponse, error) {
	v.called++
	v.endpoint = apiEndpoint
	v.siteKey = siteKey
	v.secret = secretKey
	v.token = token
	v.remoteIP = remoteIP
	if v.err != nil {
		return nil, v.err
	}
	return v.response, nil
}

func TestCapService_VerifyTokenWithConfig(t *testing.T) {
	cfg := CapConfig{
		Enabled:     true,
		ApiEndpoint: "https://cap.example.com",
		SiteKey:     "test-site-key",
		SecretKey:   "test-secret-key",
	}

	t.Run("success", func(t *testing.T) {
		stub := &capVerifierStub{
			response: &CapVerifyResponse{Success: true},
		}
		svc := NewCapService(nil, stub)

		err := svc.VerifyTokenWithConfig(context.Background(), cfg, "valid-token", "1.2.3.4")
		require.NoError(t, err)
		require.Equal(t, 1, stub.called)
		require.Equal(t, "https://cap.example.com", stub.endpoint)
		require.Equal(t, "test-site-key", stub.siteKey)
		require.Equal(t, "test-secret-key", stub.secret)
		require.Equal(t, "valid-token", stub.token)
		require.Equal(t, "1.2.3.4", stub.remoteIP)
	})

	t.Run("empty token", func(t *testing.T) {
		stub := &capVerifierStub{}
		svc := NewCapService(nil, stub)

		err := svc.VerifyTokenWithConfig(context.Background(), cfg, "", "1.2.3.4")
		require.ErrorIs(t, err, ErrCapVerificationFailed)
		require.Equal(t, 0, stub.called)
	})

	t.Run("missing config", func(t *testing.T) {
		stub := &capVerifierStub{}
		svc := NewCapService(nil, stub)

		err := svc.VerifyTokenWithConfig(context.Background(), CapConfig{
			Enabled:     true,
			ApiEndpoint: "",
			SiteKey:     "key",
			SecretKey:   "secret",
		}, "token", "1.2.3.4")
		require.ErrorIs(t, err, ErrCapNotConfigured)
	})

	t.Run("verification rejected by upstream", func(t *testing.T) {
		stub := &capVerifierStub{
			response: &CapVerifyResponse{
				Success:    false,
				ErrorCodes: []string{"invalid-token"},
			},
		}
		svc := NewCapService(nil, stub)

		err := svc.VerifyTokenWithConfig(context.Background(), cfg, "bad-token", "1.2.3.4")
		require.ErrorIs(t, err, ErrCapVerificationFailed)
	})

	t.Run("network error", func(t *testing.T) {
		stub := &capVerifierStub{
			err: errors.New("connection refused"),
		}
		svc := NewCapService(nil, stub)

		err := svc.VerifyTokenWithConfig(context.Background(), cfg, "token", "1.2.3.4")
		require.ErrorIs(t, err, ErrCapVerificationFailed)
	})
}

func TestCapService_ValidateCredentials(t *testing.T) {
	t.Run("valid credentials", func(t *testing.T) {
		stub := &capVerifierStub{
			response: &CapVerifyResponse{
				Success:    false,
				ErrorCodes: []string{"invalid-input-response"}, // Dummy token rejected, but secret is valid
			},
		}
		svc := NewCapService(nil, stub)

		err := svc.ValidateCredentials(context.Background(), "https://cap.example.com", "site", "secret")
		require.NoError(t, err)
		require.Equal(t, 1, stub.called)
	})

	t.Run("invalid secret key error code", func(t *testing.T) {
		stub := &capVerifierStub{
			response: &CapVerifyResponse{
				Success:    false,
				ErrorCodes: []string{"invalid-input-secret"},
			},
		}
		svc := NewCapService(nil, stub)

		err := svc.ValidateCredentials(context.Background(), "https://cap.example.com", "site", "bad-secret")
		require.ErrorIs(t, err, ErrCapInvalidSecretKey)
	})

	t.Run("incomplete parameters", func(t *testing.T) {
		stub := &capVerifierStub{}
		svc := NewCapService(nil, stub)

		err := svc.ValidateCredentials(context.Background(), "", "site", "secret")
		require.ErrorIs(t, err, ErrCapNotConfigured)
	})
}
