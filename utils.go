package main

import (
	"fmt"
	"os"
	"resty.dev/v3"
	"strconv"
	"time"
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

func FetchString(url string) (string, error) {
	L().Info(fmt.Sprintf("Fetching %s", url))

	client := resty.New().SetRetryCount(3)
	res, err := client.R().Get(url)

	if err != nil {
		return "", err
	}

	return res.String(), client.Close()
}
