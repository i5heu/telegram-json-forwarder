package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// Get environment variables for Telegram Bot Token and Chat ID
var TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
var TelegramChatID = os.Getenv("TELEGRAM_CHAT_ID")

func main() {
	// Check if the required environment variables are set
	if TelegramBotToken == "" || TelegramChatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID must be set as environment variables")
	}

	http.HandleFunc("/webhook", webhookHandler)

	// Start the server on port 8080
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
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
		http.Error(w, "Could not send message to Telegram", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Message forwarded to Telegram successfully")
}

func sendToTelegram(data map[string]interface{}) error {
	message := "*Received message:*\n"

	for key, value := range data {
		message += fmt.Sprintf("*%s:* %v\n", key, value)
	}

	// Telegram API URL
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", TelegramBotToken)

	// Create a map to hold the message payload
	payload := map[string]string{
		"chat_id":    TelegramChatID,
		"text":       message,
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
