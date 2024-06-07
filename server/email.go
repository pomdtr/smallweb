package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Email struct {
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Cc      string `json:"cc,omitempty"`
	Bcc     string `json:"bcc,omitempty"`
	Subject string `json:"subject,omitempty"`
	Text    string `json:"text,omitempty"`
	Html    string `json:"html,omitempty"`
}

type ValTownEmail struct {
	token string
}

func NewValTownEmail(token string) *ValTownEmail {
	return &ValTownEmail{
		token: token,
	}
}

func (me *ValTownEmail) SendEmail(email Email) error {
	email.From = "pomdtr.smallweb@valtown.email"
	body, err := json.Marshal(email)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.val.town/v1/email", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", me.token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 202 {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("could not send email: %v", string(msg))
	}

	return nil
}
