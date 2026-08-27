//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func newAuthServiceForCaptchaRepoTest(repo *settingRepoStub, required bool, turnstileVerifier TurnstileVerifier, tencentVerifier TencentCaptchaVerifier) *AuthService {
	cfg := &config.Config{
		Server:    config.ServerConfig{Mode: "release"},
		Turnstile: config.TurnstileConfig{Required: required},
	}
	settingService := NewSettingService(repo, cfg)
	turnstileService := NewTurnstileService(settingService, turnstileVerifier)
	tencentService := NewTencentCaptchaService(settingService, tencentVerifier)
	svc := NewAuthService(nil, &userRepoStub{}, nil, nil, cfg, settingService, nil, turnstileService, nil, nil, nil, nil, nil)
	svc.SetTencentCaptchaService(tencentService)
	return svc
}

func newAuthServiceForCaptchaTest(settings map[string]string, required bool, turnstileVerifier TurnstileVerifier, tencentVerifier TencentCaptchaVerifier) *AuthService {
	cfg := &config.Config{
		Server:    config.ServerConfig{Mode: "release"},
		Turnstile: config.TurnstileConfig{Required: required},
	}
	settingService := NewSettingService(&settingRepoStub{values: settings}, cfg)
	var turnstileService *TurnstileService
	if turnstileVerifier != nil {
		turnstileService = NewTurnstileService(settingService, turnstileVerifier)
	}
	svc := NewAuthService(nil, &userRepoStub{}, nil, nil, cfg, settingService, nil, turnstileService, nil, nil, nil, nil, nil)
	if tencentVerifier != nil {
		svc.SetTencentCaptchaService(NewTencentCaptchaService(settingService, tencentVerifier))
	}
	return svc
}

func tencentCaptchaSettings() map[string]string {
	return map[string]string{
		SettingKeyTencentCaptchaEnabled:        "true",
		SettingKeyTencentCaptchaAppID:          "123456789",
		SettingKeyTencentCaptchaAppSecretKey:   "app-secret",
		SettingKeyTencentCaptchaCloudSecretID:  "cloud-secret-id",
		SettingKeyTencentCaptchaCloudSecretKey: "cloud-secret-key",
	}
}

func TestVerifyCaptchaUsesTencentWhenEnabled(t *testing.T) {
	verifier := &tencentCaptchaVerifierStub{response: &TencentCaptchaVerifyResponse{CaptchaCode: 1}}
	svc := newAuthServiceForCaptchaTest(tencentCaptchaSettings(), false, nil, verifier)

	err := svc.VerifyCaptcha(context.Background(), CaptchaProof{
		TencentTicket:  "ticket",
		TencentRandstr: "@rand",
	}, "203.0.113.10")

	require.NoError(t, err)
	require.Equal(t, 1, verifier.calls)
}

func TestVerifyCaptchaRejectsDirtyDoubleEnabledSettings(t *testing.T) {
	settings := tencentCaptchaSettings()
	settings[SettingKeyTurnstileEnabled] = "true"
	settings[SettingKeyTurnstileSecretKey] = "turnstile-secret"
	turnstileVerifier := &turnstileVerifierSpy{}
	tencentVerifier := &tencentCaptchaVerifierStub{response: &TencentCaptchaVerifyResponse{CaptchaCode: 1}}
	svc := newAuthServiceForCaptchaTest(settings, false, turnstileVerifier, tencentVerifier)

	err := svc.VerifyCaptcha(context.Background(), CaptchaProof{
		TurnstileToken: "turnstile-token",
		TencentTicket:  "ticket",
		TencentRandstr: "@rand",
	}, "203.0.113.10")

	require.ErrorIs(t, err, ErrCaptchaProviderConflict)
	require.Zero(t, turnstileVerifier.called)
	require.Zero(t, tencentVerifier.calls)
}

