package mail

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"backend/config"
)

type SMTPMailer struct {
	config *config.SMTP
}

func NewSMTPMailer(cfg *config.SMTP) *SMTPMailer {
	return &SMTPMailer{config: cfg}
}

func (m *SMTPMailer) Send(ctx context.Context, recipient string, subject string, body string) error {
	address := net.JoinHostPort(m.config.Host, strconv.Itoa(int(m.config.Port)))
	connection, err := m.dial(ctx, address)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer connection.Close()

	if err := connection.SetDeadline(connectionDeadline(ctx, m.config.Timeout)); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}

	client, err := smtp.NewClient(connection, m.config.Host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	if m.config.TLSMode == "starttls" {
		if err := client.StartTLS(m.tlsConfig()); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}

	if m.config.Username != "" {
		auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate SMTP client: %w", err)
		}
	}
	if err := client.Mail(m.config.FromAddress); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message: %w", err)
	}
	message, err := m.message(recipient, subject, body)
	if err != nil {
		_ = writer.Close()
		return fmt.Errorf("encode SMTP message: %w", err)
	}
	bufferedWriter := bufio.NewWriter(writer)
	if _, err := bufferedWriter.WriteString(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := bufferedWriter.Flush(); err != nil {
		_ = writer.Close()
		return fmt.Errorf("flush SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("submit SMTP message: %w", err)
	}

	_ = client.Quit()
	return nil
}

func (m *SMTPMailer) dial(ctx context.Context, address string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: m.config.Timeout}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	if m.config.TLSMode != "implicit" {
		return connection, nil
	}

	tlsConnection := tls.Client(connection, m.tlsConfig())
	if err := tlsConnection.HandshakeContext(ctx); err != nil {
		_ = connection.Close()
		return nil, err
	}
	return tlsConnection, nil
}

func (m *SMTPMailer) tlsConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: m.config.Host,
	}
}

func (m *SMTPMailer) message(recipient string, subject string, body string) (string, error) {
	var encodedBody strings.Builder
	quotedPrintableWriter := quotedprintable.NewWriter(&encodedBody)
	if _, err := quotedPrintableWriter.Write([]byte(strings.ReplaceAll(body, "\n", "\r\n"))); err != nil {
		return "", err
	}
	if err := quotedPrintableWriter.Close(); err != nil {
		return "", err
	}

	from := (&mail.Address{Name: m.config.FromName, Address: m.config.FromAddress}).String()
	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)
	lines := []string{
		"From: " + from,
		"To: " + recipient,
		"Subject: " + encodedSubject,
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: quoted-printable",
		"",
		encodedBody.String(),
	}
	return strings.Join(lines, "\r\n") + "\r\n", nil
}

func connectionDeadline(ctx context.Context, timeout time.Duration) time.Time {
	deadline := time.Now().Add(timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		return contextDeadline
	}
	return deadline
}
