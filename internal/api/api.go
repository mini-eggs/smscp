package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/ttacon/libphonenumber"
	"smscp.xyz/internal/common"
)

type App struct {
	data dataLayer
	sms  smsLayer
	csv  csvLayer
	sec  securityLayer
	cfg  cfg
}

type cfg struct {
	resetPassswordLink string
}

const (
	perPage             = 20
	sessionKeyUserToken = "USER_TOKEN"
)

type dataLayer interface {
	// user
	UserGet(ctx context.Context, token string) (common.User, error)
	UserGetByNumber(ctx context.Context, number string) (common.User, error)
	UserGetByUsername(ctx context.Context, username string) (common.User, error)
	UserLogin(ctx context.Context, username, pass string) (common.User, error)
	UserCreate(ctx context.Context, username, pass, phone string) (common.User, error)
	// notes
	NoteGetList(ctx context.Context, user common.User, page, count int) ([]common.Note, bool, error)
	NoteGetLatest(ctx context.Context, user common.User) (common.Note, error)
	NoteGetLatestWithTime(ctx context.Context, user common.User, t time.Duration) (common.Note, error)
	NoteCreate(ctx context.Context, user common.User, text string) (common.Note, error)
	// special gdpr
	UserAll(context.Context, common.User) ([]common.Note, error)
	UserDel(context.Context, common.User) error
}

type csvLayer interface {
	ToFile(common.User, []common.Note) (*os.File, error)
}

type smsLayer interface {
	Send(number, text string) error
	Hook(c *gin.Context) (number, text string, err error)
}

type securityLayer interface {
	TokenCreate(val jwt.Claims) (string, error)
	TokenFrom(tokenString string) (jwt.MapClaims, error)
}

func AppDefault(data dataLayer, sms smsLayer, csv csvLayer, sec securityLayer) App {
	return App{
		data,
		sms,
		csv,
		sec,
		cfg{"https://smscp.xyz/reset/%s"},
	}
}

func (app App) HookSMS(c *gin.Context) {
	num, text, err := app.sms.Hook(c)
	if err != nil {
		app.error(c, err)
		return
	}

	user, err := app.data.UserGetByNumber(c, num)
	if err != nil {
		app.error(c, err)
		return
	}

	_, err = app.data.NoteCreate(c, user, text)
	if err != nil {
		app.error(c, err)
		return
	}

	c.String(http.StatusOK, "message received")
}

