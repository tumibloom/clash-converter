package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
		sub := c.Query("sub")
		scriptUrl := c.Query("script")
		templateUrl := c.Query("template")
		userToken := c.Query("token")

		if userToken != Token {
			L().Warn("Unauthorized request received")
			c.String(http.StatusUnauthorized, "Unauthorized request")
			return
		}

		if sub == "" || scriptUrl == "" || templateUrl == "" {
			c.String(http.StatusBadRequest, "sub, script and template are required")
		}

		proxies, err := ExtractProxies(sub)
		if err != nil {
			L().Error(err.Error())
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

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

		result, err := ExecJs(script, template, proxies)
		if err != nil {
			L().Error(err.Error())
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		for h, v := range proxies.TransparentHeaders {
			c.Header(h, v)
		}
		c.String(http.StatusOK, result)
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
