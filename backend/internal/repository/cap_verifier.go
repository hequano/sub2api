package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type capVerifier struct {
	httpClient *http.Client
}

func NewCapVerifier() service.CapVerifier {
	sharedClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:            10 * time.Second,
		ValidateResolvedIP: false,
		AllowPrivateHosts:  true,
	})
	if err != nil {
		sharedClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &capVerifier{
		httpClient: sharedClient,
	}
}

// BuildCapVerifyURL 规范化 Cap 服务端校验接口 URL: {apiEndpoint}/{siteKey}/siteverify
func BuildCapVerifyURL(apiEndpoint, siteKey string) string {
	endpoint, err := service.NormalizeCapAPIEndpoint(apiEndpoint)
	if err != nil || service.ValidateCapSiteKey(siteKey) != nil {
		return ""
	}
	siteKey = strings.TrimSpace(siteKey)
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	if path.Base(strings.TrimRight(parsed.Path, "/")) != siteKey {
		parsed.Path = path.Join(parsed.Path, siteKey)
	}
	parsed.Path = path.Join(parsed.Path, "siteverify")
	return parsed.String()
}

type capVerifyRequestBody struct {
	Secret   string `json:"secret"`
	Response string `json:"response"`
	RemoteIP string `json:"remoteip,omitempty"`
}

func (v *capVerifier) VerifyToken(ctx context.Context, apiEndpoint, siteKey, secretKey, token, remoteIP string) (*service.CapVerifyResponse, error) {
	verifyURL := BuildCapVerifyURL(apiEndpoint, siteKey)
	if verifyURL == "" {
		return nil, fmt.Errorf("invalid cap verification endpoint")
	}

	reqBody := capVerifyRequestBody{
		Secret:   secretKey,
		Response: token,
	}
	if remoteIP != "" {
		reqBody.RemoteIP = remoteIP
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result service.CapVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return nil, fmt.Errorf("decode response: %w", err)
	}
	result.StatusCode = resp.StatusCode

	return &result, nil
}
