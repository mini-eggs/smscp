package fs

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"golang.org/x/exp/utf8string"
	"google.golang.org/api/iterator"
	"smscp.xyz/internal/common"
)

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

func (fs fs) getconn(ctx context.Context) (*firestore.Client, error) {
	val, ok := ctx.Value(keyConn).(*firestore.Client)
	if !ok {
		return nil, errors.New("database connection has not begun")
	}
	return val, nil
}

func (fs fs) snaptouser(ctx context.Context, doc *firestore.DocumentSnapshot) (common.User, error) {
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

func (fs fs) itertouser(ctx context.Context, iter *firestore.DocumentIterator) (common.User, error) {
	doc, err := iter.Next()
	if err != nil {
		return nil, errors.New("failed to find user")
	}
	return fs.snaptouser(ctx, doc)
}

func (fs fs) toshort(text string) string {
	top := 50
	str := utf8string.NewString(text)
	if str.RuneCount() > top {
		return str.Slice(0, top) + "..."
	}
	return str.String()
}

// public

func (fs fs) Middleware(c *gin.Context) {
	// Firestore connection per request.
	client, err := firestore.NewClient(c, fs.firestoreProjectID)
	if err != nil {
		// Error will be caught later.
		c.Next()
		return
	}
	c.Set(keyConn, client)
	c.Next()
	defer client.Close()
}

func (fs fs) UserAll(ctx context.Context, user common.User) ([]common.Note, error) {
	conn, err := fs.getconn(ctx)
	if err != nil {
		return nil, err
	}

	iter := conn.Collection("notes").
		Where("UserID", "==", user.ID()).
		OrderBy("NoteCreatedAt", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var ret []common.Note
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, errors.Wrap(err, "failed to read all note values")
		}

		note := Note{ref: doc.Ref}
		if err := doc.DataTo(&note); err != nil {
			return nil, errors.Wrap(err, "note value corrupted")
		}

		token, err := fs.sec.TokenCreate(jwt.MapClaims{"NoteID": note.ID()})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create unique token for note")
		}

		note.token = token
		note.fs = fs

		ret = append(ret, note)
	}

	return ret, nil
}

func (fs fs) UserDel(ctx context.Context, user common.User) error {
	conn, err := fs.getconn(ctx)
	if err != nil {
		return err
	}

	// Delete notes

	iter := conn.Collection("notes").
		Where("UserID", "==", user.ID()).
		Documents(ctx)
	defer iter.Stop()

	for {

		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return errors.Wrap(err, "failed to read all note values")
		}

		if _, err := doc.Ref.Delete(ctx); err != nil {
			return errors.Wrap(err, "failed to delete note")
		}
	}

	// Delete user

	if _, err := conn.Collection("users").Doc(user.ID()).Delete(ctx); err != nil {
		return errors.Wrap(err, "failed to delete user")
	}

	return nil
}

func (fs fs) NoteGetLatest(ctx context.Context, user common.User) (common.Note, error) {
	conn, err := fs.getconn(ctx)
	if err != nil {
		return nil, err
	}

	iter := conn.Collection("notes").
		Where("UserID", "==", user.ID()).
		OrderBy("NoteCreatedAt", firestore.Desc).
		Limit(1).
		Documents(ctx)

	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to find note")
	}

	note := Note{ref: doc.Ref}
	if err := doc.DataTo(&note); err != nil {
		return nil, errors.Wrap(err, "note value corrupted")
	}

	token, err := fs.sec.TokenCreate(jwt.MapClaims{"NoteID": note.ID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create unique token for note")
	}

	note.token = token
	note.fs = fs

	return &note, nil
}

func (fs fs) NoteGetLatestWithTime(ctx context.Context, user common.User, t time.Duration) (common.Note, error) {
	conn, err := fs.getconn(ctx)
	if err != nil {
		return nil, err
	}

	iter := conn.Collection("notes").
		Where("UserID", "==", user.ID()).
		Where("NoteCreatedAt", ">=", time.Now().UTC().Add(-t).Unix()). // Negate .Add, awesome.
		OrderBy("NoteCreatedAt", firestore.Desc).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to find note")
	}

	note := Note{ref: doc.Ref}
	if err := doc.DataTo(&note); err != nil {
		return nil, errors.Wrap(err, "note value corrupted")
	}

	token, err := fs.sec.TokenCreate(jwt.MapClaims{"NoteID": note.ID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create unique token for note")
	}

	note.token = token
	note.fs = fs

	return &note, nil
}

