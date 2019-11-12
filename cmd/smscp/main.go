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
	// BASE = "http://localhost:3000"
	BASE        = "https://smscp.xyz"
	APILogin    = BASE + "/cli/user/login"
	APIRegister = BASE + "/cli/user/create"
	APICreate   = BASE + "/cli/note/create"
	APILatest   = BASE + "/cli/note/latest"
)

type config struct {
	Token string
}

type hash map[string]string

// helper

func post(dest string, values hash) (*http.Response, error) {
	next := url.Values{}
	for key, val := range values {
		// Remote white spaces, new lines, and windows line endings.
		val = strings.Trim(strings.Trim(strings.Trim(val, " "), "\n"), "\r\n")
		next.Add(key, val)
	}
	return http.PostForm(dest, next) // nolint - dest is always a constant (look up)
}

// cli commands

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

	fmt.Printf("Phone number (ten or eleven digit, US number): ")
	phone, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return errors.Wrap(err, "failed to read phone number from standard in")
	}
	phone = strings.Trim(strings.Trim(phone, " "), "\n")

	/* make http req */

	resp, err := post(APIRegister, hash{
		"Email":    email,
		"Phone":    phone,
		"Password": string(pass),
		"Verify":   string(verify),
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

	err = ioutil.WriteFile(path.Join(usr.HomeDir, ".smscp"), res, 0644)
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

	resp, err := post(APILogin, hash{
		"Email":    email,
		"Password": string(pass),
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

	err = ioutil.WriteFile(path.Join(usr.HomeDir, ".smscp"), res, 0644)
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

	bytes, err := ioutil.ReadFile(path.Join(usr.HomeDir, ".smscp"))
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

	resp, err := post(APICreate, hash{
		"Token": cfg.Token,
		"Text":  string(text),
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

func latest(c *cli.Context) error {
	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve current user from operating system")
	}

	bytes, err := ioutil.ReadFile(path.Join(usr.HomeDir, ".smscp"))
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

	resp, err := post(APILatest, hash{"Token": cfg.Token})
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

	var response struct {
		Note struct {
			NoteText string
		}
	}
	err = json.Unmarshal(res, &response)
	if err != nil {
		return errors.Wrap(err, "invalid response from to remote server")
	}

	if response.Note.NoteText == "" {
		return errors.New("no note availavle; you have not made any?")
	}

	fmt.Println(strings.TrimSpace(response.Note.NoteText))
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "smscp"
	app.Usage = "CLI for https://smscp.xyz/"
	app.Version = "0.1.4"

	app.Commands = []cli.Command{
		{Name: "register", Action: register},
		{Name: "login", Action: login},
		{Name: "new", Action: create},
		{Name: "latest", Action: latest},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
