package server

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/pomdtr/smallweb/server/storage"
// 	"golang.org/x/oauth2"
// )

// type AuthMiddleware struct {
// 	next http.Handler
// 	db   *storage.DB
// }

// func NewAuthMiddleware(handler http.Handler, db *storage.DB) *AuthMiddleware {
// 	return &AuthMiddleware{
// 		next: handler,
// 		db:   db,
// 	}
// }

// func (me *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	var oauthConfig = &oauth2.Config{
// 		ClientID:    fmt.Sprintf("https://%s/", r.Host),
// 		RedirectURL: fmt.Sprintf("https://%s/_smallweb/auth/callback", r.Host),
// 		Scopes:      []string{"email"},
// 		Endpoint: oauth2.Endpoint{
// 			AuthURL:   "https://lastlogin.io/auth",
// 			TokenURL:  "https://lastlogin.io/token",
// 			AuthStyle: oauth2.AuthStyleInParams,
// 		},
// 	}

// 	if r.URL.Path == "/_smallweb/auth/callback" {
// 		state, err := r.Cookie("oauth_state")
// 		if err != nil {
// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		}

// 		// delete the cookie
// 		http.SetCookie(w, &http.Cookie{
// 			Name:     "oauth_state",
// 			Value:    "",
// 			HttpOnly: true,
// 			SameSite: http.SameSiteLaxMode,
// 			Secure:   true,
// 			Expires:  time.Now(),
// 		})

// 		if r.FormValue("state") != state.Value {
// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 			return
// 		}

// 		code := r.FormValue("code")
// 		token, err := oauthConfig.Exchange(r.Context(), code)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}

// 		req, err := http.NewRequest(http.MethodGet, "https://lastlogin.io/userinfo", nil)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}

// 		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
// 		resp, err := http.DefaultClient.Do(req)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}
// 		defer resp.Body.Close()

// 		var userinfo struct {
// 			Email string `json:"email"`
// 		}
// 		if err := json.NewDecoder(resp.Body).Decode(&userinfo); err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}

// 		sessionID, err := me.db.CreateSession(userinfo.Email, r.Host)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}

// 		http.SetCookie(w, &http.Cookie{
// 			Name:     "smallweb_session",
// 			Value:    sessionID,
// 			HttpOnly: true,
// 			Path:     "/",
// 			SameSite: http.SameSiteLaxMode,
// 			Secure:   true,
// 			MaxAge:   60 * 60 * 24 * 30,
// 		})

// 		http.Redirect(w, r, "https://"+r.Host, http.StatusTemporaryRedirect)
// 		return
// 	}

// 	if r.URL.Path == "/_smallweb/auth/logout" {
// 		http.SetCookie(w, &http.Cookie{
// 			Name:     "smallweb_session",
// 			Value:    "",
// 			Path:     "/",
// 			HttpOnly: true,
// 			SameSite: http.SameSiteLaxMode,
// 			Secure:   true,
// 			Expires:  time.Now(),
// 		})
// 	}

// 	sessionID, err := r.Cookie("smallweb_session")
// 	if err != nil {
// 		state := uuid.New().String()
// 		url := oauthConfig.AuthCodeURL(state)
// 		cookie := &http.Cookie{
// 			Name:     "oauth_state",
// 			Value:    state,
// 			Path:     "/",
// 			HttpOnly: true,
// 			SameSite: http.SameSiteLaxMode,
// 			Secure:   true,
// 		}
// 		http.SetCookie(w, cookie)
// 		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
// 		return
// 	}

// 	session, err := me.db.GetSession(sessionID.Value)
// 	if err != nil {
// 		state := uuid.New().String()
// 		url := oauthConfig.AuthCodeURL(state)
// 		cookie := &http.Cookie{
// 			Name:     "oauth_state",
// 			Value:    state,
// 			Path:     "/",
// 			HttpOnly: true,
// 			SameSite: http.SameSiteLaxMode,
// 			Secure:   true,
// 		}
// 		http.SetCookie(w, cookie)
// 		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
// 		return
// 	}

// 	if session.Host != r.Host {
// 		http.Error(w, "session not valid for this host", http.StatusUnauthorized)
// 		return
// 	}

// 	ctx := context.WithValue(r.Context(), ContextKeyEmail, session.Email)
// 	me.next.ServeHTTP(w, r.WithContext(ctx))
// }
