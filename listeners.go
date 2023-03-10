package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"github.com/disgoorg/paginator"
)

func checkMusicPlayer(event *events.ApplicationCommandInteractionCreate) *MusicPlayer {
	musicPlayer, ok := musicPlayers[*event.GuildID()]
	if !ok {
		var embed discord.EmbedBuilder
		embed.SetDescription("No MusicPlayer found for this guild")
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEphemeral(true).SetEmbeds(embed.Build()).Build())
		return nil
	}
	return musicPlayer
}

func onApplicationCommand(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	var embed discord.EmbedBuilder
	embed.SetColor(16705372)
	switch data.CommandName() {
	case "queue":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayer == nil {
			return
		}

		if len(musicPlayer.queue) == 0 {
			embed.SetDescription("The queue is empty")
			_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())
			return
		}
		err := manager.Create(event.Respond, paginator.Pages{
			ID: event.ID().String(),
			PageFunc: func(page int, embed *discord.EmbedBuilder) {
				embed.SetTitlef("Queue ( %d )", len(musicPlayer.queue))

				description := ""
				for i := 0; i < 10; i++ {
					if page*10+i >= len(musicPlayer.queue) {
						break
					}
					queueIndex := page*10 + i
					track := musicPlayer.queue[queueIndex]
					description += fmt.Sprintf("%d. [%s](%s) - `%s` [%s]\n", queueIndex+1, track.Info().Title, *track.Info().URI, fmtDuration(track.Info().Length), track.UserData().(discord.Member).Mention())
				}
				embed.SetDescription(description)

				var loopStatus, queueDuration discord.EmbedField
				isInline := true
				loopStatus.Name = "Looping:"
				loopStatus.Value = "❌"
				loopStatus.Inline = &isInline
				if musicPlayer.isLooping {
					loopStatus.Value = "✅"
				}
				queueDuration.Name = "Queue duration:"
				queueDuration.Value = fmtDuration(musicPlayer.queueDuration)
				queueDuration.Inline = &isInline
				embed.SetFields(queueDuration, loopStatus)
			},
			Pages:      int(math.Ceil(float64(len(musicPlayer.queue)) / 10)),
			ExpireMode: paginator.ExpireModeAfterLastUsage,
		}, false)
		if err != nil {
			log.Error(err)
		}

	case "pause":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayer == nil {
			return
		}

		if musicPlayer.Paused() {
			embed.SetDescription("Track was already paused!")
			_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())
			return
		}

		embed.SetDescription("Track was already paused!")
		_ = musicPlayer.Pause(true)
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())

	case "resume":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayer == nil {
			return
		}

		if !musicPlayer.Paused() {
			embed.SetDescription("Track was not paused!")
			_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())
			return
		}

		embed.SetDescription("Resumed track")
		_ = musicPlayer.Pause(false)
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())

	case "play":
		_ = event.DeferCreateMessage(false)
		playOrQueue(*event.GuildID(), event.Member().Member, data.String("query"), func(embed discord.Embed) {
			_, _ = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetEmbeds(embed).Build())
		})

	case "volume":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayer == nil {
			return
		}

		volume := data.Int("level")
		embed.SetDescription(strings.Join([]string{"Volume set to ", strconv.Itoa(volume)}, ""))
		_ = musicPlayer.SetVolume(volume)
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())

	case "stop":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayer == nil {
			return
		}

		embed.SetDescription("Stopped music player")
		_ = musicPlayer.Stop()
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())

	case "skip":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayer == nil {
			return
		}

		description := "Track skipped"
		if len(musicPlayer.queue) > 0 {
			nextTrack := musicPlayer.queue[0]
			description += fmt.Sprintf("now playing: [%s](%s) `%s`", nextTrack.Info().Title, *nextTrack.Info().URI, fmtDuration(nextTrack.Info().Length))
			musicPlayer.PlayNextInQueue()
		} else {
			musicPlayer.isLooping = false
			_ = musicPlayer.Seek(musicPlayer.PlayingTrack().Info().Length)
		}
		embed.SetDescription(description)
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())

	case "clear":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayer == nil {
			return
		}

		embed.SetDescription("Queue cleared")
		musicPlayer.queue = nil
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())

	case "remove":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayer == nil {
			return
		}

		id := data.Int("id")
		removedTrack := musicPlayer.queue[id-1]
		musicPlayer.queue = append(musicPlayer.queue[:id-1], musicPlayer.queue[id:]...)
		embed.SetDescriptionf("Removed [%s](%s)", removedTrack.Info().Title, *removedTrack.Info().URI)
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())

	case "loop":
		musicPlayer := checkMusicPlayer(event)
		if musicPlayer == nil {
			return
		}

		musicPlayer.isLooping = !musicPlayer.isLooping
		status := "disabled"
		if musicPlayer.isLooping {
			status = "enabled"
		}
		embed.SetDescriptionf("Looping is %s", status)
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).Build())
		musicPlayer.UpdatePlayerMessage()

	case "setup":
		var (
			message *discord.Message
		)
		//_ = event.DeferCreateMessage(true)
		guild, err := entClient.Guild.Get(context.TODO(), *event.GuildID())
		if err != nil {
			log.Error(err)
		}

		if guild.PlayerChannelID == nil || *guild.PlayerChannelID != event.ChannelID() {
			embed.SetTitle("Nothing currently playing")
			message, err = client.Rest().CreateMessage(event.ChannelID(), discord.NewMessageCreateBuilder().SetEmbeds(embed.Build()).SetContent("Join a voice channel and queue songs by name or url in here.").Build())
			if err != nil {
				log.Error(err)
			}
			guild, err = guild.Update().SetPlayerChannelID(event.ChannelID()).SetPlayerMessageID(message.ID).Save(context.TODO())
			if err != nil {
				log.Error(err)
			}
		} else {
			embed.SetDescription("This channel was already a player!")
			err = event.CreateMessage(discord.NewMessageCreateBuilder().SetEphemeral(true).SetEmbeds(embed.Build()).Build())
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func onGuildVoiceStateUpdate(event *events.GuildVoiceStateUpdate) {
	//fmt.Println(event)
}

func onGuildJoin(event *events.GuildJoin) {
	err := entClient.Guild.Create().SetID(event.GuildID).SetName(event.Guild.Name).OnConflictColumns("id").UpdateNewValues().Exec(context.Background())
	if err != nil {
		log.Error(err)
	}
}

func OnGuildMessageCreate(event *events.GuildMessageCreate) {
	if event.Message.Author.Bot {
		return
	}
	guild, err := entClient.Guild.Get(context.TODO(), event.GuildID)
	if err != nil {
		log.Error(err)
	}
	if guild.PlayerChannelID.String() == event.ChannelID.String() && guild.PlayerMessageID != nil {
		playOrQueue(event.GuildID, *event.Message.Member, event.Message.Content, func(embed discord.Embed) {
			messageCreate := discord.NewMessageCreateBuilder()
			messageCreate.SetMessageReference(event.Message.MessageReference)
			messageCreate.SetEmbeds(embed)
			responseMessage, _ := event.Client().Rest().CreateMessage(event.ChannelID, messageCreate.Build())
			err = event.Client().Rest().DeleteMessage(event.ChannelID, event.MessageID)
			if err != nil {
				log.Error(err)
				return
			}
			time.Sleep(time.Second * 5)
			_ = event.Client().Rest().DeleteMessage(responseMessage.ChannelID, responseMessage.ID)
		})
	}
}
