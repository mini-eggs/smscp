package common

type User interface {
	ID() uint
	Username() string
	Phone() string
	Token() string /* Stored in session, secret, unique per session. */
	SetUsername(string)
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

type Msg interface {
	ID() uint
	Short() string
	Text() string
	Token() string /* Unique per msg (i.e. like an ID), only let author see. */
	From() string
	To() string
}
