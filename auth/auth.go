package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mssola/user_agent"
	"github.com/pomdtr/smallweb/utils"
	"golang.org/x/oauth2"
)

func isBrowser(ua *user_agent.UserAgent) bool {
	if ua.Bot() {
		return false
	}

	// Check for common browser engines
	engine, _ := ua.Engine()
	engineKeywords := []string{"Gecko", "AppleWebKit", "Blink"}
	for _, keyword := range engineKeywords {
		if strings.Contains(strings.ToLower(engine), strings.ToLower(keyword)) {
			return true
		}
	}

	// Additional checks for browser-specific strings
	browser, _ := ua.Browser()
	browserKeywords := []string{"Mozilla", "Chrome", "Safari", "Firefox", "Edge", "Opera"}
	for _, keyword := range browserKeywords {
		if strings.Contains(strings.ToLower(browser), strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

func Middleware(provider string, email string, appname string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		sessionCookieName := "smallweb-session"
		oauthCookieName := "smallweb-oauth-store"
		type oauthStore struct {
			State    string `json:"state"`
			Redirect string `json:"redirect"`
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, _, ok := r.BasicAuth()
			if ok {
				if err := VerifyToken(username, appname); err != nil {
					w.Header().Add("WWW-Authenticate", `Basic realm="smallweb"`)
					// here we return unauthorized instead of forbidden to trigger the basic auth prompt
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			authorization := r.Header.Get("Authorization")
			if strings.HasPrefix(authorization, "Bearer ") {
				token := strings.TrimPrefix(authorization, "Bearer ")
				if err := VerifyToken(token, appname); err != nil {
					w.Header().Add("WWW-Authenticate", `Basic realm="smallweb"`)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			if ua := user_agent.New(r.UserAgent()); !isBrowser(ua) {
				w.Header().Add("WWW-Authenticate", `Basic realm="smallweb"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if email == "" {
				w.Header().Add("WWW-Authenticate", `Basic realm="smallweb"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			authUrl := "https://lastlogin.net/auth"
			if provider != "" {
				authUrl += "?provider=" + provider
			}

			oauth2Config := oauth2.Config{
				ClientID: fmt.Sprintf("https://%s/", r.Host),
				Endpoint: oauth2.Endpoint{
					AuthURL:   authUrl,
					TokenURL:  "https://lastlogin.net/token",
					AuthStyle: oauth2.AuthStyleInParams,
				},
				Scopes:      []string{"email"},
				RedirectURL: fmt.Sprintf("https://%s/_auth/callback", r.Host),
			}

			if r.URL.Path == "/_auth/login" {
				query := r.URL.Query()
				state, err := utils.GenerateBase62String(16)
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				store := oauthStore{
					State:    state,
					Redirect: query.Get("redirect"),
				}

				value, err := json.Marshal(store)
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}

				http.SetCookie(w, &http.Cookie{
					Name:     oauthCookieName,
					Value:    url.QueryEscape(string(value)),
					Expires:  time.Now().Add(5 * time.Minute),
					Path:     "/",
					SameSite: http.SameSiteLaxMode,
					HttpOnly: true,
					Secure:   true,
				})

				url := oauth2Config.AuthCodeURL(state)
				http.Redirect(w, r, url, http.StatusSeeOther)
				return
			}

			if r.URL.Path == "/_auth/callback" {
				query := r.URL.Query()
				oauthCookie, err := r.Cookie(oauthCookieName)
				if err != nil {
					log.Printf("failed to get oauth cookie: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				var oauthStore oauthStore
				value, err := url.QueryUnescape(oauthCookie.Value)
				if err != nil {
					log.Printf("failed to unescape oauth cookie: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				if err := json.Unmarshal([]byte(value), &oauthStore); err != nil {
					log.Printf("failed to unmarshal oauth cookie: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				if query.Get("state") != oauthStore.State {
					log.Printf("state mismatch: %s != %s", query.Get("state"), oauthStore.State)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				code := query.Get("code")
				if code == "" {
					log.Printf("code not found")
					http.Error(w, "Bad Request", http.StatusBadRequest)
					return
				}

				token, err := oauth2Config.Exchange(r.Context(), code)
				if err != nil {
					log.Printf("failed to exchange code: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				req, err := http.NewRequest("GET", "https://lastlogin.net/userinfo", nil)
				if err != nil {
					log.Printf("failed to create userinfo request: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Printf("failed to execute userinfo request: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					log.Printf("userinfo request failed: %s", resp.Status)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				var userinfo struct {
					Email string `json:"email"`
				}

				if err := json.NewDecoder(resp.Body).Decode(&userinfo); err != nil {
					log.Printf("failed to decode userinfo: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				sessionID, err := CreateSession(userinfo.Email, r.Host)
				if err != nil {
					log.Printf("failed to create session: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				// delete oauth cookie
				http.SetCookie(w, &http.Cookie{
					Name:     oauthCookieName,
					Expires:  time.Now().Add(-1 * time.Hour),
					Path:     "/",
					SameSite: http.SameSiteLaxMode,
					HttpOnly: true,
					Secure:   true,
				})

				// set session cookie
				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookieName,
					Value:    sessionID,
					Expires:  time.Now().Add(14 * 24 * time.Hour),
					SameSite: http.SameSiteLaxMode,
					HttpOnly: true,
					Secure:   true,
					Path:     "/",
				})

				http.Redirect(w, r, oauthStore.Redirect, http.StatusSeeOther)
				return
			}

			if r.URL.Path == "/_auth/logout" {
				cookie, err := r.Cookie(sessionCookieName)
				if err != nil {
					log.Printf("failed to get session cookie: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				if err := DeleteSession(cookie.Value); err != nil {
					log.Printf("failed to delete session: %v", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookieName,
					Expires:  time.Now().Add(-1 * time.Hour),
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteLaxMode,
					Path:     "/",
				})

				redirect := r.URL.Query().Get("redirect")
				if redirect == "" {
					redirect = fmt.Sprintf("https://%s/", r.Host)
				}

				http.Redirect(w, r, redirect, http.StatusSeeOther)
				return
			}

			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				http.Redirect(w, r, fmt.Sprintf("/_auth/login?redirect=%s", r.URL.Path), http.StatusSeeOther)
				return
			}

			session, err := GetSession(cookie.Value)
			if session.Domain != r.Host || session.Email != email {
				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookieName,
					Expires:  time.Now().Add(-1 * time.Hour),
					SameSite: http.SameSiteLaxMode,
					HttpOnly: true,
					Secure:   true,
				})

				if err := DeleteSession(cookie.Value); err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				http.Redirect(w, r, fmt.Sprintf("/_auth/login?redirect=%s", r.URL.Path), http.StatusSeeOther)
				return
			}

			if err != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookieName,
					Expires:  time.Now().Add(-1 * time.Hour),
					SameSite: http.SameSiteLaxMode,
					HttpOnly: true,
					Secure:   true,
				})

				http.Redirect(w, r, fmt.Sprintf("/_auth/login?redirect=%s", r.URL.Path), http.StatusSeeOther)
				return
			}

			if time.Now().After(session.ExpiresAt) {
				if err := DeleteSession(cookie.Value); err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookieName,
					Expires:  time.Now().Add(-1 * time.Hour),
					SameSite: http.SameSiteLaxMode,
					HttpOnly: true,
					Secure:   true,
				})

				http.Redirect(w, r, fmt.Sprintf("/_auth/login?redirect=%s", r.URL.Path), http.StatusSeeOther)
				return
			}

			if session.Email != email {
				log.Printf("email mismatch: %s != %s", session.Email, email)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// if session is near expiration, extend it
			if time.Now().Add(7 * 24 * time.Hour).After(session.ExpiresAt) {
				if err := ExtendSession(cookie.Value, time.Now().Add(14*24*time.Hour)); err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				return
			}

			next.ServeHTTP(w, r)
		})

	}
}
