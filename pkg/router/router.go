package router

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/xuewenG/subscribe-proxy/pkg/config"
	"github.com/xuewenG/subscribe-proxy/pkg/handler"
	"github.com/xuewenG/subscribe-proxy/pkg/metrics"
)

func Bind(r *gin.Engine) {
	r.Use(cors.New(cors.Config{
		AllowOrigins: strings.Split(config.Config.CorsOrigin, ","),
		AllowMethods: []string{"GET", "OPTIONS"},
	}))

	api := r.Group(config.Config.ContextPath)
	api.GET("/subscribe", handler.SubscribeProxy)
	api.GET("/subscribe/get", handler.SubscribeProxy)
	api.GET("/metrics", metrics.GetMetrics())
}
