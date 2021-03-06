// Package cookie provides standard CouchDB cookie auth as described at
// http://docs.couchdb.org/en/2.0.0/api/server/authn.html#cookie-authentication
package cookie

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/auth"
	"github.com/flimzy/kivik/authdb"
	"github.com/flimzy/kivik/errors"
	"github.com/flimzy/kivik/serve"
)

// Auth provides CouchDB Cookie authentication.
type Auth struct{}

var _ auth.Handler = &Auth{}

// MethodName returns "cookie"
func (a *Auth) MethodName() string {
	return "cookie" // For compatibility with the name used by CouchDB
}

// Authenticate authenticates a request with cookie auth against the user store.
func (a *Auth) Authenticate(w http.ResponseWriter, r *http.Request) (*authdb.UserContext, error) {
	if r.URL.Path == "/_session" {
		switch r.Method {
		case kivik.MethodPost:
			return nil, postSession(w, r)
		case kivik.MethodDelete:
			return nil, deleteSession(w, r)
		}
	}
	return a.validateCookie(w, r)
}

func (a *Auth) validateCookie(w http.ResponseWriter, r *http.Request) (*authdb.UserContext, error) {
	store := serve.GetService(r).UserStore
	cookie, err := r.Cookie(kivik.SessionCookieName)
	if err != nil {
		return nil, nil
	}
	name, _, err := serve.DecodeCookie(cookie.Value)
	if err != nil {
		return nil, nil
	}
	user, err := store.UserCtx(r.Context(), name)
	if err != nil {
		// Failed to look up the user
		return nil, nil
	}
	s := serve.GetService(r)
	valid, err := s.ValidateCookie(r.Context(), user, cookie.Value)
	if err != nil || !valid {
		return nil, nil
	}
	return user, nil
}

func postSession(w http.ResponseWriter, r *http.Request) error {
	authData := struct {
		Name     *string `form:"name" json:"name"`
		Password string  `form:"password" json:"password"`
	}{}
	if err := serve.BindParams(r, &authData); err != nil {
		return errors.Status(kivik.StatusBadRequest, "unable to parse request data")
	}
	if authData.Name == nil {
		return errors.Status(kivik.StatusBadRequest, "request body must contain a username")
	}
	s := serve.GetService(r)
	user, err := s.UserStore.Validate(r.Context(), *authData.Name, authData.Password)
	if err != nil {
		return err
	}
	timeout, err := getSessionTimeout(r.Context(), s)
	if err != nil {
		return err
	}
	next, err := redirectURL(r)
	if err != nil {
		return err
	}

	// Success, so create a cookie
	token, err := s.CreateAuthToken(r.Context(), *authData.Name, user.Salt, time.Now().Unix())
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "must-revalidate")
	http.SetCookie(w, &http.Cookie{
		Name:     kivik.SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   timeout,
		HttpOnly: true,
	})
	w.Header().Add("Content-Type", kivik.TypeJSON)
	if next != "" {
		w.Header().Add("Location", next)
		w.WriteHeader(kivik.StatusFound)
	}
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":    true,
		"name":  user.Name,
		"roles": user.Roles,
	})
}

func redirectURL(r *http.Request) (string, error) {
	next, ok := serve.StringQueryParam(r, "next")
	if !ok {
		return "", nil
	}
	if !strings.HasPrefix(next, "/") {
		return "", errors.Status(kivik.StatusBadRequest, "redirection url must be relative to server root")
	}
	if strings.HasPrefix(next, "//") {
		// Possible schemaless url
		return "", errors.Status(kivik.StatusBadRequest, "invalid redirection url")
	}
	parsed, err := url.Parse(next)
	if err != nil {
		return "", errors.Status(kivik.StatusBadRequest, "invalid redirection url")
	}
	return parsed.String(), nil
}

func deleteSession(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:     kivik.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	w.Header().Add("Content-Type", kivik.TypeJSON)
	w.Header().Set("Cache-Control", "must-revalidate")
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
	})
}

func getSessionTimeout(ctx context.Context, s *serve.Service) (int, error) {
	timeout, err := s.Config().GetContext(ctx, "couch_httpd_auth", "timeout")
	if errors.StatusCode(err) == kivik.StatusNotFound {
		return serve.DefaultSessionTimeout, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(timeout)
}
