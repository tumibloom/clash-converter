package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type File struct {
	gorm.Model
	Url     string
	Content string `gorm:"type:text"`
}

type ShortUrl struct {
	gorm.Model
	Code      string `gorm:"uniqueIndex"`
	LongUrl   string `gorm:"type:text"`
	ExpiredAt int64  // 过期时间戳
}

var orm *gorm.DB

func InitDb() {
	var err error
	err = os.MkdirAll("./data", 0755)

	orm, err = gorm.Open(sqlite.Open(DbPath), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				IgnoreRecordNotFoundError: true,
			},
		),
	})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	err = orm.AutoMigrate(&File{}, &ShortUrl{})
	if err != nil {
		panic("failed to migrate: " + err.Error())
	}

	return
}

func GetOrPut(url string, contentSupplier func(string) (string, error)) (result string, err error) {
	if strings.Contains(url, "?") {
		return contentSupplier(url)
	}

	var file File
	err = orm.First(&file, "url = ?", url).Error
	if err == nil {
		if file.UpdatedAt.After(time.Now().Add(-CacheExpire)) {
			L().Info(fmt.Sprintf("Using cache: %s", url))
			result = file.Content
			return
		}

		L().Info(fmt.Sprintf("Cache missed: %s", url))

		var content string
		content, err = contentSupplier(url)
		if err != nil {
			return
		}

		file.Content = content
		err = orm.Save(&file).Error

		if err != nil {
			panic("failed to update: " + err.Error())
		}

		result = content
		return
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		L().Info(fmt.Sprintf("Cache missed: %s", url))

		var content string
		content, err = contentSupplier(url)
		if err != nil {
			return
		}

		err = orm.Create(&File{Url: url, Content: content}).Error
		if err != nil {
			panic("failed to insert: " + err.Error())
		}

		result = content
		return
	} else {
		panic("failed to query: " + err.Error())
	}
}
