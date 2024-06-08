package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pomdtr/smallweb/server/storage"
)

type SubdomainHandler struct {
	db        *storage.DB
	forwarder *Forwarder
}

var upgrader = websocket.Upgrader{} // use default options
func (me *SubdomainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	visitorEmail := r.Context().Value(ContextKeyEmail).(string)
	subdomain := strings.Split(r.Host, ".")[0]
	parts := strings.Split(subdomain, "-")
	username := parts[len(parts)-1]
	app := strings.Join(parts[:len(parts)-1], "-")

	user, err := me.db.GetUserWithName(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if user.Email != visitorEmail {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	port, ok := me.forwarder.ports[user.Name]
	if !ok {
		http.Error(w, fmt.Sprintf("User %s not found", user.Name), http.StatusNotFound)
		return
	}

	if r.Header.Get("Upgrade") == "websocket" {
		serverConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading connection: %v", err)
			return
		}
		defer serverConn.Close()

		clientConn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://127.0.0.1:%d%s", port, r.URL.Path), http.Header{
			"X-Smallweb-App": []string{app},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer clientConn.Close()

		go func() {
			for {
				messageType, p, err := clientConn.ReadMessage()
				if err != nil {
					log.Printf("Error reading message: %v", err)
					break
				}

				if err := serverConn.WriteMessage(messageType, p); err != nil {
					log.Printf("Error writing message: %v", err)
					break
				}
			}
		}()

		for {
			messageType, p, err := serverConn.ReadMessage()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				break
			}

			if err := clientConn.WriteMessage(messageType, p); err != nil {
				log.Printf("Error writing message: %v", err)
				break
			}
		}

		return
	}

	req, err := http.NewRequest(r.Method, fmt.Sprintf("http://127.0.0.1:%d%s", port, r.URL.String()), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range r.Header {
		req.Header[k] = v
	}
	req.Header.Add("X-Smallweb-App", app)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	flusher := w.(http.Flusher)
	// Stream the response body to the client
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := w.Write(buf[:n])
			if writeErr != nil {
				http.Error(w, writeErr.Error(), http.StatusInternalServerError)
				return
			}
			flusher.Flush() // flush the buffer to the client
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

}

func NewSubdomainHandler(db *storage.DB, forwarder *Forwarder) *SubdomainHandler {
	return &SubdomainHandler{
		db:        db,
		forwarder: forwarder,
	}
}
