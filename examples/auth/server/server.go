package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"github.com/satori/go.uuid"
	"gopkg.in/beatgammit/turnpike.v2"
)

// this is just an example, please don't actually use it
type exampleAuth struct {
	password string
}

func (e *exampleAuth) Challenge(details map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"challenge": uuid.NewV4().String()}, nil
}

func (e *exampleAuth) Authenticate(c map[string]interface{}, signature string) (map[string]interface{}, error) {
	// we assume this will work because turnpike gives us the same data the Challenge method returned
	challenge := c["challenge"].(string)
	mac := hmac.New(sha256.New, []byte(e.password))
	mac.Write([]byte(challenge))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	log.Info("challenge:", challenge)
	log.Info("expected:", expected)
	log.Info("given:", signature)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return nil, fmt.Errorf("Invalid password")
	}
	return nil, nil
}

func main() {
	turnpike.Debug()
	s, err := turnpike.NewWebsocketServer(map[string]turnpike.Realm{
		"turnpike.examples": {
			CRAuthenticators: map[string]turnpike.CRAuthenticator{
				"example-auth": &exampleAuth{password: "password"},
			},
		},
	})
	if err != nil {
		panic("error creating websocket server: " + err.Error())
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Connection from", r.RemoteAddr)
		s.ServeHTTP(w, r)
	})
	log.Info("turnpike server starting on port 8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
