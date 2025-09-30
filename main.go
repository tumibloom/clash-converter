package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func mergeProxies(allProxies []ProxiesContainer) ProxiesContainer {
	merged := ProxiesContainer{
		Proxies:            make([]map[string]any, 0),
		TransparentHeaders: make(map[string]string),
		SubInfos:           make([]*SubscriptionInfo, 0, len(allProxies)),
	}

	var totalUpload, totalDownload, totalTotal, maxExpire int64
	filenames := make([]string, 0, len(allProxies))

	for _, pc := range allProxies {
		merged.Proxies = append(merged.Proxies, pc.Proxies...)

		for _, info := range pc.SubInfos {
			totalUpload += info.Upload
			totalDownload += info.Download
			totalTotal += info.Total
			if info.Expire > maxExpire {
				maxExpire = info.Expire
			}
			merged.SubInfos = append(merged.SubInfos, info)
			filenames = append(filenames, info.Name)
		}
	}

	if totalTotal > 0 {
		merged.TransparentHeaders["Subscription-Userinfo"] = fmt.Sprintf(
			"upload=%d; download=%d; total=%d; expire=%d",
			totalUpload, totalDownload, totalTotal, maxExpire,
		)
	}

	if len(filenames) > 0 {
		combinedName := strings.Join(filenames, " | ")
		encodedName := url.PathEscape(combinedName)
		merged.TransparentHeaders["Content-Disposition"] = fmt.Sprintf(
			"attachment; filename*=UTF-8''%s", encodedName,
		)
	}

	return merged
}

func addSubInfoGroup(yamlStr string, subInfos []*SubscriptionInfo) (string, error) {
	var config map[string]any
	err := yaml.Unmarshal([]byte(yamlStr), &config)
	if err != nil {
		return "", err
	}

	proxies, ok := config["proxies"].([]any)
	if !ok {
		proxies = make([]any, 0)
	}

	infoNodeNames := make([]string, 0, len(subInfos))
	for _, info := range subInfos {
		used := float64(info.Upload+info.Download) / 1024 / 1024 / 1024
		total := float64(info.Total) / 1024 / 1024 / 1024
		nodeName := fmt.Sprintf("%s：%.1f/%.1f", info.Name, used, total)
		infoNodeNames = append(infoNodeNames, nodeName)

		dummyNode := map[string]any{
			"name":     nodeName,
			"type":     "ss",
			"server":   "127.0.0.1",
			"port":     1080,
			"cipher":   "aes-128-gcm",
			"password": "dummy",
		}
		proxies = append([]any{dummyNode}, proxies...)
	}

	config["proxies"] = proxies

	subInfoGroup := map[string]any{
		"name":    "Sub Info",
		"type":    "select",
		"proxies": infoNodeNames,
	}

	proxyGroups, ok := config["proxy-groups"].([]any)
	if ok {
		config["proxy-groups"] = append([]any{subInfoGroup}, proxyGroups...)
	} else {
		config["proxy-groups"] = []any{subInfoGroup}
	}

	resultBytes, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(resultBytes), nil
}

func setupRouter() (r *gin.Engine) {
	r = gin.Default()
	err := r.SetTrustedProxies([]string{"127.0.0.1", "172.17.0.0/16"})
	if err != nil {
		panic(err)
	}

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.GET("/sub", func(c *gin.Context) {
		subs := c.QueryArray("sub")
		scriptUrl := c.Query("script")
		templateUrl := c.Query("template")
		userToken := c.Query("token")

		if userToken != Token {
			L().Warn("Unauthorized request received")
			c.String(http.StatusUnauthorized, "Unauthorized request")
			return
		}

		if len(subs) == 0 || scriptUrl == "" || templateUrl == "" {
			c.String(http.StatusBadRequest, "sub, script and template are required")
			return
		}

		allProxies := make([]ProxiesContainer, 0, len(subs))
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

		mergedProxies := mergeProxies(allProxies)

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

		result, err := ExecJs(script, template, mergedProxies)
		if err != nil {
			L().Error(err.Error())
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		finalResult, err := addSubInfoGroup(result, mergedProxies.SubInfos)
		if err != nil {
			L().Error(err.Error())
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		for h, v := range mergedProxies.TransparentHeaders {
			c.Header(h, v)
		}
		c.String(http.StatusOK, finalResult)
	})

	return
}

func main() {
	InitDb()
	ginEngine := setupRouter()
	err := ginEngine.Run(":8080")
	if err != nil {
		L().Error(err.Error())
	}
}
