package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	BASE = "http://localhost:3000"
	// BASE       = "https://tophone.evanjon.es"
	API_LOGIN    = BASE + "/cli/user/login"
	API_REGISTER = BASE + "/cli/user/create"
	API_CREATE   = BASE + "/cli/note/create"
)

type config struct {
	Token string
}

func register(c *cli.Context) error {
	fmt.Printf("Email: ")
	email, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return errors.Wrap(err, "failed to read email from standard in")
	}
	email = strings.Trim(strings.Trim(email, " "), "\n")

	fmt.Printf("Password: ")
	pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return errors.Wrap(err, "failed to read password from standard in")
	}
	fmt.Println()

	fmt.Printf("Verify password: ")
	verify, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return errors.Wrap(err, "failed to read password from standard in")
	}
	fmt.Println()

	fmt.Printf("Phone number (nine or ten digits): ")
	phone, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return errors.Wrap(err, "failed to read phone number from standard in")
	}
	phone = strings.Trim(strings.Trim(phone, " "), "\n")

	/* make http req */

	resp, err := http.PostForm(API_REGISTER, url.Values{
		"Email":    {email},
		"Phone":    {phone},
		"Password": {string(pass)},
		"Verify":   {string(verify)},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create request to remote server")
	}

	defer resp.Body.Close()

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response from to remote server")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(fmt.Errorf(string(res)), "OK not received from remote server")
	}

	var cfg config
	err = json.Unmarshal(res, &cfg)
	if err != nil {
		return errors.Wrap(err, "failed to read remote server response")
	} else if cfg.Token == "" {
		return fmt.Errorf("no token received from remote server")
	}

	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve current user from operating system")
	}

	err = ioutil.WriteFile(path.Join(usr.HomeDir, ".tophone"), res, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write config file")
	}

	return nil
}

func login(c *cli.Context) error {
	fmt.Printf("Email: ")
	email, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return errors.Wrap(err, "failed to read email from standard in")
	}
	email = strings.Trim(strings.Trim(email, " "), "\n")

	fmt.Printf("Password: ")
	pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return errors.Wrap(err, "failed to read password from standard in")
	}
	fmt.Println()

	/* make http req */

	resp, err := http.PostForm(API_LOGIN, url.Values{
		"Email":    {email},
		"Password": {string(pass)},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create request to remote server")
	}

	defer resp.Body.Close()

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response from to remote server")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(fmt.Errorf(string(res)), "OK not received from remote server")
	}

	var cfg config
	err = json.Unmarshal(res, &cfg)
	if err != nil {
		return errors.Wrap(err, "failed to read remote server response")
	} else if cfg.Token == "" {
		return fmt.Errorf("no token received from remote server")
	}

	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve current user from operating system")
	}

	err = ioutil.WriteFile(path.Join(usr.HomeDir, ".tophone"), res, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write config file")
	}

	return nil
}

func create(c *cli.Context) error {
	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve current user from operating system")
	}

	bytes, err := ioutil.ReadFile(path.Join(usr.HomeDir, ".tophone"))
	if err != nil {
		return errors.New("failed to read local file; please login")
	}

	var cfg config
	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return errors.New("failed to read local file; file of wrong format; please login")
	} else if cfg.Token == "" {
		return fmt.Errorf("failed to get user token; please login")
	}

	text, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	resp, err := http.PostForm(API_CREATE, url.Values{
		"Token": {cfg.Token},
		"Text":  {string(text)},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create request to remote server")
	}

	defer resp.Body.Close()

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response from to remote server")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(fmt.Errorf(string(res)), "OK not received from remote server")
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "tophone"
	app.Usage = "CLI for https://tophone.evanjon.es/"
	app.Version = "0.1.0"

	app.Commands = []cli.Command{
		{Name: "register", Action: register},
		{Name: "login", Action: login},
		{Name: "new", Action: create},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
