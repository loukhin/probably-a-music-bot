package main

import (
	"math/rand"
	"time"

	"github.com/disgoorg/disgolink/v3/lavalink"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type QueueType string

const (
	QueueTypeNoRepeat    QueueType = "no_repeat"
	QueueTypeRepeatTrack QueueType = "repeat_track"
	QueueTypeRepeatQueue QueueType = "repeat_queue"
)

func (q QueueType) String() string {
	switch q {
	case QueueTypeNoRepeat:
		return "No Repeat"
	case QueueTypeRepeatTrack:
		return "Repeat Track"
	case QueueTypeRepeatQueue:
		return "Repeat Queue"
	default:
		return "unknown"
	}
}

type Queue struct {
	Length lavalink.Duration
	Tracks []lavalink.Track
	Type   QueueType
}

func (q *Queue) RecalculateDuration() {
	q.Length = 0
	for _, track := range q.Tracks {
		q.Length += track.Info.Length
	}
}

func (q *Queue) Shuffle() {
	rand.Shuffle(len(q.Tracks), func(i, j int) {
		q.Tracks[i], q.Tracks[j] = q.Tracks[j], q.Tracks[i]
	})
}

func (q *Queue) Add(tracks ...lavalink.Track) {
	q.Tracks = append(q.Tracks, tracks...)
	q.RecalculateDuration()
}

func (q *Queue) Next() (lavalink.Track, bool) {
	return q.Skip(1)
}

func (q *Queue) Skip(amount int) (lavalink.Track, bool) {
	if len(q.Tracks) == 0 {
		return lavalink.Track{}, false
	}
	if amount > len(q.Tracks) {
		amount = len(q.Tracks)
	}
	var nextTrack lavalink.Track
	nextTrack, q.Tracks = q.Tracks[amount-1], q.Tracks[amount:]
	q.RecalculateDuration()
	return nextTrack, true
}

func (q *Queue) Remove(index int) (removedTrack lavalink.Track, ok bool) {
	if len(q.Tracks) == 0 || index < 0 || index >= len(q.Tracks) {
		return lavalink.Track{}, false
	}
	removedTrack = q.Tracks[index]
	q.Tracks = append(q.Tracks[:index], q.Tracks[index+1:]...)
	q.RecalculateDuration()
	return removedTrack, true
}

func (q *Queue) Clear() {
	q.Tracks = make([]lavalink.Track, 0)
	q.Length = 0
}
