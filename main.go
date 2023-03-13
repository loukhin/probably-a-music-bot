package main

import (
	"context"
	"github.com/loukhin/probably-a-music-bot/ent"
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
	"github.com/disgoorg/disgolink/v2/disgolink"
	"github.com/disgoorg/log"
	"github.com/disgoorg/snowflake/v2"
	"github.com/getsentry/sentry-go"
	_ "github.com/lib/pq"
)

var (
	urlPattern = regexp.MustCompile("^https?://[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]?")

	Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))

	Token   = os.Getenv("TOKEN")
	GuildId = snowflake.GetEnv("GUILD_ID")

	NodeName      = os.Getenv("NODE_NAME")
	NodeAddress   = os.Getenv("NODE_ADDRESS")
	NodePassword  = os.Getenv("NODE_PASSWORD")
	NodeSecure, _ = strconv.ParseBool(os.Getenv("NODE_SECURE"))

	SentryDsn           = os.Getenv("SENTRY_DSN")
	SentrySampleRate, _ = strconv.ParseFloat(os.Getenv("SENTRY_SAMPLE_RATE"), 64)
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetLevel(log.LevelInfo)
	if Debug {
		log.SetLevel(log.LevelDebug)
	}
	log.Info("disgo version: ", disgo.Version)
	log.Info("disgolink version: ", disgolink.Version)

	var err error

	if SentryDsn != "" {
		err = sentry.Init(sentry.ClientOptions{
			Dsn: SentryDsn,
			// Set TracesSampleRate to 1.0 to capture 100%
			// of transactions for performance monitoring.
			// We recommend adjusting this value in production,
			TracesSampleRate: SentrySampleRate,
		})
		if err != nil {
			log.Fatalf("Can't initialize sentry: %s", err)
		}
		defer sentry.Flush(2 * time.Second)
	}

	b := newBot()

	entClient, err := ent.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Can't initialize ent: %s", err)
		return
	}
	err = entClient.Ping()
	if err != nil {
		log.Fatalf("Can't initialize database connection: %s", err)
		return
	}
	if os.Args[1] == "--migrate" {
		log.Info("--migrate flag present, migrating database changes...")
		err = migrateDatabase(entClient)
		if err != nil {
			log.Fatalf("failed creating schema resources: %v", err)
			return
		}
	}
	b.EntClient = entClient

	client, err := disgo.New(Token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentGuilds|gateway.IntentGuildVoiceStates|gateway.IntentGuildMessages|gateway.IntentMessageContent),
			gateway.WithPresenceOpts(gateway.WithPlayingActivity("something")),
		),
		bot.WithCacheConfigOpts(cache.WithCaches(cache.FlagVoiceStates, cache.FlagMessages, cache.FlagChannels, cache.FlagGuilds)),
		bot.WithEventListenerFunc(b.onApplicationCommand),
		bot.WithEventListenerFunc(b.onVoiceStateUpdate),
		bot.WithEventListenerFunc(b.onVoiceServerUpdate),
		bot.WithEventListenerFunc(b.onGuildJoin),
		bot.WithEventListenerFunc(b.onGuildMessageCreate),
		//bot.WithEventListeners(manager),
	)
	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("error while building disgo instance: %s", err)
		return
	}
	b.Client = client

	registerCommands(client)

	b.Lavalink = disgolink.New(client.ApplicationID(),
		disgolink.WithListenerFunc(b.onPlayerPause),
		disgolink.WithListenerFunc(b.onPlayerResume),
		disgolink.WithListenerFunc(b.onTrackStart),
		disgolink.WithListenerFunc(b.onTrackEnd),
		disgolink.WithListenerFunc(b.onTrackException),
		disgolink.WithListenerFunc(b.onTrackStuck),
		disgolink.WithListenerFunc(b.onWebSocketClosed),
	)
	b.Handlers = map[string]func(event *events.ApplicationCommandInteractionCreate, data discord.SlashCommandInteractionData) error{
		"play":        b.play,
		"pause":       b.pause,
		"now-playing": b.nowPlaying,
		"stop":        b.stop,
		"queue":       b.queue,
		"clear-queue": b.clearQueue,
		"repeat":      b.repeatType,
		"shuffle":     b.shuffle,
		"seek":        b.seek,
		"volume":      b.volume,
		"skip":        b.skip,
		"disconnect":  b.disconnect,
		"setup":       b.setup,
		"remove":      b.removeQueue,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = client.OpenGateway(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Close(context.TODO())
	defer func(client *ent.Client) {
		_ = client.Close()
	}(b.EntClient)

	node, err := b.Lavalink.AddNode(ctx, disgolink.NodeConfig{
		Name:     NodeName,
		Address:  NodeAddress,
		Password: NodePassword,
		Secure:   NodeSecure,
	})
	if err != nil {
		log.Fatal(err)
	}
	version, err := node.Version(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("node version: %s", version)

	log.Infof("example is now running. Press CTRL and C on your keyboard together to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-s
}
