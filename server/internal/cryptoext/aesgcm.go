package cryptoext

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encrypt 使用AES-GCM加密明文
// 参数:
//   - key: 32字节的AES密钥
//   - plaintext: 要加密的明文字符串
//
// 返回:
//   - string: Base64编码的加密结果（包含nonce）
//   - error: 加密过程中的错误
func Encrypt(key []byte, plaintext string) (string, error) {
	// 使用密钥创建AES加密块
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 创建GCM模式的加密器
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机nonce（盐）
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 使用GCM模式加密明文
	// nonce被预先追加到密文前面
	blob := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回Base64编码的加密结果
	return base64.StdEncoding.EncodeToString(blob), nil
}

// Decrypt 解密使用AES-GCM加密的密文
// 参数:
//   - key: 32字节的AES密钥（必须与加密时使用的密钥相同）
//   - ciphertext: Base64编码的加密字符串
//
// 返回:
//   - string: 解密后的明文
//   - error: 解密过程中的错误
func Decrypt(key []byte, ciphertext string) (string, error) {
	// 解码Base64字符串
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// 使用密钥创建AES加密块
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 创建GCM模式的加密器
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 验证密文长度是否足够包含nonce
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	// 提取nonce和密文
	nonce := raw[:gcm.NonceSize()]
	data := raw[gcm.NonceSize():]

	// 解密
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}
