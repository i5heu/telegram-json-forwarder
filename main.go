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

var TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
var TelegramChatID = os.Getenv("TELEGRAM_CHAT_ID")
var AllowedCORSOrigin = os.Getenv("ALLOWED_CORS_ORIGIN")

func main() {
	if TelegramBotToken == "" || TelegramChatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID must be set as environment variables")
	}

	http.HandleFunc("/webhook", corsMiddleware(webhookHandler))
	http.HandleFunc("/", corsMiddleware(ok))

	log.Println("Starting server on :80")
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if AllowedCORSOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", AllowedCORSOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

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
	// Start from navigationStart
	navigationStart := timingData["navigationStart"].(float64)

	calculatedTimes := map[string]float64{
		"Redirect":          timingData["redirectEnd"].(float64) - timingData["redirectStart"].(float64),
		"AppCache":          timingData["domainLookupStart"].(float64) - timingData["fetchStart"].(float64),
		"DNS Lookup":        timingData["domainLookupEnd"].(float64) - timingData["domainLookupStart"].(float64),
		"TCP Connection":    timingData["connectEnd"].(float64) - timingData["connectStart"].(float64),
		"SSL Handshake":     timingData["connectEnd"].(float64) - timingData["secureConnectionStart"].(float64),
		"Request Sent":      timingData["responseStart"].(float64) - timingData["requestStart"].(float64),
		"Response Received": timingData["responseEnd"].(float64) - timingData["responseStart"].(float64),
		"DOM Processing":    timingData["domComplete"].(float64) - timingData["domLoading"].(float64),
		"Load Event":        timingData["loadEventEnd"].(float64) - timingData["loadEventStart"].(float64),
		"DOMContentLoaded":  timingData["domContentLoadedEventEnd"].(float64) - navigationStart,
		"Finish":            timingData["loadEventEnd"].(float64) - navigationStart,
	}

	// Create a waterfall-like output
	waterfall := "*Waterfall Timing (in ms):*\n"
	for key, value := range calculatedTimes {
		if value > 0 {
			waterfall += fmt.Sprintf("*%s:* %.2f ms\n", key, value)
		}
	}

	// Final DOM loading time from navigation start to domLoading
	domLoadingTime := timingData["domLoading"].(float64) - navigationStart
	waterfall += fmt.Sprintf("\n*DOM Loading Time:* %.2f ms\n", domLoadingTime)

	return waterfall
}

func sendToTelegram(data map[string]interface{}) error {
	var message strings.Builder
	message.WriteString("*Received message:*\n\n")

	for key, value := range data {
		switch key {
		case "timing":
			if timingMap, ok := value.(map[string]interface{}); ok {
				message.WriteString(formatTimingData(timingMap))
			}
		default:
			message.WriteString(fmt.Sprintf("*%s:* %v\n", key, value))
		}
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", TelegramBotToken)

	payload := map[string]string{
		"chat_id":    TelegramChatID,
		"text":       message.String(),
		"parse_mode": "Markdown",
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
