package pubsub

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ariandi/gocom/config"
	"gitlab.com/bot3342545/il-dashboard/constans"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	customLogger "gitlab.com/bot3342545/il-dashboard/logger"
	"gitlab.com/bot3342545/il-dashboard/repositories/messageConversation"
	"gitlab.com/bot3342545/il-dashboard/services"
)

// ============================================================================
// Image Processor - Download and OCR images from WhatsApp
// ============================================================================

type ImageProcessor struct {
	messageRouter *MessageRouter
}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		messageRouter: NewMessageRouter(),
	}
}

// ProcessImage processes incoming image messages
func (p *ImageProcessor) ProcessImage(event dtos.WAImageMessageEvent) {
	customLogger.InfoWithData("Processing image message", map[string]interface{}{
		"component":   "ImageProcessor",
		"function":    "ProcessImage",
		"from_number": event.FromNumber,
		"image_id":    event.ImageID,
		"session_id":  event.SessionID,
	})

	// Step 1: Download image from WhatsApp
	imageData, err := p.downloadWhatsAppImage(event.ImageID)
	if err != nil {
		customLogger.ErrorWithData("Failed to download image", map[string]interface{}{
			"component":   "ImageProcessor",
			"function":    "ProcessImage",
			"error":       err.Error(),
			"from_number": event.FromNumber,
			"image_id":    event.ImageID,
		})
		p.sendErrorMessage(event.FromNumber, "Maaf, gagal mengunduh gambar. Silakan coba lagi.")
		return
	}

	customLogger.InfoWithData("Image downloaded successfully", map[string]interface{}{
		"component":   "ImageProcessor",
		"function":    "ProcessImage",
		"from_number": event.FromNumber,
		"image_size":  len(imageData),
		"image_id":    event.ImageID,
	})

	// Step 2: Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Step 3: Extract text from image using Groq Vision API
	extractedText, err := p.extractTextFromImage(base64Image, event.ImageMimeType, event.ImageCaption)
	if err != nil {
		customLogger.ErrorWithData("Failed to extract text from image", map[string]interface{}{
			"component":   "ImageProcessor",
			"function":    "ProcessImage",
			"error":       err.Error(),
			"from_number": event.FromNumber,
			"image_id":    event.ImageID,
		})
		p.sendErrorMessage(event.FromNumber, "Maaf, gagal membaca gambar. Silakan coba kirim ulang atau ketik manual.")
		return
	}

	customLogger.InfoWithData("Text extracted successfully", map[string]interface{}{
		"component":      "ImageProcessor",
		"function":       "ProcessImage",
		"from_number":    event.FromNumber,
		"extracted_text": extractedText,
		"text_length":    len(extractedText),
	})

	// Step 4: Send OCR result to user with confirmation buttons
	p.sendOCRConfirmation(event.FromNumber, event.ClientID, extractedText)

	customLogger.InfoWithData("OCR result sent to user", map[string]interface{}{
		"component":   "ImageProcessor",
		"function":    "ProcessImage",
		"from_number": event.FromNumber,
		"status":      "awaiting_confirmation",
	})
}

// downloadWhatsAppImage downloads image from WhatsApp using Media ID
func (p *ImageProcessor) downloadWhatsAppImage(mediaID string) ([]byte, error) {
	customLogger.DebugWithData("Downloading WhatsApp image", map[string]interface{}{
		"component": "ImageProcessor",
		"function":  "downloadWhatsAppImage",
		"media_id":  mediaID,
	})

	// Get WhatsApp credentials
	accessToken := config.Get(constans.WaAuthToken)
	if accessToken == "" {
		return nil, fmt.Errorf("WhatsApp access token not configured")
	}

	// Step 1: Get media URL from WhatsApp
	mediaURL, mimeType, err := p.getMediaURL(mediaID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get media URL: %v", err)
	}

	customLogger.InfoWithData("Media URL obtained", map[string]interface{}{
		"component": "ImageProcessor",
		"function":  "downloadWhatsAppImage",
		"media_id":  mediaID,
		"mime_type": mimeType,
	})

	// Step 2: Download media from URL
	imageData, err := p.downloadMedia(mediaURL, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %v", err)
	}

	return imageData, nil
}

