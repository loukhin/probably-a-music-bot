package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/loukhin/probably-a-music-bot/ent"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/log"
	"github.com/disgoorg/snowflake/v2"
)

func newBot() *Bot {
	b := new(Bot)
	b.Guilds = &GuildManager{
		bot:    b,
		guilds: make(map[snowflake.ID]*Guild),
	}
	return b
}

type Bot struct {
	Client    bot.Client
	EntClient *ent.Client
	Guilds    *GuildManager
	Handlers  map[string]func(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) error
	Lavalink  disgolink.Client
}

func (b *Bot) updateVoiceState(guildID snowflake.ID, channelID *snowflake.ID) bool {
	if err := b.Client.UpdateVoiceState(context.TODO(), guildID, channelID, false, true); err != nil {
		log.Errorf("error while connecting to channel: %s", err)
		return false
	}
	return true
}

func (b *Bot) updatePlayerMessage(guildID snowflake.ID) {
	guildPlayer := b.Guilds.GetGuildPlayer(guildID)
	if guildPlayer.channelID == nil || guildPlayer.messageID == nil {
		return
	}
	playerMessage, err := b.Client.Rest().GetMessage(*guildPlayer.channelID, *guildPlayer.messageID)
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
	queue := b.Guilds.GetQueue(guildID)
	queueLength := len(queue.Tracks)
	queueEmbed.SetTitlef("Queue list (%d)", queueLength)
	if queueLength == 0 {
		description = "The queue is empty"
	} else {
		queueDuration.Name = "Queue duration:"
		queueDuration.Value = formatDuration(queue.Length)
		queueDuration.Inline = &isInline
		queueEmbed.SetFields(queueDuration)
	}
	for i := 0; i < min(10, queueLength); i++ {
		track := queue.Tracks[i]
		description += fmt.Sprintf("%d. [%s](%s) `%s`\n", i+1, track.Info.Title, *track.Info.URI, formatDuration(track.Info.Length))
	}
	if queueLength > 10 {
		description += fmt.Sprintf("**and other %d tracks...**\n", queueLength-10)
	}
	queueEmbed.SetDescription(description)

	player := b.Lavalink.Player(guildID)
	playingTrack := player.Track()
	if playingTrack != nil {
		playStatus := "▶️"
		if player.Paused() {
			playStatus = "⏸️"
		}
		playerEmbed.SetTitlef("%s %s", playStatus, playingTrack.Info.Title)
		playerEmbed.SetURL(*playingTrack.Info.URI)
		if playingTrack.Info.ArtworkURL != nil {
			playerEmbed.SetImage(*playingTrack.Info.ArtworkURL)
		} else {
			playerEmbed.SetImage("https://images.pexels.com/videos/3045163/free-video-3045163.jpg?auto=compress&cs=tinysrgb&dpr=1")
		}
	} else {
		playerEmbed.SetTitle("Nothing currently playing")
		playerEmbed.SetImage("https://images.pexels.com/videos/3045163/free-video-3045163.jpg?auto=compress&cs=tinysrgb&dpr=1")
	}

	loopStatus.Text = fmt.Sprintf("Mode: %s", queue.Type)
	playerEmbed.SetEmbedFooter(&loopStatus)

	messageUpdate.SetContent("Join a voice channel and queue songs by name or url in here.")
	messageUpdate.SetEmbeds(playerEmbed.Build(), queueEmbed.Build())

	_, err = b.Client.Rest().UpdateMessage(playerMessage.ChannelID, playerMessage.ID, messageUpdate.Build())
	if err != nil {
		log.Error(err)
	}
}

