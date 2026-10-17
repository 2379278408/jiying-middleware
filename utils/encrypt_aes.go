package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

func pkcs7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	out := make([]byte, len(src)+padding)
	copy(out, src)
	for i := len(src); i < len(out); i++ {
		out[i] = byte(padding)
	}
	return out
}

func pkcs7Unpadding(src []byte) []byte {
	if len(src) == 0 {
		return src
	}
	n := int(src[len(src)-1])
	if n < 1 || n > len(src) {
		return src
	}
	return src[:len(src)-n]
}

func clampKey(key []byte) []byte {
	var size int
	switch {
	case len(key) < 0x11:
		size = 0x10
	case len(key) < 0x19:
		size = 0x18
	case len(key) < 0x21:
		size = 0x20
	default:
		size = 0x20
	}
	out := make([]byte, 0, size)
	for i := 0; i < size; i++ {
		if len(key) == 0 {
			out = append(out, 0)
		} else {
			out = append(out, key[i%len(key)])
		}
	}
	return out
}

const aesCBCVersion byte = 0x02

// aesEncrypt 使用随机 IV 的 AES-CBC，输出格式为 版本号 || IV || 密文。
func aesEncrypt(key []byte, origData []byte) []byte {
	fixedKey := clampKey(key)
	block, err := aes.NewCipher(fixedKey)
	if err != nil {
		fmt.Fprintln(os_StderrLike{}, err)
		return nil
	}
	blockSize := block.BlockSize()
	iv := make([]byte, blockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil
	}
	src := pkcs7Padding(origData, blockSize)
	dst := make([]byte, len(src))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(dst, src)
	out := make([]byte, 0, 1+len(iv)+len(dst))
	out = append(out, aesCBCVersion)
	out = append(out, iv...)
	return append(out, dst...)
}

func aesDecrypt(key []byte, crypted []byte) []byte {
	fixedKey := clampKey(key)
	block, err := aes.NewCipher(fixedKey)
	if err != nil {
		return nil
	}
	blockSize := block.BlockSize()
	if len(crypted) < 1+blockSize || crypted[0] != aesCBCVersion {
		return nil
	}
	iv := crypted[1 : 1+blockSize]
	body := crypted[1+blockSize:]
	if len(body) == 0 || len(body)%blockSize != 0 {
		return nil
	}
	dst := make([]byte, len(body))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(dst, body)
	return pkcs7Unpadding(dst)
}