func (app App) NoteCreate(c *gin.Context) {
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

	note, err := app.data.NoteCreate(c, user, payload.Text)
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

func (app App) NoteCreateCLI(c *gin.Context) {
	var payload struct {
		Token, Text string
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	user, err := app.currentUserFromToken(c, payload.Token)
	if err != nil {
		app.error(c, errors.New("not logged in; or something else terribly wrong"))
		return
	}

	note, err := app.data.NoteCreate(c, user, payload.Text)
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

func (app App) NoteLatestCLI(c *gin.Context) {
	var payload struct {
		Token string
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	user, err := app.currentUserFromToken(c, payload.Token)
	if err != nil {
		app.error(c, errors.New("not logged in; or something else terribly wrong"))
		return
	}

	note, err := app.data.NoteGetLatest(c, user)
	if err != nil {
		app.error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"Message": "complete", "Note": note})
}

func (app App) UserLogin(c *gin.Context) {
	var payload struct {
		Username, Password string
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	user, err := app.data.UserLogin(c, payload.Username, payload.Password)
	if err != nil {
		app.error(c, err)
		return
	}

	fsUser, err := app.data.UserLogin(c, payload.Username, payload.Password)
	if err != nil {
		app.error(c, err)
		return
	}
	_ = fsUser

	s := sessions.Default(c)
	s.Set(sessionKeyUserToken, user.Token())
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app App) UserLoginCLI(c *gin.Context) {
	var payload struct {
		Username, Password string
	}

	err := c.Bind(&payload)
	if err != nil {
		app.error(c, err)
		return
	}

	user, err := app.data.UserLogin(c, payload.Username, payload.Password)
	if err != nil {
		app.error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"Token": user.Token()})
}

func (app App) UserCreate(c *gin.Context) {
	var payload struct {
		Username, Password, Verify, Phone string
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

	phone, err := libphonenumber.Parse(payload.Phone, "US")
	if err != nil {
		app.error(c, errors.Wrap(err, "must be US number"))
		return
	} else if !libphonenumber.IsValidNumber(phone) {
		app.error(c, errors.New("invalid phone number; try again"))
		return
	}
	full := fmt.Sprintf("%d%d", phone.GetCountryCode(), phone.GetNationalNumber())

	user, err := app.data.UserCreate(c, payload.Username, payload.Password, full)
	if err != nil {
		app.error(c, err)
		return
	}

	s := sessions.Default(c)
	s.Set(sessionKeyUserToken, user.Token())
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app App) UserCreateCLI(c *gin.Context) {
	var payload struct {
		Username, Password, Verify, Phone string
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

	phone, err := libphonenumber.Parse(payload.Phone, "US")
	if err != nil {
		app.error(c, err)
		return
	} else if !libphonenumber.IsValidNumber(phone) {
		app.error(c, errors.New("invalid phone number; try again"))
		return
	}
	full := fmt.Sprintf("%d%d", phone.GetCountryCode(), phone.GetNationalNumber())

	user, err := app.data.UserCreate(c, payload.Username, payload.Password, full)
	if err != nil {
		app.error(c, err)
		return
	}

	s := sessions.Default(c)
	s.Set(sessionKeyUserToken, user.Token())
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"Token": user.Token()})
}

func (app App) UserUpdate(c *gin.Context) {
	var payload struct {
		Username, Password, Verify, Phone string
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

	if payload.Username != "" {
		user.SetUsername(payload.Username)
	}

	if payload.Password != "" {
		user.SetPass(payload.Password)
	}

	if payload.Phone != "" {
		phone, err := libphonenumber.Parse(payload.Phone, "US")
		if err != nil {
			app.error(c, errors.Wrap(err, "must be US number"))
			return
		} else if !libphonenumber.IsValidNumber(phone) {
			app.error(c, errors.New("invalid phone number; try again"))
			return
		}
		user.SetPhone(fmt.Sprintf("%d%d", phone.GetCountryCode(), phone.GetNationalNumber()))
	}

	err = user.Save(c)
	if err != nil {
		app.error(c, err)
		return
	}

	s := sessions.Default(c)
	s.Set(sessionKeyUserToken, user.Token())
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app App) UserLogout(c *gin.Context) {
	s := sessions.Default(c)
	s.Set(sessionKeyUserToken, nil)
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app App) Page(c *gin.Context) {
	user, err := app.currentUser(c)
	if err != nil {
		// Not really an error, just we don't currently have a user stored in
		// session.
		c.HTML(http.StatusOK, "main.html", gin.H{
			"HasUser": false,
		})
		return
	}

	page := 0
	notes, hasMore, err := app.data.NoteGetList(c, user, page, perPage)
	if err != nil {
		app.error(c, err)
		return
	}

	latest, err := app.data.NoteGetLatestWithTime(c, user, 5*time.Minute)
	if err != nil {
		app.error(c, err)
		return
	}

	c.HTML(http.StatusOK, "main.html", gin.H{
		"HasUser":      true,
		"User":         user,
		"Notes":        notes,
		"NotesHasMore": hasMore,
		"Latest":       latest,
	})
}

func (app App) NoteListJSON(c *gin.Context) {
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

	notes, hasMore, err := app.data.NoteGetList(c, user, page, perPage)
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

func (app App) Pong(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func (app App) currentUser(c *gin.Context) (common.User, error) {
	s := sessions.Default(c)
	token, ok := s.Get(sessionKeyUserToken).(string)
	if !ok {
		return nil, errors.New("no session available; no user")
	}
	return app.currentUserFromToken(c, token)
}

func (app App) currentUserFromToken(ctx context.Context, token string) (common.User, error) {
	user, err := app.data.UserGet(ctx, token)
	return user, err
}

func (app App) error(c *gin.Context, err error) {
	c.String(http.StatusInternalServerError, err.Error())
}

func (app App) UserExportAllData(c *gin.Context) {
	user, err := app.currentUser(c)
	if err != nil {
		app.error(c, errors.New("no user"))
		return
	}

	notes /* messages, */, err := app.data.UserAll(c, user)
	if err != nil {
		app.error(c, errors.Wrap(err, "failed to retrieve user data"))
		return
	}

	file, err := app.csv.ToFile(user, notes)
	if err != nil {
		app.error(c, errors.Wrap(err, "failed to generate csv file export"))
		return
	}

	byt, err := ioutil.ReadFile(file.Name())
	if err != nil {
		app.error(c, errors.Wrap(err, "failed to read csv file export"))
		return
	}
	defer os.Remove(file.Name()) // cleanup file

	filename := fmt.Sprintf("%s_user_data.csv", url.QueryEscape(user.Username()))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.String(http.StatusOK, string(byt))
}

func (app App) UserDeleteAllData(c *gin.Context) {
	user, err := app.currentUser(c)
	if err != nil {
		app.error(c, errors.New("no user"))
		return
	}

	if err := app.data.UserDel(c, user); err != nil {
		app.error(c, errors.Wrap(err, "failed to delete user data"))
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app App) UserForgotPassword(c *gin.Context) {
	var payload struct{ Username string }
	if err := c.Bind(&payload); err != nil {
		app.error(c, errors.Wrap(err, "failed to retreive username provided"))
		return
	}

	user, err := app.data.UserGetByUsername(c, payload.Username)
	if err != nil {
		app.error(c, errors.Wrap(err, "no user"))
		return
	}

	token, err := app.sec.TokenCreate(jwt.MapClaims{
		"UserToken": user.Token(),
		"Time":      time.Now().UTC().Format(time.UnixDate),
	})
	if err != nil {
		app.error(c, errors.Wrap(err, "failed to create magic link"))
		return
	}

	msg := `Please visit the link below to reset your password.

`

	err = app.sms.Send(user.Phone(), msg+fmt.Sprintf(app.cfg.resetPassswordLink, token))
	if err != nil {
		app.error(c, errors.Wrap(err, "failed to send sms"))
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app App) PageForgotPassword(c *gin.Context) {
	c.HTML(http.StatusOK, "forgot-password.html", gin.H{"HasUser": false})
}

func (app App) UserForgotPasswordNewPassword(c *gin.Context) {
	var payload struct{ Password, Verify string }
	if err := c.Bind(&payload); err != nil {
		app.error(c, errors.Wrap(err, "failed to retreive form payload"))
		return
	}

	if payload.Password != payload.Verify || payload.Password == "" {
		app.error(c, errors.New("invalid password; either not equal or no password entered"))
		return
	}

	data, err := app.sec.TokenFrom(c.Param("hash"))
	if err != nil {
		app.error(c, errors.Wrap(err, "could not read magic link"))
		return
	}

	timeData, ok := data["Time"]
	if !ok {
		app.error(c, errors.New("could not read magic link; no time value"))
		return
	}

	timeStr, ok := timeData.(string)
	if !ok {
		app.error(c, errors.New("could not read magic link; invalid time type"))
		return
	}

	t, err := time.Parse(time.UnixDate, timeStr)
	if err != nil {
		app.error(c, errors.Wrap(err, "could not read magic link; invalid time value"))
		return
	}

	if t.Add(5 * time.Minute).Before(time.Now().UTC()) {
		app.error(c, errors.New("this link has expired; may not reset password here"))
		return
	}

	tokenData, ok := data["UserToken"]
	if !ok {
		app.error(c, errors.New("could not read magic link; no user token"))
		return
	}

	tokenStr, ok := tokenData.(string)
	if !ok {
		app.error(c, errors.New("could not read magic link; invalid user token"))
		return
	}

	user, err := app.currentUserFromToken(c, tokenStr)
	if err != nil {
		app.error(c, errors.Wrap(err, "this token does not represent a user; broken token"))
		return
	}

	user.SetPass(payload.Password)
	if err := user.Save(c); err != nil {
		app.error(c, errors.Wrap(err, "failed to update password"))
		return
	}

	s := sessions.Default(c)
	s.Set(sessionKeyUserToken, user.Token())
	if err := s.Save(); err != nil {
		app.error(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}
