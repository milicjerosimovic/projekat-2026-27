package mail

import (
	"errors"
	"net/smtp"
	"strings"
)

type Mailer interface {
	Send(to, subject, body string) error
}

type SMTP struct {
	host, port, user, pass, from string
}

func NewSMTP(host, port, user, pass, from string) *SMTP {
	return &SMTP{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *SMTP) Send(to, subject, body string) error {
	if strings.ContainsAny(to+subject+s.from, "\r\n") {
		return errors.New("neispravna vrednost zaglavlja")
	}
	msg := "From: " + s.from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body

	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}
	return smtp.SendMail(s.host+":"+s.port, auth, s.from, []string{to}, []byte(msg))
}