func EncryptByAes(key, s string) string {
	data := aesEncrypt([]byte(key), []byte(s))
	if data == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func DecryptByAes(key, s string) string {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	out := aesDecrypt([]byte(key), data)
	return string(out)
}

var fkSM4 = [4]uint32{0xa3b1bac6, 0x56aa3350, 0x677d9197, 0xb27022dc}
var ckSM4 = [32]uint32{
	0x00070e15, 0x1c232a31, 0x383f464d, 0x545b6269,
	0x70777e85, 0x8c939aa1, 0xa8afb6bd, 0xc4cbd2d9,
	0xe0e7eef5, 0xfc030a11, 0x181f262d, 0x343b4249,
	0x50575e65, 0x6c737a81, 0x888f969d, 0xa4abb2b9,
	0xc0c7ced5, 0xdce3eaf1, 0xf8ff060d, 0x141b2229,
	0x30373e45, 0x4c535a61, 0x686f767d, 0x848b9299,
	0xa0a7aeb5, 0xbcc3cad1, 0xd8dfe6ed, 0xf4fb0209,
	0x10171e25, 0x2c333a41, 0x484f565d, 0x646b7279,
}

var sboxSM4 = [256]byte{
	0xd6, 0x90, 0xe9, 0xfe, 0xcc, 0xe1, 0x3d, 0xb7, 0x16, 0xb6, 0x14, 0xc2, 0x28, 0xfb, 0x2c, 0x05,
	0x2b, 0x67, 0x9a, 0x76, 0x2a, 0xbe, 0x04, 0xc3, 0xaa, 0x44, 0x13, 0x26, 0x49, 0x86, 0x06, 0x99,
	0x9c, 0x42, 0x50, 0xf4, 0x91, 0xef, 0x98, 0x7a, 0x33, 0x54, 0x0b, 0x43, 0xed, 0xcf, 0xac, 0x62,
	0xe4, 0xb3, 0x1c, 0xa9, 0xc9, 0x08, 0xe8, 0x95, 0x80, 0xdf, 0x94, 0xfa, 0x75, 0x8f, 0x3f, 0xa6,
	0x47, 0x07, 0xa7, 0xfc, 0xf3, 0x73, 0x17, 0xba, 0x83, 0x59, 0x3c, 0x19, 0xe6, 0x85, 0x4f, 0xa8,
	0x68, 0x6b, 0x81, 0xb2, 0x71, 0x64, 0xda, 0x8b, 0xf8, 0xeb, 0x0f, 0x4b, 0x70, 0x56, 0x9d, 0x35,
	0x1e, 0x24, 0x0e, 0x5e, 0x63, 0x58, 0xd1, 0xa2, 0x25, 0x22, 0x7c, 0x3b, 0x01, 0x21, 0x78, 0x87,
	0xd4, 0x00, 0x46, 0x57, 0x9f, 0xd3, 0x27, 0x52, 0x4c, 0x36, 0x02, 0xe7, 0xa0, 0xc4, 0xc8, 0x9e,
	0xea, 0xbf, 0x8a, 0xd2, 0x40, 0xc7, 0x38, 0xb5, 0xa3, 0xf7, 0xf2, 0xce, 0xf9, 0x61, 0x15, 0xa1,
	0xe0, 0xae, 0x5d, 0xa4, 0x9b, 0x34, 0x1a, 0x55, 0xad, 0x93, 0x32, 0x30, 0xf5, 0x8c, 0xb1, 0xe3,
	0x1d, 0xf6, 0xe2, 0x2e, 0x82, 0x66, 0xca, 0x60, 0xc0, 0x29, 0x23, 0xab, 0x0d, 0x53, 0x4e, 0x6f,
	0xd5, 0xdb, 0x37, 0x45, 0xde, 0xfd, 0x8e, 0x2f, 0x03, 0xff, 0x6a, 0x72, 0x6d, 0x6c, 0x5b, 0x51,
	0x8d, 0x1b, 0xaf, 0x92, 0xbb, 0xdd, 0xbc, 0x7f, 0x11, 0xd9, 0x5c, 0x41, 0x1f, 0x10, 0x5a, 0xd8,
	0x0a, 0xc1, 0x31, 0x88, 0xa5, 0xcd, 0x7b, 0xbd, 0x2d, 0x74, 0xd0, 0x12, 0xb8, 0xe5, 0xb4, 0xb0,
	0x89, 0x69, 0x97, 0x4a, 0x0c, 0x96, 0x77, 0x7e, 0x65, 0xb9, 0xf1, 0x09, 0xc5, 0x6e, 0xc6, 0x84,
	0x18, 0xf0, 0x7d, 0xec, 0x3a, 0xdc, 0x4d, 0x20, 0x79, 0xee, 0x5f, 0x3e, 0xd7, 0xcb, 0x39, 0x48,
}

func keyRotl(x uint32, k uint) uint32 { return (x<<k | x>>(32-k)) & 0xffffffff }

func sm4Tau(x uint32) uint32 {
	return uint32(sboxSM4[byte(x>>24)])<<24 |
		uint32(sboxSM4[byte(x>>16)])<<16 |
		uint32(sboxSM4[byte(x>>8)])<<8 |
		uint32(sboxSM4[byte(x)])
}

func sm4L(b uint32) uint32 {
	return b ^ keyRotl(b, 2) ^ keyRotl(b, 10) ^ keyRotl(b, 18) ^ keyRotl(b, 24)
}
func sm4Lp(b uint32) uint32 {
	return b ^ keyRotl(b, 13) ^ keyRotl(b, 23)
}

func sm4KeyExpansion(key [16]byte) (rk [32]uint32) {
	var k [36]uint32
	for i := 0; i < 4; i++ {
		k[i] = uint32(key[4*i])<<24 | uint32(key[4*i+1])<<16 | uint32(key[4*i+2])<<8 | uint32(key[4*i+3])
		k[i] ^= fkSM4[i]
	}
	for i := 0; i < 32; i++ {
		k[i+4] = k[i] ^ sm4Lp(sm4Tau(k[i+1]^k[i+2]^k[i+3]^ckSM4[i]))
		rk[i] = k[i+4]
	}
	return
}

func sm4BlockEnc(block []byte, rk [32]uint32) {
	var x [36]uint32
	for i := 0; i < 4; i++ {
		x[i] = uint32(block[4*i])<<24 | uint32(block[4*i+1])<<16 | uint32(block[4*i+2])<<8 | uint32(block[4*i+3])
	}
	for i := 0; i < 32; i++ {
		x[i+4] = x[i] ^ sm4L(sm4Tau(x[i+1]^x[i+2]^x[i+3]^rk[i]))
	}
	r := [4]uint32{x[35], x[34], x[33], x[32]}
	for i := 0; i < 4; i++ {
		block[4*i] = byte(r[i] >> 24)
		block[4*i+1] = byte(r[i] >> 16)
		block[4*i+2] = byte(r[i] >> 8)
		block[4*i+3] = byte(r[i])
	}
}

func sm4BlockDec(b []byte, rk [32]uint32) {
	var rr [32]uint32
	for i := 0; i < 32; i++ {
		rr[i] = rk[31-i]
	}
	sm4BlockEnc(b, rr)
}

const sm4CBCVersion byte = 0x02

// Sm4EncryptCbc 使用随机 IV，输出格式为 版本号 || IV || 密文。
func Sm4EncryptCbc(key []byte, data []byte) []byte {
	if len(key) < 16 {
		return nil
	}
	var mk [16]byte
	copy(mk[:], key[:16])
	rk := sm4KeyExpansion(mk)
	src := pkcs7Padding(data, 16)
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return nil
	}
	dst := make([]byte, len(src))
	prev := append([]byte(nil), iv...)
	for i := 0; i < len(src); i += 16 {
		for j := 0; j < 16; j++ {
			prev[j] ^= src[i+j]
		}
		sm4BlockEnc(prev, rk)
		copy(dst[i:], prev)
	}
	out := make([]byte, 0, 1+len(iv)+len(dst))
	out = append(out, sm4CBCVersion)
	out = append(out, iv...)
	return append(out, dst...)
}