func (b *Bot) playOrQueue(guildID snowflake.ID, user discord.Member, query string, responseFunc func(embed discord.Embed)) {
	var embed discord.EmbedBuilder
	embed.SetColor(16705372)
	voiceState, ok := b.Client.Caches().VoiceState(guildID, user.User.ID)
	if !ok || voiceState.ChannelID == nil {
		embed.SetDescription("Please join a VoiceChannel to use this command")
		responseFunc(embed.Build())
		return
	}
	botVoiceState, ok := b.Client.Caches().VoiceState(guildID, b.Client.ID())
	if ok && botVoiceState.ChannelID != nil && botVoiceState.ChannelID.String() != voiceState.ChannelID.String() {
		embed.SetDescription("Bot was already in other channel")
		responseFunc(embed.Build())
		return
	}

	if !urlPattern.MatchString(query) {
		query = lavalink.SearchTypeYouTube.Apply(query)
	}

	queue := b.Guilds.GetQueue(guildID)
	player := b.Lavalink.Player(guildID)
	_ = player.Update(context.TODO(), lavalink.WithVolume(40))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	loadResult, err := b.Lavalink.BestNode().LoadTracks(ctx, query)
	if err != nil {
		log.Error(err)
	}
	tracks := loadResult.Data

	switch loadResult.LoadType {
	case lavalink.LoadTypeTrack, lavalink.LoadTypeSearch:
		var track lavalink.Track
		if searchResult, ok := tracks.(lavalink.Search); ok {
			track = searchResult[0]
		} else {
			track = tracks.(lavalink.Track)
		}
		if player.Track() == nil {
			message := fmt.Sprintf("▶ Playing [%s](%s) `%s`", track.Info.Title, *track.Info.URI, formatDuration(track.Info.Length))
			embed.SetDescription(message)
		} else {
			message := fmt.Sprintf("Queued [%s](%s) `%s`", track.Info.Title, *track.Info.URI, formatDuration(track.Info.Length))
			embed.SetDescription(message)
		}
		queue.Add(track)
	case lavalink.LoadTypePlaylist:
		var playlistLength lavalink.Duration
		playlists := tracks.(lavalink.Playlist)
		tracks := playlists.Tracks
		for _, track := range tracks {
			playlistLength += track.Info.Length
		}
		if player.Track() == nil {
			message := fmt.Sprintf("▶ Playing %d tracks from [%s](%s) playlist `%s`", len(tracks), playlists.Info.Name, query, formatDuration(playlistLength))
			embed.SetDescription(message)
		} else {
			message := fmt.Sprintf("Queued %d tracks from [%s](%s) playlist `%s`", len(tracks), playlists.Info.Name, query, formatDuration(playlistLength))
			embed.SetDescription(message)
		}
		queue.Add(tracks...)
	case lavalink.LoadTypeEmpty:
		embed.SetDescription("No tracks found")
		responseFunc(embed.Build())
		return
	case lavalink.LoadTypeError:
		embed.SetDescription("error while loading track:\n" + err.Error())
		responseFunc(embed.Build())
		return
	}

	if player.Track() == nil {
		if track, ok := queue.Next(); ok {
			if ok := b.updateVoiceState(guildID, voiceState.ChannelID); !ok {
				log.Info("not ok")
				return
			}
			err := player.Update(context.TODO(), lavalink.WithTrack(track))
			if err != nil {
				log.Error(err)
			}
		}
	}
	responseFunc(embed.Build())
}

