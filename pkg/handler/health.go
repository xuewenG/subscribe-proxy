package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain", []byte("ok"))
}
