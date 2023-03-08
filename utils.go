package main

import (
	"context"
	"fmt"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/log"
	"github.com/disgoorg/snowflake/v2"

	"github.com/disgoorg/disgolink/lavalink"
)

func fmtDuration(duration lavalink.Duration) string {
	return fmt.Sprintf("%02d:%02d:%02d", duration.Hours(), duration.MinutesPart(), duration.SecondsPart())
}

func connectVoiceChannel(guildID snowflake.ID, channelID *snowflake.ID) bool {
	if err := client.UpdateVoiceState(context.TODO(), guildID, channelID, false, true); err != nil {
		log.Errorf("error while connecting to channel: %s", err)
		return false
	}
	return true
}

func playOrQueue(guildID snowflake.ID, user discord.Member, query string, responseFunc func(embed discord.Embed)) {
	var embed discord.EmbedBuilder
	embed.SetColor(16705372)
	voiceState, ok := client.Caches().VoiceState(guildID, user.User.ID)
	if !ok || voiceState.ChannelID == nil {
		embed.SetDescription("Please join a VoiceChannel to use this command")
		responseFunc(embed.Build())
		return
	}
	botVoiceState, ok := client.Caches().VoiceState(guildID, client.ID())
	if ok && botVoiceState.ChannelID != nil && botVoiceState.ChannelID.String() != voiceState.ChannelID.String() {
		embed.SetDescription("Bot was already in other channel")
		responseFunc(embed.Build())
		return
	}
	go func() {
		if !URLPattern.MatchString(query) {
			query = lavalink.SearchTypeYoutube.Apply(query)
		}

		musicPlayer, ok := musicPlayers[guildID]
		if !ok {
			musicPlayer = NewMusicPlayer(client, guildID)
			_ = musicPlayer.SetVolume(30)
			musicPlayers[guildID] = musicPlayer
		}

		_ = musicPlayer.Node().RestClient().LoadItemHandler(context.TODO(), query, lavalink.NewResultHandler(
			func(track lavalink.AudioTrack) {
				if ok = connectVoiceChannel(guildID, voiceState.ChannelID); !ok {
					return
				}
				if musicPlayer.PlayingTrack() == nil {
					message := fmt.Sprintf("▶ Playing [%s](%s) `%s`", track.Info().Title, *track.Info().URI, fmtDuration(track.Info().Length))
					embed.SetDescription(message)
				} else {
					message := fmt.Sprintf("Queued [%s](%s) `%s`", track.Info().Title, *track.Info().URI, fmtDuration(track.Info().Length))
					embed.SetDescription(message)
				}
				musicPlayer.Queue(user, track)
				responseFunc(embed.Build())
			},
			func(playlist lavalink.AudioPlaylist) {
				if ok = connectVoiceChannel(guildID, voiceState.ChannelID); !ok {
					return
				}
				var playlistLength lavalink.Duration
				for _, track := range playlist.Tracks() {
					playlistLength += track.Info().Length
				}
				if musicPlayer.PlayingTrack() == nil {
					message := fmt.Sprintf("▶ Playing %d tracks from [%s](%s) playlist `%s`", len(playlist.Tracks()), playlist.Name(), query, fmtDuration(playlistLength))
					embed.SetDescription(message)
				} else {
					message := fmt.Sprintf("Queued %d tracks from [%s](%s) playlist `%s`", len(playlist.Tracks()), playlist.Name(), query, fmtDuration(playlistLength))
					embed.SetDescription(message)
				}
				musicPlayer.Queue(user, playlist.Tracks()...)
				responseFunc(embed.Build())
			},
			func(tracks []lavalink.AudioTrack) {
				if ok = connectVoiceChannel(guildID, voiceState.ChannelID); !ok {
					return
				}
				track := tracks[0]
				if musicPlayer.PlayingTrack() == nil {
					message := fmt.Sprintf("▶ Playing [%s](%s) `%s`", track.Info().Title, *track.Info().URI, fmtDuration(track.Info().Length))
					embed.SetDescription(message)
				} else {
					message := fmt.Sprintf("Queued [%s](%s) `%s`", track.Info().Title, *track.Info().URI, fmtDuration(track.Info().Length))
					embed.SetDescription(message)
				}
				musicPlayer.Queue(user, track)
				responseFunc(embed.Build())
			},
			func() {
				embed.SetDescription("No tracks found")
				responseFunc(embed.Build())
			},
			func(e lavalink.FriendlyException) {
				embed.SetDescription("error while loading track:\n" + e.Error())
				responseFunc(embed.Build())
			},
		))
	}()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
