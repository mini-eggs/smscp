package twilio

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sfreiberg/gotwilio"
	"github.com/ttacon/libphonenumber"
	"smscp.xyz/internal/common"
)

type SMS struct {
	id, secret, from string
	msg              MsgLogLayer
}

type MsgLogLayer interface {
	MsgCreate(text, to, from string) (common.Msg, error)
}

func Default(id, secret, from string, msg MsgLogLayer) SMS {
	return SMS{id, secret, from, msg}
}

func (sms SMS) Send(to, text string) error {
	twilio := gotwilio.NewTwilioClient(sms.id, sms.secret)

	if _, _, err := twilio.SendMMS(sms.from, to, text, "", "", ""); err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	if len(sms.from) < 1 {
		// This should never actually happen. The above call would fail first. But,
		// just in case, we don't want runtime stringing errors.
		return errors.New("received faulty \"from\" message; cannot send message")
	}

	if _, err := sms.msg.MsgCreate(text, to, sms.from[1:]); err != nil { // Don't want to log the "+".
		return errors.Wrap(err, "message sent successfully; but failed to create transaction for message")
	}

	return nil
}

func (sms SMS) Hook(c *gin.Context) (_number, _text string, _err error) {
	var payload struct{ Body, From, FromCountry string }

	err := c.Bind(&payload)
	if err != nil {
		return "", "", err
	}

	phone, err := libphonenumber.Parse(payload.From, payload.FromCountry)
	if err != nil {
		return "", "", err
	} else if !libphonenumber.IsValidNumber(phone) {
		return "", "", errors.New("invalid phone number; try again")
	}

	userPhone := fmt.Sprintf("%d%d", phone.GetCountryCode(), phone.GetNationalNumber())

	if len(sms.from) < 1 {
		// Just in case, we don't want runtime stringing errors.
		return "", "", errors.New("received faulty \"from\" message; failed to receive message properly")
	}

	if _, err := sms.msg.MsgCreate(payload.Body, sms.from[1:], userPhone); err != nil { // Don't want to log the "+".
		return "", "", errors.Wrap(err, "message received successfully; but failed to create transaction for message")
	}

	return userPhone, payload.Body, nil
}
