package asana

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

func NewClient(pat string) *http.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tok := &oauth2.Token{AccessToken: pat}
	conf := &oauth2.Config{}
	client := conf.Client(ctx, tok)
	return client
}