// getMediaURL retrieves media URL from WhatsApp API
func (p *ImageProcessor) getMediaURL(mediaID, accessToken string) (string, string, error) {
	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s", mediaID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("WhatsApp API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var mediaInfo struct {
		URL      string `json:"url"`
		MimeType string `json:"mime_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mediaInfo); err != nil {
		return "", "", err
	}

	return mediaInfo.URL, mediaInfo.MimeType, nil
}

// downloadMedia downloads media file from URL
func (p *ImageProcessor) downloadMedia(mediaURL, accessToken string) ([]byte, error) {
	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to download media (status %d)", resp.StatusCode)
	}

	// Read media data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return imageData, nil
}

// extractTextFromImage uses Groq Vision API to extract text from image
func (p *ImageProcessor) extractTextFromImage(base64Image, mimeType, caption string) (string, error) {
	customLogger.DebugWithData("Extracting text using Groq Vision API", map[string]interface{}{
		"component":    "ImageProcessor",
		"function":     "extractTextFromImage",
		"mime_type":    mimeType,
		"has_caption":  caption != "",
		"image_size":   len(base64Image),
	})

	// Get Groq API key
	groqAPIKey := config.Get("app.api.key.groq", "")
	if groqAPIKey == "" {
		return "", fmt.Errorf("Groq API key not configured")
	}

	// Construct prompt for OCR
	systemPrompt := `Kamu adalah asisten OCR (Optical Character Recognition) untuk PDAM Kabupaten Tangerang.

Tugasmu:
1. Baca dan extract SEMUA text yang terlihat di gambar
2. Jika gambar adalah:
   - Meteran air → extract angka meteran
   - Tagihan → extract nomor pelanggan, periode, jumlah tagihan
   - Dokumen → extract text yang relevan
   - Foto kerusakan/masalah → deskripsikan masalahnya

3. Format output:
   - Tulis semua text yang kamu baca
   - Jika ada nomor penting (nomor pelanggan, meteran), tulis dengan jelas
   - Jika ada angka tagihan, tulis dengan format: Rp X.XXX

PENTING: Output HANYA berisi text hasil OCR, tanpa tambahan penjelasan atau formatting markdown.`

	userPrompt := "Baca semua text yang ada di gambar ini dengan teliti."
	if caption != "" {
		userPrompt = fmt.Sprintf("User caption: %s\n\nBaca semua text yang ada di gambar ini dengan teliti.", caption)
	}

	// Construct request body for Groq Vision API
	requestBody := map[string]interface{}{
		"model": constans.AIModel, // Use same model (llama-4-scout supports vision)
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": userPrompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image),
						},
					},
				},
			},
		},
		"temperature": 0.2, // Low temperature for more accurate OCR
		"max_tokens":  2000,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// Call Groq API
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+groqAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Groq API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var groqResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", err
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq API")
	}

	extractedText := groqResp.Choices[0].Message.Content
	customLogger.InfoWithData("OCR completed", map[string]interface{}{
		"component":      "ImageProcessor",
		"function":       "extractTextFromImage",
		"extracted_text": extractedText,
		"text_length":    len(extractedText),
	})

	return extractedText, nil
}

// sendOCRConfirmation sends OCR result to user with confirmation buttons
func (p *ImageProcessor) sendOCRConfirmation(userPhone, clientID, extractedText string) {
	customLogger.InfoWithData("Sending OCR confirmation", map[string]interface{}{
		"component":   "ImageProcessor",
		"function":    "sendOCRConfirmation",
		"user_phone":  userPhone,
		"client_id":   clientID,
		"text_length": len(extractedText),
	})

	// Store OCR result in conversation metadata for later use
	sessionID := fmt.Sprintf("ocr_%s_%d", userPhone, time.Now().Unix())

	// Save OCR result to conversation
	conv := &messageConversation.MessageConversation{
		SessionID:          sessionID,
		ClientId:           clientID,
		FromNumber:         userPhone,
		Role:               "system",
		Message:            extractedText,
		Status:             "pending_confirmation",
		ConversationStatus: "active",
		Model:              constans.AIModel,
	}
	messageConversation.GetRepo().Create(conv)

	// Create confirmation message
	bodyText := fmt.Sprintf("📸 *Hasil Pembacaan Gambar:*\n\n%s\n\n_Apakah pembacaan ini sudah benar?_", extractedText)

	// Create interactive buttons
	buttons := []map[string]string{
		{
			"id":    fmt.Sprintf("ocr_confirm_%s", sessionID),
			"title": "✅ Benar",
		},
		{
			"id":    fmt.Sprintf("ocr_reject_%s", sessionID),
			"title": "❌ Salah",
		},
	}

	// Send interactive button message
	_, _, err := services.GetWASendSvc().SendInteractiveButtonMessage(
		userPhone,
		bodyText,
		buttons,
		clientID,
		"", // businessPhone not available - will use fallback config
	)

	if err != nil {
		customLogger.ErrorWithData("Failed to send OCR confirmation", map[string]interface{}{
			"component":  "ImageProcessor",
			"function":   "sendOCRConfirmation",
			"error":      err.Message,
			"user_phone": userPhone,
		})
		// Fallback: send as text
		services.GetWASendSvc().SendTextMessageWithLog(userPhone, bodyText, clientID, "")
	} else {
		customLogger.InfoWithData("OCR confirmation sent successfully", map[string]interface{}{
			"component":   "ImageProcessor",
			"function":    "sendOCRConfirmation",
			"user_phone":  userPhone,
			"session_id":  fmt.Sprintf("ocr_%s_%d", userPhone, time.Now().Unix()),
			"status":      "success",
		})
	}
}

// sendErrorMessage sends error message to user
func (p *ImageProcessor) sendErrorMessage(userPhone, errorMsg string) {
	customLogger.WarnWithData("Sending error message to user", map[string]interface{}{
		"component":   "ImageProcessor",
		"function":    "sendErrorMessage",
		"user_phone":  userPhone,
		"error_msg":   errorMsg,
	})

	// Use empty clientID - will use default
	services.GetWASendSvc().SendTextMessage(userPhone, errorMsg)
}
