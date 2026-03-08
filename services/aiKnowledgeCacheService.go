package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ariandi/gocom"
	"gitlab.com/bot3342545/il-dashboard/constans"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/repositories/aiKnowledgeBase"
	"gitlab.com/bot3342545/il-dashboard/utils"
)

const (
	AIKnowledgeCacheTTL = 24 * time.Hour
)

type AIKnowledgeCacheSvc interface {
	RefreshCache(clientId string) *gocom.CodedError
	GetKnowledge(clientId string) ([]aiKnowledgeBase.AiKnowledgeBase, *gocom.CodedError)
	RefreshAllClients() *gocom.CodedError
}

type AIKnowledgeCacheSvcImpl struct {
}

// RefreshCache loads knowledge from DB and stores in Redis for specific client
func (o *AIKnowledgeCacheSvcImpl) RefreshCache(clientId string) *gocom.CodedError {
	customLogger.InfoWithData("RefreshCache started", map[string]interface{}{
		"component": "AIKnowledgeCacheService",
		"function":  "RefreshCache",
		"client_id": clientId,
	})

	// FORCE DELETE old cache first to ensure fresh data from DB
	cacheKey := utils.GetCacheKey(constans.AIKnowledgeCachePrefix + clientId)
	delErr := gocom.KeyVal().Del(cacheKey) // Delete old cache
	if delErr != nil {
		customLogger.WarnWithData("Failed to delete old cache", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "RefreshCache",
			"error":     delErr.Error(),
		})
	}

	// Load from database
	knowledge := aiKnowledgeBase.GetRepo().GetActiveKnowledge(clientId)
	customLogger.InfoWithData("DB query completed", map[string]interface{}{
		"component":    "AIKnowledgeCacheService",
		"function":     "RefreshCache",
		"client_id":    clientId,
		"record_count": len(knowledge),
	})

	if len(knowledge) == 0 {
		customLogger.WarnWithData("No active knowledge found", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "RefreshCache",
			"client_id": clientId,
		})
		// Store empty array to cache to avoid repeated DB queries
		knowledge = []aiKnowledgeBase.AiKnowledgeBase{}
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(knowledge)
	if err != nil {
		customLogger.ErrorWithData("Failed to marshal knowledge", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "RefreshCache",
			"error":     err.Error(),
		})
		return gocom.NewError(500, fmt.Sprintf("Failed to marshal knowledge: %v", err))
	}

	// Store in Redis with 24 hour TTL
	setErr := gocom.KeyVal().Set(cacheKey, string(jsonData), AIKnowledgeCacheTTL)
	if setErr != nil {
		customLogger.ErrorWithData("Failed to set cache", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "RefreshCache",
			"error":     setErr.Error(),
		})
	}

	// Verify the data was stored
	verifyData := gocom.KeyVal().Get(cacheKey)
	if verifyData == "" {
		customLogger.ErrorWithData("Cache verification failed", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "RefreshCache",
			"cache_key": cacheKey,
		})
	} else {
		customLogger.InfoWithData("Cache stored successfully", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "RefreshCache",
			"data_size": len(verifyData),
		})
	}

	// Store last refresh time
	lastRefreshKey := utils.GetCacheKey(constans.AIKnowledgeCacheLastRefresh + clientId)
	_ = gocom.KeyVal().Set(lastRefreshKey, time.Now().Format(time.RFC3339), AIKnowledgeCacheTTL)

	customLogger.InfoWithData("RefreshCache completed", map[string]interface{}{
		"component":       "AIKnowledgeCacheService",
		"function":        "RefreshCache",
		"client_id":       clientId,
		"knowledge_count": len(knowledge),
	})
	return nil
}

