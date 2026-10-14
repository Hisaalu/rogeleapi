package mailer

import "context"

type Message struct {
	To      string
	ToName  string
	Subject string
	HTML    string
	Text    string
	ReplyTo string
}

type Sender interface {
	Name() string
	Send(ctx context.Context, msg Message) error
}
