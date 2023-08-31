package mail

import "context"

type Sender interface {
	SingleSend(Ctx context.Context, mailer *Mailer) error
}

type MailerType int

const (
	MailerTypeNormal = iota
	MailerTypeTemplate
)

type Mailer struct {
	Type     MailerType
	Mail     *Mail
	Template *Template
}

type Mail struct {
	From    string
	To      []string
	CC      []string
	BCC     []string
	Subject string
	Body    string
}

type Template struct {
	ID   string
	Data map[string]interface{}
}
