package asana

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type Asana struct {
	Client *http.Client
}

func (a *Asana) Connect(pat string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tok := &oauth2.Token{AccessToken: pat}
	conf := &oauth2.Config{}
	a.Client = conf.Client(ctx, tok)
}
