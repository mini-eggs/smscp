package twilio

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
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
	_, _, err := twilio.SendMMS(sms.from, to, text, "", "", "")
	return err
}

func (sms SMS) Hook(c *gin.Context) (number, text string, err error) {
	var payload struct {
		Body, From, FromCountry string
	}

	err = c.Bind(&payload)
	if err != nil {
		return
	}

	phone, err := libphonenumber.Parse(payload.From, payload.FromCountry)
	if err != nil {
		return
	} else if !libphonenumber.IsValidNumber(phone) {
		err = errors.New("invalid phone number; try again")
		return
	}

	return fmt.Sprintf("%d%d", phone.GetCountryCode(), phone.GetNationalNumber()), payload.Body, nil
}
