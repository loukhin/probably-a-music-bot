package main

import (
	"context"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
)

func (b *Bot) onApplicationCommand(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()

	_ = event.DeferCreateMessage(false)

	handler, ok := b.Handlers[data.CommandName()]
	if !ok {
		log.Info("unknown command: ", data.CommandName())
		return
	}
	if err := handler(event, data); err != nil {
		log.Error("error handling command: ", err)
	}
}

func (b *Bot) onVoiceStateUpdate(event *events.GuildVoiceStateUpdate) {
	if event.VoiceState.UserID != b.Client.ApplicationID() {
		return
	}
	b.Lavalink.OnVoiceStateUpdate(context.TODO(), event.VoiceState.GuildID, event.VoiceState.ChannelID, event.VoiceState.SessionID)
	if event.VoiceState.ChannelID == nil {
		b.Guilds.Delete(event.VoiceState.GuildID)
	}
}

func (b *Bot) onVoiceServerUpdate(event *events.VoiceServerUpdate) {
	b.Lavalink.OnVoiceServerUpdate(context.TODO(), event.GuildID, event.Token, *event.Endpoint)
}

func (b *Bot) onGuildJoin(event *events.GuildJoin) {
	err := b.EntClient.Guild.Create().SetID(event.GuildID).SetName(event.Guild.Name).OnConflictColumns("id").UpdateNewValues().Exec(context.Background())
	if err != nil {
		log.Error(err)
	}
}

func (b *Bot) onGuildMessageCreate(event *events.GuildMessageCreate) {
	guildPlayer := b.Guilds.GetGuildPlayer(event.GuildID)
	if guildPlayer.channelId.String() == event.ChannelID.String() {
		go func() {
			time.Sleep(5 * time.Second)
			_ = event.Client().Rest().DeleteMessage(event.ChannelID, event.MessageID)
		}()
		if event.Message.Author.Bot {
			return
		}
		if guildPlayer.messageId != nil {
			b.playOrQueue(event.GuildID, *event.Message.Member, event.Message.Content, func(embed discord.Embed) {
				messageCreate := discord.NewMessageCreateBuilder()
				messageCreate.SetMessageReference(event.Message.MessageReference)
				messageCreate.SetEmbeds(embed)
				_, err := event.Client().Rest().CreateMessage(event.ChannelID, messageCreate.Build())
				if err != nil {
					log.Error(err)
				}
				b.updatePlayerMessage(event.GuildID)
			})
		}
	}
}
