package main

import (
	"context"
	"errors"
	"os"
	"sort"
	"strconv"
	"strings"

	kclib "code.it4i.cz/lexis/wp4/keycloak-lib"
	"github.com/Nerzal/gocloak/v7"
	"github.com/coreos/go-oidc"
	l "gitlab.com/cyclops-utilities/logging"
	"golang.org/x/oauth2"
)

const (
	NIL     = "00000000-0000-0000-0000-000000000000"
	ORG_ATT = "org_read"
	PRJ_ATT = "prj_list"
)

var (
	//roles = []string{"lex_adm", "lex_sup", "org_mgr", "prj_mgr", "dat_mgr", "end_usr"}
	roles = []string{"org_mgr", "prj_mgr", "dat_mgr", "end_usr"}
)

type UserInfo struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	EmailAddress  string   `json:"email"`
	EmailVerified bool     `json:"emailverified"`
	Firstname     string   `json:"firstname"`
	Lastname      string   `json:"lastname"`
	Role          string   `json:"role"`
	Organization  string   `json:"organization"`
	Projects      []string `json:"projects"`
	DDIProjects   []string `json:"ddi-projects"`
	Token         string
	Permissions   map[string]interface{}
}

// createOAuth2Config creates an OAuth2Config struct populated with the appropriate
// data based on what is in the confiuration
func createOauth2Config(c keycloakConfig) (returnConfig oauth2.Config) {

	ctx := context.Background()

	keycloakService := getKeycloakService(c)

	configUrl := keycloakService + "/auth/realms/" + c.Realm

	l.Info.Printf("Connecting to OIDC provider at %v\n", configUrl)

	provider, e := oidc.NewProvider(ctx, configUrl)

	if e != nil {

		l.Error.Printf("Error creating provider: %v\n", e)

		os.Exit(1)

	}

	returnConfig = oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURL,
		Endpoint:     provider.Endpoint(),        // Discovery returns the OAuth2 endpoints
		Scopes:       []string{oidc.ScopeOpenID}, //, "profile", "email", "roles"}, // "openid" is a required scope for OpenID Connect flows
	}

	return

}

// getKeycloaktService returns the keycloak service; note that there has to be exceptional
// handling of port 80 and port 443
func getKeycloakService(c keycloakConfig) (s string) {

	if c.UseHttp {

		s = "http://" + c.Host

	} else {

		s = "https://" + c.Host

	}

	if c.Port != 80 && c.Port != 443 {

		s = s + ":" + strconv.Itoa(c.Port)

	}

	return

}

// getUserInfo gets info pertaining to the users
func getUserInfo(token string) (u UserInfo, returnError error) {

	l.Debug.Printf("[KEYCLOAK] Performing authentication check. Token [ ****%v... ]\n", token[:13])

	keycloakService := getKeycloakService(cfg.Keycloak)
	client := gocloak.NewClient(keycloakService)
	ctx := context.Background()

	_, e := client.LoginClient(ctx, cfg.Keycloak.ClientID, cfg.Keycloak.ClientSecret, cfg.Keycloak.Realm)

	if e != nil {

		l.Warning.Printf("[KEYCLOAK] Problems logging into keycloak. Error: %v\n", e)
		returnError = errors.New("unable to log in to keycloak")

		return

	}

	retroinspection, e := client.RetrospectToken(ctx, token, cfg.Keycloak.ClientID, cfg.Keycloak.ClientSecret, cfg.Keycloak.Realm)

	if e != nil {

		l.Warning.Printf("[KEYCLOAK] Problems retroinspecting the token. Error: %v\n", e)
		returnError = errors.New("unable to retroinspect the token")

		return

	}

	if !*retroinspection.Active {

		l.Warning.Printf("[KEYCLOAK] The token seems to be no longer valid.\n")
		returnError = errors.New("token no longer valid")

		return

	}

	attributes, e := client.GetRawUserInfo(ctx, token, cfg.Keycloak.Realm)

	if e != nil {

		l.Warning.Printf("[KEYCLOAK] Problems retrieving the user info. Error: %v\n", e)
		returnError = errors.New("unable to get the user info")

		return

	}

	u.Token = token

	// end_usr for default role
	u.Role = roles[len(roles)-1]

	if attributes["sub"] != nil {

		u.ID = attributes["sub"].(string)
		u.EmailAddress = attributes["email"].(string)
		u.EmailVerified = attributes["email_verified"].(bool)
		u.Firstname = attributes["given_name"].(string)
		u.Lastname = attributes["family_name"].(string)
		u.Username = attributes["preferred_username"].(string)

	}

	if attributes["attributes"] != nil {

		l.Warning.Printf("[KC<->UO] Attributes received from Keycloak for the user: %+v\n", attributes["attributes"])

		u.Organization, u.Projects, u.DDIProjects = getIDs(attributes["attributes"].(map[string]interface{}))
		u.Role = getRole(attributes["attributes"].(map[string]interface{}), u.Organization, strings.Join(u.Projects, " "))

		u.Permissions = attributes["attributes"].(map[string]interface{})

	} else {

		l.Warning.Printf("[AUTHZ] The user attributes from Keycloak are nil, if this is not a newly created user then there's something wrong with Keycloak!\n")

	}

	return

}

