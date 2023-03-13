package main

import (
	"context"

	"github.com/disgoorg/disgolink/v2/lavalink"
	"github.com/disgoorg/log"
	"github.com/disgoorg/snowflake/v2"
)

type GuildPlayer struct {
	channelID *snowflake.ID
	messageID *snowflake.ID
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
			Tracks: make([]lavalink.Track, 0),
			Type:   QueueTypeNoRepeat,
		}
		dbGuild, err := gm.bot.EntClient.Guild.Get(context.TODO(), guildID)
		if err != nil {
			log.Error(err)
		}
		guildPlayer := &GuildPlayer{
			channelID: dbGuild.PlayerChannelID,
			messageID: dbGuild.PlayerMessageID,
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

func (gp *GuildPlayer) IsPlayerChannel(channelID snowflake.ID) bool {
	return gp.channelID != nil && gp.channelID.String() == channelID.String()
}

func (gp *GuildPlayer) IsPlayerMessage(messageID snowflake.ID) bool {
	return gp.messageID != nil && gp.messageID.String() == messageID.String()
}