func TestVerifyCaptchaRequiredModeAcceptsCompleteTencentProvider(t *testing.T) {
	verifier := &tencentCaptchaVerifierStub{response: &TencentCaptchaVerifyResponse{CaptchaCode: 1}}
	svc := newAuthServiceForCaptchaTest(tencentCaptchaSettings(), true, nil, verifier)

	err := svc.VerifyCaptcha(context.Background(), CaptchaProof{
		TencentTicket:  "ticket",
		TencentRandstr: "@rand",
	}, "203.0.113.10")

	require.NoError(t, err)
}

func TestVerifyCaptchaForRegisterSkipsDuplicateTencentTicketAfterEmailCode(t *testing.T) {
	settings := tencentCaptchaSettings()
	settings[SettingKeyEmailVerifyEnabled] = "true"
	verifier := &tencentCaptchaVerifierStub{response: &TencentCaptchaVerifyResponse{CaptchaCode: 1}}
	svc := newAuthServiceForCaptchaTest(settings, true, nil, verifier)

	err := svc.VerifyCaptchaForRegister(context.Background(), CaptchaProof{}, "203.0.113.10", "123456")

	require.NoError(t, err)
	require.Zero(t, verifier.calls)
}

func TestVerifyCaptchaFailsClosedWhenProviderSettingsCannotBeRead(t *testing.T) {
	repo := &settingRepoStub{err: errors.New("settings unavailable")}
	svc := newAuthServiceForCaptchaRepoTest(repo, false, &turnstileVerifierSpy{}, &tencentCaptchaVerifierStub{})

	err := svc.VerifyCaptcha(context.Background(), CaptchaProof{}, "203.0.113.10")

	require.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestVerifyCaptchaReadsProviderConfigurationOnce(t *testing.T) {
	repo := &settingRepoStub{values: tencentCaptchaSettings()}
	verifier := &tencentCaptchaVerifierStub{response: &TencentCaptchaVerifyResponse{CaptchaCode: 1}}
	svc := newAuthServiceForCaptchaRepoTest(repo, false, &turnstileVerifierSpy{}, verifier)

	err := svc.VerifyCaptcha(context.Background(), CaptchaProof{
		TencentTicket:  "ticket",
		TencentRandstr: "@rand",
	}, "203.0.113.10")

	require.NoError(t, err)
	require.Equal(t, 1, repo.getMultipleCalls)
	require.Zero(t, repo.getValueCalls)
	require.Equal(t, 1, verifier.calls)
}

func TestVerifyCaptchaRejectsEnabledTencentProviderWithIncompleteCredentials(t *testing.T) {
	repo := &settingRepoStub{values: map[string]string{
		SettingKeyTencentCaptchaEnabled: "true",
		SettingKeyTencentCaptchaAppID:   "123456789",
	}}
	verifier := &tencentCaptchaVerifierStub{response: &TencentCaptchaVerifyResponse{CaptchaCode: 1}}
	svc := newAuthServiceForCaptchaRepoTest(repo, false, &turnstileVerifierSpy{}, verifier)

	err := svc.VerifyCaptcha(context.Background(), CaptchaProof{
		TencentTicket:  "ticket",
		TencentRandstr: "@rand",
	}, "203.0.113.10")

	require.ErrorIs(t, err, ErrTencentCaptchaNotConfigured)
	require.Equal(t, 1, repo.getMultipleCalls)
	require.Zero(t, verifier.calls)
}

func TestVerifyActionCaptchaIfEnabledVerifiesTencentProof(t *testing.T) {
	verifier := &tencentCaptchaVerifierStub{response: &TencentCaptchaVerifyResponse{CaptchaCode: 1}}
	svc := newAuthServiceForCaptchaTest(tencentCaptchaSettings(), false, nil, verifier)

	err := svc.VerifyActionCaptchaIfEnabled(context.Background(), CaptchaProof{
		TencentTicket:  "ticket",
		TencentRandstr: "@rand",
	}, "203.0.113.10")

	require.NoError(t, err)
	require.Equal(t, 1, verifier.calls)
	require.Equal(t, TencentCaptchaProof{Ticket: "ticket", Randstr: "@rand"}, verifier.proof)
}

func TestVerifyActionCaptchaIfEnabledDoesNotExpandTurnstileCoverage(t *testing.T) {
	settings := map[string]string{
		SettingKeyTurnstileEnabled:   "true",
		SettingKeyTurnstileSecretKey: "turnstile-secret",
	}
	turnstileVerifier := &turnstileVerifierSpy{}
	svc := newAuthServiceForCaptchaTest(settings, false, turnstileVerifier, nil)

	err := svc.VerifyActionCaptchaIfEnabled(context.Background(), CaptchaProof{}, "203.0.113.10")

	require.NoError(t, err)
	require.Zero(t, turnstileVerifier.called)
}

func TestVerifyActionCaptchaIfEnabledFailsClosedOnSettingReadError(t *testing.T) {
	repo := &settingRepoStub{err: errors.New("settings unavailable")}
	svc := newAuthServiceForCaptchaRepoTest(repo, false, &turnstileVerifierSpy{}, &tencentCaptchaVerifierStub{})

	err := svc.VerifyActionCaptchaIfEnabled(context.Background(), CaptchaProof{}, "203.0.113.10")

	require.ErrorIs(t, err, ErrServiceUnavailable)
}

func capCaptchaSettings() map[string]string {
	return map[string]string{
		SettingKeyCapEnabled:     "true",
		SettingKeyCapApiEndpoint: "https://cap.example.com",
		SettingKeyCapSiteKey:     "cap-site-key",
		SettingKeyCapSecretKey:   "cap-secret-key",
	}
}

func TestVerifyCaptchaUsesCapWhenEnabled(t *testing.T) {
	verifier := &capVerifierStub{response: &CapVerifyResponse{Success: true}}
	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
	}
	settingService := NewSettingService(&settingRepoStub{values: capCaptchaSettings()}, cfg)
	svc := NewAuthService(nil, &userRepoStub{}, nil, nil, cfg, settingService, nil, nil, nil, nil, nil, nil, nil)
	svc.SetCapService(NewCapService(settingService, verifier))

	err := svc.VerifyCaptcha(context.Background(), CaptchaProof{
		TurnstileToken: "valid-cap-token",
	}, "203.0.113.10")

	require.NoError(t, err)
	require.Equal(t, 1, verifier.called)
	require.Equal(t, "valid-cap-token", verifier.token)
}