// GetKnowledge retrieves knowledge from Redis cache, falls back to DB if not found
func (o *AIKnowledgeCacheSvcImpl) GetKnowledge(clientId string) ([]aiKnowledgeBase.AiKnowledgeBase, *gocom.CodedError) {
	cacheKey := utils.GetCacheKey(constans.AIKnowledgeCachePrefix + clientId)

	// Try to get from Redis cache first
	cachedData := gocom.KeyVal().Get(cacheKey)

	if cachedData != "" {
		var knowledge []aiKnowledgeBase.AiKnowledgeBase
		if err := json.Unmarshal([]byte(cachedData), &knowledge); err != nil {
			customLogger.ErrorWithData("Failed to unmarshal cached data", map[string]interface{}{
				"component": "AIKnowledgeCacheService",
				"function":  "GetKnowledge",
				"error":     err.Error(),
			})
			// Fall through to DB load
		} else {
			customLogger.DebugWithData("Cache hit", map[string]interface{}{
				"component": "AIKnowledgeCacheService",
				"function":  "GetKnowledge",
				"client_id": clientId,
				"count":     len(knowledge),
			})
			return knowledge, nil
		}
	}

	// Cache miss - load from DB and refresh cache
	customLogger.DebugWithData("Cache miss, loading from DB", map[string]interface{}{
		"component": "AIKnowledgeCacheService",
		"function":  "GetKnowledge",
		"client_id": clientId,
	})
	codedErr := o.RefreshCache(clientId)
	if codedErr != nil {
		return nil, codedErr
	}

	// Get from cache after refresh
	cachedData = gocom.KeyVal().Get(cacheKey)
	if cachedData == "" {
		customLogger.WarnWithData("Cache empty after refresh", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "GetKnowledge",
			"client_id": clientId,
		})
		return []aiKnowledgeBase.AiKnowledgeBase{}, nil
	}

	var knowledge []aiKnowledgeBase.AiKnowledgeBase
	if err := json.Unmarshal([]byte(cachedData), &knowledge); err != nil {
		customLogger.ErrorWithData("Failed to unmarshal after refresh", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "GetKnowledge",
			"error":     err.Error(),
		})
		return nil, gocom.NewError(500, fmt.Sprintf("Failed to unmarshal knowledge: %v", err))
	}

	customLogger.DebugWithData("GetKnowledge completed", map[string]interface{}{
		"component": "AIKnowledgeCacheService",
		"function":  "GetKnowledge",
		"client_id": clientId,
		"count":     len(knowledge),
	})
	return knowledge, nil
}

// RefreshAllClients refreshes cache for all clients with active knowledge
func (o *AIKnowledgeCacheSvcImpl) RefreshAllClients() *gocom.CodedError {
	customLogger.InfoWithData("RefreshAllClients started", map[string]interface{}{
		"component": "AIKnowledgeCacheService",
		"function":  "RefreshAllClients",
	})

	// Get distinct client IDs from database
	var clientIds []string
	err := aiKnowledgeBase.GetRepo().
		Model(&aiKnowledgeBase.AiKnowledgeBase{}).
		Where("is_active = ?", true).
		Where("deleted_at IS NULL").
		Distinct().
		Pluck("client_id", &clientIds).Error

	if err != nil {
		customLogger.ErrorWithData("Failed to get client IDs", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "RefreshAllClients",
			"error":     err.Error(),
		})
		return gocom.NewError(500, fmt.Sprintf("Failed to get client IDs: %v", err))
	}

	if len(clientIds) == 0 {
		customLogger.WarnWithData("No active clients found", map[string]interface{}{
			"component": "AIKnowledgeCacheService",
			"function":  "RefreshAllClients",
		})
		return nil
	}

	customLogger.InfoWithData("Refreshing caches", map[string]interface{}{
		"component":    "AIKnowledgeCacheService",
		"function":     "RefreshAllClients",
		"client_count": len(clientIds),
	})

	// Refresh cache for each client
	successCount := 0
	failCount := 0
	for _, clientId := range clientIds {
		if codedErr := o.RefreshCache(clientId); codedErr != nil {
			customLogger.ErrorWithData("Failed to refresh cache for client", map[string]interface{}{
				"component": "AIKnowledgeCacheService",
				"function":  "RefreshAllClients",
				"client_id": clientId,
				"error":     codedErr.Message,
			})
			failCount++
		} else {
			successCount++
		}
	}

	customLogger.InfoWithData("RefreshAllClients completed", map[string]interface{}{
		"component":     "AIKnowledgeCacheService",
		"function":      "RefreshAllClients",
		"success_count": successCount,
		"fail_count":    failCount,
	})

	if failCount > 0 {
		return gocom.NewError(500, fmt.Sprintf("Refresh completed with %d failures out of %d clients", failCount, len(clientIds)))
	}

	return nil
}

//---------------------------------------

var aiKnowledgeCacheSvc *AIKnowledgeCacheSvcImpl
var aiKnowledgeCacheSvcOnce sync.Once

func GetAIKnowledgeCacheSvc() AIKnowledgeCacheSvc {
	if aiKnowledgeCacheSvc == nil {
		aiKnowledgeCacheSvcOnce.Do(func() {
			aiKnowledgeCacheSvc = &AIKnowledgeCacheSvcImpl{}
		})
	}
	return aiKnowledgeCacheSvc
}
