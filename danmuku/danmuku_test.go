package danmuku

import (
	"testing"
	"time"
)

func TestDanmuku(t *testing.T) {
	danmukuRoom := NewDanmukuRoom(3258)
	if err := danmukuRoom.Start(); err != nil {
		t.Error(err)
		return
	}
	defer danmukuRoom.Stop()

	time.Sleep(30 * time.Second)
}
