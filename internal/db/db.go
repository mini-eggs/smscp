package db

import (
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"

	// for mysql support
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/pkg/errors"
	"smscp.xyz/internal/common"
	"smscp.xyz/pkg/mode"
)

type DB struct {
	conn         *gorm.DB
	security     securityLayer
	migrationKey string
}

type securityLayer interface {
	HashCreate(pass string) (string, error)
	HashCompare(pass, hash string) error
	TokenCreate(val jwt.Claims) (string, error)
	TokenFrom(tokenString string) (jwt.MapClaims, error)
}

func Default(conn *gorm.DB, security securityLayer, mkey string) DB {
	return DB{
		conn,
		security,
		mkey,
	}
}

func ConnDefault() (*gorm.DB, error) {
	connStr := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_NAME"),
	)

	conn, err := gorm.Open("mysql", connStr)
	if err != nil {
		return conn, errors.Wrap(err, "failed to connect to database")
	}

	conn.DB().SetMaxIdleConns(10)
	conn.DB().SetMaxOpenConns(100)
	conn.DB().SetConnMaxLifetime(time.Hour)

	return conn, nil
}

func (db DB) SetMode(m mode.Mode) {
	switch m {
	case mode.Test:
		gorm.DefaultTableNameHandler = func(db *gorm.DB, n string) string { return n + "_test" }
		return
	case mode.Dev:
		gorm.DefaultTableNameHandler = func(db *gorm.DB, n string) string { return n + "_dev" }
		return
	default:
		break
	}
}

func (db DB) Migrate(key string) error {
	if key != db.migrationKey {
		return errors.New("invalid migration key")
	}
	if res := db.conn.AutoMigrate(&SmsCpUser{}); res.Error != nil {
		return res.Error
	}
	if res := db.conn.AutoMigrate(&SmsCpNote{}); res.Error != nil {
		return res.Error
	}
	return nil
}

func (db DB) UserLogin(username, plaintext string) (common.User, error) {
	var user SmsCpUser
	status := db.conn.Where(&SmsCpUser{UserUsername: username}).First(&user)

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

func (db DB) UserGet(token string) (common.User, error) {
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

func (db DB) UserGetByNumber(number string) (common.User, error) {
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

func (db DB) UserCreate(username, plaintext, phone string) (common.User, error) {
	pass, err := db.security.HashCreate(plaintext)
	if err != nil {
		return nil, errors.New("failed to create user; password hash not obtained")
	}

	user := SmsCpUser{
		UserUsername: username,
		UserPass:     pass,
		UserPhone:    phone,
		db:           db,
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

func (db DB) NoteGetList(user common.User, page, count int) ([]common.Note, bool, error) {
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

	var notes []common.Note
	for _, note := range dbnotes {
		note.NoteShort = note.Short() /* special case, bc in templates we can just call method */
		notes = append(notes, note)
	}

	return notes, hasMore, status.Error
}

func (db DB) NoteGetLatest(user common.User) (common.Note, error) {
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

func (db DB) NoteGetLatestWithTime(user common.User, t time.Duration) (common.Note, error) {
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

func (db DB) NoteCreate(user common.User, text string) (common.Note, error) {
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

func (this DB) UserAll(user common.User) ([]common.Note, []common.Msg, error) {
	// get notes
	var dbnotes []SmsCpNote
	status := this.conn.
		Where(&SmsCpNote{UserID: user.ID()}).
		Order("id ASC").
		Find(&dbnotes)
	if status.Error != nil {
		return nil, nil, status.Error
	}

	var notes []common.Note
	for _, note := range dbnotes {
		note.NoteShort = note.Short() /* special case, bc in templates we can just call method */
		notes = append(notes, note)
	}

	// get msgs
	var dbmsgs []SmsCpMsg
	status = this.conn.
		Where(&SmsCpMsg{UserID: user.ID()}).
		Order("id ASC").
		Find(&dbmsgs)
	if status.Error != nil {
		return nil, nil, status.Error
	}

	var msgs []common.Note
	for _, msg := range dbmsgs {
		msgs = append(msgs, msg)
	}

	return notes, msgs, nil
}

func (this DB) UserDel(common.User) error {
	return errors.New("not complete")
}

// SmsCpUser class

type SmsCpUser struct {
	gorm.Model
	UserUsername string `gorm:"unique;not null"`
	UserPass     string `gorm:"not null"`
	UserPhone    string `gorm:"unique;not null"`
	UserNotes    []SmsCpNote
	token        string
	err          error
	db           DB
}

func (SmsCpUser *SmsCpUser) Username() string { return SmsCpUser.UserUsername }
func (SmsCpUser *SmsCpUser) Phone() string    { return SmsCpUser.UserPhone }
func (SmsCpUser *SmsCpUser) ID() uint         { return SmsCpUser.Model.ID }
func (SmsCpUser *SmsCpUser) Token() string    { return SmsCpUser.token }

func (SmsCpUser *SmsCpUser) SetUsername(value string) {
	SmsCpUser.UserUsername = value
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
	db        DB
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
