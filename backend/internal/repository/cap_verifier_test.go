//go:build unit

package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCapVerifyURL(t *testing.T) {
	tests := []struct {
		name        string
		apiEndpoint string
		siteKey     string
		expected    string
	}{
		{
			name:        "standard domain without slash",
			apiEndpoint: "https://cap.example.com",
			siteKey:     "my-site",
			expected:    "https://cap.example.com/my-site/siteverify",
		},
		{
			name:        "domain with trailing slash",
			apiEndpoint: "https://cap.example.com/",
			siteKey:     "my-site",
			expected:    "https://cap.example.com/my-site/siteverify",
		},
		{
			name:        "missing scheme",
			apiEndpoint: "cap.example.com",
			siteKey:     "my-site",
			expected:    "https://cap.example.com/my-site/siteverify",
		},
		{
			name:        "endpoint already includes siteKey",
			apiEndpoint: "https://cap.example.com/my-site",
			siteKey:     "my-site",
			expected:    "https://cap.example.com/my-site/siteverify",
		},
		{
			name:        "endpoint already includes siteKey with trailing slash",
			apiEndpoint: "https://cap.example.com/my-site/",
			siteKey:     "my-site",
			expected:    "https://cap.example.com/my-site/siteverify",
		},
		{
			name:        "http scheme preserved",
			apiEndpoint: "http://localhost:8080",
			siteKey:     "dev-site",
			expected:    "http://localhost:8080/dev-site/siteverify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCapVerifyURL(tt.apiEndpoint, tt.siteKey)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestCapVerifier_VerifyToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/test-site/siteverify", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req capVerifyRequestBody
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		require.Equal(t, "secret-123", req.Secret)
		require.Equal(t, "token-456", req.Response)
		require.Equal(t, "192.0.2.1", req.RemoteIP)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":      true,
			"challenge_ts": "2026-08-27T00:00:00Z",
			"hostname":     "example.com",
		})
	}))
	defer server.Close()

	verifier := &capVerifier{httpClient: server.Client()}
	resp, err := verifier.VerifyToken(context.Background(), server.URL, "test-site", "secret-123", "token-456", "192.0.2.1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Success)
	require.Equal(t, "example.com", resp.Hostname)
}
