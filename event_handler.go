package main

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"github.com/loukhin/probably-a-music-bot/ent/guild"
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
		botVoiceState, ok := b.Client.Caches().VoiceState(event.VoiceState.GuildID, b.Client.ID())
		if !ok || botVoiceState.ChannelID != event.VoiceState.ChannelID {
			return
		}
		audioChannel, ok := b.Client.Caches().GuildAudioChannel(*botVoiceState.ChannelID)
		if !ok {
			return
		}
		members := b.Client.Caches().AudioChannelMembers(audioChannel)
		if len(members) == 1 {
			b.updateVoiceState(event.VoiceState.GuildID, nil)
		}
		return
	}
	b.Lavalink.OnVoiceStateUpdate(context.TODO(), event.VoiceState.GuildID, event.VoiceState.ChannelID, event.VoiceState.SessionID)
	if event.VoiceState.ChannelID == nil {
		b.Guilds.Delete(event.VoiceState.GuildID)
		b.updatePlayerMessage(event.VoiceState.GuildID)
	}
}

func (b *Bot) onVoiceServerUpdate(event *events.VoiceServerUpdate) {
	b.Lavalink.OnVoiceServerUpdate(context.TODO(), event.GuildID, event.Token, *event.Endpoint)
}

func (b *Bot) onGuildJoin(event *events.GuildJoin) {
	err := b.EntClient.Guild.Create().SetID(event.GuildID).SetName(event.Guild.Name).
		OnConflict(
			sql.ConflictColumns("id"),
			sql.ResolveWithNewValues(),
			sql.ResolveWith(func(u *sql.UpdateSet) {
				u.SetIgnore(guild.FieldID)
				u.SetIgnore(guild.FieldCreatedAt)
			}),
		).Exec(context.Background())
	if err != nil {
		log.Error(err)
	}
}

func (b *Bot) onGuildMessageCreate(event *events.GuildMessageCreate) {
	guildPlayer := b.Guilds.GetGuildPlayer(event.GuildID)
	if guildPlayer.IsPlayerChannel(event.ChannelID) {
		if !guildPlayer.IsPlayerMessage(event.MessageID) {
			go func() {
				time.Sleep(5 * time.Second)
				_ = event.Client().Rest().DeleteMessage(event.ChannelID, event.MessageID)
			}()
		}
		if event.Message.Author.Bot {
			return
		}
		if guildPlayer.messageID != nil {
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

func (b *Bot) onGuildMessageUpdate(event *events.GuildMessageUpdate) {
	guildPlayer := b.Guilds.GetGuildPlayer(event.GuildID)
	newMessageEmbed := event.Message.Embeds
	if guildPlayer.IsPlayerChannel(event.ChannelID) && guildPlayer.IsPlayerMessage(event.MessageID) && len(newMessageEmbed) == 0 {
		guild, err := b.EntClient.Guild.Get(context.TODO(), event.GuildID)
		if err != nil {
			log.Error(err)
			return
		}
		_, err = guild.Update().ClearPlayerChannelID().ClearPlayerMessageID().Save(context.TODO())
		if err != nil {
			log.Error(err)
			return
		}
		err = event.Client().Rest().DeleteMessage(event.ChannelID, event.MessageID)
		if err != nil {
			log.Error(err)
			return
		}
		if ok := b.createPlayerMessage(event.GuildID, event.ChannelID); ok {
			b.updatePlayerMessage(event.GuildID)
		}
	}
}
