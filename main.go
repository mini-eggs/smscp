package handler

// package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/sfreiberg/gotwilio"
	"golang.org/x/crypto/bcrypt"
)

// common types

type User interface {
	ID() uint
	Email() string
	Phone() int
	Token() string /* Stored in session, secret, unique per session. */
	SetEmail(string)
	SetPass(string)
	SetPhone(int)
	Save() error
}

type Note interface {
	ID() uint
	Text() string
	Token() string /* Unique per note (i.e. like an ID), only let author see. */
}

// app class

type app struct {
	data dataLayer
	sms  smsLayer
}

const (
	PER_PAGE               = 20
	KEY_USER_SESSION_TOKEN = iota
)

type dataLayer interface {
	UserGet(token string) (User, error)
	UserLogin(email, pass string) (User, error)
	UserCreate(email, pass string, phone int) (User, error)
	NoteGetList(user User, page, count int) ([]Note, bool, error)
	NoteCreate(user User, text string) (Note, error)
}

type smsLayer interface {
	Send(number int, text string) error
}

func AppDefault(data dataLayer, sms smsLayer) app {
	return app{
		data,
		sms,
	}
}

func (app app) NoteCreate(c *gin.Context) {
	var payload struct {
		Text string
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	user, err := app.currentUser(c)
	if err != nil {
		app.error(c, errors.New("not logged in; or something else terribly wrong"))
		return
	}

	note, err := app.data.NoteCreate(user, payload.Text)
	if err != nil {
		app.error(c, err)
		return
	}

	if err := app.sms.Send(user.Phone(), note.Text()); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app app) NoteCreateCLI(c *gin.Context) {
	var payload struct {
		Token, Text string
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	user, err := app.currentUserFromToken(payload.Token)
	if err != nil {
		app.error(c, errors.New("not logged in; or something else terribly wrong"))
		return
	}

	note, err := app.data.NoteCreate(user, payload.Text)
	if err != nil {
		app.error(c, err)
		return
	}

	if err := app.sms.Send(user.Phone(), note.Text()); err != nil {
		app.error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"Message": "complete"})
}

func (app app) UserLogin(c *gin.Context) {
	var payload struct {
		Email, Password string
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	user, err := app.data.UserLogin(payload.Email, payload.Password)
	if err != nil {
		app.error(c, err)
		return
	}

	s := sessions.Default(c)
	s.Set(KEY_USER_SESSION_TOKEN, user.Token())
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app app) UserLoginCLI(c *gin.Context) {
	var payload struct {
		Email, Password string
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	user, err := app.data.UserLogin(payload.Email, payload.Password)
	if err != nil {
		app.error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"Token": user.Token()})
}

func (app app) UserCreate(c *gin.Context) {
	var payload struct {
		Email, Password, Verify string
		Phone                   int
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	if payload.Password != payload.Verify || payload.Password == "" {
		app.error(c, errors.New("invalid password; either not equal or no password entered"))
		return
	}

	user, err := app.data.UserCreate(payload.Email, payload.Password, payload.Phone)
	if err != nil {
		app.error(c, err)
		return
	}

	s := sessions.Default(c)
	s.Set(KEY_USER_SESSION_TOKEN, user.Token())
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app app) UserCreateCLI(c *gin.Context) {
	var payload struct {
		Email, Password, Verify string
		Phone                   int
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	if payload.Password != payload.Verify || payload.Password == "" {
		app.error(c, errors.New("invalid password; either not equal or no password entered"))
		return
	}

	user, err := app.data.UserCreate(payload.Email, payload.Password, payload.Phone)
	if err != nil {
		app.error(c, err)
		return
	}

	s := sessions.Default(c)
	s.Set(KEY_USER_SESSION_TOKEN, user.Token())
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"Token": user.Token()})
}

func (app app) UserUpdate(c *gin.Context) {
	var payload struct {
		Email, Password, Verify string
		Phone                   int
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	if payload.Password != payload.Verify {
		app.error(c, errors.New("invalid password; not equal"))
		return
	}

	user, err := app.currentUser(c)
	if err != nil {
		app.error(c, err)
		return
	}

	if payload.Email != "" {
		user.SetEmail(payload.Email)
	}

	if payload.Password != "" {
		user.SetPass(payload.Password)
	}

	if payload.Phone != 0 {
		user.SetPhone(payload.Phone)
	}

	err = user.Save()
	if err != nil {
		app.error(c, err)
		return
	}

	s := sessions.Default(c)
	s.Set(KEY_USER_SESSION_TOKEN, user.Token())
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app app) UserLogout(c *gin.Context) {
	s := sessions.Default(c)
	s.Set(KEY_USER_SESSION_TOKEN, nil)
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app app) Page(c *gin.Context) {
	user, err := app.currentUser(c)
	if err != nil {
		// Not really an error, just we don't currently have a user stored in
		// session.
		c.HTML(http.StatusOK, "main.html", gin.H{
			"HasUser": false,
		})
		return
	}

	/*
		page := 0
		notes, hasMore, err := app.data.NoteGetList(user, page, PER_PAGE)
		if err != nil {
			app.error(c, err)
			return
		}
	*/

	c.HTML(http.StatusOK, "main.html", gin.H{
		"HasUser": true,
		"User":    user,
		/*
			"Notes":        notes,
			"NotesHasMore": hasMore,
		*/
	})
}

func (app app) NoteListJSON(c *gin.Context) {
	user, err := app.currentUser(c)
	if err != nil {
		// Not really an error, just we don't currently have a user stored in
		// session.
		app.error(c, errors.New("no user"))
		return
	}

	strPage := c.Param("page")
	page, err := strconv.Atoi(strPage)
	if err != nil {
		app.error(c, err)
		return
	}

	notes, hasMore, err := app.data.NoteGetList(user, page, PER_PAGE)
	if err != nil {
		app.error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"HasUser":      true,
		"User":         user,
		"Notes":        notes,
		"NotesHasMore": hasMore,
	})
}

func (app app) currentUser(c *gin.Context) (User, error) {
	s := sessions.Default(c)
	token, ok := s.Get(KEY_USER_SESSION_TOKEN).(string)
	if !ok {
		return nil, errors.New("session key type assertion failed")
	}
	return app.currentUserFromToken(token)
}

func (app app) currentUserFromToken(token string) (User, error) {
	user, err := app.data.UserGet(token)
	return user, err
}

func (app app) error(c *gin.Context, err error) {
	c.String(http.StatusInternalServerError, err.Error())
}

// data class

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

	fmt.Println(connStr)

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

func (db db) UserCreate(email, plaintext string, phone int) (User, error) {
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
		notes = append(notes, note)
	}

	return notes, hasMore, status.Error
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
	UserPhone int
	UserNotes []ToPhoneNote
	token     string
	err       error
	db        db
}

func (ToPhoneUser *ToPhoneUser) Email() string { return ToPhoneUser.UserEmail }
func (ToPhoneUser *ToPhoneUser) Phone() int    { return ToPhoneUser.UserPhone }
func (ToPhoneUser *ToPhoneUser) ID() uint      { return ToPhoneUser.Model.ID }
func (ToPhoneUser *ToPhoneUser) Token() string { return ToPhoneUser.token }

func (ToPhoneUser *ToPhoneUser) SetEmail(value string) {
	ToPhoneUser.UserEmail = value
}

func (ToPhoneUser *ToPhoneUser) SetPhone(value int) {
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
	NoteText string `sql:"type:text"`
	UserID   uint
	token    string
	db       db
}

func (ToPhoneNote ToPhoneNote) Text() string  { return ToPhoneNote.NoteText }
func (ToPhoneNote ToPhoneNote) ID() uint      { return ToPhoneNote.Model.ID }
func (ToPhoneNote ToPhoneNote) Token() string { return ToPhoneNote.token }

// security

type security struct {
	secret string
}

func SecurityDefault(secret string) security {
	return security{
		secret,
	}
}

func (sec security) HashCreate(pass string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pass), 4)
	return string(bytes), err
}

