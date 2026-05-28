package runtimeenv

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func CurrentGinMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("GIN_MODE")))

	switch mode {
	case "", gin.ReleaseMode:
		return gin.ReleaseMode
	case gin.DebugMode, gin.TestMode:
		return mode
	default:
		return gin.ReleaseMode
	}
}

func EnvBool(key string, defaultValue bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return defaultValue
	}

	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}
