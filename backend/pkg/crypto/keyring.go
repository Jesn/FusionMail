package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// CiphertextVersion 是当前密文格式的版本前缀。
//
// 格式：v2:<keyID>:<base64(nonce||ciphertext)>
//
// 历史密文没有前缀（直接是 base64），Decrypt 会回退到"用密钥环里的每一把密钥
// 依次尝试"，因此升级不需要停机重加密。
const CiphertextVersion = "v2"

// keyIDLength 是 key-id 的十六进制字符数（取密钥 SHA-256 前 4 字节）。
// 它只用于在密钥环里定位密钥，不泄露密钥本身。
const keyIDLength = 8

// keyEntry 是密钥环里的一把密钥。
type keyEntry struct {
	id  string
	key []byte
}

// KeyRing 持有一把当前密钥与若干把历��密钥，支持密钥轮换。
//
// 为什么需要它：此前 pkg/crypto 只读单个 ENCRYPTION_KEY，密文里不带任何密钥标识，
// 因此换密钥会让全部历史密文永久不可解，且没有任何迁移路径
// （AGENTS.md 里写的 ENCRYPTION_KEY_PREVIOUS 在代码里零引用）。
//
// 轮换流程：
//  1. 把当前密钥移到 ENCRYPTION_KEY_PREVIOUS，把新密钥设为 ENCRYPTION_KEY
//  2. 新写入的密文带新 key-id；旧密文仍可用 previous 密钥解开
//  3. 跑一次重加密（遍历并重新 Encrypt）后即可移除 previous
type KeyRing struct {
	current  keyEntry
	previous []keyEntry
}

// deriveKeyID 由密钥内容派生稳定的短标识。
func deriveKeyID(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:keyIDLength/2])
}

// NewKeyRing 构造密钥环。current 为当前密钥，previous 为历史密钥（解密用）。
func NewKeyRing(current string, previous ...string) (*KeyRing, error) {
	curKey, err := deriveKey(current)
	if err != nil {
		return nil, fmt.Errorf("当前加密密钥无效: %w", err)
	}
	ring := &KeyRing{current: keyEntry{id: deriveKeyID(curKey), key: curKey}}

	seen := map[string]bool{ring.current.id: true}
	for _, p := range previous {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		prevKey, prevErr := deriveKey(p)
		if prevErr != nil {
			return nil, fmt.Errorf("历史加密密钥无效: %w", prevErr)
		}
		id := deriveKeyID(prevKey)
		if seen[id] {
			continue // 与当前或已有历史密钥重复，跳过
		}
		seen[id] = true
		ring.previous = append(ring.previous, keyEntry{id: id, key: prevKey})
	}
	return ring, nil
}

// CurrentKeyID 返回当前密钥的标识，供运维确认轮换是否生效。
func (r *KeyRing) CurrentKeyID() string { return r.current.id }

// PreviousKeyCount 返回历史密钥数量。
func (r *KeyRing) PreviousKeyCount() int { return len(r.previous) }

// Encrypt 用当前密钥加密，并在密文前写入 key-id。
func (r *KeyRing) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	body, err := seal(r.current.key, plaintext)
	if err != nil {
		return "", err
	}
	return CiphertextVersion + ":" + r.current.id + ":" + body, nil
}

// Decrypt 解密。
//
// 支持两种输入：
//   - 带版本前缀的新格式：按 key-id 精确选密钥
//   - 无前缀的历史格式：依次尝试密钥环里的每一把密钥（轮换过渡期必需）
func (r *KeyRing) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	if strings.HasPrefix(ciphertext, CiphertextVersion+":") {
		rest := strings.TrimPrefix(ciphertext, CiphertextVersion+":")
		idx := strings.Index(rest, ":")
		if idx <= 0 {
			return "", errors.New("密文格式错误：缺少 key-id 分隔符")
		}
		keyID, body := rest[:idx], rest[idx+1:]

		if keyID == r.current.id {
			return open(r.current.key, body)
		}
		for _, p := range r.previous {
			if p.id == keyID {
				return open(p.key, body)
			}
		}
		return "", fmt.Errorf("密文使用了未知的 key-id %q；"+
			"请把对应密钥加入 ENCRYPTION_KEY_PREVIOUS 后再读取", keyID)
	}

	// 历史格式：无 key-id，逐把密钥尝试
	var lastErr error
	for _, e := range append([]keyEntry{r.current}, r.previous...) {
		plaintext, err := open(e.key, ciphertext)
		if err == nil {
			return plaintext, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("解密失败（已尝试 %d 把密钥）: %w", 1+len(r.previous), lastErr)
}

// seal 用 AES-256-GCM 加密，返回 base64(nonce||ciphertext)。
func seal(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil)), nil
}

// open 解密 base64(nonce||ciphertext)。
func open(key []byte, body string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
