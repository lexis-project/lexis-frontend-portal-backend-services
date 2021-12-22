package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	l "gitlab.com/cyclops-utilities/logging"
)

type SessionInfo struct {
	Authenticated bool     `json:"authenticated"`
	ID            string   `json:"id"`
	Token         string   `json:"token"`
	User          UserInfo `json:"auth"`
}

// updateSession is called from the callback after a successful authentication; it
// populates the session info with the user data.
func updateSession(w http.ResponseWriter, r *http.Request, u UserInfo, t string, refT string) (returnErr error) {

	session, e := store.Get(r, sessionName)

	if session.IsNew {

		l.Debug.Printf("[SESSION] New session created with ID %v\n", session.ID)

	}

	if e != nil {

		returnErr = fmt.Errorf("[SESSION] Error getting session info - %v - unable to update session\n", e)

		return

	}

	session.Values["authenticated"] = true
	session.Values["token"] = t
	session.Values["refToken"] = refT
	session.Values["firstname"] = u.Firstname
	session.Values["lastname"] = u.Lastname
	session.Values["email"] = u.EmailAddress
	session.Values["username"] = u.Username
	session.Values["emailverified"] = u.EmailVerified
	session.Values["keycloakid"] = u.ID
	session.Values["role"] = u.Role
	session.Values["permissions"] = u.Permissions
	session.Values["ddi-projects"] = u.DDIProjects

	e = session.Save(r, w)

	if e != nil {

		returnErr = fmt.Errorf("[SESSION] Error saving session information - %v\n", e)

	}

	return

}

// getStringValueFromSession obtains a string value from a session with the given
// key
func getStringValueFromSession(s *sessions.Session, k string) (str string) {

	stemp := s.Values[k]

	if stemp != nil {

		str = stemp.(string)

	}

	return

}

func getStringArrayValueFromSession(s *sessions.Session, k string) (strArray []string) {

	satemp := s.Values[k]

	if satemp != nil {

		strArray = satemp.([]string)

	}

	return

}

// sessionInfo is called when there is a request to obtain the information for the
// session
func sessionInfo(w http.ResponseWriter, r *http.Request) {

	s, e := store.Get(r, sessionName)

	if e != nil {

		l.Error.Printf("[SESSION] Error getting session: %v\n", e)

	}

	if s.IsNew {

		l.Info.Printf("[SESSION] New session created with ID: %v\n", s.ID)

		e := s.Save(r, w)

		if e != nil {

			l.Warning.Printf("[SESSION] Error saving session information: %v\n", e)

		}

	}

	u := UserInfo{
		EmailAddress:  getStringValueFromSession(s, "email"),
		EmailVerified: s.Values["emailverified"] == "true",
		Firstname:     getStringValueFromSession(s, "firstname"),
		ID:            getStringValueFromSession(s, "keycloakid"),
		Lastname:      getStringValueFromSession(s, "lastname"),
		Role:          getStringValueFromSession(s, "role"),
		Username:      getStringValueFromSession(s, "username"),
		DDIProjects:   getStringArrayValueFromSession(s, "ddi-projects"),
	}

	if permissions, exists := s.Values["permissions"]; exists {

		u.Permissions = permissions.(map[string]interface{})

	}

	i := SessionInfo{
		ID:            s.ID, // session ID
		Authenticated: isAuthenticated(s),
		Token:         getStringValueFromSession(s, "token"),
		User:          u,
	}

	j, _ := json.Marshal(i)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)

}

// isAuthenticated checks is a session is authenticated or not
func isAuthenticated(s *sessions.Session) bool {

	auth := s.Values["authenticated"]

	if auth == nil {

		return false

	}

	return auth.(bool)

}

// createSessionStore creates a session store which is stored in an directory defined
// at compile time. There was an issue with the default behaviour; if no session directory
// is specified, then /tmp is assumed, However, for minimal containers, /tmp is not always
// present.- hence we went with this approach
func createSessionStore() {

	os.Mkdir(sessionDir, 0744)

	key := []byte(cfg.General.SessionKey)

	store = sessions.NewFilesystemStore(sessionDir, key)

	store.Options = &sessions.Options{
		Domain: cfg.General.SessionDomain,
		Path:   "/",
		MaxAge: 3600, // 1h
		Secure: true,
	}

	store.MaxLength(1048576) // 1MB

}
