package main

import (
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
	"github.com/disgoorg/log"
)

var commands = []discord.ApplicationCommandCreate{
	discord.SlashCommandCreate{
		Name:                     "setup",
		Description:              "Setup dedicated music player channel",
		DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionAdministrator),
	},
	discord.SlashCommandCreate{
		Name:        "play",
		Description: "Queue tracks",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "query",
				Description: "Search query or links (Youtube, Spotify, etc.)",
				Required:    true,
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "pause",
		Description: "Pauses the current song",
	},
	discord.SlashCommandCreate{
		Name:        "tts",
		Description: "Make TTS voice",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "text",
				Description: "Text for TTS to speak",
				Required:    true,
				MaxLength:   json.Ptr(130),
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "bits",
		Description: "Donate fake bits to streameringzation",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "amount",
				Description: "Amount of fake bits",
				Required:    true,
				MinValue:    json.Ptr(1),
			},
			discord.ApplicationCommandOptionString{
				Name:        "text",
				Description: "Donate message",
				Required:    true,
				MaxLength:   json.Ptr(130),
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "now-playing",
		Description: "Shows the current playing song",
	},
	discord.SlashCommandCreate{
		Name:        "stop",
		Description: "Stops the current song and stops the player",
	},
	discord.SlashCommandCreate{
		Name:        "disconnect",
		Description: "Disconnects the player",
	},
	discord.SlashCommandCreate{
		Name:        "queue",
		Description: "Show queue",
	},
	discord.SlashCommandCreate{
		Name:        "clear-queue",
		Description: "Clear queue",
	},
	discord.SlashCommandCreate{
		Name:        "repeat",
		Description: "Select repeat type",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "mode",
				Description: "Repeat mode",
				Required:    true,
				Choices: []discord.ApplicationCommandOptionChoiceString{
					{
						Name:  QueueTypeNoRepeat.String(),
						Value: string(QueueTypeNoRepeat),
					},
					{
						Name:  QueueTypeRepeatTrack.String(),
						Value: string(QueueTypeRepeatTrack),
					},
					{
						Name:  QueueTypeRepeatQueue.String(),
						Value: string(QueueTypeRepeatQueue),
					},
				},
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "remove",
		Description: "Remove item from queue",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "id",
				Description: "ID of the track in queue",
				Required:    true,
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "skip",
		Description: "Skips the current song",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "amount",
				Description: "The amount of songs to skip",
				Required:    false,
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "volume",
		Description: "Sets the volume of the player",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "level",
				Description: "The volume level to set",
				Required:    true,
				MaxValue:    json.Ptr(100),
				MinValue:    json.Ptr(0),
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "seek",
		Description: "Seeks to a specific position in the current song",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "position",
				Description: "The position to seek to",
				Required:    true,
			},
			discord.ApplicationCommandOptionInt{
				Name:        "unit",
				Description: "The unit of the position",
				Required:    false,
				Choices: []discord.ApplicationCommandOptionChoiceInt{
					{
						Name:  "Milliseconds",
						Value: int(lavalink.Millisecond),
					},
					{
						Name:  "Seconds",
						Value: int(lavalink.Second),
					},
					{
						Name:  "Minutes",
						Value: int(lavalink.Minute),
					},
					{
						Name:  "Hours",
						Value: int(lavalink.Hour),
					},
				},
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "shuffle",
		Description: "Shuffles the current queue",
	},
}

func registerCommands(client bot.Client) {
	if Debug {
		if _, err := client.Rest().SetGuildCommands(client.ApplicationID(), GuildId, commands); err != nil {
			log.Warn(err)
		}
	} else {
		if _, err := client.Rest().SetGlobalCommands(client.ApplicationID(), commands); err != nil {
			log.Error(err)
		}
	}
}
