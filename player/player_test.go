package player

import (
	"log"
	"testing"
	"time"
)

func TestPlay(t *testing.T) {
	if err := Init(); err != nil {
		t.Error(err)
	}
	defer DeInit()

	player := NewPlayer("/Users/zzz/1.mp4")
	go func() {
		err := <-player.ErrorChannel()
		if err != nil {
			log.Panic(err)
			t.Error(err)
		}
	}()
	t.Log("start play")
	player.Play()

	for i := 0; i < 5; i++ {
		t.Logf("play for %d seconds\n", i)
		time.Sleep(time.Second * 1)
	}

	player.Stop()
	t.Log("stop play")

	t.Log("start play")
	player.ChangeLiveStreamUrl("/Users/zzz/2.mp4")
	player.Play()
	for i := 0; i < 5; i++ {
		t.Logf("play for %d seconds\n", i)
		time.Sleep(time.Second * 1)
	}
	player.Stop()
	t.Log("stop play")
}
