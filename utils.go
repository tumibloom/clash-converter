// 更新了函数
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"resty.dev/v3"
)

// ENV
var (
	DefaultExpire = 24 * time.Hour

	CacheExpire = func() time.Duration {
		expireStr, exist := os.LookupEnv("CACHE_EXPIRE_SEC")
		if !exist {
			return DefaultExpire
		}

		expire, err := strconv.ParseInt(expireStr, 10, 64)
		if err != nil {
			return DefaultExpire
		}

		return time.Duration(expire) * time.Second
	}()

	DbPath = func() string {
		path, exist := os.LookupEnv("DB_PATH")
		if !exist {
			path = "./data/database.db"
		}
		return path
	}()

	Token = os.Getenv("ACCESS_TOKEN")
)

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GenerateShortCode 生成32位安全的短码
func GenerateShortCode() string {
	const (
		chars  = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		base   = 62
		length = 32
	)

	// 使用时间戳、随机数和nonce创建唯一输入
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	nonceBytes := make([]byte, 4)
	rand.Read(nonceBytes)
	nonce := binary.BigEndian.Uint32(nonceBytes)

	// 计算SHA-256哈希
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d-%x-%d", timestamp, randomBytes, nonce)))
	hash := h.Sum(nil)

	// 将哈希转换为Base62编码
	num := new(big.Int).SetBytes(hash)
	result := make([]byte, 0, length)

	for len(result) < length {
		remainder := new(big.Int)
		num.DivMod(num, big.NewInt(base), remainder)
		result = append(result, chars[remainder.Int64()])
	}

	// 反转结果以保持正确的顺序
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// CreateShortUrl 创建短链接
func CreateShortUrl(longUrl string) (string, error) {
	// 检查是否已存在相同的长链接且未过期
	var existingShortUrl ShortUrl
	err := orm.Where("long_url = ? AND expired_at > ?", longUrl, time.Now().Unix()).First(&existingShortUrl).Error
	if err == nil {
		return existingShortUrl.Code, nil
	}

	// 生成新的短码
	for i := 0; i < 5; i++ { // 最多尝试5次，由于使用了更安全的算法，碰撞概率极低
		code := GenerateShortCode()
		shortUrl := ShortUrl{
			Code:      code,
			LongUrl:   longUrl,
			ExpiredAt: time.Now().Add(180 * 24 * time.Hour).Unix(), // 180天有效期
		}

		err = orm.Create(&shortUrl).Error
		if err == nil {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique short code")
}

func FetchString(url string) (string, error) {
	L().Info(fmt.Sprintf("Fetching %s", url))

	client := resty.New().SetRetryCount(3)
	res, err := client.R().Get(url)

	if err != nil {
		return "", err
	}

	return res.String(), client.Close()
}