func (sec security) HashCompare(pass, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
}

func (sec security) TokenCreate(val jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, val)
	return token.SignedString([]byte(sec.secret))
}

func (sec security) TokenFrom(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(sec.secret), nil
	})

	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return token.Claims.(jwt.MapClaims), nil
	} else {
		return jwt.MapClaims{}, err
	}
}

// sms class

type sms struct {
	id, secret, from string
}

func SMSDefault(id, secret, from string) sms {
	return sms{id, secret, from}
}

func (sms sms) Send(number int, text string) error {
	var to string
	if number < 10000000000 {
		to = fmt.Sprintf("1%d", number)
	} else {
		to = fmt.Sprintf("%d", number)
	}

	twilio := gotwilio.NewTwilioClient(sms.id, sms.secret)
	_, _, err := twilio.SendMMS(sms.from, to, text, "", "", "")
	return err
}

// startup

var databaseConn, databaseErr = DBConnDefault()

type server interface {
	Run(...string) error
	ServeHTTP(http.ResponseWriter, *http.Request)
}

func build(db *gorm.DB) server {
	router := gin.Default()
	router.LoadHTMLGlob("html/*")
	store := cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))
	router.Use(sessions.Sessions("lasso_sessions", store))

	sms := SMSDefault(os.Getenv("TWILIO_ID"), os.Getenv("TWILIO_SECRET"), os.Getenv("TWILIO_FROM"))
	security := SecurityDefault(os.Getenv("JWT_SECRET"))
	data := DBDefault(db, security)
	app := AppDefault(data, sms)

	router.GET("/", app.Page)
	router.POST("/", app.Page)

	router.POST("/user/login", app.UserLogin)
	router.POST("/user/create", app.UserCreate)
	router.POST("/user/update", app.UserUpdate)
	router.POST("/user/logout", app.UserLogout)

	router.POST("/note/create", app.NoteCreate)
	router.GET("/note/list/:page", app.NoteListJSON)

	router.POST("/cli/user/login", app.UserLoginCLI)
	router.POST("/cli/user/create", app.UserCreateCLI)
	router.POST("/cli/note/create", app.NoteCreateCLI)

	return router
}

func main() {
	if databaseErr != nil {
		log.Fatal(databaseErr)
		return
	}
	if err := build(databaseConn).Run(); err != nil {
		log.Fatal(err)
		return
	}
}

func H(w http.ResponseWriter, r *http.Request) {
	if databaseErr != nil {
		log.Fatal(databaseErr)
		return
	}
	build(databaseConn).ServeHTTP(w, r)
}
