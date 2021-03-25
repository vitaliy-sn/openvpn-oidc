package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var authURL = "http://127.0.0.1:9999/auth"

func init() {
	log.SetLevel(log.DebugLevel)
	if os.Getenv("SHELL") != "/bin/bash" {
		file, _ := os.OpenFile("/tmp/openvpn-auth-client-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		log.SetOutput(file)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Debug("missing argument (path to file with username and password)")
		os.Exit(1)
	}

	authFilePath := os.Args[1]
	authFileContent, err := ioutil.ReadFile(authFilePath)
	if err != nil {
		log.Debug(err)
	}

	username := strings.Split(string(authFileContent), "\n")[0]
	password := strings.Split(string(authFileContent), "\n")[1]

	client := &http.Client{
		Timeout: 7 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.PostForm(authURL, url.Values{"username": {username}, "password": {password}})
	if err != nil {
		log.Debug("failed to get ", authURL)
		os.Exit(1)
	}

	switch resp.StatusCode {
	case 200:
		fmt.Println("authorized")
		os.Exit(0)
	case 401:
		fmt.Println("unauthorized")
		os.Exit(1)
	default:
		fmt.Println("unknown")
		os.Exit(2)
	}
}
