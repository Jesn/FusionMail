package crypto

import (
	"strings"
	"testing"
)

const (
	keyA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 32 字节
	keyB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" // 32 字节
)

// TestKeyRingEncryptCarriesKeyID 校验新密文带版本前缀与 key-id。
func TestKeyRingEncryptCarriesKeyID(t *testing.T) {
	ring, err := NewKeyRing(keyA)
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}

	ciphertext, err := ring.Encrypt("hello")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	parts := strings.SplitN(ciphertext, ":", 3)
	if len(parts) != 3 {
		t.Fatalf("密文格式应为 v2:<keyid>:<body>，实际 %q", ciphertext)
	}
	if parts[0] != CiphertextVersion {
		t.Errorf("版本前缀应为 %q，实际 %q", CiphertextVersion, parts[0])
	}
	if parts[1] != ring.CurrentKeyID() {
		t.Errorf("key-id 应为 %q，实际 %q", ring.CurrentKeyID(), parts[1])
	}

	plaintext, err := ring.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if plaintext != "hello" {
		t.Errorf("往返后应得到 %q，实际 %q", "hello", plaintext)
	}
}

// TestKeyRingDecryptsLegacyCiphertext 校验无 key-id 的历史密文仍可解密。
//
// 这是升级不停机的关键：历史密文由旧版（单密钥、无前缀）写入，
// 升级后必须仍能被解开，否则全部存量凭据不可用。
func TestKeyRingDecryptsLegacyCiphertext(t *testing.T) {
	// 用 seal 直接生成"旧格式"密文（无版本前��）
	legacy, err := seal([]byte(keyA), "legacy-secret")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if strings.Contains(legacy, ":") && strings.HasPrefix(legacy, CiphertextVersion) {
		t.Fatal("测试前置条件错误：legacy 密文不应带版本前缀")
	}

	ring, err := NewKeyRing(keyA)
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}
	plaintext, err := ring.Decrypt(legacy)
	if err != nil {
		t.Fatalf("解密历史密文失败: %v", err)
	}
	if plaintext != "legacy-secret" {
		t.Errorf("应得到 %q，实际 %q", "legacy-secret", plaintext)
	}
}

// TestKeyRingRotation 校验轮换：用旧密钥写的密文，在新密钥为当前密钥时仍可解。
func TestKeyRingRotation(t *testing.T) {
	oldRing, err := NewKeyRing(keyA)
	if err != nil {
		t.Fatalf("NewKeyRing(old): %v", err)
	}
	ciphertext, err := oldRing.Encrypt("rotate-me")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// 轮换：当前密钥换成 B，A 进入 previous
	newRing, err := NewKeyRing(keyB, keyA)
	if err != nil {
		t.Fatalf("NewKeyRing(new): %v", err)
	}
	if newRing.CurrentKeyID() == oldRing.CurrentKeyID() {
		t.Fatal("轮换后 key-id 应发生变化")
	}
	if newRing.PreviousKeyCount() != 1 {
		t.Fatalf("previous 密钥数应为 1，实际 %d", newRing.PreviousKeyCount())
	}

	plaintext, err := newRing.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("轮换后解密旧密文失败: %v", err)
	}
	if plaintext != "rotate-me" {
		t.Errorf("应得到 %q，实际 %q", "rotate-me", plaintext)
	}

	// 新写入的密文应带新 key-id
	fresh, err := newRing.Encrypt("fresh")
	if err != nil {
		t.Fatalf("Encrypt(fresh): %v", err)
	}
	if !strings.HasPrefix(fresh, CiphertextVersion+":"+newRing.CurrentKeyID()+":") {
		t.Errorf("新密文应带新 key-id，实际 %q", fresh)
	}
}

// TestKeyRingUnknownKeyIDFailsLoudly 校验未知 key-id 会明确报错而不是静默失败。
func TestKeyRingUnknownKeyIDFailsLoudly(t *testing.T) {
	otherRing, err := NewKeyRing(keyB)
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}
	ciphertext, err := otherRing.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	ring, err := NewKeyRing(keyA) // 没有 B
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}
	_, err = ring.Decrypt(ciphertext)
	if err == nil {
		t.Fatal("用未知 key-id 解密应报错")
	}
	if !strings.Contains(err.Error(), "key-id") {
		t.Errorf("错误信息应提示 key-id 问题，实际: %v", err)
	}
	if !strings.Contains(err.Error(), "ENCRYPTION_KEY_PREVIOUS") {
		t.Errorf("错误信息应给出补救方式（ENCRYPTION_KEY_PREVIOUS），实际: %v", err)
	}
}

// TestKeyRingRejectsInvalidKeys 校验非法长度密钥被拒绝（不静默补零）。
func TestKeyRingRejectsInvalidKeys(t *testing.T) {
	if _, err := NewKeyRing("short"); err == nil {
		t.Error("短密钥应被拒绝")
	}
	if _, err := NewKeyRing(keyA, "short"); err == nil {
		t.Error("非法的 previous 密钥应被拒绝")
	}
	if _, err := NewKeyRing(""); err == nil {
		t.Error("空密钥应被拒绝")
	}
}

// TestKeyRingDeduplicatesPrevious 校验与当前密钥重复的 previous 会被忽略。
func TestKeyRingDeduplicatesPrevious(t *testing.T) {
	ring, err := NewKeyRing(keyA, keyA, keyA)
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}
	if ring.PreviousKeyCount() != 0 {
		t.Errorf("与当前密钥重复的 previous 应被忽略，实际保留 %d 把", ring.PreviousKeyCount())
	}
}

// TestNewEncryptorReadsPreviousFromEnv 校验 ENCRYPTION_KEY_PREVIOUS 被正确解析。
func TestNewEncryptorReadsPreviousFromEnv(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", keyB)
	t.Setenv("ENCRYPTION_KEY_PREVIOUS", keyA+", "+keyA) // 含空格与重复

	enc, err := NewEncryptor()
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}
	ae, ok := enc.(*aesEncryptor)
	if !ok {
		t.Fatalf("类型断言失败: %T", enc)
	}
	if ae.ring.PreviousKeyCount() != 1 {
		t.Errorf("应解析出 1 把去重后的 previous 密钥，实际 %d", ae.ring.PreviousKeyCount())
	}
}
