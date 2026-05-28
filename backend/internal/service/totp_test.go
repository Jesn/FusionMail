package service

import "testing"

func TestGenerateBackupCodesReturnsUniqueCodes(t *testing.T) {
	totpService := NewTOTPService("FusionMail")

	codes, err := totpService.GenerateBackupCodes(100)
	if err != nil {
		t.Fatalf("GenerateBackupCodes returned error: %v", err)
	}
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if _, exists := seen[code]; exists {
			t.Fatalf("expected unique backup code, got duplicate %q", code)
		}
		seen[code] = struct{}{}
	}
}

func TestTwoFactorSecretEncryptionRoundTrip(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "test-two-factor-secret-key-32-bytes")

	totpService := NewTOTPService("FusionMail")
	secret := "JBSWY3DPEHPK3PXP"

	encrypted, err := totpService.EncryptSecret(secret)
	if err != nil {
		t.Fatalf("EncryptSecret returned error: %v", err)
	}
	if encrypted == secret {
		t.Fatal("expected encrypted secret to differ from plaintext")
	}

	decrypted, err := totpService.DecryptSecret(encrypted)
	if err != nil {
		t.Fatalf("DecryptSecret returned error: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("expected decrypted secret %q, got %q", secret, decrypted)
	}

	legacySecret, err := totpService.DecryptSecret(secret)
	if err != nil {
		t.Fatalf("DecryptSecret should accept legacy plaintext secret: %v", err)
	}
	if legacySecret != secret {
		t.Fatalf("expected legacy secret %q, got %q", secret, legacySecret)
	}
}

func TestBackupCodesAreStoredAsHashesAndConsumedOnce(t *testing.T) {
	totpService := NewTOTPService("FusionMail")
	codes := []string{"12345678", "87654321"}

	hashedCodes, err := totpService.HashBackupCodes(codes)
	if err != nil {
		t.Fatalf("HashBackupCodes returned error: %v", err)
	}
	if len(hashedCodes) != len(codes) {
		t.Fatalf("expected %d hashes, got %d", len(codes), len(hashedCodes))
	}
	for i, hashedCode := range hashedCodes {
		if hashedCode == codes[i] {
			t.Fatalf("expected backup code %d to be hashed", i)
		}
	}

	valid, remaining, err := totpService.ValidateBackupCode(hashedCodes, "12345678")
	if err != nil {
		t.Fatalf("ValidateBackupCode returned error: %v", err)
	}
	if !valid {
		t.Fatal("expected hashed backup code to validate")
	}
	if len(remaining) != 1 {
		t.Fatalf("expected one remaining code, got %d", len(remaining))
	}

	valid, _, err = totpService.ValidateBackupCode(remaining, "12345678")
	if err != nil {
		t.Fatalf("ValidateBackupCode returned error: %v", err)
	}
	if valid {
		t.Fatal("expected consumed backup code to be rejected")
	}

	valid, _, err = totpService.ValidateBackupCode(remaining, "87654321")
	if err != nil {
		t.Fatalf("ValidateBackupCode returned error: %v", err)
	}
	if !valid {
		t.Fatal("expected remaining backup code to validate")
	}
}

func TestBackupCodeValidationSupportsLegacyPlaintextAndNormalizesRemaining(t *testing.T) {
	totpService := NewTOTPService("FusionMail")
	legacyCodes := []string{"11111111", "22222222"}

	valid, remaining, err := totpService.ValidateBackupCode(legacyCodes, "11111111")
	if err != nil {
		t.Fatalf("ValidateBackupCode returned error: %v", err)
	}
	if !valid {
		t.Fatal("expected legacy plaintext backup code to validate")
	}
	if len(remaining) != 1 {
		t.Fatalf("expected one remaining code, got %d", len(remaining))
	}
	if remaining[0] == "22222222" {
		t.Fatal("expected remaining legacy backup code to be normalized to a hash")
	}

	valid, _, err = totpService.ValidateBackupCode(remaining, "22222222")
	if err != nil {
		t.Fatalf("ValidateBackupCode returned error: %v", err)
	}
	if !valid {
		t.Fatal("expected normalized legacy backup code to validate")
	}
}
