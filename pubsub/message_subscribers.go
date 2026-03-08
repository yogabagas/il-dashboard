package pubsub

import (
	"encoding/json"
	"sync"

	"github.com/ariandi/gocom/pubsub"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
)

// ============================================================================
// Message Subscribers - Subscribe to pubsub topics and route to handlers
// ============================================================================

const (
	logMsgSubscribed = "Subscribed to topic"
)

// MessageSubscribers manages all message topic subscriptions
type MessageSubscribers struct {
	ppobHandler      *WhatsappWebHookImpl
	csHandler        *CSEscalationHandler
	knowledgeHandler *CSEscalationHandler // Reuse same handler, different routing
	imageProcessor   *ImageProcessor      // Image OCR processor
}

// NewMessageSubscribers creates new message subscribers
func NewMessageSubscribers() *MessageSubscribers {
	return &MessageSubscribers{
		ppobHandler:      NewWhatsappWebHook(),
		csHandler:        NewCSEscalationHandler(),
		knowledgeHandler: NewCSEscalationHandler(),
		imageProcessor:   NewImageProcessor(),
	}
}

// SubscribeAll subscribes to all message routing topics
func (s *MessageSubscribers) SubscribeAll() {
	customLogger.InfoWithData("Subscribing to all message routing topics", map[string]interface{}{
		"component": "MessageSubscribers",
		"action":    "subscribe_all",
	})

	// Subscribe to PPOB topic
	pubsub.Get().Subscribe(dtos.TopicWAMessageAIPPOB, s.handlePPOBMessage())
	customLogger.InfoWithData(logMsgSubscribed, map[string]interface{}{
		"component": "MessageSubscribers",
		"topic":     dtos.TopicWAMessageAIPPOB,
		"type":      "PPOB",
	})

	// Subscribe to CS topic
	pubsub.Get().Subscribe(dtos.TopicWAMessageAICS, s.handleCSMessage())
	customLogger.InfoWithData(logMsgSubscribed, map[string]interface{}{
		"component": "MessageSubscribers",
		"topic":     dtos.TopicWAMessageAICS,
		"type":      "CS",
	})

	// Subscribe to Knowledge topic
	pubsub.Get().Subscribe(dtos.TopicWAMessageAIKnowledge, s.handleKnowledgeMessage())
	customLogger.InfoWithData(logMsgSubscribed, map[string]interface{}{
		"component": "MessageSubscribers",
		"topic":     dtos.TopicWAMessageAIKnowledge,
		"type":      "Knowledge",
	})

	// Subscribe to Image processing topic
	pubsub.Get().Subscribe(dtos.TopicWAMessageImage, s.handleImageMessage())
	customLogger.InfoWithData(logMsgSubscribed, map[string]interface{}{
		"component": "MessageSubscribers",
		"topic":     dtos.TopicWAMessageImage,
		"type":      "Image",
	})

	customLogger.InfoWithData("All subscriptions registered successfully", map[string]interface{}{
		"component":       "MessageSubscribers",
		"topics_count":    4,
		"topics":          []string{"PPOB", "CS", "Knowledge", "Image"},
		"status":          "success",
	})
}

// handlePPOBMessage handles messages routed to PPOB topic
func (s *MessageSubscribers) handlePPOBMessage() pubsub.PubSubEventHandler {
	return func(name, message string) {
		customLogger.InfoWithData("Received PPOB message", map[string]interface{}{
			"component": "MessageSubscribers",
			"topic":     name,
			"type":      "PPOB",
			"message":   message,
		})

		// Parse message event
		var event dtos.WAMessageEvent
		if err := json.Unmarshal([]byte(message), &event); err != nil {
			customLogger.ErrorWithData("Failed to unmarshal PPOB event", map[string]interface{}{
				"component": "MessageSubscribers",
				"topic":     name,
				"type":      "PPOB",
				"error":     err.Error(),
				"message":   message,
			})
			return
		}

		customLogger.InfoWithData("Processing PPOB flow", map[string]interface{}{
			"component":   "MessageSubscribers",
			"type":        "PPOB",
			"from_number": event.FromNumber,
			"to_number":   event.ToNumber,
			"message":     event.Message,
			"session_id":  event.SessionID,
		})

		// Call existing PPOB handler (triggerAIConversation in updateStatus.go)
		s.ppobHandler.triggerAIConversation(
			event.SessionID,
			event.FromNumber,
			event.Message,
			event.ToNumber,
		)
	}
}

