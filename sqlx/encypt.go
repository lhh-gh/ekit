package sqlx

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// EncryptColumn 代表一个加密的数据库列，使用AES-GCM模式进行加密和解密。
// 泛型参数T表示被加密数据的类型。
// 注意：Key必须是16、24或32字节长度的字符串（对应AES-128、AES-192、AES-256）。
// Valid标记该值是否有效，类似于sql.Null类型的行为。
type EncryptColumn[T any] struct {
	Val   T      // 存储实际的值，类型由泛型T指定
	Valid bool   // 标记值是否有效，为false时Value返回nil
	Key   string // 加密密钥，必须为16/24/32字节长度
}

// 错误定义
var (
	errInvalid          = errors.New("ekit EncryptColumn无效")
	errKeyLengthInvalid = errors.New("ekit EncryptColumn仅支持 16/24/32 byte 的key")
)

// Value 实现driver.Valuer接口，将值加密后存入数据库。
// 返回值可能为[]byte类型（加密后的数据）或错误。
// 如果 T 是基本类型，那么会对 T 进行直接加密
// 否则，将 T 按照 JSON 序列化之后进行加密，返回加密后的数据
func (e EncryptColumn[T]) Value() (driver.Value, error) {
	//检查值有效性
	if !e.Valid {
		return nil, errInvalid
	}
	// 验证密钥长度
	if len(e.Key) != 16 && len(e.Key) != 24 && len(e.Key) != 32 {
		return nil, errKeyLengthInvalid
	}
	var (
		val any = e.Val // 将值转为interface{}以进行类型断言
		b   []byte
		err error
	)

	// 根据值的实际类型进行序列化处理
	switch valT := val.(type) {
	case string: // 字符串直接转为字节切片
		b = []byte(valT)
	case []byte: // 字节切片直接使用
		b = valT
		// 处理所有整数和浮点数类型，使用二进制序列化
	case int8, int16, int32, int64, uint8, uint16, uint32, uint64,
		float32, float64:
		buffer := new(bytes.Buffer)
		err = binary.Write(buffer, binary.BigEndian, val)
		b = buffer.Bytes()
	case int:
		tmp := int64(valT)
		buffer := new(bytes.Buffer)
		err = binary.Write(buffer, binary.BigEndian, tmp)
		b = buffer.Bytes()
	case uint:
		tmp := uint64(valT)
		buffer := new(bytes.Buffer)
		err = binary.Write(buffer, binary.BigEndian, tmp)
		b = buffer.Bytes()
	default: // 其他类型使用JSON序列化
		b, err = json.Marshal(e.Val)
	}
	if err != nil {
		return nil, err
	}
	//对序列化后的数据进行AES-GCM加密
	return e.aesEncrypt(b)
}

// Scan 实现sql.Scanner接口，从数据库读取并解密数据。
// 参数src为数据库读取的原始数据（[]byte或string类型）。
// 并将解密后的数据进行反序列化，构造 T
func (e *EncryptColumn[T]) Scan(src any) error {
	var (
		b   []byte
		err error
	)
	// 根据数据库返回类型转换数据
	switch value := src.(type) {
	case []byte:
		b, err = e.aesDecrypt(value)
	case string:
		b, err = e.aesDecrypt([]byte(value))
	default:
		return fmt.Errorf("ekit：EncryptColumn.Scan 不支持 src 类型 %v", src)
	}
	if err != nil {
		return err
	}
	// 解密后反序列化到目标类型
	err = e.setValAfterDecrypt(b)
	e.Valid = err == nil // 根据反序列化结果设置有效性标志
	return err
}

// setValAfterDecrypt 将解密后的数据反序列化到结构体的Val字段。
func (e *EncryptColumn[T]) setValAfterDecrypt(deEncrypt []byte) error {
	var val any = &e.Val // 获取Val的指针用于反序列化
	var err error
	// 根据目标类型进行反序列化
	switch valT := val.(type) {
	case *string:
		*valT = string(deEncrypt)
	case *[]byte:
		*valT = deEncrypt
	case *int8, *int16, *int32, *int64, *uint8, *uint16, *uint32, *uint64,
		*float32, *float64:
		reader := bytes.NewReader(deEncrypt)
		err = binary.Read(reader, binary.BigEndian, valT)
	case *int:
		tmp := new(int64)
		reader := bytes.NewReader(deEncrypt)
		err = binary.Read(reader, binary.BigEndian, tmp)
		*valT = int(*tmp)
	case *uint:
		tmp := new(uint64)
		reader := bytes.NewReader(deEncrypt)
		err = binary.Read(reader, binary.BigEndian, tmp)
		*valT = uint(*tmp)
	default: // 其他类型使用JSON反序列化
		err = json.Unmarshal(deEncrypt, &e.Val)
	}
	return err
}

// aesEncrypt 使用AES-GCM模式加密数据，返回nonce和密文的组合。
func (e *EncryptColumn[T]) aesEncrypt(data []byte) ([]byte, error) {
	// 创建AES cipher实例
	block, err := aes.NewCipher([]byte(e.Key))
	if err != nil {
		return nil, err
	}
	// 创建GCM模式实例
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	// 加密并组合nonce和密文
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// aesDecrypt 使用AES-GCM模式解密数据。
func (e *EncryptColumn[T]) aesDecrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(e.Key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// 分离nonce和密文
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	// 解密数据
	return gcm.Open(nil, nonce, ciphertext, nil)
}
