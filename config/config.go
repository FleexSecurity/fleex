package config

import (
	"context"
	"net/http"

	"github.com/digitalocean/godo"
	"github.com/linode/linodego"
	"github.com/vultr/govultr/v2"
	"golang.org/x/oauth2"
)

func GetLinodeClient(token string) linodego.Client {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	linodeClient := linodego.NewClient(oauth2Client)
	linodeClient.SetDebug(false)

	return linodeClient
}

func GetVultrClient(token string) *govultr.Client {
	config := &oauth2.Config{}
	ctx := context.Background()
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: token})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, ts))
	return vultrClient
}

func GetDigitaloaceanToken(token string) *godo.Client {
	return godo.NewFromToken(token)
}
