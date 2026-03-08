package utils

import (
	"github.com/ariandi/gocom/config"
	"gitlab.com/bot3342545/il-dashboard/constans"
)

// GetCacheKey returns cache key with environment prefix
// Example: GetCacheKey("ai:knowledge:client:123") -> "DEV:ai:knowledge:client:123"
func GetCacheKey(suffix string) string {
	envPrefix := config.Get(constans.StaticDefaultEnv, "DEV")
	return envPrefix + ":" + suffix
}
