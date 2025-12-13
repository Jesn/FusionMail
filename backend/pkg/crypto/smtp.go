package crypto

// SMTPPasswordEncryptor SMTP 密码加密器
// 封装 AES-256 加密，专门用于 SMTP 密码的加密存储
// Requirements: 3.5
type SMTPPasswordEncryptor struct {
	encryptor Encryptor
}

// NewSMTPPasswordEncryptor 创建 SMTP 密码加密器
func NewSMTPPasswordEncryptor() (*SMTPPasswordEncryptor, error) {
	encryptor, err := NewEncryptor()
	if err != nil {
		return nil, err
	}
	return &SMTPPasswordEncryptor{encryptor: encryptor}, nil
}

// NewSMTPPasswordEncryptorWithKey 使用指定密钥创建 SMTP 密码加密器
func NewSMTPPasswordEncryptorWithKey(key string) (*SMTPPasswordEncryptor, error) {
	service, err := NewService(key)
	if err != nil {
		return nil, err
	}
	return &SMTPPasswordEncryptor{encryptor: service.encryptor}, nil
}

// Encrypt 加密 SMTP 密码
// 使用 AES-256-GCM 加密，返回 base64 编码的密文
func (e *SMTPPasswordEncryptor) Encrypt(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	return e.encryptor.Encrypt(password)
}

// Decrypt 解密 SMTP 密码
// 解密 base64 编码的密文，返回原始密码
func (e *SMTPPasswordEncryptor) Decrypt(encryptedPassword string) (string, error) {
	if encryptedPassword == "" {
		return "", nil
	}
	return e.encryptor.Decrypt(encryptedPassword)
}

// EncryptSMTPPassword 便捷函数：加密 SMTP 密码
// 使用默认加密器（从环境变量获取密钥）
func EncryptSMTPPassword(password string) (string, error) {
	encryptor, err := NewSMTPPasswordEncryptor()
	if err != nil {
		return "", err
	}
	return encryptor.Encrypt(password)
}

// DecryptSMTPPassword 便捷函数：解密 SMTP 密码
// 使用默认加密器（从环境变量获取密钥）
func DecryptSMTPPassword(encryptedPassword string) (string, error) {
	encryptor, err := NewSMTPPasswordEncryptor()
	if err != nil {
		return "", err
	}
	return encryptor.Decrypt(encryptedPassword)
}
