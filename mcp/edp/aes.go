package edp

//加密解密工具

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"errors"
)

type Padding int

const (
	Padding5 Padding = iota + 1
	Padding7
)

// AESCBCEncrypt AesCBC加密PKCS5
//
// @param origData []byte 加密的字节数组
//
// @param key []byte 密钥字节数组
func AESCBCEncrypt(origData []byte, key []byte, padding ...Padding) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	paddingFunc := PKCS5Padding
	if len(padding) > 0 {
		switch padding[0] {
		case Padding7:
			paddingFunc = PKCS7Padding
		}
	}
	origData = paddingFunc(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	cypted := make([]byte, len(origData))
	blockMode.CryptBlocks(cypted, origData)
	return cypted, nil
}

// AESECBEncrypt Aes ECB加密PKCS5
//
// @param origData []byte 加密的字节数组
//
// @param key []byte 密钥字节数组
func AESECBEncrypt(origData []byte, key []byte, padding ...Padding) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	paddingFunc := PKCS5Padding
	if len(padding) > 0 {
		switch padding[0] {
		case Padding7:
			paddingFunc = PKCS7Padding
		}
	}
	blockSize := block.BlockSize()
	origData = paddingFunc(origData, blockSize)
	blockMode := newECBEncrypter(block)
	cypted := make([]byte, len(origData))
	blockMode.CryptBlocks(cypted, origData)
	return cypted, nil
}

// AESCBCDecrypt Aes CBC解密PKCS5
//
// @param origData []byte 解密的字节数组
//
// @param key []byte 密钥字节数组
func AESCBCDecrypt(cypted []byte, key []byte, padding ...Padding) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	UnPaddingFunc := PKCS5UnPadding
	if len(padding) > 0 {
		switch padding[0] {
		case Padding7:
			UnPaddingFunc = PKCS7UnPadding
		}
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(cypted))
	blockMode.CryptBlocks(origData, cypted)
	origData = UnPaddingFunc(origData)
	return origData, nil
}

// AESECBDecrypt Aes ECB解密PKCS5
//
// @param origData []byte 解密的字节数组
//
// @param key []byte 密钥字节数组
func AESECBDecrypt(cypted []byte, key []byte, padding ...Padding) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	UnPaddingFunc := PKCS5UnPadding
	if len(padding) > 0 {
		switch padding[0] {
		case Padding7:
			UnPaddingFunc = PKCS7UnPadding
		}
	}
	blockMode := newECBDecrypter(block)
	origData := make([]byte, len(cypted))
	blockMode.CryptBlocks(origData, cypted)
	origData = UnPaddingFunc(origData)
	return origData, nil
}

// PKCS5Padding PKCS5密钥填充方式
func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// PKCS5UnPadding PKCS5密钥填充方式返填充
func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func MD5(key []byte) []byte {
	m := md5.New()
	m.Write(key)
	return m.Sum(nil)
}

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

// NewECBEncrypter returns a BlockMode which encrypts in electronic code book
// mode, using the given Block.
func newECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}

func (x *ecbEncrypter) BlockSize() int { return x.blockSize }

func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

type ecbDecrypter ecb

// NewECBDecrypter returns a BlockMode which decrypts in electronic code book
// mode, using the given Block.
func newECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}

func (x *ecbDecrypter) BlockSize() int { return x.blockSize }

func (x *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Decrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

const encryptoKey = "FE4F3F7F3FCDC334F14F8CB34CB88932"

// Encrypto 加密
func Encrypto(data string) (string, error) {
	if data == "" {
		return "", errors.New("空加密内容")
	}
	encryptoData, e := AESCBCEncrypt([]byte(data), []byte(encryptoKey))
	if e != nil {
		return "", e
	}
	return hex.EncodeToString(encryptoData), nil
}

// Decrypto 解密
func Decrypto(data string) (string, error) {
	if data == "" {
		return "", errors.New("空解密内容")
	}
	dataBts, e := hex.DecodeString(data)
	if e != nil {
		return "", nil
	}
	decryptoData, e := AESCBCDecrypt(dataBts, []byte(encryptoKey))
	if e != nil {
		return "", e
	}
	return string(decryptoData), nil
}
