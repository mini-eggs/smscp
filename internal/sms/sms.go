package sms

import "github.com/plivo/plivo-go"

type sms struct {
	id, secret, from string
}

func SMSDefault(id, secret, from string) sms {
	return sms{id, secret, from}
}

func (sms sms) Send(to, text string) error {
	client, err := plivo.NewClient(sms.id, sms.secret, &plivo.ClientOptions{})
	if err != nil {
		return err
	}

	_, err = client.Messages.Create(plivo.MessageCreateParams{
		Src:  sms.from,
		Dst:  to,
		Text: text,
	})

	return err
}