func Sm4DecryptCbc(key []byte, data []byte) []byte {
	if len(key) < 16 || len(data) < 1+16 || data[0] != sm4CBCVersion {
		return nil
	}
	body := data[1+16:]
	if len(body) == 0 || len(body)%16 != 0 {
		return nil
	}
	var mk [16]byte
	copy(mk[:], key[:16])
	rk := sm4KeyExpansion(mk)
	iv := data[1 : 1+16]
	dst := make([]byte, len(body))
	prev := append([]byte(nil), iv...)
	var cur [16]byte
	for i := 0; i < len(body); i += 16 {
		copy(cur[:], body[i:i+16])
		sm4BlockDec(cur[:], rk)
		for j := 0; j < 16; j++ {
			cur[j] ^= prev[j]
		}
		copy(dst[i:], cur[:])
		prev = body[i : i+16]
	}
	return pkcs7Unpadding(dst)
}

// rankKeyFromEnv 从部署环境读取 rank 密钥，禁止在二进制中内置可复用密钥。
func rankKeyFromEnv(name, label string) ([]byte, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if len(raw) < 32 {
		return nil, false
	}
	key := make([]byte, 32)
	copy(key, raw)
	return key, true
}

func rankKDF(key []byte, saltTxt string) []byte {
	k1 := pbkdf2.Key(key, key, 1000, 32, sha1.New)
	return argon2.IDKey(k1, []byte(saltTxt), 1, 64*1024, 4, 32)
}

func EncodeRank3(s string) string {
	key, ok := rankKeyFromEnv("JIYING_RANK3_KEY", "rank3")
	if !ok {
		return ""
	}
	return hex.EncodeToString(Sm4EncryptCbc(rankKDF(key, "rank3"), []byte(s)))
}

func DecodeRank3(s string) string {
	key, ok := rankKeyFromEnv("JIYING_RANK3_KEY", "rank3")
	if !ok {
		return ""
	}
	c, err := hex.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(Sm4DecryptCbc(rankKDF(key, "rank3"), c))
}

func EncodeRank4(s string) string {
	key, ok := rankKeyFromEnv("JIYING_RANK4_KEY", "rank4")
	if !ok {
		return ""
	}
	return hex.EncodeToString(Sm4EncryptCbc(rankKDF(key, "rank4"), []byte(s)))
}

func DecodeRank4(s string) string {
	key, ok := rankKeyFromEnv("JIYING_RANK4_KEY", "rank4")
	if !ok {
		return ""
	}
	c, err := hex.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(Sm4DecryptCbc(rankKDF(key, "rank4"), c))
}

type os_StderrLike struct{}

func (os_StderrLike) Write(p []byte) (int, error) { return fmt.Print(string(p)) }
