package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

type Payload struct {
	SMTPHost   string   `json:"smtp_host"`
	SMTPPort   int      `json:"smtp_port"`
	SMTPUser   string   `json:"smtp_user"`
	SMTPPass   string   `json:"smtp_pass"`
	Encryption string   `json:"encryption"` // "STARTTLS" | "SSLTLS" | "NONE"
	From       string   `json:"from"`
	To         []string `json:"to"`
	Subject    string   `json:"subject"`
	HTML       string   `json:"html"`
	Text       string   `json:"text,omitempty"` // opsiyonel fallback
}

func encFromString(s string) mail.Encryption {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "STARTTLS":
		return mail.EncryptionSTARTTLS
	case "SSLTLS", "SSL", "TLS":
		return mail.EncryptionSSLTLS
	case "NONE", "":
		return mail.EncryptionNone
	default:
		return mail.EncryptionSTARTTLS
	}
}

func main() {
	// 1) JSON oku (dosyadan veya stdin)
	//   - dosyadan: go run main.go payload.json
	//   - stdin:    cat payload.json | go run main.go
	var data []byte
	var err error

	if len(os.Args) > 1 {
		data, err = os.ReadFile(os.Args[1])
	} else {
		data, err = os.ReadFile("payload.json")
		// stdin istersen:
		// data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		log.Fatal("JSON okunamadı:", err)
	}

	var p Payload
	if err := json.Unmarshal(data, &p); err != nil {
		log.Fatal("JSON parse hatası:", err)
	}

	// 2) Basit validasyon
	if p.SMTPHost == "" || p.SMTPPort == 0 || p.SMTPUser == "" || p.SMTPPass == "" {
		log.Fatal(errors.New("smtp_host, smtp_port, smtp_user, smtp_pass zorunlu"))
	}
	if len(p.To) == 0 {
		log.Fatal(errors.New("en az bir 'to' alıcısı gerekli"))
	}
	if p.From == "" {
		p.From = p.SMTPUser // çoğu SMTP'de kimliklenen adresi kullanmak daha sorunsuz
	}

	// Gmail App Password boşluksuz girilmeli (Gmail arayüzü boşluklu gösterir)
	p.SMTPPass = strings.ReplaceAll(p.SMTPPass, " ", "")

	// 3) SMTP client hazırla
	client := mail.NewSMTPClient()
	client.Host = p.SMTPHost
	client.Port = p.SMTPPort
	client.Username = p.SMTPUser
	client.Password = p.SMTPPass
	client.Encryption = encFromString(p.Encryption)
	client.Authentication = mail.AuthAuto
	client.ConnectTimeout = 20 * time.Second
	client.SendTimeout = 30 * time.Second
	client.KeepAlive = false

	smtpClient, err := client.Connect()
	if err != nil {
		log.Fatalf("SMTP bağlantı hatası: %v", err)
	}

	// 4) Mesajı hazırla
	msg := mail.NewMSG().
		SetFrom(p.From).
		SetSubject(p.Subject)

	for _, to := range p.To {
		msg.AddTo(strings.TrimSpace(to))
	}

	// HTML + (opsiyonel) text fallback
	if p.HTML != "" {
		msg.SetBody(mail.TextHTML, p.HTML)
		if p.Text != "" {
			msg.AddAlternative(mail.TextPlain, p.Text)
		}
	} else if p.Text != "" {
		// HTML yoksa düz metin gönder
		msg.SetBody(mail.TextPlain, p.Text)
	} else {
		log.Fatal("Ne 'html' ne de 'text' verilmiş; gövde boş olamaz")
	}

	if msg.Error != nil {
		log.Fatal("Mesaj oluşturma hatası:", msg.Error)
	}

	// 5) Gönder
	if err := msg.Send(smtpClient); err != nil {
		log.Fatalf("Gönderim hatası: %v", err)
	}

	log.Println("E-posta başarıyla gönderildi ✔")
}