func (fs fs) NoteGetList(ctx context.Context, user common.User, page, count int) ([]common.Note, bool, error) {
	conn, err := fs.getconn(ctx)
	if err != nil {
		return nil, false, err
	}

	iter := conn.Collection("notes").
		Where("UserID", "==", user.ID()).
		Offset(page*count).
		Limit(count+1).
		OrderBy("NoteCreatedAt", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var ret []common.Note
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, false, errors.Wrap(err, "failed to read all note values")
		}

		note := Note{ref: doc.Ref}
		if err := doc.DataTo(&note); err != nil {
			return nil, false, errors.Wrap(err, "note value corrupted")
		}

		token, err := fs.sec.TokenCreate(jwt.MapClaims{"NoteID": note.ID()})
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to create unique token for note")
		}

		note.token = token
		note.fs = fs

		ret = append(ret, note)
	}

	hasMore := len(ret) > count
	if hasMore {
		ret = ret[:len(ret)-1] /* all but last */
	}

	return ret, hasMore, nil
}

func (fs fs) NoteCreate(ctx context.Context, user common.User, text string) (common.Note, error) {
	conn, err := fs.getconn(ctx)
	if err != nil {
		return nil, err
	}

	note := Note{
		ref:           conn.Collection("notes").NewDoc(),
		NoteText:      text,
		NoteShort:     fs.toshort(text),
		NoteCreatedAt: time.Now().UTC().Unix(),
		UserID:        user.ID(),
	}

	_, err = note.ref.Set(ctx, note)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new note")
	}

	token, err := fs.sec.TokenCreate(jwt.MapClaims{"NoteID": note.ID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create unique token for note")
	}

	note.token = token
	note.fs = fs

	return &note, nil
}

func (fs fs) UserGet(ctx context.Context, token string) (common.User, error) {
	conn, err := fs.getconn(ctx)
	if err != nil {
		return nil, err
	}

	claims, err := fs.sec.TokenFrom(token)
	if err != nil {
		return nil, errors.Wrap(err, "corrupted token")
	}

	id, ok := claims["UserID"].(string)
	if !ok {
		return nil, errors.New("invalid token or no user in token")
	}

	snap, err := conn.Collection("users").Doc(id).Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find user")
	}

	return fs.snaptouser(ctx, snap)
}

func (fs fs) UserGetByNumber(ctx context.Context, phone string) (common.User, error) {
	conn, err := fs.getconn(ctx)
	if err != nil {
		return nil, err
	}

	iter := conn.Collection("users").Where("UserPhone", "==", phone).Documents(ctx)
	defer iter.Stop()
	return fs.itertouser(ctx, iter)
}

func (fs fs) UserGetByUsername(ctx context.Context, username string) (common.User, error) {
	conn, err := fs.getconn(ctx)
	if err != nil {
		return nil, err
	}

	iter := conn.Collection("users").Where("UserUsername", "==", username).Documents(ctx)
	defer iter.Stop()
	return fs.itertouser(ctx, iter)
}

func (fs fs) UserLogin(ctx context.Context, username, plaintext string) (common.User, error) {
	conn, err := fs.getconn(ctx)
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

	if err := fs.sec.HashCompare(user.ref.ID+plaintext, user.UserEncryptedPassword); err != nil {
		return nil, errors.New("failed to login user; password hash not matched")
	}

	token, err := fs.sec.TokenCreate(jwt.MapClaims{"UserID": user.ID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create unique token for user")
	}

	user.token = token
	user.fs = fs

	return &user, nil
}

func (fs fs) UserCreate(ctx context.Context, username, plaintext, phone string) (common.User, error) {
	conn, err := fs.getconn(ctx)
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

	ref := conn.Collection("users").NewDoc()
	pass, err := fs.sec.HashCreate(ref.ID + plaintext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create hash for user password")
	}

	user := User{
		ref:                   ref,
		UserUsername:          username,
		UserPhone:             phone,
		UserEncryptedPassword: pass,
		UserCreatedAt:         time.Now().UTC().Unix(),
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
	UserCreatedAt         int64

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
	if User.err != nil {
		return
	}

	pass, err := User.fs.sec.HashCreate(User.ref.ID + plaintext)
	if err != nil {
		User.err = err
		return
	}

	User.UserEncryptedPassword = pass
}

func (User *User) Save(ctx context.Context) error {
	if User.err != nil {
		return User.err
	}

	conn, err := User.fs.getconn(ctx)
	if err != nil {
		return err
	}

	if _, err = conn.Collection("users").Doc(User.ID()).Set(ctx, User); err != nil {
		return errors.Wrap(err, "failed to update user")
	}

	return nil
}

// note type

type Note struct {
	ref           *firestore.DocumentRef
	NoteText      string
	NoteShort     string
	NoteCreatedAt int64

	// Relations:
	UserID string

	// Set when retrieved:
	token string
	fs    fs
}

func (Note Note) Short() string { return Note.NoteShort }
func (Note Note) Text() string  { return Note.NoteText }
func (Note Note) ID() string    { return Note.ref.ID }
func (Note Note) Token() string { return Note.token }
