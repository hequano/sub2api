//go:build unit

package admin

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type capVerifierSpy struct {
	called int
	url    string
	secret string
	token  string
}

func (s *capVerifierSpy) VerifyToken(ctx context.Context, apiEndpoint, siteKey, secret, token, remoteIP string) (*service.CapVerifyResponse, error) {
	s.called++
	s.secret = secret
	s.token = token
	return &service.CapVerifyResponse{Success: true}, nil
}

func TestUpdateSettings_Cap_RejectsMultipleProviders(t *testing.T) {
	h, _ := newStepUpSwitchTestHandler(t, map[string]string{
		service.SettingKeyTurnstileEnabled: "true",
	})

	rec := doUpdateSettings(t, h, map[string]any{
		"cap_enabled":        true,
		"cap_api_endpoint":   "https://cap.example.com",
		"cap_site_key":       "site-key",
		"cap_secret_key":     "secret-key",
	}, nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Multiple captcha providers")
}

func TestUpdateSettings_Cap_RequiresMandatoryFields(t *testing.T) {
	h, _ := newStepUpSwitchTestHandler(t, nil)

	rec := doUpdateSettings(t, h, map[string]any{
		"cap_enabled":      true,
		"cap_api_endpoint": "",
		"cap_site_key":     "site-key",
		"cap_secret_key":   "secret-key",
	}, nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Cap API Endpoint is required")

	rec2 := doUpdateSettings(t, h, map[string]any{
		"cap_enabled":      true,
		"cap_api_endpoint": "https://cap.example.com",
		"cap_site_key":     "",
		"cap_secret_key":   "secret-key",
	}, nil)

	require.Equal(t, http.StatusBadRequest, rec2.Code)
	require.Contains(t, rec2.Body.String(), "Cap Site Key is required")
}

func TestUpdateSettings_Cap_SuccessAndSecretRetention(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, nil)
	verifier := &capVerifierSpy{}
	h.SetCapService(service.NewCapService(h.settingService, verifier))

	// First save: set all fields
	rec := doUpdateSettings(t, h, map[string]any{
		"cap_enabled":      true,
		"cap_api_endpoint": "https://cap.example.com",
		"cap_site_key":     "test-site-key",
		"cap_secret_key":   "test-secret-key",
	}, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyCapEnabled])
	require.Equal(t, "https://cap.example.com", repo.values[service.SettingKeyCapApiEndpoint])
	require.Equal(t, "test-site-key", repo.values[service.SettingKeyCapSiteKey])
	require.Equal(t, "test-secret-key", repo.values[service.SettingKeyCapSecretKey])
	require.Equal(t, 1, verifier.called)

	// Second save: omit secret key, should retain existing secret without error
	rec2 := doUpdateSettings(t, h, map[string]any{
		"cap_enabled":      true,
		"cap_api_endpoint": "https://cap.example.com",
		"cap_site_key":     "test-site-key",
	}, nil)

	require.Equal(t, http.StatusOK, rec2.Code)
	require.Equal(t, "test-secret-key", repo.values[service.SettingKeyCapSecretKey])
}
