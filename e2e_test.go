package main_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/Pallinder/go-randomdata"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/go-playground/assert.v1"
	"smscp.xyz/pkg/builder"
	"smscp.xyz/pkg/mode"
)

var (
	server, _ = builder.Build(mode.Test)
)

// helpers

func fromSession(in *httptest.ResponseRecorder, req *http.Request) *httptest.ResponseRecorder {
	out := httptest.NewRecorder()
	for _, item := range in.Result().Cookies() {
		req.AddCookie(item)
	}
	return out
}

// Prehook to remove test db data
func TestMain(m *testing.M) {
	os.Exit(m.Run())
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
	form.Add("Username", "__test__"+randomdata.SillyName())
	form.Add("Password", pass)
	form.Add("Verify", pass)
	form.Add("Phone", fmt.Sprintf("(208) %d-%d", randomdata.Number(100, 999), randomdata.Number(1000, 9999)))
	return form
}

func badUserPass() url.Values {
	form := url.Values{}
	pass := randomdata.SillyName()
	pass2 := randomdata.SillyName()
	form.Add("Username", "__test__"+randomdata.SillyName())
	form.Add("Password", pass)
	form.Add("Verify", pass2)
	form.Add("Phone", fmt.Sprintf("(208) %d-%d", randomdata.Number(100, 999), randomdata.Number(1000, 9999)))
	return form
}

func badUserPhone() url.Values {
	form := url.Values{}
	pass := randomdata.SillyName()
	form.Add("Username", "__test__"+randomdata.SillyName())
	form.Add("Password", pass)
	form.Add("Verify", pass)
	form.Add("Phone", randomdata.PhoneNumber()) // unsupported national
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
	user := goodUser()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/create", nil)
	req.PostForm = user
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(w, req)
	if w.Code != http.StatusTemporaryRedirect {
		spew.Dump("TestUserCreationGood", w.Body.String(), user)
	}
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
	if w.Code != http.StatusTemporaryRedirect {
		spew.Dump("TestUserUpdateGood", w.Body.String(), user)
	}
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// modify user
	user.Del("Username")
	user.Add("Username", randomdata.SillyName())

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
		badUser.Del("Username")
		badUser.Set("Username", user.Get("Username"))

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
	if w.Code != http.StatusTemporaryRedirect {
		spew.Dump("TestUserLoginBad", w.Body.String(), user)
	}
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
