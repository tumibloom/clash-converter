package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"resty.dev/v3"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

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

// GenerateShortCode 生成6位随机短码
func GenerateShortCode() string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const codeLength = 6
	result := make([]byte, codeLength)
	for i := 0; i < codeLength; i++ {
		result[i] = chars[rand.Intn(len(chars))]
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
	for i := 0; i < 3; i++ { // 最多尝试3次
		code := GenerateShortCode()
		shortUrl := ShortUrl{
			Code:      code,
			LongUrl:   longUrl,
			ExpiredAt: time.Now().Add(7 * 24 * time.Hour).Unix(), // 7天有效期
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
