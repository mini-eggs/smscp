package db

import (
	"errors"
	"fmt"
	"os"
	"time"

	. "tophone.evanjon.es/internal/common"

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type db struct {
	conn     *gorm.DB
	security securityLayer
}

type securityLayer interface {
	HashCreate(pass string) (string, error)
	HashCompare(pass, hash string) error
	TokenCreate(val jwt.MapClaims) (string, error)
	TokenFrom(tokenString string) (jwt.MapClaims, error)
}

func DBDefault(conn *gorm.DB, security securityLayer) db {
	// conn.AutoMigrate(&ToPhoneUser{})
	// conn.AutoMigrate(&ToPhoneNote{})
	return db{
		conn,
		security,
	}
}

func DBConnDefault() (*gorm.DB, error) {
	connStr := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_NAME"),
	)

	conn, err := gorm.Open("mysql", connStr)
	if err != nil {
		return conn, err
	}

	conn.DB().SetMaxIdleConns(10)
	conn.DB().SetMaxOpenConns(100)
	conn.DB().SetConnMaxLifetime(time.Hour)

	return conn, nil
}

func (db db) UserLogin(email, plaintext string) (User, error) {
	var user ToPhoneUser
	status := db.conn.Where(&ToPhoneUser{UserEmail: email}).First(&user)

	if status.Error != nil {
		return nil, status.Error
	}

	err := db.security.HashCompare(plaintext, user.UserPass)
	if err != nil {
		return nil, errors.New("failed to login user; password hash not matched")
	}

	token, err := db.security.TokenCreate(jwt.MapClaims{"UserID": user.Model.ID})
	if err != nil {
		return nil, err
	}

	user.token = token
	user.db = db
	user.db = db

	return &user, nil
}

func (db db) UserGet(token string) (User, error) {
	claims, err := db.security.TokenFrom(token)
	if err != nil {
		return nil, err
	}

	var user ToPhoneUser
	status := db.conn.Where("id = ?", claims["UserID"]).First(&user)

	if status.Error != nil {
		return nil, status.Error
	}

	user.token = token
	user.db = db

	return &user, nil
}

func (db db) UserGetByNumber(number string) (User, error) {
	var user ToPhoneUser
	status := db.conn.Where("user_phone = ?", number).First(&user)
	if status.Error != nil {
		return nil, status.Error
	}

	token, err := db.security.TokenCreate(jwt.MapClaims{"UserID": user.Model.ID})
	if err != nil {
		return nil, err
	}

	user.token = token
	user.db = db

	return &user, nil
}

func (db db) UserCreate(email, plaintext, phone string) (User, error) {
	pass, err := db.security.HashCreate(plaintext)
	if err != nil {
		return nil, errors.New("failed to create user; password hash not obtained")
	}

	user := ToPhoneUser{
		UserEmail: email,
		UserPass:  pass,
		UserPhone: phone,
		db:        db,
	}

	status := db.conn.Create(&user)

	if status.Error != nil {
		return nil, status.Error
	}

	token, err := db.security.TokenCreate(jwt.MapClaims{"UserID": user.Model.ID})
	if err != nil {
		return nil, err
	}

	user.token = token
	user.db = db

	return &user, nil
}

func (db db) NoteGetList(user User, page, count int) ([]Note, bool, error) {
	var dbnotes []ToPhoneNote
	status := db.conn.
		Where(&ToPhoneNote{UserID: user.ID()}).
		Offset(page * count).
		Limit(count + 1).
		Order("id DESC").
		Find(&dbnotes)
	if status.Error != nil {
		return nil, false, status.Error
	}

	hasMore := len(dbnotes) > count
	if hasMore {
		dbnotes = dbnotes[:len(dbnotes)-1] /* all but last */
	}

	var notes []Note
	for _, note := range dbnotes {
		note.NoteShort = note.Short() /* special case, bc in templates we can just call method */
		notes = append(notes, note)
	}

	return notes, hasMore, status.Error
}

func (db db) NoteGetLatest(user User) (Note, error) {
	var latest ToPhoneNote
	status := db.conn.
		Where("user_id = ?", user.ID()).
		Order("created_at ASC").
		Find(&latest)
	if status.Error != nil {
		return nil, nil /* not having this is not an error */
	}
	latest.NoteShort = latest.Short() /* special case, bc in templates we can just call method */
	return latest, nil
}

func (db db) NoteGetLatestWithTime(user User, t time.Duration) (Note, error) {
	var latest ToPhoneNote
	status := db.conn.
		Where("user_id = ? AND created_at >= NOW() - INTERVAL ? SECOND", user.ID(), t.Seconds()).
		Order("created_at ASC").
		Find(&latest)
	if status.Error != nil {
		return nil, nil /* not having this is not an error */
	}
	latest.NoteShort = latest.Short() /* special case, bc in templates we can just call method */
	return latest, nil
}

func (db db) NoteCreate(user User, text string) (Note, error) {
	note := ToPhoneNote{
		NoteText: text,
		UserID:   user.ID(),
		db:       db,
	}

	status := db.conn.Create(&note)

	if status.Error != nil {
		return nil, status.Error
	}

	token, err := db.security.TokenCreate(jwt.MapClaims{"NoteID": note.Model.ID})
	if err != nil {
		return nil, err
	}

	note.token = token
	note.db = db

	return note, nil
}

// ToPhoneUser class

type ToPhoneUser struct {
	gorm.Model
	UserEmail string `gorm:"unique;not null"`
	UserPass  string
	UserPhone string
	UserNotes []ToPhoneNote
	token     string
	err       error
	db        db
}

func (ToPhoneUser *ToPhoneUser) Email() string { return ToPhoneUser.UserEmail }
func (ToPhoneUser *ToPhoneUser) Phone() string { return ToPhoneUser.UserPhone }
func (ToPhoneUser *ToPhoneUser) ID() uint      { return ToPhoneUser.Model.ID }
func (ToPhoneUser *ToPhoneUser) Token() string { return ToPhoneUser.token }

func (ToPhoneUser *ToPhoneUser) SetEmail(value string) {
	ToPhoneUser.UserEmail = value
}

func (ToPhoneUser *ToPhoneUser) SetPhone(value string) {
	ToPhoneUser.UserPhone = value
}

func (ToPhoneUser *ToPhoneUser) SetPass(plaintext string) {
	pass, err := ToPhoneUser.db.security.HashCreate(plaintext)
	if err != nil {
		ToPhoneUser.err = err
		return
	}

	ToPhoneUser.UserPass = pass
}

func (ToPhoneUser *ToPhoneUser) Save() error {
	if ToPhoneUser.err != nil {
		return ToPhoneUser.err
	}

	status := ToPhoneUser.db.conn.Save(&ToPhoneUser)

	return status.Error
}

// ToPhoneNote class

type ToPhoneNote struct {
	gorm.Model
	NoteText  string `sql:"type:text"`
	NoteShort string `gorm:"-"` /* ignore! */
	UserID    uint
	token     string
	db        db
}

func (ToPhoneNote ToPhoneNote) Short() string {
	top := 50
	if len(ToPhoneNote.NoteText) > top {
		return ToPhoneNote.NoteText[:top] + "..."
	}
	return ToPhoneNote.NoteText
}

func (ToPhoneNote ToPhoneNote) Text() string  { return ToPhoneNote.NoteText }
func (ToPhoneNote ToPhoneNote) ID() uint      { return ToPhoneNote.Model.ID }
func (ToPhoneNote ToPhoneNote) Token() string { return ToPhoneNote.token }
