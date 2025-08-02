package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/xuewenG/subscribe-proxy/pkg/config"
	"github.com/xuewenG/subscribe-proxy/pkg/router"
)

var mode string
var version string
var commitId string

func main() {
	err := config.InitConfig()
	if err != nil {
		log.Fatalf("Init config failed, %v", err)
	}

	log.Printf(
		"Server is starting:\nmode: %s\nversion: %s\ncommitId: %s\nport: %s\n",
		mode,
		version,
		commitId,
		config.Config.Port,
	)

	if config.Config.Port == "" {
		log.Fatalf("Invalid port: %s\n", config.Config.Port)
		return
	}

	if mode == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.SetTrustedProxies(nil)

	router.Bind(r)

	r.Run(fmt.Sprintf(":%s", config.Config.Port))
}
