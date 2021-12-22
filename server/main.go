package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/sessions"
	l "gitlab.com/cyclops-utilities/logging"
	"golang.org/x/oauth2"
)

var (
	cfg         configuration
	version     string
	serviceName = "Lexis Portal"
	state       = "foobar" // this should not be a global but is currently for testing..

	Oauth2Config oauth2.Config
	sessionDir   = "./sessions"
	store        *sessions.FilesystemStore
	sessionName  = "lexis-session"
)

// init function - reads in configuration file and creates logger
func init() {

	confFile := flag.String("conf", "./config", "configuration file path (without toml extension)")

	flag.Parse()

	//placeholder code as the default value will ensure this situation will never arise
	if len(*confFile) == 0 {

		fmt.Println("Usage: go-lexis-portal --conf=/path/to/configuration/file")

		os.Exit(0)

	}

	readConfigFile(*confFile)

	cfg = parseConfig()

	// when communicating with other services, they may not be secured with valid Https
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	Oauth2Config = createOauth2Config(cfg.Keycloak)

	createSessionStore()

	l.InitLogger(cfg.General.LogFile, cfg.General.LogLevel, cfg.General.LogToConsole)

	dumpConfig(cfg)

	l.Info.Printf("%v version %v initialized", serviceName, version)

}

// main function creates the database connection and launches the endpoint handlers
func main() {

	f := FileServerMiddleware()

	// note that this runs on all interfaces right now
	serviceLocation := ":" + strconv.Itoa(cfg.General.ServerPort)

	l.Info.Printf("Starting to serve %v, access server on http://localhost:%v\n", serviceName, serviceLocation)

	// Run the standard http server
	if cfg.General.HttpsEnabled {

		l.Error.Printf((http.ListenAndServeTLS(serviceLocation, cfg.General.CertificateFile, cfg.General.CertificateKey, f)).Error())

	} else {

		l.Warning.Printf("Running without TLS security - do not use in production scenario...")

		l.Error.Printf((http.ListenAndServe(serviceLocation, f)).Error())

	}

}
