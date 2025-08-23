package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

type SendReq struct {
	SMTPHost   string   `json:"smtp_host"`
	SMTPPort   int      `json:"smtp_port"`
	SMTPUser   string   `json:"smtp_user"`
	SMTPPass   string   `json:"smtp_pass"`
	Encryption string   `json:"encryption"` // STARTTLS | SSL | NONE
	From       string   `json:"from"`
	To         []string `json:"to"`
	Cc         []string `json:"cc,omitempty"`
	Bcc        []string `json:"bcc,omitempty"`
	Subject    string   `json:"subject"`
	Text       string   `json:"text,omitempty"`
	HTML       string   `json:"html,omitempty"`
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req SendReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	smtp := mail.NewSMTPClient()
	smtp.Host = req.SMTPHost
	smtp.Port = req.SMTPPort
	smtp.Username = req.SMTPUser
	smtp.Password = req.SMTPPass
	smtp.ConnectTimeout = 15 * time.Second
	smtp.SendTimeout = 30 * time.Second
	switch req.Encryption {
	case "STARTTLS":
		smtp.Encryption = mail.EncryptionSTARTTLS
	case "SSL":
		smtp.Encryption = mail.EncryptionSSL
	default:
		smtp.Encryption = mail.EncryptionNone
	}
	client, err := smtp.Connect()
	if err != nil {
		http.Error(w, "SMTP connect failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	email := mail.NewMSG()
	email.SetFrom(req.From)
	for _, t := range req.To { email.AddTo(t) }
	for _, c := range req.Cc { email.AddCc(c) }
	for _, b := range req.Bcc { email.AddBcc(b) }
	email.SetSubject(req.Subject)
	if req.HTML != "" {
		if req.Text != "" {
			email.SetBody(mail.TextPlain, req.Text)
			email.AddAlternative(mail.TextHTML, req.HTML)
		} else {
			email.SetBody(mail.TextHTML, req.HTML)
		}
	} else {
		email.SetBody(mail.TextPlain, req.Text)
	}
	if email.Error != nil {
		http.Error(w, "email build error: "+email.Error.Error(), http.StatusBadRequest)
		return
	}
	if err := email.Send(client); err != nil {
		http.Error(w, "send failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func main() {
	http.HandleFunc("/send", sendHandler)
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	log.Println("HTTP→SMTP listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
