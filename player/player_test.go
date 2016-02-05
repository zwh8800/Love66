package player

import (
	"testing"
	"time"
)

func TestPlay(t *testing.T) {
	if err := Init(); err != nil {
		t.Error(err)
	}
	defer DeInit()

	player := NewPlayer("/Users/zzz/1.mp4")
	t.Log("start play")
	player.Play()
	if err := player.Error(); err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		t.Logf("play for %d seconds\n", i)
		time.Sleep(time.Second * 1)
	}

	player.Stop()
	t.Log("stop play")
}
