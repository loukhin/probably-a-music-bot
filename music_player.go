package main

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/lavalink"
	"github.com/disgoorg/log"
	"github.com/disgoorg/snowflake/v2"
	sourcePlugins "github.com/disgoorg/source-plugins"
)

func NewMusicPlayer(client bot.Client, guildID snowflake.ID) *MusicPlayer {
	player := dgoLink.Player(guildID)
	musicPlayer := &MusicPlayer{
		Player:        player,
		client:        client,
		isLooping:     false,
		queueDuration: 0,
	}
	player.AddListener(musicPlayer)
	return musicPlayer
}

var _ lavalink.PlayerEventListener = (*MusicPlayer)(nil)

type MusicPlayer struct {
	lavalink.Player
	queue         []lavalink.AudioTrack
	client        bot.Client
	isLooping     bool
	queueDuration lavalink.Duration
}

func (p *MusicPlayer) PlayNextInQueue() {
	var track lavalink.AudioTrack
	track, p.queue = p.queue[0], p.queue[1:]
	p.queueDuration -= track.Info().Length
	_ = p.Play(track)
	p.UpdatePlayerMessage()
}

func (p *MusicPlayer) Queue(userData discord.Member, tracks ...lavalink.AudioTrack) {
	for _, track := range tracks {
		track.SetUserData(userData)
		p.queue = append(p.queue, track)
		p.queueDuration += track.Info().Length
	}

	track := tracks[0]
	if p.PlayingTrack() == nil {
		p.PlayNextInQueue()
		message := fmt.Sprintf("▶ Playing: [%s](%s) `%s`", track.Info().Title, *track.Info().URI, fmtDuration(track.Info().Length))
		if len(tracks) > 1 {
			message += fmt.Sprintf("\nand queued %d tracks", len(tracks)-1)
		}
	} else {
		p.UpdatePlayerMessage()
		message := fmt.Sprintf("Queued [%s](%s) `%s`", track.Info().Title, *track.Info().URI, fmtDuration(track.Info().Length))
		if len(tracks) > 1 {
			message += fmt.Sprintf("\nand other %d tracks", len(tracks)-1)
		}
	}
}

func (p *MusicPlayer) OnPlayerPause(player lavalink.Player) {

}

func (p *MusicPlayer) OnPlayerResume(player lavalink.Player) {

}

func (p *MusicPlayer) OnPlayerUpdate(player lavalink.Player, state lavalink.PlayerState) {

}

func (p *MusicPlayer) OnTrackStart(player lavalink.Player, track lavalink.AudioTrack) {

}

func (p *MusicPlayer) OnTrackEnd(player lavalink.Player, track lavalink.AudioTrack, endReason lavalink.AudioTrackEndReason) {
	p.UpdatePlayerMessage()
	if p.isLooping && endReason == lavalink.AudioTrackEndReasonFinished {
		_ = player.Play(track)
		return
	}
	if endReason.MayStartNext() && len(p.queue) > 0 {
		p.PlayNextInQueue()
		return
	}
	if endReason != lavalink.AudioTrackEndReasonStopped && len(p.queue) > 0 || endReason == lavalink.AudioTrackEndReasonReplaced {
		return
	}
	err := p.client.UpdateVoiceState(context.TODO(), p.GuildID(), nil, false, true)
	if err != nil {
		log.Error(err)
		return
	}
	delete(musicPlayers, p.GuildID())
	p.RemoveListener(p)
	defer dgoLink.RemovePlayer(p.GuildID())
}

func (p *MusicPlayer) OnTrackException(player lavalink.Player, track lavalink.AudioTrack, exception lavalink.FriendlyException) {
	log.Debug("track except")
}

func (p *MusicPlayer) OnTrackStuck(player lavalink.Player, track lavalink.AudioTrack, thresholdMs lavalink.Duration) {
	log.Debug("track stuck")
}

func (p *MusicPlayer) OnWebSocketClosed(player lavalink.Player, code int, reason string, byRemote bool) {
	log.Debug("ws closed")
}

func (p *MusicPlayer) UpdatePlayerMessage() {
	guild, err := entClient.Guild.Get(context.TODO(), p.GuildID())
	if err != nil {
		log.Debug(err)
		return
	}
	if guild.PlayerMessageID == nil || guild.PlayerChannelID == nil {
		return
	}
	playerMessage, err := client.Rest().GetMessage(*guild.PlayerChannelID, *guild.PlayerMessageID)
	if err != nil || playerMessage == nil {
		log.Debug(err)
		return
	}

	var (
		playerEmbed, queueEmbed discord.EmbedBuilder
		description             string
		messageUpdate           discord.MessageUpdateBuilder
		queueDuration           discord.EmbedField
		loopStatus              discord.EmbedFooter
	)
	isInline := true

	queueLength := len(p.queue)
	queueEmbed.SetTitlef("Queue list (%d)", queueLength)
	if queueLength == 0 {
		description = "The queue is empty"
	} else {
		queueDuration.Name = "Queue duration:"
		queueDuration.Value = fmtDuration(p.queueDuration)
		queueDuration.Inline = &isInline
		queueEmbed.SetFields(queueDuration)
		if queueLength > 10 {
			description = fmt.Sprintf("**and other %d tracks...**\n", queueLength-10)
		}
	}
	for i := min(9, queueLength-1); i >= 0; i-- {
		track := p.queue[i]
		description += fmt.Sprintf("%d. [%s](%s) `%s`\n", i+1, track.Info().Title, *track.Info().URI, fmtDuration(track.Info().Length))
	}
	queueEmbed.SetDescription(description)

	playingTrack := p.PlayingTrack()
	if playingTrack != nil {
		playerEmbed.SetTitle(playingTrack.Info().Title)
		playerEmbed.SetURL(*playingTrack.Info().URI)
	} else {
		playerEmbed.SetTitle("Nothing currently playing")
	}

	if spotifyTrack, ok := playingTrack.(*sourcePlugins.SpotifyAudioTrack); ok {
		playerEmbed.SetImage(*spotifyTrack.ArtworkURL)
	}
	loopStatus.Text = "Looping: ❌"
	if p.isLooping {
		loopStatus.Text = "Looping: ✅"
	}
	playerEmbed.SetEmbedFooter(&loopStatus)

	messageUpdate.SetContent("Join a voice channel and queue songs by name or url in here.")
	messageUpdate.SetEmbeds(queueEmbed.Build(), playerEmbed.Build())

	_, err = client.Rest().UpdateMessage(playerMessage.ChannelID, playerMessage.ID, messageUpdate.Build())
	if err != nil {
		log.Error(err)
	}
}
