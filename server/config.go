package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	l "gitlab.com/cyclops-utilities/logging"
)

type generalConfig struct {
	CertificateFile string `json:"certificate_file"`
	CertificateKey  string `json:"certificate_key"`
	FrontEndDir     string `json:"front_end_dir"`
	HttpsEnabled    bool   `json:"https_enabled"`
	LogFile         string `json:"log_file"`
	LogLevel        string `json:"log_level"`
	LogToConsole    bool   `json:"log_to_console"`
	ServerPort      int    `json:"server_port"`
	SessionDomain   string `json:"session_domain"`
	SessionKey      string `json:"session_key"`
}

type keycloakConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Realm        string `json:"realm"`
	RedirectURL  string `json:"redirect_url"`
	UseHttp      bool   `json:"use_http"`
}

type configuration struct {
	General  generalConfig  `json:"general"`
	Keycloak keycloakConfig `json:"keycloak"`
}

// masked returns asterisks in place of string except for last um=nmakedChars chars
func masked(s string, unmaskedChars int) (returnString string) {

	if len(s) <= unmaskedChars {

		returnString = s

		return

	}

	asteriskString := strings.Repeat("*", (len(s) - unmaskedChars))

	returnString = asteriskString + string(s[len(s)-unmaskedChars:])

	return

}

// readConfigfile reads in a config file in toml format with the name specified by
// the input parameter filename
func readConfigFile(f string) {

	viper.SetConfigName(f) // name of config file (without extension)
	viper.SetConfigType("toml")
	viper.AddConfigPath(".") // path to look for the config file in

	if e := viper.ReadInConfig(); e != nil {

		fmt.Printf("Fatal error reading config file. Error: %v.\n", e)

		os.Exit(1)

	}

}

// parseConfig creates a valid configuration from the input file read with viper
func parseConfig() (c configuration) {

	c = configuration{

		General: generalConfig{
			CertificateFile: viper.GetString("general.certificatefile"),
			CertificateKey:  viper.GetString("general.certificatekey"),
			FrontEndDir:     viper.GetString("general.frontenddir"),
			HttpsEnabled:    viper.GetBool("general.httpsenabled"),
			LogFile:         viper.GetString("general.logfile"),
			LogLevel:        viper.GetString("general.loglevel"),
			LogToConsole:    viper.GetBool("general.logtoconsole"),
			ServerPort:      viper.GetInt("general.serverport"),
			SessionDomain:   viper.GetString("general.sessiondomain"),
			SessionKey:      viper.GetString("general.sessionkey"),
		},

		Keycloak: keycloakConfig{
			ClientID:     viper.GetString("keycloak.clientid"),
			ClientSecret: viper.GetString("keycloak.clientsecret"),
			Host:         viper.GetString("keycloak.host"),
			Port:         viper.GetInt("keycloak.port"),
			Realm:        viper.GetString("keycloak.realm"),
			RedirectURL:  viper.GetString("keycloak.redirecturl"),
			UseHttp:      viper.GetBool("keycloak.usehttp"),
		},
	}

	return
}

// dumpConfig dumps the configuration in json format to the log system
func dumpConfig(c configuration) {

	cfgCopy := c

	// deal with configuration params that should be masked
	cfgCopy.General.SessionKey = masked(c.General.SessionKey, 4)
	cfgCopy.Keycloak.ClientSecret = masked(c.Keycloak.ClientSecret, 4)

	// mmrshalindent creates a string containing newlines; each line starts with
	// two spaces and two spaces are added for each indent...
	configJson, _ := json.MarshalIndent(cfgCopy, "  ", "  ")

	l.Info.Printf("Configuration settings:\n")

	l.Info.Printf("%v\n", string(configJson))

}