// handleCSMessage handles messages routed to CS topic
func (s *MessageSubscribers) handleCSMessage() pubsub.PubSubEventHandler {
	return func(name, message string) {
		customLogger.InfoWithData("Received CS message", map[string]interface{}{
			"component": "MessageSubscribers",
			"topic":     name,
			"type":      "CS",
			"message":   message,
		})

		// Parse message event
		var event dtos.WAMessageEvent
		if err := json.Unmarshal([]byte(message), &event); err != nil {
			customLogger.ErrorWithData("Failed to unmarshal CS event", map[string]interface{}{
				"component": "MessageSubscribers",
				"topic":     name,
				"type":      "CS",
				"error":     err.Error(),
				"message":   message,
			})
			return
		}

		customLogger.InfoWithData("Processing CS escalation", map[string]interface{}{
			"component":   "MessageSubscribers",
			"type":        "CS",
			"from_number": event.FromNumber,
			"to_number":   event.ToNumber,
			"message":     event.Message,
			"session_id":  event.SessionID,
		})

		// Call CS escalation handler
		s.csHandler.TriggerAIWithEscalation(
			event.SessionID,
			event.FromNumber,
			event.Message,
			event.ToNumber,
		)
	}
}

// handleKnowledgeMessage handles messages routed to Knowledge topic
func (s *MessageSubscribers) handleKnowledgeMessage() pubsub.PubSubEventHandler {
	return func(name, message string) {
		customLogger.InfoWithData("Received Knowledge message", map[string]interface{}{
			"component": "MessageSubscribers",
			"topic":     name,
			"type":      "Knowledge",
			"message":   message,
		})

		// Parse message event
		var event dtos.WAMessageEvent
		if err := json.Unmarshal([]byte(message), &event); err != nil {
			customLogger.ErrorWithData("Failed to unmarshal Knowledge event", map[string]interface{}{
				"component": "MessageSubscribers",
				"topic":     name,
				"type":      "Knowledge",
				"error":     err.Error(),
				"message":   message,
			})
			return
		}

		customLogger.InfoWithData("Processing Knowledge conversation", map[string]interface{}{
			"component":   "MessageSubscribers",
			"type":        "Knowledge",
			"from_number": event.FromNumber,
			"to_number":   event.ToNumber,
			"message":     event.Message,
			"session_id":  event.SessionID,
			"note":        "simple chat without function calling",
		})

		// Use simple knowledge chat WITHOUT function calling
		// This prevents markdown link issues and is more reliable for general Q&A
		// Function calling (PPOB inquiry/payment) is handled by PPOB topic
		s.knowledgeHandler.TriggerSimpleKnowledgeChat(
			event.SessionID,
			event.FromNumber,
			event.Message,
			event.ToNumber,
		)
	}
}

// handleImageMessage handles messages routed to Image topic
func (s *MessageSubscribers) handleImageMessage() pubsub.PubSubEventHandler {
	return func(name, message string) {
		customLogger.InfoWithData("Received Image message", map[string]interface{}{
			"component": "MessageSubscribers",
			"topic":     name,
			"type":      "Image",
			"message":   message,
		})

		// Parse message event
		var event dtos.WAImageMessageEvent
		if err := json.Unmarshal([]byte(message), &event); err != nil {
			customLogger.ErrorWithData("Failed to unmarshal Image event", map[string]interface{}{
				"component": "MessageSubscribers",
				"topic":     name,
				"type":      "Image",
				"error":     err.Error(),
				"message":   message,
			})
			return
		}

		customLogger.InfoWithData("Processing Image OCR", map[string]interface{}{
			"component":   "MessageSubscribers",
			"type":        "Image",
			"from_number": event.FromNumber,
			"to_number":   event.ToNumber,
			"image_id":    event.ImageID,
			"session_id":  event.SessionID,
		})

		// Call image processor
		s.imageProcessor.ProcessImage(event)
	}
}

// ============================================================================
// Singleton Pattern for Subscribers
// ============================================================================

var (
	subscribersInstance *MessageSubscribers
	subscribersOnce     sync.Once
)

// InitMessageSubscribers initializes and starts message subscribers (singleton)
func InitMessageSubscribers() {
	subscribersOnce.Do(func() {
		customLogger.InfoWithData("Initializing message routing subscribers", map[string]interface{}{
			"component": "MessageSubscribers",
			"action":    "initialize",
		})
		subscribersInstance = NewMessageSubscribers()
		subscribersInstance.SubscribeAll()
		customLogger.InfoWithData("Message routing subscribers initialized successfully", map[string]interface{}{
			"component": "MessageSubscribers",
			"action":    "initialize",
			"status":    "success",
		})
	})
}

// GetMessageSubscribers returns the singleton instance
func GetMessageSubscribers() *MessageSubscribers {
	if subscribersInstance == nil {
		InitMessageSubscribers()
	}
	return subscribersInstance
}
