package main

import (
	"context"

	"github.com/disgoorg/disgolink/v2/lavalink"
	"github.com/disgoorg/log"
	"github.com/disgoorg/snowflake/v2"
)

type GuildPlayer struct {
	channelId *snowflake.ID
	messageId *snowflake.ID
}

type Guild struct {
	guildPlayer *GuildPlayer
	queue       *Queue
}

type GuildManager struct {
	bot    *Bot
	guilds map[snowflake.ID]*Guild
}

func (gm *GuildManager) Get(guildID snowflake.ID) *Guild {
	guild, ok := gm.guilds[guildID]
	if !ok {
		queue := &Queue{
			bot:     gm.bot,
			guildId: GuildId,
			Tracks:  make([]lavalink.Track, 0),
			Type:    QueueTypeNoRepeat,
		}
		entClient := getEntClient()
		dbGuild, err := entClient.Guild.Get(context.TODO(), guildID)
		if err != nil {
			log.Error(err)
		}
		guildPlayer := &GuildPlayer{
			channelId: dbGuild.PlayerChannelID,
			messageId: dbGuild.PlayerMessageID,
		}
		gm.guilds[guildID] = &Guild{
			queue:       queue,
			guildPlayer: guildPlayer,
		}
		guild, _ = gm.guilds[guildID]
	}
	return guild
}

func (gm *GuildManager) GetQueue(guildID snowflake.ID) *Queue {
	guild := gm.Get(guildID)
	return guild.queue
}

func (gm *GuildManager) GetGuildPlayer(guildID snowflake.ID) *GuildPlayer {
	guild := gm.Get(guildID)
	return guild.guildPlayer
}

func (gm *GuildManager) Delete(guildID snowflake.ID) {
	delete(gm.guilds, guildID)
}
