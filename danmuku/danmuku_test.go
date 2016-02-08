package danmuku

import (
	"log"
	"testing"
)

func TestDanmuku(t *testing.T) {
	danmukuRoom := NewDanmukuRoom(3258)
	if err := danmukuRoom.Start(); err != nil {
		t.Error(err)
		return
	}
	defer danmukuRoom.Stop()

	for {
		danmuku := danmukuRoom.PeekDanmuku()
		log.Printf("%s: %s\n", danmuku.User, danmuku.Content)
	}
}
