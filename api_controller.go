package main

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

//go:embed ui.html
var uiHTML string

// setupRouter 配置路由
func setupRouter() (r *gin.Engine) {
	r = gin.Default()
	err := r.SetTrustedProxies([]string{"127.0.0.1", "172.17.0.0/16"})
	if err != nil {
		panic(err)
	}

	r.GET("/ping", handlePing)
	r.GET("/sub", handleSubscription)
	r.GET("/ui", handleUI)
	r.GET("/s/:code", handleShortUrl)
	r.POST("/s/create", handleCreateShortUrl)

	// 配置静态文件服务以提供 config 目录下的文件
	r.Static("/config", "./config")

	return
}

func handlePing(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func handleUI(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, uiHTML)
}

// handleShortUrl 处理短链接重定向
func handleShortUrl(c *gin.Context) {
	code := c.Param("code")
	var shortUrl ShortUrl
	err := orm.Where("code = ? AND expired_at > ?", code, time.Now().Unix()).First(&shortUrl).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.String(http.StatusNotFound, "Short URL not found or expired")
			return
		}
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Redirect(http.StatusMovedPermanently, shortUrl.LongUrl)
}

// handleCreateShortUrl 处理创建短链接的请求
func handleCreateShortUrl(c *gin.Context) {
	var body struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	code, err := CreateShortUrl(body.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": code})
}

// handleSubscription 处理订阅转换请求
// 支持多个订阅合并、流量统计、用量信息显示
func handleSubscription(c *gin.Context) {
	subs := c.QueryArray("sub")
	scriptUrl := c.Query("script")
	templateUrl := c.Query("template")
	userToken := c.Query("token")

	// 鉴权
	if userToken != Token {
		L().Warn("Unauthorized request received")
		c.String(http.StatusUnauthorized, "Unauthorized request")
		return
	}

	// 参数校验
	if len(subs) == 0 {
		c.String(http.StatusBadRequest, "sub is required")
		return
	}

	// 如果未提供 script 或 template，使用默认文件
	if scriptUrl == "" {
		defaultScriptPath := "./config/script.js"
		if !FileExists(defaultScriptPath) {
			c.String(http.StatusNotFound, "Default script file not found")
			return
		}
		scheme := "http"
		if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		scriptUrl = scheme + "://" + c.Request.Host + "/config/script.js"
	}
	if templateUrl == "" {
		defaultTemplatePath := "./config/template.yaml"
		if !FileExists(defaultTemplatePath) {
			c.String(http.StatusNotFound, "Default template file not found")
			return
		}
		scheme := "http"
		if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		templateUrl = scheme + "://" + c.Request.Host + "/config/template.yaml"
	}

	// 提取所有订阅的节点
	allProxies := make([]SubscriptionData, 0, len(subs))
	for i, sub := range subs {
		name := fmt.Sprintf("订阅%02d", i+1)
		proxies, err := ExtractProxies(sub, name)
		if err != nil {
			L().Error(err.Error())
			c.String(http.StatusInternalServerError, fmt.Sprintf("%s:\n%s", sub, err.Error()))
			return
		}
		allProxies = append(allProxies, proxies)
	}

	// 合并订阅数据
	mergedProxies := mergeProxies(allProxies)

	// 获取模板和脚本
	template, err := FetchString(templateUrl)
	if err != nil {
		L().Error(err.Error())
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	script, err := FetchString(scriptUrl)
	if err != nil {
		L().Error(err.Error())
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// 执行 JS 脚本生成配置
	result, err := ExecJs(script, template, mergedProxies)
	if err != nil {
		L().Error(err.Error())
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// 添加用量信息节点组
	finalResult, err := addSubInfoGroup(result, mergedProxies.SubInfos)
	if err != nil {
		L().Error(err.Error())
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// 设置响应头
	for h, v := range mergedProxies.TransparentHeaders {
		c.Header(h, v)
	}
	c.String(http.StatusOK, finalResult)
}
