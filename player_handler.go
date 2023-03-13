package main

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgolink/v2/disgolink"
	"github.com/disgoorg/disgolink/v2/lavalink"
	"github.com/disgoorg/log"
)

func (b *Bot) onPlayerPause(_ disgolink.Player, event lavalink.PlayerPauseEvent) {
	fmt.Printf("onPlayerPause: %v\n", event)
}

func (b *Bot) onPlayerResume(_ disgolink.Player, event lavalink.PlayerResumeEvent) {
	fmt.Printf("onPlayerResume: %v\n", event)
}

func (b *Bot) onTrackStart(_ disgolink.Player, event lavalink.TrackStartEvent) {
	b.updatePlayerMessage(event.GuildID())
	fmt.Printf("onTrackStart: %v\n", event)
}

func (b *Bot) onTrackEnd(player disgolink.Player, event lavalink.TrackEndEvent) {
	if !event.Reason.MayStartNext() {
		return
	}

	queue := b.Guilds.GetQueue(event.GuildID())
	var (
		nextTrack lavalink.Track
		ok        bool
	)
	event.Track.Info.Position = 0
	switch queue.Type {
	case QueueTypeNoRepeat:
		nextTrack, ok = queue.Next()

	case QueueTypeRepeatTrack:
		nextTrack = event.Track
		ok = true

	case QueueTypeRepeatQueue:
		queue.Add(event.Track)
		nextTrack, ok = queue.Next()
	}

	if !ok {
		b.updatePlayerMessage(event.GuildID())
		b.updateVoiceState(event.GuildID(), nil)
		return
	}
	if err := player.Update(context.TODO(), lavalink.WithTrack(nextTrack)); err != nil {
		log.Error("Failed to play next track: ", err)
	}
}

func (b *Bot) onTrackException(_ disgolink.Player, event lavalink.TrackExceptionEvent) {
	fmt.Printf("onTrackException: %v\n", event)
}

func (b *Bot) onTrackStuck(_ disgolink.Player, event lavalink.TrackStuckEvent) {
	fmt.Printf("onTrackStuck: %v\n", event)
}

func (b *Bot) onWebSocketClosed(_ disgolink.Player, event lavalink.WebSocketClosedEvent) {
	fmt.Printf("onWebSocketClosed: %v\n", event)
}
