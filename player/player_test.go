package player

import (
	"testing"
	"time"
)

func TestPlay(t *testing.T) {
	player := NewPlayer("/Users/zzz/1.flac")
	t.Log("start play")
	player.Play()

	for i := 0; i < 5; i++ {
		t.Logf("play for %d seconds\n", i)
		time.Sleep(time.Second * 1)
	}

	player.Stop()
	t.Log("stop play")

	t.Log("start play")
	player.ChangeLiveStreamUrl("/Users/zzz/2.flac")
	player.Play()
	for i := 0; i < 5; i++ {
		t.Logf("play for %d seconds\n", i)
		time.Sleep(time.Second * 1)
	}
	player.Stop()
	t.Log("stop play")
}