func TestVerifyCaptchaRejectsCapWithTurnstileConflict(t *testing.T) {
	settings := capCaptchaSettings()
	settings[SettingKeyTurnstileEnabled] = "true"
	settings[SettingKeyTurnstileSecretKey] = "turnstile-secret"

	verifier := &capVerifierStub{response: &CapVerifyResponse{Success: true}}
	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
	}
	settingService := NewSettingService(&settingRepoStub{values: settings}, cfg)
	svc := NewAuthService(nil, &userRepoStub{}, nil, nil, cfg, settingService, nil, nil, nil, nil, nil, nil, nil)
	svc.SetCapService(NewCapService(settingService, verifier))

	err := svc.VerifyCaptcha(context.Background(), CaptchaProof{
		TurnstileToken: "token",
	}, "203.0.113.10")

	require.ErrorIs(t, err, ErrCaptchaProviderConflict)
	require.Zero(t, verifier.called)
}

func TestVerifyActionCaptchaIfEnabledVerifiesCapProof(t *testing.T) {
	verifier := &capVerifierStub{response: &CapVerifyResponse{Success: true}}
	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
	}
	settingService := NewSettingService(&settingRepoStub{values: capCaptchaSettings()}, cfg)
	svc := NewAuthService(nil, &userRepoStub{}, nil, nil, cfg, settingService, nil, nil, nil, nil, nil, nil, nil)
	svc.SetCapService(NewCapService(settingService, verifier))

	err := svc.VerifyActionCaptchaIfEnabled(context.Background(), CaptchaProof{
		TurnstileToken: "cap-action-token",
	}, "203.0.113.10")

	require.NoError(t, err)
	require.Equal(t, 1, verifier.called)
	require.Equal(t, "cap-action-token", verifier.token)
}

