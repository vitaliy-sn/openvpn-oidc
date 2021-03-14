package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	IssuerURL   string `yaml:"issuerURL"`
	RedirectURI string `yaml:"redirectURI"`
	ClientID    string `yaml:"clientID"`
	Debug       bool   `yaml:"debug"`
}

var (
	config       Config
	issuerURL    string
	clientID     string
	clientSecret string
	redirectURI  string
	userID       string
	userPassword string
	// allowedGroups string

	client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

func init() {
	configYaml, err := ioutil.ReadFile("/openvpn/oidc-config.yaml")
	if err != nil {
		logrus.Debug(err)

		configYaml, err = ioutil.ReadFile("oidc-config.yaml")
		if err != nil {
			logrus.Debug(err)
			os.Exit(1)
		}
	}

	err = yaml.Unmarshal([]byte(configYaml), &config)
	if err != nil {
		logrus.Debug("error unmarshal yaml:", err)
	}

	if config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		file, _ := os.OpenFile("/tmp/openvpn-oidc-auth-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		logrus.SetOutput(file)
	}
}

func main() {
	if len(os.Args) < 2 {
		logrus.Debug("missing argument (path to file with login and password)")
		os.Exit(1)
	}
	authFilePath := os.Args[1]
	authFileContent, _ := ioutil.ReadFile(authFilePath)
	userID = strings.Split(string(authFileContent), "\n")[0]
	userPassword = strings.Split(string(authFileContent), "\n")[1]

	// userID = os.Getenv("username")
	// userPassword = os.Getenv("password")

	issuerURL = config.IssuerURL
	clientID = config.ClientID
	redirectURI = config.RedirectURI

	fmt.Println(issuerURL)

	clientSecretFileContent, err := ioutil.ReadFile("/openvpn/oauth2/clientSecret")
	if err != nil {
		clientSecret = os.Getenv("CLIENT_SECRET")
	} else {
		clientSecret = string(clientSecretFileContent)
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	ctx := oidc.ClientContext(context.Background(), client)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		logrus.Debug("failed to query provider\nissuerURL:", issuerURL, "error:", err)
	}

	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	oauth2Config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	var state string
	redirectURL := oauth2Config.AuthCodeURL(state)

	logrus.Debug("redirectURL:", redirectURL)

	resp2, err := client.Get(redirectURL)
	if err != nil {
		logrus.Debug("failed to get", redirectURL)
		os.Exit(1)
	}
	dexAuthLocalURL, _ := resp2.Location()
	logrus.Debug("dexAuthLocalURL: ", dexAuthLocalURL.String())

	resp3, err := client.PostForm(dexAuthLocalURL.String(), url.Values{"login": {userID}, "password": {userPassword}})
	if err != nil {
		logrus.Debug("failed to get", dexAuthLocalURL.String())
		os.Exit(1)
	}
	dexApprovalURL, _ := resp3.Location()
	logrus.Debug("dexApprovalURL:", dexApprovalURL.String())

	resp4, err := client.Get(dexApprovalURL.String())
	if err != nil {
		logrus.Debug("failed to get", dexApprovalURL.String())
		os.Exit(1)
	}
	callbackURL, _ := resp4.Location()
	logrus.Debug("callbackURL:", callbackURL.String())

	code := callbackURL.Query().Get("code")
	logrus.Debug("code:", code)

	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		logrus.Debug("failed to get oauth2Token", err)
		os.Exit(1)
	}
	logrus.Debug("oauth2Token:", oauth2Token)

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		logrus.Debug("no id_token in oauth2Token response")
		os.Exit(1)
	}
	idToken, _ := idTokenVerifier.Verify(context.Background(), rawIDToken)
	logrus.Debug("idToken:", idToken)

	var claims struct {
		Email    string   `json:"email"`
		Verified bool     `json:"email_verified"`
		Groups   []string `json:"groups"`
	}

	err = idToken.Claims(&claims)
	if err != nil {
		logrus.Debug("error decoding ID token claims:", err)
	}

	logrus.Debug("claims.Verified:", claims.Verified)
	logrus.Debug("claims.Email:", claims.Email)
	logrus.Debug("claims.Groups:", claims.Groups)

	if claims.Verified {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
