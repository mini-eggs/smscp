package main_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Pallinder/go-randomdata"
	"gopkg.in/go-playground/assert.v1"
	"smscp.xyz/pkg/builder"
	"smscp.xyz/pkg/mode"
)

var (
	server, _ = builder.Build(mode.ModeTest)
)

// helpers

func fromSession(in *httptest.ResponseRecorder, req *http.Request) *httptest.ResponseRecorder {
	out := httptest.NewRecorder()
	for _, item := range in.Result().Cookies() {
		req.AddCookie(item)
	}
	return out
}

// data gen

func goodNote() url.Values {
	form := url.Values{}
	form.Add("Text", randomdata.Paragraph())
	return form
}

func badNote() url.Values {
	form := url.Values{}
	return form
}

func goodUser() url.Values {
	form := url.Values{}
	pass := randomdata.SillyName()
	form.Add("Email", randomdata.Email())
	form.Add("Password", pass)
	form.Add("Verify", pass)
	form.Add("Phone", "(202) 555-0139") // fake number
	return form
}

func badUserPass() url.Values {
	form := url.Values{}
	pass := randomdata.SillyName()
	pass2 := randomdata.SillyName()
	form.Add("Email", randomdata.Email())
	form.Add("Password", pass)
	form.Add("Verify", pass2)
	form.Add("Phone", "(202) 555-0139") // fake number
	return form
}

func badUserPhone() url.Values {
	form := url.Values{}
	pass := randomdata.SillyName()
	form.Add("Email", randomdata.Email())
	form.Add("Password", pass)
	form.Add("Verify", pass)
	form.Add("Phone", "+233 007 4 41 36014") // fake number - not supported
	return form
}

// tests

func TestBasic(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ping", nil)
	server.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "pong", w.Body.String())
}

func TestUserCreationGood(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/create", nil)
	req.PostForm = goodUser()
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
}

func TestUserCreationBadPass(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/create", nil)
	req.PostForm = badUserPass()
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserCreationBadPhone(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/create", nil)
	req.PostForm = badUserPhone()
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserUpdateGood(t *testing.T) {
	t.Parallel()
	user := goodUser()

	// create user
	req, _ := http.NewRequest("POST", "/user/create", nil)
	req.PostForm = user
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// modify user
	user.Del("Email")
	user.Add("Email", randomdata.Email())

	// update user
	req, _ = http.NewRequest("POST", "/user/update", nil)
	req.PostForm = user
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w = fromSession(w, req)
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// logout user
	req, _ = http.NewRequest("POST", "/user/logout", nil)
	w = fromSession(w, req)
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// login user with new creds
	req, _ = http.NewRequest("POST", "/user/login", nil)
	req.PostForm = user
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
}

func TestUserUpdateBad(t *testing.T) {
	t.Parallel()

	badUsers := []url.Values{
		badUserPass(),
		badUserPhone(),
	}

	for _, badUser := range badUsers {
		user := goodUser()

		// create user
		req, _ := http.NewRequest("POST", "/user/create", nil)
		req.PostForm = user
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

		// modify user
		badUser.Del("Email")
		badUser.Set("Email", user.Get("Email"))

		// update user
		req, _ = http.NewRequest("POST", "/user/update", nil)
		req.PostForm = badUser
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		w = fromSession(w, req)
		server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// logout user
		req, _ = http.NewRequest("POST", "/user/logout", nil)
		w = fromSession(w, req)
		server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

		// login user with bad creds
		req, _ = http.NewRequest("POST", "/user/login", nil)
		req.PostForm = badUser
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		w = fromSession(w, req)
		server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// login user with good creds
		req, _ = http.NewRequest("POST", "/user/login", nil)
		req.PostForm = user
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		w = fromSession(w, req)
		server.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	}
}

func TestUserLoginBad(t *testing.T) {
	t.Parallel()
	user := goodUser()

	// create user
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/create", nil)
	req.PostForm = user
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// modify user
	user.Del("Password")
	user.Add("Password", randomdata.SillyName())

	// login bad user
	req, _ = http.NewRequest("POST", "/user/login", nil)
	req.PostForm = user
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w = fromSession(w, req)
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNoteGood(t *testing.T) {
	t.Parallel()
	user := goodUser()

	// create user
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/create", nil)
	req.PostForm = user
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// create note
	req, _ = http.NewRequest("POST", "/note/create", nil)
	req.PostForm = goodNote()
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w = fromSession(w, req)
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
}

func TestNoteBad(t *testing.T) {
	t.Parallel()
	user := goodUser()

	// create user
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/create", nil)
	req.PostForm = user
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// create note
	req, _ = http.NewRequest("POST", "/note/create", nil)
	req.PostForm = badNote()
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w = fromSession(w, req)
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
}

func TestNoteNoUser(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/note/create", nil)
	req.PostForm = badNote()
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
