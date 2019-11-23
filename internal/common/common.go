package common

import "context"

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

// Temp, for firestore rewrite.

type User2 interface {
	ID() string
	Username() string
	Phone() string
	Token() string /* Stored in session, secret, unique per session. */
	SetUsername(string)
	SetPass(string)
	SetPhone(string)
	Save(context.Context) error
}

type Note2 interface {
	ID() string
	Short() string
	Text() string
	Token() string /* Unique per note (i.e. like an ID), only let author see. */
}
