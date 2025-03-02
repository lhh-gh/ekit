package sqlx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/vmihailenco/msgpack/v5"
	"sync"
	"unsafe"
)

// SecureField 代表一个安全加密的数据库字段
type SecureField[T any] struct {
	value     T
	isValid   bool
	secret    []byte
	aead      cipher.AEAD
	initOnce  sync.Once
	initError error
}

// NewSecureField 创建新的安全字段实例
func NewSecureField[T any](secret []byte, initialValue T) *SecureField[T] {
	sf := &SecureField[T]{
		value:   initialValue,
		isValid: true,
		secret:  make([]byte, len(secret)),
	}
	copy(sf.secret, secret)
	sf.setupCrypto()
	return sf
}

// 初始化加密组件
func (sf *SecureField[T]) setupCrypto() {
	sf.initOnce.Do(func() {
		if len(sf.secret) != 16 && len(sf.secret) != 24 && len(sf.secret) != 32 {
			sf.initError = errors.New("安全密钥长度必须为16/24/32字节")
			return
		}

		block, err := aes.NewCipher(sf.secret)
		if err != nil {
			sf.initError = fmt.Errorf("密码初始化失败: %w", err)
			return
		}

		sf.aead, err = cipher.NewGCM(block)
		if err != nil {
			sf.initError = fmt.Errorf("加密模式创建失败: %w", err)
		}
	})
}

// Value 实现driver.Valuer接口
func (sf *SecureField[T]) Value() (driver.Value, error) {
	if sf.initError != nil {
		return nil, sf.initError
	}
	if !sf.isValid {
		return nil, errors.New("无效的安全字段值")
	}

	var (
		data []byte
		err  error
	)

	// 类型特化处理
	switch v := any(sf.value).(type) {
	case string:
		data = convertStringToBytes(v)
	case []byte:
		data = v
	case int8:
		data = []byte{byte(v)}
	case int16:
		data = make([]byte, 2)
		binary.BigEndian.PutUint16(data, uint16(v))
	// 其他数值类型处理...
	default:
		data, err = msgpack.Marshal(sf.value)
	}

	if err != nil {
		return nil, fmt.Errorf("数据序列化失败: %w", err)
	}

	nonce := make([]byte, sf.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("随机数生成失败: %w", err)
	}

	return sf.aead.Seal(nonce, nonce, data, nil), nil
}

// Scan 实现sql.Scanner接口
func (sf *SecureField[T]) Scan(src any) error {
	if sf.initError != nil {
		return sf.initError
	}

	var encrypted []byte
	switch v := src.(type) {
	case []byte:
		encrypted = v
	case string:
		encrypted = []byte(v)
	default:
		return fmt.Errorf("不支持的数据类型: %T", src)
	}

	nonceSize := sf.aead.NonceSize()
	if len(encrypted) < nonceSize {
		return errors.New("无效的加密数据长度")
	}

	plainData, err := sf.aead.Open(
		nil,
		encrypted[:nonceSize],
		encrypted[nonceSize:],
		nil,
	)
	if err != nil {
		sf.isValid = false
		return fmt.Errorf("数据解密失败: %w", err)
	}

	// 类型特化处理
	switch v := any(&sf.value).(type) {
	case *string:
		*v = convertBytesToString(plainData)
	case *[]byte:
		*v = plainData
	case *int8:
		if len(plainData) != 1 {
			return errors.New("int8类型数据长度不匹配")
		}
		*v = int8(plainData[0])
	case *int16:
		if len(plainData) != 2 {
			return errors.New("int16类型数据长度不匹配")
		}
		*v = int16(binary.BigEndian.Uint16(plainData))
	// 其他数值类型处理...
	default:
		if err := msgpack.Unmarshal(plainData, &sf.value); err != nil {
			sf.isValid = false
			return fmt.Errorf("数据反序列化失败: %w", err)
		}
	}

	sf.isValid = true
	return nil
}

// 零拷贝转换（需确保数据安全）
func convertStringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

func convertBytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
