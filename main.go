package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// Get environment variables for Telegram Bot Token, Chat ID, and Allowed CORS Domain
var TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
var TelegramChatID = os.Getenv("TELEGRAM_CHAT_ID")
var AllowedCORSOrigin = os.Getenv("ALLOWED_CORS_ORIGIN")

func main() {
	// Check if the required environment variables are set
	if TelegramBotToken == "" || TelegramChatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID must be set as environment variables")
	}

	http.HandleFunc("/webhook", corsMiddleware(webhookHandler))
	http.HandleFunc("/", corsMiddleware(ok))

	// Start the server on port 80
	log.Println("Starting server on :80")
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}

// CORS Middleware to add CORS headers
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers only if AllowedCORSOrigin is set
		if AllowedCORSOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", AllowedCORSOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true") // Allow credentials
		}

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func ok(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Could not parse JSON", http.StatusBadRequest)
		return
	}

	if err := sendToTelegram(data); err != nil {
		log.Printf("Error sending message to Telegram: %s\n", err.Error())
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func formatTimingData(timingData map[string]interface{}) string {
	formatted := ""
	for key, value := range timingData {
		if num, ok := value.(float64); ok {
			formatted += fmt.Sprintf("*%s:* %.2f ms\n", key, num/1e6) // convert nanoseconds to milliseconds
		} else {
			formatted += fmt.Sprintf("*%s:* %v\n", key, value)
		}
	}
	return formatted
}

func formatResourcesData(resourcesData []interface{}) string {
	formatted := ""
	for _, resource := range resourcesData {
		if resourceMap, ok := resource.(map[string]interface{}); ok {
			for key, value := range resourceMap {
				if num, ok := value.(float64); ok {
					formatted += fmt.Sprintf("*%s:* %.2f ms\n", key, num/1e3) // convert microseconds to milliseconds
				} else {
					formatted += fmt.Sprintf("*%s:* %v\n", key, value)
				}
			}
			formatted += "\n"
		}
	}
	return formatted
}

func sendToTelegram(data map[string]interface{}) error {
	var message strings.Builder
	message.WriteString("*Received message:*\n\n")

	for key, value := range data {
		switch key {
		case "timing":
			if timingMap, ok := value.(map[string]interface{}); ok {
				message.WriteString("*Timing:*\n")
				message.WriteString(formatTimingData(timingMap))
			}
		case "resources":
			if resourcesArray, ok := value.([]interface{}); ok {
				message.WriteString("*Resources:*\n")
				message.WriteString(formatResourcesData(resourcesArray))
			}
		default:
			message.WriteString(fmt.Sprintf("*%s:* %v\n", key, value))
		}
	}

	// Telegram API URL
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", TelegramBotToken)

	// Create a map to hold the message payload
	payload := map[string]string{
		"chat_id":    TelegramChatID,
		"text":       message.String(),
		"parse_mode": "Markdown", // To format the text with bold etc.
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message to Telegram, status code: %d", resp.StatusCode)
	}

	return nil
}
