package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ariandi/gocom/config"
	"github.com/ariandi/gocom/logger"
	"gitlab.com/bot3342545/il-dashboard/constans"
)

// ============================================================================
// WhatsApp Interactive Message Helpers
// ============================================================================

// WAInteractiveButton represents a single reply button
type WAInteractiveButton struct {
	Type  string                   `json:"type"`
	Reply WAInteractiveButtonReply `json:"reply"`
}

type WAInteractiveButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// WAInteractiveMessage represents WhatsApp interactive message structure
type WAInteractiveMessage struct {
	MessagingProduct string                      `json:"messaging_product"`
	RecipientType    string                      `json:"recipient_type"`
	To               string                      `json:"to"`
	Type             string                      `json:"type"`
	Interactive      WAInteractiveMessageContent `json:"interactive"`
}

type WAInteractiveMessageContent struct {
	Type   string                     `json:"type"`
	Body   WAInteractiveMessageBody   `json:"body"`
	Action WAInteractiveMessageAction `json:"action"`
}

type WAInteractiveMessageBody struct {
	Text string `json:"text"`
}

type WAInteractiveMessageAction struct {
	Buttons []WAInteractiveButton `json:"buttons"`
}

// SendInteractiveButtonMessage sends WhatsApp message with reply buttons
// maxButtons: 1-3 buttons (WhatsApp limitation)
func SendInteractiveButtonMessage(
	phoneNumberID string,
	accessToken string,
	to string,
	bodyText string,
	buttons []WAInteractiveButton,
) (string, error) {
	logger.Infof("[WAHelper SendInteractiveButtonMessage] Sending to %s with %d buttons", to, len(buttons))

	// Validation
	if len(buttons) == 0 || len(buttons) > 3 {
		return "", fmt.Errorf("buttons count must be 1-3, got %d", len(buttons))
	}

	// Prepare request payload
	payload := WAInteractiveMessage{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "interactive",
		Interactive: WAInteractiveMessageContent{
			Type: "button",
			Body: WAInteractiveMessageBody{
				Text: bodyText,
			},
			Action: WAInteractiveMessageAction{
				Buttons: buttons,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("[WAHelper SendInteractiveButtonMessage] Failed to marshal payload: %v", err)
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	logger.Debugf("[WAHelper SendInteractiveButtonMessage] Payload: %s", string(jsonData))

	// Send to WhatsApp API
	url := fmt.Sprintf("%s/%s/messages", config.Get(constans.BaseURLMeta), phoneNumberID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Errorf("[WAHelper SendInteractiveButtonMessage] Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("[WAHelper SendInteractiveButtonMessage] Failed to send request: %v", err)
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("[WAHelper SendInteractiveButtonMessage] Failed to read response: %v", err)
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	logger.Infof("[WAHelper SendInteractiveButtonMessage] WhatsApp API response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("[WAHelper SendInteractiveButtonMessage] WhatsApp API error: %s", string(body))
		return "", fmt.Errorf("WhatsApp API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response to get message ID
	var response struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		logger.Errorf("[WAHelper SendInteractiveButtonMessage] Failed to unmarshal response: %v", err)
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(response.Messages) == 0 {
		logger.Errorf("[WAHelper SendInteractiveButtonMessage] No message ID in response")
		return "", fmt.Errorf("no message ID in response")
	}

	messageID := response.Messages[0].ID
	logger.Infof("[WAHelper SendInteractiveButtonMessage] Message sent successfully. MessageID: %s", messageID)

	return messageID, nil
}

// CreateCSEscalationButton creates a button for CS escalation
func CreateCSEscalationButton() WAInteractiveButton {
	return WAInteractiveButton{
		Type: "reply",
		Reply: WAInteractiveButtonReply{
			ID:    "btn_escalate_cs",
			Title: "Hubungkan dengan CS",
		},
	}
}
