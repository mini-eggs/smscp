package db

import (
	"errors"
	"fmt"
	"os"
	"time"

	. "smscp.xyz/internal/common"

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
	// conn.AutoMigrate(&SmsCpUser{})
	// conn.AutoMigrate(&SmsCpNote{})
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
	var user SmsCpUser
	status := db.conn.Where(&SmsCpUser{UserEmail: email}).First(&user)

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

	var user SmsCpUser
	status := db.conn.Where("id = ?", claims["UserID"]).First(&user)

	if status.Error != nil {
		return nil, status.Error
	}

	user.token = token
	user.db = db

	return &user, nil
}

func (db db) UserGetByNumber(number string) (User, error) {
	var user SmsCpUser
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

	user := SmsCpUser{
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
	var dbnotes []SmsCpNote
	status := db.conn.
		Where(&SmsCpNote{UserID: user.ID()}).
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
	var latest SmsCpNote
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
	var latest SmsCpNote
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
	note := SmsCpNote{
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

// SmsCpUser class

type SmsCpUser struct {
	gorm.Model
	UserEmail string `gorm:"unique;not null"`
	UserPass  string
	UserPhone string
	UserNotes []SmsCpNote
	token     string
	err       error
	db        db
}

func (SmsCpUser *SmsCpUser) Email() string { return SmsCpUser.UserEmail }
func (SmsCpUser *SmsCpUser) Phone() string { return SmsCpUser.UserPhone }
func (SmsCpUser *SmsCpUser) ID() uint      { return SmsCpUser.Model.ID }
func (SmsCpUser *SmsCpUser) Token() string { return SmsCpUser.token }

func (SmsCpUser *SmsCpUser) SetEmail(value string) {
	SmsCpUser.UserEmail = value
}

func (SmsCpUser *SmsCpUser) SetPhone(value string) {
	SmsCpUser.UserPhone = value
}

func (SmsCpUser *SmsCpUser) SetPass(plaintext string) {
	pass, err := SmsCpUser.db.security.HashCreate(plaintext)
	if err != nil {
		SmsCpUser.err = err
		return
	}

	SmsCpUser.UserPass = pass
}

func (SmsCpUser *SmsCpUser) Save() error {
	if SmsCpUser.err != nil {
		return SmsCpUser.err
	}

	status := SmsCpUser.db.conn.Save(&SmsCpUser)

	return status.Error
}

// SmsCpNote class

type SmsCpNote struct {
	gorm.Model
	NoteText  string `sql:"type:text"`
	NoteShort string `gorm:"-"` /* ignore! */
	UserID    uint
	token     string
	db        db
}

func (SmsCpNote SmsCpNote) Short() string {
	top := 50
	if len(SmsCpNote.NoteText) > top {
		return SmsCpNote.NoteText[:top] + "..."
	}
	return SmsCpNote.NoteText
}

func (SmsCpNote SmsCpNote) Text() string  { return SmsCpNote.NoteText }
func (SmsCpNote SmsCpNote) ID() uint      { return SmsCpNote.Model.ID }
func (SmsCpNote SmsCpNote) Token() string { return SmsCpNote.token }
