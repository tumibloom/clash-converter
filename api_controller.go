package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupRouter 配置路由
func setupRouter() (r *gin.Engine) {
	r = gin.Default()
	err := r.SetTrustedProxies([]string{"127.0.0.1", "172.17.0.0/16"})
	if err != nil {
		panic(err)
	}

	r.GET("/ping", handlePing)
	r.GET("/sub", handleSubscription)

	return
}

func handlePing(c *gin.Context) {
	c.String(http.StatusOK, "pong")
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
	if len(subs) == 0 || scriptUrl == "" || templateUrl == "" {
		c.String(http.StatusBadRequest, "sub, script and template are required")
		return
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
