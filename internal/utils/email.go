package utils

import (
	"log"
	"os"
	"strconv"

	gomail "gopkg.in/mail.v2"
)

func SendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

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

	if err := dialer.DialAndSend(msg); err != nil {
		log.Printf("[EMAIL ERROR] ❌ gagal kirim email ke %s: %v\n", to, err)
		return err
	}

	log.Printf("[EMAIL SENT] ✅ ke %s | Subject: %s\n", to, subject)
	return nil
}
