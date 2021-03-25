package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type app struct {
	listenHost       string
	listenPort       string
	domain           string
	redirectURI      string
	clientID         string
	clientSecret     string
	additionalScoups []string
	db               map[string]string
	verifier         *oidc.IDTokenVerifier
	provider         *oidc.Provider
}

func (a *app) client() *http.Client {
	return &http.Client{
		Timeout: 7 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (a *app) oauth2Config(scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.clientID,
		ClientSecret: a.clientSecret,
		Endpoint:     a.provider.Endpoint(),
		RedirectURL:  a.redirectURI,
		Scopes:       append([]string{"openid"}, scopes...),
	}
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI != "/" {
		w.WriteHeader(404)
		return
	}
	log.Info(r.RemoteAddr, " ", r.RequestURI)
	var state string
	authCodeURL := a.oauth2Config(a.additionalScoups).AuthCodeURL(state)
	log.Println(authCodeURL)
	http.Redirect(w, r, authCodeURL, http.StatusSeeOther)
}

func (a *app) handleCallback(w http.ResponseWriter, r *http.Request) {
	log.Info(r.RemoteAddr, r.RequestURI)

	code := r.URL.Query().Get("code")
	ctx := oidc.ClientContext(context.Background(), a.client())
	oauth2Token, err := a.oauth2Config(a.additionalScoups).Exchange(ctx, code)
	if err != nil {
		log.Debug("failed to get oauth2Token ", err)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug("no id_token in oauth2Token response")
		return
	}
	idToken, _ := a.verifier.Verify(context.Background(), rawIDToken)

	var claims struct {
		Email    string   `json:"email"`
		Verified bool     `json:"email_verified"`
		Groups   []string `json:"groups"`
	}

	err = idToken.Claims(&claims)
	if err != nil {
		log.Debug("error decoding ID token claims: ", err)
	}

	log.Debug("claims.Verified: ", claims.Verified)
	log.Debug("claims.Email: ", claims.Email)
	log.Debug("claims.Groups: ", claims.Groups)

	if claims.Verified {
		a.db[claims.Email] = code
		fmt.Fprintln(w, grantAccess(claims.Email, code))
	}
}

func grantAccess(login, password string) string {
	type params struct {
		Host    string
		Port    string
		CA      string
		TLSAuth string
	}

	var config params

	if os.Getenv("OPENVPN_SERVER_HOST") != "" {
		config.Host = os.Getenv("OPENVPN_SERVER_HOST")
	}

	if os.Getenv("OPENVPN_SERVER_PORT") != "" {
		config.Port = os.Getenv("OPENVPN_SERVER_PORT")
	}

	ca, err := ioutil.ReadFile("/app/certificates-and-keys/ca.crt")
	if err != nil {
		log.Debug(err)
	} else {
		config.CA = string(ca)
	}

	ta, err := ioutil.ReadFile("/app/certificates-and-keys/ta.key")
	if err != nil {
		log.Debug(err)
	} else {
		config.TLSAuth = string(ta)
	}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	t, _ := template.ParseFiles(dir + "/client.ovpn.tpl")
	var buf bytes.Buffer

	err = t.Execute(&buf, config)
	if err != nil {
		log.Debug(err)
	}
	clientConfig := buf.String()
	return (fmt.Sprintf("login: %s\npassword: %s\n###\n%s\n###", login, password, clientConfig))
}

func (a *app) handleAuth(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if a.db[username] == password {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(401)
	}
}

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	var (
		a         app
		err       error
		issuerURL string
	)

	clientSecretFileContent, err := ioutil.ReadFile("/app/oauth2/clientSecret")
	if err != nil {
		a.clientSecret = os.Getenv("CLIENT_SECRET")
	} else {
		a.clientSecret = string(clientSecretFileContent)
	}

	if os.Getenv("DOMAIN") != "" {
		a.domain = os.Getenv("DOMAIN")
	} else {
		log.Println("ERROR: environment variable [DOMAIN] is not set")
		os.Exit(1)
	}
	a.redirectURI = fmt.Sprintf("https://%s/callback", a.domain)

	if os.Getenv("CLIENT_ID") != "" {
		a.clientID = os.Getenv("CLIENT_ID")
	}

	a.additionalScoups = append(a.additionalScoups, []string{"groups", "email"}...)
	if os.Getenv("ADDITIONAL_SCOPES") != "" {
		a.additionalScoups = strings.Split(os.Getenv("ADDITIONAL_SCOPES"), " ")
	}

	a.listenHost = "0.0.0.0"
	if os.Getenv("LISTEN_HOST") != "" {
		a.listenHost = os.Getenv("LISTEN_HOST")
	}

	a.listenPort = "9999"
	if os.Getenv("LISTEN_PORT") != "" {
		a.listenPort = os.Getenv("LISTEN_PORT")
	}

	a.db = make(map[string]string)

	if os.Getenv("ISSUER_URL") != "" {
		issuerURL = os.Getenv("ISSUER_URL")
	}

	ctx := oidc.ClientContext(context.Background(), a.client())
	a.provider, err = oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		log.Debug("failed to query provider\nissuerURL:", issuerURL, "error:", err)
	}

	a.verifier = a.provider.Verifier(&oidc.Config{ClientID: a.clientID})

	http.HandleFunc("/", a.handleLogin)
	http.HandleFunc("/callback", a.handleCallback)
	http.HandleFunc("/auth", a.handleAuth)
	log.Println(a.listenHost + ":" + a.listenPort)
	log.Fatal(http.ListenAndServe(a.listenHost+":"+a.listenPort, nil))
}
