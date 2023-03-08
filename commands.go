package main

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/json"
)

var minVolume, maxVolume = 0, 100

var commands = []discord.ApplicationCommandCreate{
	discord.SlashCommandCreate{
		Name:        "pause",
		Description: "Pauses current track",
	},
	discord.SlashCommandCreate{
		Name:        "resume",
		Description: "Resume current track",
	},
	discord.SlashCommandCreate{
		Name:        "queue",
		Description: "Show queue",
	},
	discord.SlashCommandCreate{
		Name:        "clear",
		Description: "Clear queue",
	},
	discord.SlashCommandCreate{
		Name:        "remove",
		Description: "Remove item from queue",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "id",
				Description: "ID of the track in queue command",
				Required:    true,
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "play",
		Description: "Queue music to play",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "query",
				Description: "Search query or links (Youtube, Spotify, etc.)",
				Required:    true,
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "volume",
		Description: "Set volume",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionInt{
				Name:        "level",
				Description: "Level (0-100)",
				Required:    true,
				MinValue:    &minVolume,
				MaxValue:    &maxVolume,
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "stop",
		Description: "Stop music player",
	},
	discord.SlashCommandCreate{
		Name:        "skip",
		Description: "Skip current track",
	},
	discord.SlashCommandCreate{
		Name:        "loop",
		Description: "Loop current track",
	},
	discord.SlashCommandCreate{
		Name:                     "setup",
		Description:              "Setup dedicated music player channel",
		DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionAdministrator),
	},
}
