package main

import (
	"os"

	"github.com/disgoorg/log"
	"github.com/loukhin/probably-a-music-bot/ent"
)

type EntClient struct {
	client *ent.Client
}

func getEntClient() *ent.Client {
	var e EntClient
	if e.client == nil {
		entClient, err := ent.Open("postgres", os.Getenv("DATABASE_URL"))
		if err != nil {
			log.Fatalf("Can't initialize ent: %s", err)
		}
		err = entClient.Ping()
		if err != nil {
			log.Fatalf("Can't initialize database connection: %s", err)
			return nil
		}
		e.client = entClient
	}
	return e.client
}
