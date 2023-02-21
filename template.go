package fiberextend

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"strings"
)

func (p *IFiberEx) ExecuteTemplate(out io.Writer, src string, values interface{}) error {
	t, err := template.New("instant").Parse(src)
	if err != nil {
		return err
	}
	if err := t.Execute(out, values); err != nil {
		return err
	}
	return nil
}

const MailFormat string = `From: %s
To: %s
%s
MIME-Version: 1.0
Content-Type: text/html; charset="utf-8"
Content-Transfer-Encoding: base64

%s`

func (p *IFiberEx) Mail(to []string, subject string, body string, values interface{}) error {
	title := bytes.NewBufferString("")
	if err := p.ExecuteTemplate(title, subject, values); err != nil {
		return err
	}
	text := bytes.NewBufferString("")
	if err := p.ExecuteTemplate(text, body, values); err != nil {
		return err
	}
	message := fmt.Sprintf(
		MailFormat,
		p.Config.SmtpFrom,
		strings.Join(to, ","),
		fmt.Sprintf("=?utf-8?B?%s?=", base64.StdEncoding.EncodeToString(title.Bytes())),
		base64.StdEncoding.EncodeToString(text.Bytes()),
	)
	var auth smtp.Auth
	if p.Config.SmtpUseMd5 {
		auth = smtp.CRAMMD5Auth(*p.Config.SmtpUser, *p.Config.SmtpPass)
	} else {
		auth = smtp.PlainAuth("", *p.Config.SmtpUser, *p.Config.SmtpPass, p.Config.SmtpAddr)
	}
	if err := smtp.SendMail(p.Config.SmtpAddr, auth, p.Config.SmtpFrom, to, []byte(message)); err != nil {
		return err
	}
	return nil
}
