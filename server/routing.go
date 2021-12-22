package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v7"
	"github.com/coreos/go-oidc"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	l "gitlab.com/cyclops-utilities/logging"
)

// login is called when the front end tries to log in. It currently is implemented
// as a primitive redirect; it should probably check the login status first and act
// accordingly
func login(w http.ResponseWriter, r *http.Request) middleware.Responder {

	// implements the login with a redirection
	return middleware.ResponderFunc(

		func(w http.ResponseWriter, pr runtime.Producer) {

			l.Info.Printf("[ROUTING] Returning redirect...\n")

			http.Redirect(w, r, Oauth2Config.AuthCodeURL(state), http.StatusFound)

		})

}

// callback is called after the OpenID login process has completed - it obtains
// information from the openid provider and determines whether the login was
// successful. Note that as per standard OpenID flows, we expect the callbadk to
// contain a state and a code.
func callback(w http.ResponseWriter, r *http.Request) (string, error) {

	if r.URL.Query().Get("state") != state {

		l.Info.Printf("[ROUTING] State did not match\n")

		return "", fmt.Errorf("state did not match")

	}

	myClient := &http.Client{}

	parentContext := context.Background()

	ctx := oidc.ClientContext(parentContext, myClient)

	// Exchange converts an authorization code into an access token.
	// Under the hood, the oauth2 client POST a request to do so
	// at tokenURL, then redirects...
	authCode := r.URL.Query().Get("code")

	oauth2Token, e := Oauth2Config.Exchange(ctx, authCode)

	if e != nil {

		l.Info.Printf("[ROUTING] Failed to exchange token: %v\n", e)

		return "", fmt.Errorf("failed to exchange token")

	}

	// the authorization server's returned token
	// l.Debug.Printf("[ROUTING] Access token : %v\n", oauth2Token.AccessToken)
	// l.Debug.Printf("[ROUTING] Refresh token : %v\n", oauth2Token.RefreshToken)

	u, _ := getUserInfo(oauth2Token.AccessToken)

	e = updateSession(w, r, u, oauth2Token.AccessToken, oauth2Token.RefreshToken)

	if e != nil {

		l.Info.Printf("[ROUTING] Error updating session: %v\n", e)

		return "", nil

	}

	http.Redirect(w, r, "/", http.StatusFound)

	// never called...
	return "", nil

}

func logout(w http.ResponseWriter, r *http.Request) {

	keycloakService := getKeycloakService(cfg.Keycloak)

	client := gocloak.NewClient(keycloakService)

	ctx := context.Background()

	// adminToken, e := client.LoginClient(ctx, cfg.Keycloak.ClientID, cfg.Keycloak.ClientSecret, cfg.Keycloak.Realm)

	// if e != nil {

	// 	l.Warning.Printf("[ROUTING] Error logging in to keycloak: %v\n", e)

	// }

	s, e := store.Get(r, sessionName)

	if e != nil {

		l.Error.Printf("[ROUTING] Error getting session: %v\n", e)

	}

	if s.IsNew {

		l.Info.Printf("[ROUTING] New session created with ID %v\n", s.ID)

		e := s.Save(r, w)

		if e != nil {

			l.Warning.Printf("[ROUTING] Error saving session information: %v\n", e)

		}

	}

	e = client.Logout(ctx, cfg.Keycloak.ClientID, cfg.Keycloak.ClientSecret, cfg.Keycloak.Realm, getStringValueFromSession(s, "refToken"))

	if e != nil {

		l.Warning.Printf("[ROUTING] Error logging out the user from keycloak: %v\n", e)

	}

	s.Values["authenticated"] = false
	s.Values["token"] = ""
	s.Values["refToken"] = ""
	s.Values["firstname"] = ""
	s.Values["lastname"] = ""
	s.Values["email"] = ""
	s.Values["username"] = ""
	s.Values["emailverified"] = ""
	s.Values["role"] = ""
	s.Values["keycloakid"] = ""
	s.Values["permissions"] = ""

	s.Options.MaxAge = -1

	e = s.Save(r, w)

	if e != nil {

		l.Error.Printf("[ROUTING] Error saving session information: %v\n", e)

	}

}

func FileServerMiddleware() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		session, _ := store.Get(r, sessionName)

		if session.IsNew {

			l.Info.Printf("[ROUTING] New session\n")

		}

		l.Info.Printf("[ROUTING] Session id = %v\n", session.ID)

		session.Values["test"] = false

		e := session.Save(r, w)

		if e != nil {

			l.Warning.Printf("[ROUTING] Error saving session information: %v\n", e)

		}

		l.Info.Printf("[ROUTING] Serving endpoint request %v\n", r.URL)

		switch {

		case strings.HasPrefix(r.URL.Path, "/auth/login"):

			l.Info.Printf("[ROUTING] Calling login function\n")

			http.Redirect(w, r, Oauth2Config.AuthCodeURL(state), http.StatusFound)
			// login(w, r)

		case strings.HasPrefix(r.URL.Path, "/auth/logout"):

			logout(w, r)

		case strings.HasPrefix(r.URL.Path, "/auth/callback"):

			callback(w, r)

		case strings.HasPrefix(r.URL.Path, "/auth/session-info"):

			sessionInfo(w, r)

		case strings.HasPrefix(r.URL.Path, "/dataset"),
			strings.HasPrefix(r.URL.Path, "/organization"),
			strings.HasPrefix(r.URL.Path, "/project"),
			strings.HasPrefix(r.URL.Path, "/user"),
			strings.HasPrefix(r.URL.Path, "/workflow"),
			strings.HasPrefix(r.URL.Path, "/error"):

			http.ServeFile(w, r, cfg.General.FrontEndDir+"/index.html")

		default:

			http.FileServer(http.Dir(cfg.General.FrontEndDir)).ServeHTTP(w, r)

		}

	})

}
