package service

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var (
	ErrCapVerificationFailed = infraerrors.BadRequest("CAP_VERIFICATION_FAILED", "cap verification failed")
	ErrCapNotConfigured      = infraerrors.ServiceUnavailable("CAP_NOT_CONFIGURED", "cap captcha not configured")
	ErrCapInvalidSecretKey   = infraerrors.BadRequest("CAP_INVALID_SECRET_KEY", "invalid cap secret key")
)

// CapConfig Cap 人机验证配置
type CapConfig struct {
	Enabled     bool
	ApiEndpoint string
	SiteKey     string
	SecretKey   string
}

// CapVerifier 验证 Cap token 的接口
type CapVerifier interface {
	VerifyToken(ctx context.Context, apiEndpoint, siteKey, secretKey, token, remoteIP string) (*CapVerifyResponse, error)
}

// CapVerifyResponse Cap (TryCap) 验证响应
type CapVerifyResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts,omitempty"`
	Hostname    string   `json:"hostname,omitempty"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
}

// CapService Cap 验证服务
type CapService struct {
	settingService *SettingService
	verifier       CapVerifier
}

// NewCapService 创建 Cap 服务实例
func NewCapService(settingService *SettingService, verifier CapVerifier) *CapService {
	return &CapService{
		settingService: settingService,
		verifier:       verifier,
	}
}

// VerifyTokenWithConfig 使用指定配置验证 Cap token
func (s *CapService) VerifyTokenWithConfig(ctx context.Context, config CapConfig, token, remoteIP string) error {
	if s == nil || s.verifier == nil {
		return ErrCapNotConfigured
	}
	apiEndpoint := strings.TrimSpace(config.ApiEndpoint)
	siteKey := strings.TrimSpace(config.SiteKey)
	secretKey := strings.TrimSpace(config.SecretKey)
	if apiEndpoint == "" || siteKey == "" || secretKey == "" {
		logger.LegacyPrintf("service.cap", "%s", "[Cap] credentials not configured")
		return ErrCapNotConfigured
	}

	if strings.TrimSpace(token) == "" {
		logger.LegacyPrintf("service.cap", "%s", "[Cap] Token is empty")
		return ErrCapVerificationFailed
	}

	logger.LegacyPrintf("service.cap", "[Cap] Verifying token for IP: %s", remoteIP)
	result, err := s.verifier.VerifyToken(ctx, apiEndpoint, siteKey, secretKey, token, remoteIP)
	if err != nil {
		logger.LegacyPrintf("service.cap", "[Cap] Request failed: %v", err)
		return fmt.Errorf("%w: send request: %v", ErrCapVerificationFailed, err)
	}

	if result == nil || !result.Success {
		if result != nil {
			logger.LegacyPrintf("service.cap", "[Cap] Verification failed, error codes: %v", result.ErrorCodes)
		}
		return ErrCapVerificationFailed
	}

	logger.LegacyPrintf("service.cap", "%s", "[Cap] Verification successful")
	return nil
}

// ValidateCredentials 验证 Cap 凭证是否有效（后台保存设置时调用）
func (s *CapService) ValidateCredentials(ctx context.Context, apiEndpoint, siteKey, secretKey string) error {
	if s == nil || s.verifier == nil {
		return ErrCapNotConfigured
	}
	apiEndpoint = strings.TrimSpace(apiEndpoint)
	siteKey = strings.TrimSpace(siteKey)
	secretKey = strings.TrimSpace(secretKey)
	if apiEndpoint == "" || siteKey == "" || secretKey == "" {
		return ErrCapNotConfigured
	}

	result, err := s.verifier.VerifyToken(ctx, apiEndpoint, siteKey, secretKey, "sub2api-credential-validation", "")
	if err != nil {
		return fmt.Errorf("validate cap credentials: %w", err)
	}

	if result != nil {
		for _, code := range result.ErrorCodes {
			if code == "invalid-input-secret" || code == "invalid_secret" {
				return ErrCapInvalidSecretKey
			}
		}
	}

	return nil
}
