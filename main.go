package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

type MailRequest struct {
	SMTPHost  string   `json:"smtp_host"`
	SMTPPort  int      `json:"smtp_port"`
	SMTPUser  string   `json:"smtp_user"`
	SMTPPass  string   `json:"smtp_pass"`
	Encryption string  `json:"encryption"`
	From      string   `json:"from"`
	To        []string `json:"to"`
	Subject   string   `json:"subject"`
	Text      string   `json:"text"`
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload MailRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Timeout ENV (default 10 saniye)
	timeoutSec := 30
	if v := os.Getenv("TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
		}
	}

	// SMTP client ayarları
	client := mail.NewSMTPClient()
	client.Host = payload.SMTPHost
	client.Port = payload.SMTPPort
	client.Username = payload.SMTPUser
	client.Password = payload.SMTPPass
	client.ConnectTimeout = time.Duration(timeoutSec) * time.Second
	client.SendTimeout = time.Duration(timeoutSec) * time.Second
	client.KeepAlive = false

	switch payload.Encryption {
	case "SSL":
		client.Encryption = mail.EncryptionSSL
	case "STARTTLS":
		client.Encryption = mail.EncryptionSTARTTLS
	default:
		client.Encryption = mail.EncryptionNone
	}

	// SMTP server’a bağlan
	smtpClient, err := client.Connect()
	if err != nil {
		http.Error(w, fmt.Sprintf("SMTP connect failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Mail hazırla
	email := mail.NewMSG()
	email.SetFrom(payload.From).
		AddTo(payload.To...).
		SetSubject(payload.Subject)
	email.SetBody(mail.TextPlain, payload.Text)

	if email.Error != nil {
		http.Error(w, fmt.Sprintf("Mail build error: %v", email.Error), http.StatusBadRequest)
		return
	}

	// Mail gönder
	if err := email.Send(smtpClient); err != nil {
		http.Error(w, fmt.Sprintf("Mail send error: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Mail sent successfully"))
}

func main() {
	http.HandleFunc("/send", sendHandler)

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Printf("Listening on :%s ...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