func (b *Bot) textToSpeech(guildID snowflake.ID, user discord.Member, text string, bitsAmount int, responseFunc func(embed discord.Embed)) {
	var embed discord.EmbedBuilder
	embed.SetColor(16705372)
	voiceState, ok := b.Client.Caches().VoiceState(guildID, user.User.ID)
	if !ok || voiceState.ChannelID == nil {
		embed.SetDescription("Please join a VoiceChannel to use this command")
		responseFunc(embed.Build())
		return
	}
	botVoiceState, ok := b.Client.Caches().VoiceState(guildID, b.Client.ID())
	if ok && botVoiceState.ChannelID != nil && botVoiceState.ChannelID.String() != voiceState.ChannelID.String() {
		embed.SetDescription("Bot was already in other channel")
		responseFunc(embed.Build())
		return
	}

	queue := b.Guilds.GetQueue(guildID)
	player := b.Lavalink.Player(guildID)
	_ = player.Update(context.TODO(), lavalink.WithVolume(100))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if bitsAmount != 0 {
		text = fmt.Sprintf("%d bits / / / / %s", bitsAmount, text)
		loadBitsResult, err := b.Lavalink.BestNode().LoadTracks(ctx, "https://files.loukhin.com/bits.ogg")
		if err != nil {
			log.Error(err)
		}
		bitsTracks := loadBitsResult.Data.(lavalink.Track)
		queue.Add(bitsTracks)
	}

	type Input struct {
		Text string `json:"text"`
	}

	type Voice struct {
		LanguageCode string `json:"languageCode"`
		Name         string `json:"name"`
		SsmlGender   string `json:"ssmlGender"`
	}

	type AudioConfig struct {
		AudioEncoding string  `json:"audioEncoding"`
		SpeakingRate  float64 `json:"speakingRate"`
	}

	type Config struct {
		Input       Input       `json:"input"`
		Voice       Voice       `json:"voice"`
		AudioConfig AudioConfig `json:"audioConfig"`
	}

	jsonStr, _ := json.Marshal(&Config{
		Input: Input{
			Text: fmt.Sprintf("%s / / / /", text), // add padding to prevent cutoff?
		},
		Voice: Voice{
			LanguageCode: "th-TH",
			Name:         "th-TH-Neural2-C",
		},
		AudioConfig: AudioConfig{
			AudioEncoding: "OGG_OPUS",
			SpeakingRate:  0.8,
		},
	})

	query := url.Values{}
	query.Add("config", string(jsonStr))

	loadResult, err := b.Lavalink.BestNode().LoadTracks(ctx, fmt.Sprintf("tts://?%s", query.Encode()))
	if err != nil {
		log.Error(err)
		embed.SetDescription("Text too long?")
		responseFunc(embed.Build())
		return
	}
	track := loadResult.Data.(lavalink.Track)

	queue.Add(track)

	if player.Track() == nil {
		if track, ok := queue.Next(); ok {
			if ok := b.updateVoiceState(guildID, voiceState.ChannelID); !ok {
				log.Info("not ok")
				return
			}
			err := player.Update(context.TODO(), lavalink.WithTrack(track))
			if err != nil {
				log.Error(err)
			}
		}
	}
	embed.SetDescription(text)
	responseFunc(embed.Build())
}

func (b *Bot) createPlayerMessage(guildID snowflake.ID, channelID snowflake.ID) bool {
	guild, err := b.EntClient.Guild.Get(context.TODO(), guildID)
	if err != nil {
		log.Error(err)
		return false
	}

	if guild.PlayerChannelID == nil || *guild.PlayerChannelID != channelID {
		var message *discord.Message
		guildPlayer := b.Guilds.GetGuildPlayer(guildID)
		message, err = b.Client.Rest().CreateMessage(channelID, discord.NewMessageCreateBuilder().SetContent("Join a voice channel and queue songs by name or url in here.").Build())
		if err != nil {
			log.Error(err)
		}
		guild, err = guild.Update().SetPlayerChannelID(channelID).SetPlayerMessageID(message.ID).Save(context.TODO())
		if err != nil {
			log.Error(err)
		}
		guildPlayer.channelID = guild.PlayerChannelID
		guildPlayer.messageID = guild.PlayerMessageID
		b.updatePlayerMessage(guildID)
		return true
	}
	return false
}

func formatDuration(duration lavalink.Duration) string {
	return fmt.Sprintf("%02d:%02d:%02d", duration.Hours(), duration.MinutesPart(), duration.SecondsPart())
}

func updateInteractionResponse(event *events.ApplicationCommandInteractionCreate, text string) error {
	var embed discord.EmbedBuilder
	embed.SetDescription(text)
	_, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetEmbeds(embed.Build()).Build())
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