// getIDs job is to parse the attributes received from Keycloak and extract the
// ORG and PRJ UUIDs.
// The design assumes that the first UUID tha matches the attributes is the only
// one around and doesn't double check.
func getIDs(att map[string]interface{}) (org string, prj, sn []string) {

	prjs := make(map[string]int)
	orgs := make(map[string]int)
	sns := make(map[string]int)

	org = NIL

	attArray, exists := att[PRJ_ATT].([]interface{})

	if exists {

		for i := range attArray {

			attMap, exists := attArray[i].(map[string]interface{})

			if exists {

				//orgs[attMap["ORG_UUID"].(string)]++
				prjs[attMap["PRJ_UUID"].(string)]++
				sns[attMap["PRJ"].(string)]++

			}

		}

	}

	attArray, exists = att[ORG_ATT].([]interface{})

	if exists {

		for i := range attArray {

			attMap, exists := attArray[i].(map[string]interface{})

			if exists {

				orgs[attMap["ORG_UUID"].(string)]++

			}
		}

	}

	if len(orgs) > 0 {

		keys := make([]string, len(orgs))

		i := 0

		for k := range orgs {

			keys[i] = k

			i++

		}

		sort.Strings(keys)

		org = strings.Join(keys, " ")

	}

	if len(prjs) > 0 {

		keys := make([]string, len(prjs))

		i := 0

		for k := range prjs {

			keys[i] = k

			i++

		}

		sort.Strings(keys)

		prj = keys

	}

	if len(sns) > 0 {

		keys := make([]string, len(sns))

		i := 0

		for k := range sns {

			keys[i] = k

			i++

		}

		sort.Strings(keys)

		sn = keys

	}

	return

}

func getRole(att map[string]interface{}, orgs, prjs string) string {

	at := make(map[string][]string)
	pickRole := make(map[string]int)

	var prj string

	for v, k := range att {

		var b [][]string

		for _, m := range k.([]interface{}) {

			for r, i := range m.(map[string]interface{}) {

				var a []string

				a = append(a, strings.ToUpper(r))
				a = append(a, i.(string))

				b = append(b, a)

			}

		}

		at[strings.ToUpper(v)] = []string{createJsonOfAttributes(b)}

	}

	for _, org := range strings.Fields(orgs) {

		for _, p := range strings.Fields(prjs) {

			for _, role := range roles {

				prj = ""

				if p != NIL {

					prj = p

				}

				if kclib.CheckAccess(at, role, org, prj) {

					pickRole[role]++

				}

			}

		}

		if len(strings.Fields(prjs)) < 1 {

			prj = ""

			for _, role := range roles {

				if kclib.CheckAccess(at, role, org, prj) {

					pickRole[role]++

				}

			}
		}

	}

	for i := 0; i < len(roles); i++ {

		if _, exists := pickRole[roles[i]]; exists {

			return roles[i]

		}

	}

	return roles[len(roles)-1]

}

func createJsonOfAttributes(attributes [][]string) string {

	var res = "{"

	for index, attribute := range attributes {

		res = res + "\"" + attribute[0] + "\":\"" + attribute[1] + "\""

		if index < len(attributes)-1 {

			res += ","

		}

	}

	res += "}"

	return res

}
