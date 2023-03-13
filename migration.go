package main

import (
	"context"

	"github.com/loukhin/probably-a-music-bot/ent"
	"github.com/loukhin/probably-a-music-bot/ent/migrate"
)

func migrateDatabase(client *ent.Client) error {
	ctx := context.Background()
	err := client.Schema.Create(
		ctx,
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	)
	return err
}
