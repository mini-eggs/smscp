package twilio

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sfreiberg/gotwilio"
	"github.com/ttacon/libphonenumber"
)

type SMS struct {
	id, secret, from string
}

func Default(id, secret, from string) SMS {
	return SMS{id, secret, from}
}

func (sms SMS) Send(to, text string) error {
	twilio := gotwilio.NewTwilioClient(sms.id, sms.secret)

	if _, _, err := twilio.SendMMS(sms.from, to, text, "", "", ""); err != nil {
		return errors.Wrap(err, "failed to send message")
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

	return userPhone, payload.Body, nil
}
