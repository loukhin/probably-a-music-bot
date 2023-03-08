package main

import (
	"context"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgolink/disgolink"
	"github.com/disgoorg/disgolink/lavalink"
	"github.com/disgoorg/log"
	"github.com/disgoorg/paginator"
	"github.com/disgoorg/snowflake/v2"
	"github.com/disgoorg/source-plugins"
	"github.com/getsentry/sentry-go"
	_ "github.com/lib/pq"

	"definitelynotmusicbot/ent"
)

var (
	URLPattern = regexp.MustCompile("^https?://[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]?")

	token        = os.Getenv("disgolink_token")
	guildID      = snowflake.GetEnv("guild_id")
	client       bot.Client
	dgoLink      disgolink.Link
	musicPlayers = map[snowflake.ID]*MusicPlayer{}
	manager      *paginator.Manager
	entClient    *ent.Client
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetLevel(log.LevelDebug)
	log.Info("starting music bot probably...")

	var err error
	err = sentry.Init(sentry.ClientOptions{
		Dsn: os.Getenv("sentry_dsn"),
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for performance monitoring.
		// We recommend adjusting this value in production,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("Can't initialize sentry: %s", err)
	}
	defer sentry.Flush(2 * time.Second)

	entClient, err = ent.Open("postgres", os.Getenv("database_url"))
	if err != nil {
		log.Fatalf("Can't initialize ent: %s", err)
	}
	defer entClient.Close()

	manager = paginator.New()
	client, err = disgo.New(token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentGuilds|gateway.IntentGuildVoiceStates|gateway.IntentGuildMessages|gateway.IntentMessageContent),
			gateway.WithPresenceOpts(gateway.WithPlayingActivity("something")),
		),
		bot.WithCacheConfigOpts(cache.WithCaches(cache.FlagVoiceStates, cache.FlagMessages, cache.FlagChannels, cache.FlagGuilds)),
		bot.WithEventListeners(&events.ListenerAdapter{
			OnApplicationCommandInteraction: onApplicationCommand,
			OnGuildVoiceStateUpdate:         onGuildVoiceStateUpdate,
			OnGuildJoin:                     onGuildJoin,
			OnGuildMessageCreate:            OnGuildMessageCreate,
		}),
		bot.WithEventListeners(manager),
	)
	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("error while building disgolink instance: %s", err)
		return
	}

	defer client.Close(context.TODO())

	dgoLink = disgolink.New(client)
	registerNodes()

	defer dgoLink.Close()

	_, err = client.Rest().SetGuildCommands(client.ApplicationID(), guildID, commands)
	if err != nil {
		log.Errorf("error while registering guild commands: %s", err)
	}

	err = client.OpenGateway(context.TODO())
	if err != nil {
		log.Fatalf("error while connecting to discord: %s", err)
	}

	log.Infof("example is now running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-s
}

func connect(event *events.ApplicationCommandInteractionCreate, voiceState discord.VoiceState) bool {
	if err := event.Client().UpdateVoiceState(context.TODO(), voiceState.GuildID, voiceState.ChannelID, false, true); err != nil {
		_, _ = event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().SetContent("error while connecting to channel:\n"+err.Error()).Build())
		log.Errorf("error while connecting to channel: %s", err)
		return false
	}
	return true
}

func registerNodes() {
	secure, _ := strconv.ParseBool(os.Getenv("lavalink_secure"))
	_, _ = dgoLink.AddNode(context.TODO(), lavalink.NodeConfig{
		Name:        "test",
		Host:        os.Getenv("lavalink_host"),
		Port:        os.Getenv("lavalink_port"),
		Password:    os.Getenv("lavalink_password"),
		Secure:      secure,
		ResumingKey: os.Getenv("lavalink_resuming_key"),
	})
	if os.Getenv("lavalink_resuming_key") != "" {
		_ = dgoLink.BestNode().ConfigureResuming(os.Getenv("lavalink_resuming_key"), 20)
	}
	dgoLink.AddPlugins(
		source_plugins.NewSpotifyPlugin(),
		source_plugins.NewAppleMusicPlugin(),
	)
}
