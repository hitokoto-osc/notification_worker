package mailer

import "context"

type Sender interface {
	SendSingle(ctx context.Context, mail *Mailer) error
}

type Type int

const (
	TypeNormal Type = iota
	TypeTemplate
)

type Mailer struct {
	Type     Type
	Mail     Mail
	Body     *MailBody
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

type MailBody struct {
	Subject string
	Body    string
}

type Template struct {
	ID   string
	Data map[string]interface{}
}
