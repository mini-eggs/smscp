package fs

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	"smscp.xyz/internal/common"
)

// type dataLayer interface {
// 	// user
// 	UserGet(token string) (common.User, error)
// 	UserGetByNumber(number string) (common.User, error)
// 	UserGetByUsername(username string) (common.User, error)
// 	UserLogin(username, pass string) (common.User, error)
// 	UserCreate(username, pass, phone string) (common.User, error)
// 	// notes
// 	NoteGetList(user common.User, page, count int) ([]common.Note, bool, error)
// 	NoteGetLatest(user common.User) (common.Note, error)
// 	NoteGetLatestWithTime(user common.User, t time.Duration) (common.Note, error)
// 	NoteCreate(user common.User, text string) (common.Note, error)
// 	// special database
// 	Migrate(key string) error
// 	// special gdpr
// 	UserAll(common.User) ([]common.Note /* []common.Msg, */, error)
// 	UserDel(common.User) error
// }

type securityLayer interface {
	HashCreate(pass string) (string, error)
	HashCompare(pass, hash string) error
	TokenCreate(val jwt.Claims) (string, error)
	TokenFrom(tokenString string) (jwt.MapClaims, error)
}

type fs struct {
	firestoreProjectID string
	sec                securityLayer
}

const (
	keyConn = "FIRESTORE_CONNECTION_KEY"
)

func Default(firestoreProjectID string, sec securityLayer) fs {
	return fs{firestoreProjectID, sec}
}

// private

func (fs fs) getconn(ctx *context.Context) (*firestore.Client, error) {
	val, ok := (*ctx).Value(keyConn).(*firestore.Client)
	if !ok {
		// TODO: Make sure we're not setting up multiple times.
		fmt.Println("SETTING UP FIRESTORE")

		client, err := firestore.NewClient((*ctx), fs.firestoreProjectID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to database; check if GOOGLE_APPLICATION_CREDENTIALS is set.")
		}

		*ctx = context.WithValue((*ctx), keyConn, client)

		return client, nil
	}

	return val, nil
}

// public
func (fs fs) UserLogin(ctx context.Context, username, plaintext string) (common.User2, error) {
	conn, err := fs.getconn(&ctx)
	if err != nil {
		return nil, err
	}

	iter := conn.Collection("users").Where("UserUsername", "==", username).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		return nil, errors.New("failed to find user")
	}

	user := User{ref: doc.Ref}
	if err := doc.DataTo(&user); err != nil {
		return nil, errors.Wrap(err, "user value corrupted")
	}

	token, err := fs.sec.TokenCreate(jwt.MapClaims{"UserID": user.ID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create unique token for user")
	}

	user.token = token
	user.fs = fs

	return &user, nil
}

func (fs fs) UserCreate(ctx context.Context, username, plaintext, phone string) (common.User2, error) {
	conn, err := fs.getconn(&ctx)
	if err != nil {
		return nil, err
	}

	// Check username taken.
	usernameIter := conn.Collection("users").Where("UserUsername", "==", username).Documents(ctx)
	defer usernameIter.Stop()
	if _, err = usernameIter.Next(); err != iterator.Done {
		return nil, errors.New("username already exists")
	}

	// Check phone taken.
	phoneIter := conn.Collection("users").Where("UserPhone", "==", phone).Documents(ctx)
	defer phoneIter.Stop()
	if _, err = phoneIter.Next(); err != iterator.Done {
		return nil, errors.New("phone already used; try reseting password")
	}

	pass, err := fs.sec.HashCreate(plaintext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create hash for user password")
	}

	user := User{
		ref:                   conn.Collection("users").NewDoc(),
		UserUsername:          username,
		UserPhone:             phone,
		UserEncryptedPassword: pass,
	}

	_, err = user.ref.Set(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new user")
	}

	token, err := fs.sec.TokenCreate(jwt.MapClaims{"UserID": user.ID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create unique token for user")
	}

	user.token = token
	user.fs = fs

	return &user, nil
}

// user type

type User struct {
	ref                   *firestore.DocumentRef
	UserUsername          string
	UserPhone             string
	UserEncryptedPassword string

	// Set when retrieved:
	token string
	fs    fs

	// Set while updating
	err error
}

func (user *User) Username() string { return user.UserUsername }
func (user *User) Phone() string    { return user.UserPhone }
func (User *User) ID() string       { return User.ref.ID }
func (User *User) Token() string    { return User.token }

func (User *User) SetUsername(value string) { User.UserUsername = value }
func (User *User) SetPhone(value string)    { User.UserPhone = value }

func (User *User) SetPass(plaintext string) {
	pass, err := User.fs.sec.HashCreate(plaintext)
	if err != nil {
		User.err = err
		return
	}

	User.UserEncryptedPassword = pass
}

func (User *User) Save() error {
	if User.err != nil {
		return User.err
	}

	return errors.New("todo")

	// status := User.db.conn.Save(&User)
	// return status.Error
}
