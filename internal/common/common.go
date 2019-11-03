package common

type User interface {
	ID() uint
	Email() string
	Phone() string
	Token() string /* Stored in session, secret, unique per session. */
	SetEmail(string)
	SetPass(string)
	SetPhone(string)
	Save() error
}

type Note interface {
	ID() uint
	Short() string
	Text() string
	Token() string /* Unique per note (i.e. like an ID), only let author see. */
}
