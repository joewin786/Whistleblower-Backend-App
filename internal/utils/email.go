package utils

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	gomail "gopkg.in/mail.v2"
)

func SendEmail(to, subject, body string) error {
	// ✅ Validate environment variables
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	if smtpHost == "" || smtpUser == "" || smtpPass == "" {
		log.Println("[EMAIL ERROR] ❌ SMTP configuration not set")
		return errors.New("SMTP configuration missing")
	}

	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil || smtpPort == 0 {
		smtpPort = 587
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", smtpUser)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body)

	dialer := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)
	
	// ✅ Set timeout 10 detik
	dialer.Timeout = 10 * time.Second

	// ✅ Send dengan timeout protection
	errChan := make(chan error, 1)
	go func() {
		errChan <- dialer.DialAndSend(msg)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			log.Printf("[EMAIL ERROR] ❌ gagal kirim email ke %s: %v\n", to, err)
			return err
		}
		log.Printf("[EMAIL SENT] ✅ ke %s | Subject: %s\n", to, subject)
		return nil
	case <-time.After(15 * time.Second):
		log.Printf("[EMAIL ERROR] ⏱️ timeout kirim email ke %s\n", to)
		return errors.New("email send timeout")
	}
}